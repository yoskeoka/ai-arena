package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authModeDisabled = "disabled"
	authModeEnabled  = "enabled"

	authProviderGitHub = "github"

	sessionCookieName           = "arena_session"
	pendingAuthCookieName       = "arena_github_oauth_pending"
	defaultSessionLifetime      = 24 * time.Hour
	defaultSignupInviteLifetime = 24 * time.Hour
)

var (
	// ErrAuthDisabled indicates that auth-dependent behavior was requested while auth is not configured.
	ErrAuthDisabled = errors.New("authentication is disabled")
	// ErrAuthenticationNeeded indicates that no valid authenticated session is present.
	ErrAuthenticationNeeded = errors.New("authentication required")
	// ErrOperatorRoleNeeded indicates that the authenticated account lacks the operator role.
	ErrOperatorRoleNeeded = errors.New("operator role required")
	// ErrSignupInviteRequired indicates that first-time signup requires an invite token.
	ErrSignupInviteRequired = errors.New("signup invite is required")
	// ErrInvalidSignupInvite indicates that the supplied invite token is invalid, expired, or already claimed.
	ErrInvalidSignupInvite = errors.New("signup invite is invalid or expired")
	// ErrIdentityLookupFailed indicates that provider-specific identity resolution failed after token exchange.
	ErrIdentityLookupFailed = errors.New("identity lookup failed")
)

// AuthConfig describes the runtime configuration required for GitHub-backed auth.
type AuthConfig struct {
	GitHubClientID       string
	GitHubClientSecret   string
	AllowedReturnOrigins []string
	SessionTTL           time.Duration
	CookieSigningSecret  string
}

// AuthStore persists account identities, sessions, and signup invites.
type AuthStore interface {
	ResolveIdentityLogin(ctx context.Context, identity AuthIdentity, inviteToken string, now time.Time) (AuthPrincipal, error)
	CreateSession(ctx context.Context, accountID string, expiresAt time.Time) (string, error)
	GetSession(ctx context.Context, sessionToken string, now time.Time) (AuthPrincipal, error)
	DeleteSession(ctx context.Context, sessionToken string) error
	CreateSignupInvite(ctx context.Context, role string, expiresAt time.Time) (SignupInviteRecord, error)
}

// AuthIdentity is the provider-normalized identity claim used for account linking.
type AuthIdentity struct {
	Provider string
	Subject  string
	Login    string
	Email    string
}

// OAuthIdentityProvider exchanges provider authorization codes and resolves normalized identity claims.
type OAuthIdentityProvider interface {
	AuthorizationURL(redirectURI string, state string) string
	ExchangeIdentity(ctx context.Context, code string, redirectURI string) (AuthIdentity, error)
}

// AuthPrincipal is the authenticated account identity returned to operator clients.
type AuthPrincipal struct {
	AccountID     string   `json:"account_id"`
	Provider      string   `json:"provider"`
	ProviderLogin string   `json:"provider_login"`
	ProviderEmail string   `json:"provider_email,omitempty"`
	Roles         []string `json:"roles"`
}

// SignupInviteRequest describes the operator API payload for creating an invite.
type SignupInviteRequest struct {
	Role string `json:"role"`
	TTL  string `json:"ttl,omitempty"`
}

// SignupInviteResponse returns the raw invite token and a frontend-relative login URL.
type SignupInviteResponse struct {
	SignupInviteRecord
	InviteURL string `json:"invite_url"`
}

// SessionStatusResponse reports whether the current browser session is authenticated.
type SessionStatusResponse struct {
	AuthMode      string         `json:"auth_mode"`
	Authenticated bool           `json:"authenticated"`
	Principal     *AuthPrincipal `json:"principal,omitempty"`
}

// AuthService coordinates GitHub OAuth, session issuance, and operator access control.
type AuthService struct {
	store                AuthStore
	github               OAuthIdentityProvider
	allowedReturnOrigins map[string]struct{}
	sessionTTL           time.Duration
	cookieSigningSecret  []byte
	now                  func() time.Time
}

