package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestAuthServiceGitHubLoginCallbackAndSessionStatus(t *testing.T) {
	t.Parallel()

	store := &memoryAuthStore{
		invites: map[string]string{"invite-1": "operator"},
	}
	github := fakeGitHubAuthProvider{
		identity: AuthIdentity{
			Provider: authProviderGitHub,
			Subject:  "12345",
			Login:    "arena-dev",
			Email:    "arena-dev@example.com",
		},
	}
	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:       "client-id",
		GitHubClientSecret:   "client-secret",
		AllowedReturnOrigins: []string{"http://localhost:4173"},
	}, store, github)
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	loginReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/login?return_to=http://localhost:4173/operator&invite_token=invite-1", nil)
	loginResp := httptest.NewRecorder()
	auth.GitHubLogin(loginResp, loginReq)
	if loginResp.Code != http.StatusFound {
		t.Fatalf("GitHubLogin status = %d, want %d", loginResp.Code, http.StatusFound)
	}
	redirectURL, err := url.Parse(loginResp.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Parse(redirect) error = %v", err)
	}
	if got := redirectURL.Host; got != "github.com" {
		t.Fatalf("redirect host = %q, want github.com", got)
	}
	state := redirectURL.Query().Get("state")
	if state == "" {
		t.Fatal("redirect state = empty, want oauth state")
	}
	pendingCookie := loginResp.Result().Cookies()[0]

	callbackReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/callback?code=code-1&state="+state, nil)
	callbackReq.AddCookie(pendingCookie)
	callbackResp := httptest.NewRecorder()
	auth.GitHubCallback(callbackResp, callbackReq)
	if callbackResp.Code != http.StatusFound {
		t.Fatalf("GitHubCallback status = %d, want %d", callbackResp.Code, http.StatusFound)
	}
	if got := callbackResp.Header().Get("Location"); got != "http://localhost:4173/operator" {
		t.Fatalf("callback return_to = %q, want operator route", got)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range callbackResp.Result().Cookies() {
		if cookie.Name == sessionCookieName {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie = nil, want issued auth session")
	}

	sessionReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/session", nil)
	sessionReq.AddCookie(sessionCookie)
	sessionResp := httptest.NewRecorder()
	auth.SessionStatus(sessionResp, sessionReq)
	if sessionResp.Code != http.StatusOK {
		t.Fatalf("SessionStatus status = %d, want %d", sessionResp.Code, http.StatusOK)
	}
	var payload SessionStatusResponse
	if err := json.Unmarshal(sessionResp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(SessionStatus) error = %v", err)
	}
	if !payload.Authenticated {
		t.Fatalf("SessionStatus payload = %+v, want authenticated session", payload)
	}
	if payload.Principal == nil || payload.Principal.ProviderLogin != "arena-dev" {
		t.Fatalf("SessionStatus payload = %+v, want provider login arena-dev", payload)
	}
}

func TestAuthServiceGitHubCallbackStateMismatchRedirectsToLoginAndClearsPendingCookie(t *testing.T) {
	t.Parallel()

	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:       "client-id",
		GitHubClientSecret:   "client-secret",
		AllowedReturnOrigins: []string{"http://localhost:4173"},
	}, &memoryAuthStore{}, fakeGitHubAuthProvider{})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	loginReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/login?return_to=http://localhost:4173/operator&invite_token=invite-1", nil)
	loginResp := httptest.NewRecorder()
	auth.GitHubLogin(loginResp, loginReq)
	if loginResp.Code != http.StatusFound {
		t.Fatalf("GitHubLogin status = %d, want %d", loginResp.Code, http.StatusFound)
	}
	pendingCookie := loginResp.Result().Cookies()[0]

	callbackReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/callback?code=code-1&state=wrong-state", nil)
	callbackReq.AddCookie(pendingCookie)
	callbackResp := httptest.NewRecorder()
	auth.GitHubCallback(callbackResp, callbackReq)
	if callbackResp.Code != http.StatusFound {
		t.Fatalf("GitHubCallback status = %d, want %d", callbackResp.Code, http.StatusFound)
	}
	redirectURL, err := url.Parse(callbackResp.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Parse(state mismatch redirect) error = %v", err)
	}
	if got := redirectURL.Scheme + "://" + redirectURL.Host + redirectURL.Path; got != "http://localhost:4173/login" {
		t.Fatalf("state mismatch redirect path = %q, want login route", got)
	}
	if got := redirectURL.Query().Get("return_to"); got != "http://localhost:4173/operator" {
		t.Fatalf("state mismatch return_to = %q, want operator route", got)
	}
	if got := redirectURL.Query().Get("invite_token"); got != "invite-1" {
		t.Fatalf("state mismatch invite_token = %q, want invite-1", got)
	}
	if got := redirectURL.Query().Get("error"); got != "oauth_state_mismatch" {
		t.Fatalf("state mismatch error = %q, want oauth_state_mismatch", got)
	}

	var clearedPendingCookie *http.Cookie
	for _, cookie := range callbackResp.Result().Cookies() {
		if cookie.Name == pendingAuthCookieName {
			clearedPendingCookie = cookie
			break
		}
	}
	if clearedPendingCookie == nil {
		t.Fatal("cleared pending cookie = nil, want cookie clearing on state mismatch")
	}
	if clearedPendingCookie.MaxAge != -1 {
		t.Fatalf("cleared pending cookie MaxAge = %d, want -1", clearedPendingCookie.MaxAge)
	}
}

