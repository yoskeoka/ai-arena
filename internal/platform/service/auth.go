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

	sessionCookieName      = "arena_session"
	pendingAuthCookieName  = "arena_github_oauth_pending"
	defaultSessionLifetime = 24 * time.Hour
)

var (
	ErrAuthDisabled         = errors.New("authentication is disabled")
	ErrAuthenticationNeeded = errors.New("authentication required")
	ErrOperatorRoleNeeded   = errors.New("operator role required")
	ErrSignupInviteRequired = errors.New("signup invite is required")
	ErrInvalidSignupInvite  = errors.New("signup invite is invalid or expired")
)

type AuthConfig struct {
	GitHubClientID       string
	GitHubClientSecret   string
	AllowedReturnOrigins []string
	SessionTTL           time.Duration
	CookieSigningSecret  string
}

type AuthStore interface {
	ResolveGitHubLogin(ctx context.Context, profile GitHubUserProfile, inviteToken string, now time.Time) (AuthPrincipal, error)
	CreateSession(ctx context.Context, accountID string, expiresAt time.Time) (string, error)
	GetSession(ctx context.Context, sessionToken string, now time.Time) (AuthPrincipal, error)
	DeleteSession(ctx context.Context, sessionToken string) error
}

type GitHubOAuthClient interface {
	ExchangeCode(ctx context.Context, code string, redirectURI string) (string, error)
	FetchUser(ctx context.Context, accessToken string) (GitHubUserProfile, error)
}

type GitHubUserProfile struct {
	Subject string
	Login   string
	Email   string
}

type AuthPrincipal struct {
	AccountID     string   `json:"account_id"`
	Provider      string   `json:"provider"`
	ProviderLogin string   `json:"provider_login"`
	ProviderEmail string   `json:"provider_email,omitempty"`
	Roles         []string `json:"roles"`
}

type SessionStatusResponse struct {
	AuthMode      string         `json:"auth_mode"`
	Authenticated bool           `json:"authenticated"`
	Principal     *AuthPrincipal `json:"principal,omitempty"`
}

type AuthService struct {
	store                AuthStore
	github               GitHubOAuthClient
	clientID             string
	clientSecret         string
	allowedReturnOrigins map[string]struct{}
	sessionTTL           time.Duration
	cookieSigningSecret  []byte
	now                  func() time.Time
}

type pendingGitHubAuth struct {
	StateNonce  string    `json:"state_nonce"`
	ReturnTo    string    `json:"return_to"`
	InviteToken string    `json:"invite_token,omitempty"`
	IssuedAt    time.Time `json:"issued_at"`
}

func NewAuthService(cfg AuthConfig, store AuthStore, github GitHubOAuthClient) (*AuthService, error) {
	if store == nil {
		return nil, fmt.Errorf("service: auth store is required")
	}
	if github == nil {
		return nil, fmt.Errorf("service: github oauth client is required")
	}
	if strings.TrimSpace(cfg.GitHubClientID) == "" || strings.TrimSpace(cfg.GitHubClientSecret) == "" {
		return nil, fmt.Errorf("service: github oauth client id and secret are required")
	}
	if len(cfg.AllowedReturnOrigins) == 0 {
		cfg.AllowedReturnOrigins = []string{
			"http://localhost:4173",
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
		clientID:             strings.TrimSpace(cfg.GitHubClientID),
		clientSecret:         strings.TrimSpace(cfg.GitHubClientSecret),
		allowedReturnOrigins: allowed,
		sessionTTL:           cfg.SessionTTL,
		cookieSigningSecret:  []byte(signingSecret),
		now:                  time.Now,
	}, nil
}

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
	pending := pendingGitHubAuth{
		StateNonce:  nonce,
		ReturnTo:    returnTo,
		InviteToken: strings.TrimSpace(r.URL.Query().Get("invite_token")),
		IssuedAt:    a.now().UTC(),
	}
	if err := a.setSignedCookie(w, r, pendingAuthCookieName, pending, 10*time.Minute); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	redirectURI := callbackURL(r)
	target := (&url.URL{
		Scheme: "https",
		Host:   "github.com",
		Path:   "/login/oauth/authorize",
	}).String()
	params := url.Values{}
	params.Set("client_id", a.clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", nonce)
	params.Set("scope", "read:user")
	http.Redirect(w, r, target+"?"+params.Encode(), http.StatusFound)
}

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
		writeError(w, http.StatusBadRequest, fmt.Errorf("oauth state mismatch"))
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	if code == "" {
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "missing_code")
		return
	}
	accessToken, err := a.github.ExchangeCode(r.Context(), code, callbackURL(r))
	if err != nil {
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "token_exchange_failed")
		return
	}
	profile, err := a.github.FetchUser(r.Context(), accessToken)
	if err != nil {
		a.redirectLoginError(w, r, pending.ReturnTo, pending.InviteToken, "github_profile_failed")
		return
	}
	principal, err := a.store.ResolveGitHubLogin(r.Context(), profile, pending.InviteToken, a.now().UTC())
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

func (a *AuthService) Logout(w http.ResponseWriter, r *http.Request) {
	if a == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	sessionCookie, err := r.Cookie(sessionCookieName)
	if err == nil && strings.TrimSpace(sessionCookie.Value) != "" {
		_ = a.store.DeleteSession(r.Context(), sessionCookie.Value)
	}
	a.clearCookie(w, r, sessionCookieName)
	w.WriteHeader(http.StatusNoContent)
}

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

func (a *AuthService) pendingAuth(r *http.Request) (pendingGitHubAuth, error) {
	cookie, err := r.Cookie(pendingAuthCookieName)
	if err != nil {
		return pendingGitHubAuth{}, err
	}
	var pending pendingGitHubAuth
	if err := a.decodeSignedValue(cookie.Value, &pending); err != nil {
		return pendingGitHubAuth{}, err
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

func (a *AuthService) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, ttl time.Duration) {
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

func callbackURL(r *http.Request) string {
	scheme := "https"
	if !isHTTPSRequest(r) {
		scheme = "http"
	}
	host := r.Host
	if !isHTTPSRequest(r) {
		host = strings.Replace(host, "127.0.0.1", "localhost", 1)
	}
	return scheme + "://" + host + "/auth/github/callback"
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