type pendingAuth struct {
	Provider    string    `json:"provider"`
	StateNonce  string    `json:"state_nonce"`
	ReturnTo    string    `json:"return_to"`
	InviteToken string    `json:"invite_token,omitempty"`
	IssuedAt    time.Time `json:"issued_at"`
}

// NewAuthService constructs an auth service from the configured store and GitHub client.
func NewAuthService(cfg AuthConfig, store AuthStore, github OAuthIdentityProvider) (*AuthService, error) {
	if store == nil {
		return nil, fmt.Errorf("service: auth store is required")
	}
	if github == nil {
		return nil, fmt.Errorf("service: github auth provider is required")
	}
	if strings.TrimSpace(cfg.GitHubClientID) == "" || strings.TrimSpace(cfg.GitHubClientSecret) == "" {
		return nil, fmt.Errorf("service: github oauth client id and secret are required")
	}
	if len(cfg.AllowedReturnOrigins) == 0 {
		cfg.AllowedReturnOrigins = []string{
			"http://localhost:4173",
			"http://127.0.0.1:4173",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"https://staging.ai-arena.pages.dev",
			"https://ai-arena.pages.dev",
		}
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = defaultSessionLifetime
	}
	signingSecret := strings.TrimSpace(cfg.CookieSigningSecret)
	if signingSecret == "" {
		signingSecret = strings.TrimSpace(cfg.GitHubClientSecret)
	}
	allowed := make(map[string]struct{}, len(cfg.AllowedReturnOrigins))
	for _, origin := range cfg.AllowedReturnOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	return &AuthService{
		store:                store,
		github:               github,
		allowedReturnOrigins: allowed,
		sessionTTL:           cfg.SessionTTL,
		cookieSigningSecret:  []byte(signingSecret),
		now:                  time.Now,
	}, nil
}

// SessionStatus returns the current session principal when the browser is authenticated.
func (a *AuthService) SessionStatus(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		writeJSON(w, http.StatusOK, SessionStatusResponse{AuthMode: authModeDisabled, Authenticated: false})
		return
	}
	principal, err := a.sessionPrincipal(r.Context(), r)
	if err != nil {
		writeJSON(w, http.StatusOK, SessionStatusResponse{AuthMode: authModeEnabled, Authenticated: false})
		return
	}
	writeJSON(w, http.StatusOK, SessionStatusResponse{
		AuthMode:      authModeEnabled,
		Authenticated: true,
		Principal:     &principal,
	})
}

// CreateSignupInvite creates a single-use invite for one of the durable roles.
func (a *AuthService) CreateSignupInvite(ctx context.Context, role string, ttl time.Duration) (SignupInviteRecord, error) {
	if a == nil {
		return SignupInviteRecord{}, ErrAuthDisabled
	}
	role = strings.TrimSpace(role)
	if err := validateSignupInviteRole(role); err != nil {
		return SignupInviteRecord{}, err
	}
	if ttl < 0 {
		return SignupInviteRecord{}, fmt.Errorf("service: invite ttl must not be negative")
	}
	if ttl == 0 {
		ttl = defaultSignupInviteLifetime
	}
	return a.store.CreateSignupInvite(ctx, role, a.now().UTC().Add(ttl))
}