func TestAuthServiceGitHubCallbackProfileLookupFailureRedirectsWithExistingErrorCode(t *testing.T) {
	t.Parallel()

	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:       "client-id",
		GitHubClientSecret:   "client-secret",
		AllowedReturnOrigins: []string{"http://localhost:4173"},
	}, &memoryAuthStore{}, fakeGitHubAuthProvider{exchangeErr: ErrIdentityLookupFailed})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	loginReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/login?return_to=http://localhost:4173/operator", nil)
	loginResp := httptest.NewRecorder()
	auth.GitHubLogin(loginResp, loginReq)
	pendingCookie := loginResp.Result().Cookies()[0]
	redirectURL, err := url.Parse(loginResp.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Parse(login redirect) error = %v", err)
	}

	callbackReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/callback?code=code-1&state="+redirectURL.Query().Get("state"), nil)
	callbackReq.AddCookie(pendingCookie)
	callbackResp := httptest.NewRecorder()
	auth.GitHubCallback(callbackResp, callbackReq)
	if callbackResp.Code != http.StatusFound {
		t.Fatalf("GitHubCallback status = %d, want %d", callbackResp.Code, http.StatusFound)
	}
	got, err := url.Parse(callbackResp.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Parse(callback redirect) error = %v", err)
	}
	if got.Query().Get("error") != "github_profile_failed" {
		t.Fatalf("callback error = %q, want github_profile_failed", got.Query().Get("error"))
	}
}

func TestAuthServiceRequireOperatorRejectsAnonymousAndNonOperator(t *testing.T) {
	t.Parallel()

	store := &memoryAuthStore{
		identities: map[string]AuthPrincipal{
			authIdentityKey(AuthIdentity{Provider: authProviderGitHub, Subject: "12345"}): {
				AccountID:     "account-operator",
				Provider:      "github",
				ProviderLogin: "operator-dev",
				Roles:         []string{"operator"},
			},
			authIdentityKey(AuthIdentity{Provider: authProviderGitHub, Subject: "54321"}): {
				AccountID:     "account-participant",
				Provider:      "github",
				ProviderLogin: "participant-dev",
				Roles:         []string{"participant"},
			},
		},
	}
	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:       "client-id",
		GitHubClientSecret:   "client-secret",
		AllowedReturnOrigins: []string{"http://localhost:4173"},
	}, store, fakeGitHubAuthProvider{})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	protected := auth.RequireOperator(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	anonymousResp := httptest.NewRecorder()
	protected.ServeHTTP(anonymousResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/api/v1/matches/active", nil))
	if anonymousResp.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want %d", anonymousResp.Code, http.StatusUnauthorized)
	}

	participantToken, err := store.CreateSession(context.Background(), "account-participant", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession(participant) error = %v", err)
	}
	participantReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/api/v1/matches/active", nil)
	participantReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: participantToken})
	participantResp := httptest.NewRecorder()
	protected.ServeHTTP(participantResp, participantReq)
	if participantResp.Code != http.StatusForbidden {
		t.Fatalf("participant status = %d, want %d", participantResp.Code, http.StatusForbidden)
	}

	operatorToken, err := store.CreateSession(context.Background(), "account-operator", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession(operator) error = %v", err)
	}
	operatorReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/api/v1/matches/active", nil)
	operatorReq.AddCookie(&http.Cookie{Name: sessionCookieName, Value: operatorToken})
	operatorResp := httptest.NewRecorder()
	protected.ServeHTTP(operatorResp, operatorReq)
	if operatorResp.Code != http.StatusNoContent {
		t.Fatalf("operator status = %d, want %d", operatorResp.Code, http.StatusNoContent)
	}
}

func TestAuthServiceGitHubLoginAllowsDefaultLocalReturnOrigin(t *testing.T) {
	t.Parallel()

	auth, err := NewAuthService(AuthConfig{
		GitHubClientID:     "client-id",
		GitHubClientSecret: "client-secret",
	}, &memoryAuthStore{}, fakeGitHubAuthProvider{})
	if err != nil {
		t.Fatalf("NewAuthService() error = %v", err)
	}

	loginReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://127.0.0.1:10000/auth/github/login?return_to=http://127.0.0.1:5173/operator", nil)
	loginResp := httptest.NewRecorder()
	auth.GitHubLogin(loginResp, loginReq)
	if loginResp.Code != http.StatusFound {
		t.Fatalf("GitHubLogin status = %d, want %d", loginResp.Code, http.StatusFound)
	}
}

