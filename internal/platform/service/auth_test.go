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
	github := fakeGitHubOAuthClient{
		profile: GitHubUserProfile{
			Subject: "12345",
			Login:   "arena-dev",
			Email:   "arena-dev@example.com",
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
	}, &memoryAuthStore{}, fakeGitHubOAuthClient{})
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

func TestAuthServiceRequireOperatorRejectsAnonymousAndNonOperator(t *testing.T) {
	t.Parallel()

	store := &memoryAuthStore{
		identities: map[string]AuthPrincipal{
			"12345": {
				AccountID:     "account-operator",
				Provider:      "github",
				ProviderLogin: "operator-dev",
				Roles:         []string{"operator"},
			},
			"54321": {
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
	}, store, fakeGitHubOAuthClient{})
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
	}, &memoryAuthStore{}, fakeGitHubOAuthClient{})
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

type fakeGitHubOAuthClient struct {
	profile GitHubUserProfile
}

func (f fakeGitHubOAuthClient) ExchangeCode(_ context.Context, code string, redirectURI string) (string, error) {
	return "access-token-" + code + "-" + redirectURI, nil
}

func (f fakeGitHubOAuthClient) FetchUser(_ context.Context, _ string) (GitHubUserProfile, error) {
	return f.profile, nil
}

type memoryAuthStore struct {
	identities map[string]AuthPrincipal
	invites    map[string]string
	sessions   map[string]AuthPrincipal
}

func (s *memoryAuthStore) ResolveGitHubLogin(_ context.Context, profile GitHubUserProfile, inviteToken string, _ time.Time) (AuthPrincipal, error) {
	if s.identities == nil {
		s.identities = map[string]AuthPrincipal{}
	}
	if principal, ok := s.identities[profile.Subject]; ok {
		return principal, nil
	}
	if role := s.invites[inviteToken]; role != "" {
		principal := AuthPrincipal{
			AccountID:     "account-" + profile.Subject,
			Provider:      "github",
			ProviderLogin: profile.Login,
			ProviderEmail: profile.Email,
			Roles:         []string{role},
		}
		s.identities[profile.Subject] = principal
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