// GitHubLogin starts the GitHub OAuth authorization-code flow.
func (a *AuthService) GitHubLogin(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		writeError(w, http.StatusServiceUnavailable, ErrAuthDisabled)
		return
	}
	returnTo, err := a.validatedReturnTo(r.URL.Query().Get("return_to"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	nonce, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	pending := pendingAuth{
		Provider:    authProviderGitHub,
		StateNonce:  nonce,
		ReturnTo:    returnTo,
		InviteToken: strings.TrimSpace(r.URL.Query().Get("invite_token")),
		IssuedAt:    a.now().UTC(),
	}
	if err := a.setSignedCookie(w, r, pendingAuthCookieName, pending, 10*time.Minute); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	redirectURL := a.github.AuthorizationURL(callbackURLForProvider(r, authProviderGitHub), nonce)
	// #nosec G710 -- the provider returns a GitHub authorize URL; return_to validation happens before this redirect.
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// GitHubCallback completes the GitHub OAuth flow and issues the operator session cookie.
func (a *AuthService) GitHubCallback(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		writeError(w, http.StatusServiceUnavailable, ErrAuthDisabled)
		return
	}
	pending, err := a.pendingAuth(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	queryState := strings.TrimSpace(r.URL.Query().Get("state"))
	if queryState == "" || !hmac.Equal([]byte(queryState), []byte(pending.StateNonce)) {
		a.clearCookie(w, r, pendingAuthCookieName)
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "oauth_state_mismatch")
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "missing_code")
		return
	}
	if pending.Provider != authProviderGitHub {
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "login_failed")
		return
	}
	identity, err := a.github.ExchangeIdentity(r.Context(), code, callbackURLForProvider(r, pending.Provider))
	if err != nil {
		if errors.Is(err, ErrIdentityLookupFailed) {
			a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "github_profile_failed")
			return
		}
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "token_exchange_failed")
		return
	}
	principal, err := a.store.ResolveIdentityLogin(r.Context(), identity, pending.InviteToken, a.now().UTC())
	if err != nil {
		loginError := "login_failed"
		switch {
		case errors.Is(err, ErrSignupInviteRequired):
			loginError = "signup_invite_required"
		case errors.Is(err, ErrInvalidSignupInvite):
			loginError = "signup_invite_invalid"
		}
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, loginError)
		return
	}
	sessionToken, err := a.store.CreateSession(r.Context(), principal.AccountID, a.now().UTC().Add(a.sessionTTL))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	a.clearCookie(w, r, pendingAuthCookieName)
	a.setSessionCookie(w, r, sessionToken, a.sessionTTL)
	http.Redirect(w, r, pending.ReturnTo, http.StatusFound)
}

// Logout revokes the current persisted session and clears the browser cookie.
func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		if _, ok := a.allowedReturnOrigins[origin]; !ok {
			writeError(w, http.StatusForbidden, fmt.Errorf("origin is not allowed"))
			return
		}
	}
	sessionCookie, err := r.Cookie(sessionCookieName)
	if err == nil && strings.TrimSpace(sessionCookie.Value) != "" {
		_ = a.store.DeleteSession(r.Context(), sessionCookie.Value)
	}
	a.clearCookie(w, r, sessionCookieName)
	w.WriteHeader(http.StatusNoContent)
}

// RequireOperator wraps a handler with authenticated-operator access control.
func (a *AuthService) RequireOperator(next http.Handler) http.Handler {
	if a == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, err := a.sessionPrincipal(r.Context(), r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, ErrAuthenticationNeeded)
			return
		}
		if !hasRole(principal.Roles, "operator") {
			writeError(w, http.StatusForbidden, ErrOperatorRoleNeeded)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *AuthService) sessionPrincipal(ctx context.Context, r *http.Request) (AuthPrincipal, error) {
	if a == nil {
		return AuthPrincipal{}, ErrAuthDisabled
	}
	sessionCookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return AuthPrincipal{}, ErrAuthenticationNeeded
	}
	token := strings.TrimSpace(sessionCookie.Value)
	if token == "" {
		return AuthPrincipal{}, ErrAuthenticationNeeded
	}
	return a.store.GetSession(ctx, token, a.now().UTC())
}

func (a *AuthService) validatedReturnTo(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid return_to")
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("return_to must be an absolute URL")
	}
	origin := parsed.Scheme + "://" + parsed.Host
	if _, ok := a.allowedReturnOrigins[origin]; !ok {
		return "", fmt.Errorf("return_to origin is not allowed")
	}
	parsed.Fragment = ""
	if parsed.Path == "" {
		parsed.Path = "/operator"
	}
	return parsed.String(), nil
}