func TestNormalizedOIDCIdentityUsesPreferredUsernameAndNameFallback(t *testing.T) {
	t.Parallel()

	identity, err := normalizedOIDCIdentity(context.Background(), fakeOIDCVerifier{
		claims: map[string]any{
			"sub":                "subject-1",
			"email":              "oidc@example.com",
			"preferred_username": "oidc-user",
			"name":               "OIDC User",
		},
	}, "google", "raw-token")
	if err != nil {
		t.Fatalf("normalizedOIDCIdentity() error = %v", err)
	}
	if identity.Provider != "google" || identity.Subject != "subject-1" || identity.Login != "oidc-user" {
		t.Fatalf("normalizedOIDCIdentity() = %+v, want normalized preferred username identity", identity)
	}

	fallbackIdentity, err := normalizedOIDCIdentity(context.Background(), fakeOIDCVerifier{
		claims: map[string]any{
			"sub":   "subject-2",
			"name":  "OIDC Display",
			"email": "display@example.com",
		},
	}, "google", "raw-token")
	if err != nil {
		t.Fatalf("normalizedOIDCIdentity() fallback error = %v", err)
	}
	if fallbackIdentity.Login != "OIDC Display" {
		t.Fatalf("fallback login = %q, want OIDC Display", fallbackIdentity.Login)
	}
}

type fakeGitHubAuthProvider struct {
	identity    AuthIdentity
	exchangeErr error
}

func (f fakeGitHubAuthProvider) AuthorizationURL(redirectURI string, state string) string {
	return "https://github.com/login/oauth/authorize?redirect_uri=" + url.QueryEscape(redirectURI) + "&state=" + url.QueryEscape(state)
}

func (f fakeGitHubAuthProvider) ExchangeIdentity(_ context.Context, _ string, _ string) (AuthIdentity, error) {
	if f.exchangeErr != nil {
		return AuthIdentity{}, f.exchangeErr
	}
	return f.identity, nil
}

type fakeOIDCVerifier struct {
	claims map[string]any
	err    error
}

func (f fakeOIDCVerifier) Verify(_ context.Context, _ string) (OIDCVerifiedClaims, error) {
	if f.err != nil {
		return nil, f.err
	}
	return fakeOIDCVerifiedClaims{claims: f.claims}, nil
}

type fakeOIDCVerifiedClaims struct {
	claims map[string]any
}

func (f fakeOIDCVerifiedClaims) Claims(dest any) error {
	payload, err := json.Marshal(f.claims)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, dest)
}

type memoryAuthStore struct {
	identities map[string]AuthPrincipal
	invites    map[string]string
	sessions   map[string]AuthPrincipal
}

func (s *memoryAuthStore) ResolveIdentityLogin(_ context.Context, identity AuthIdentity, inviteToken string, _ time.Time) (AuthPrincipal, error) {
	if s.identities == nil {
		s.identities = map[string]AuthPrincipal{}
	}
	identity = normalizedAuthIdentity(identity)
	if principal, ok := s.identities[authIdentityKey(identity)]; ok {
		return principal, nil
	}
	if principal, ok := s.identities[identity.Subject]; ok {
		return principal, nil
	}
	if role := s.invites[inviteToken]; role != "" {
		principal := AuthPrincipal{
			AccountID:     "account-" + identity.Subject,
			Provider:      identity.Provider,
			ProviderLogin: identity.Login,
			ProviderEmail: identity.Email,
			Roles:         []string{role},
		}
		s.identities[authIdentityKey(identity)] = principal
		delete(s.invites, inviteToken)
		return principal, nil
	}
	if strings.TrimSpace(inviteToken) == "" {
		return AuthPrincipal{}, ErrSignupInviteRequired
	}
	return AuthPrincipal{}, ErrInvalidSignupInvite
}

func (s *memoryAuthStore) CreateSession(_ context.Context, accountID string, _ time.Time) (string, error) {
	if s.sessions == nil {
		s.sessions = map[string]AuthPrincipal{}
	}
	for _, principal := range s.identities {
		if principal.AccountID == accountID {
			token := "session-" + accountID
			s.sessions[token] = principal
			return token, nil
		}
	}
	return "", ErrAuthenticationNeeded
}

func (s *memoryAuthStore) GetSession(_ context.Context, sessionToken string, _ time.Time) (AuthPrincipal, error) {
	if principal, ok := s.sessions[sessionToken]; ok {
		return principal, nil
	}
	return AuthPrincipal{}, ErrAuthenticationNeeded
}

func (s *memoryAuthStore) DeleteSession(_ context.Context, sessionToken string) error {
	delete(s.sessions, sessionToken)
	return nil
}

func authIdentityKey(identity AuthIdentity) string {
	return identity.Provider + ":" + identity.Subject
}