func (a *AuthService) pendingAuth(r *http.Request) (pendingAuth, error) {
	cookie, err := r.Cookie(pendingAuthCookieName)
	if err != nil {
		return pendingAuth{}, err
	}
	var pending pendingAuth
	if err := a.decodeSignedValue(cookie.Value, &pending); err != nil {
		return pendingAuth{}, err
	}
	if strings.TrimSpace(pending.Provider) == "" {
		pending.Provider = authProviderGitHub
	}
	return pending, nil
}

func (a *AuthService) redirectLoginError(w http.ResponseWriter, r *http.Request, returnTo string, inviteToken string, errorCode string) {
	a.clearCookie(w, r, pendingAuthCookieName)
	target, err := url.Parse(returnTo)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	loginURL := &url.URL{
		Scheme: target.Scheme,
		Host:   target.Host,
		Path:   "/login",
	}
	params := url.Values{}
	params.Set("return_to", returnTo)
	if strings.TrimSpace(inviteToken) != "" {
		params.Set("invite_token", inviteToken)
	}
	params.Set("error", errorCode)
	loginURL.RawQuery = params.Encode()
	http.Redirect(w, r, loginURL.String(), http.StatusFound)
}

func validateSignupInviteRole(role string) error {
	switch strings.TrimSpace(role) {
	case "participant", "developer", "operator":
		return nil
	default:
		return fmt.Errorf("service: invite role must be participant, developer, or operator")
	}
}

func signupInviteURL(inviteToken string) string {
	params := url.Values{}
	params.Set("invite_token", strings.TrimSpace(inviteToken))
	return (&url.URL{Path: "/login", RawQuery: params.Encode()}).String()
}

func (a *AuthService) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
	// #nosec G124 -- localhost manual auth verification requires non-Secure cookies on plain HTTP; HTTPS requests still receive Secure cookies.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPSRequest(r),
		SameSite: sessionCookieSameSite(r),
		MaxAge:   int(ttl.Seconds()),
	})
}

func (a *AuthService) setSignedCookie(w http.ResponseWriter, r *http.Request, name string, payload any, ttl time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	encoded := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, a.cookieSigningSecret)
	if _, err := io.WriteString(mac, encoded); err != nil {
		return err
	}
	value := encoded + "." + hex.EncodeToString(mac.Sum(nil))
	// #nosec G124 -- the pending OAuth cookie must remain usable on localhost HTTP during manual development; production HTTPS still receives Secure cookies.
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPSRequest(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
	return nil
}

func (a *AuthService) decodeSignedValue(raw string, dest any) error {
	parts := strings.Split(raw, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid signed cookie")
	}
	encoded := parts[0]
	sig, err := hex.DecodeString(parts[1])
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, a.cookieSigningSecret)
	if _, err := io.WriteString(mac, encoded); err != nil {
		return err
	}
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return fmt.Errorf("signed cookie verification failed")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}

func (a *AuthService) clearCookie(w http.ResponseWriter, r *http.Request, name string) {
	// #nosec G124 -- cookie clearing must match the same localhost HTTP attributes used when issuing local development cookies.
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPSRequest(r),
		SameSite: sessionCookieSameSite(r),
		MaxAge:   -1,
	})
}

func callbackURLForProvider(r *http.Request, provider string) string {
	scheme := "https"
	if !isHTTPSRequest(r) {
		scheme = "http"
	}
	host := r.Host
	if !isHTTPSRequest(r) {
		host = strings.Replace(host, "127.0.0.1", "localhost", 1)
	}
	return scheme + "://" + host + "/auth/" + provider + "/callback"
}

func isHTTPSRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func sessionCookieSameSite(r *http.Request) http.SameSite {
	if isHTTPSRequest(r) {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func randomToken(numBytes int) (string, error) {
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func hasRole(roles []string, want string) bool {
	for _, role := range roles {
		if role == want {
			return true
		}
	}
	return false
}
