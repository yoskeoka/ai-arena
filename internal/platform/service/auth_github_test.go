package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestGitHubAuthProviderUsesConfiguredEndpoints(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("token method = %s, want POST", r.Method)
		}
		if got := r.FormValue("code"); got != "code-1" {
			t.Fatalf("token code = %q, want code-1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token-1",
			"token_type":   "bearer",
		})
	})
	mux.HandleFunc("/api/user", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token-1" {
			t.Fatalf("authorization = %q, want bearer token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    424242,
			"login": "playwright-operator",
			"email": "playwright-operator@example.com",
		})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	provider, err := NewGitHubAuthProvider(GitHubAuthProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		OAuthBaseURL: server.URL + "/oauth",
		APIBaseURL:   server.URL + "/api",
	})
	if err != nil {
		t.Fatalf("NewGitHubAuthProvider() error = %v", err)
	}

	redirectURL := provider.AuthorizationURL("http://127.0.0.1:10000/auth/github/callback", "state-1")
	parsedRedirect, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("url.Parse(redirect) error = %v", err)
	}
	if got := parsedRedirect.Scheme + "://" + parsedRedirect.Host + parsedRedirect.Path; got != server.URL+"/oauth/authorize" {
		t.Fatalf("redirect auth url = %q, want test authorize endpoint", got)
	}

	identity, err := provider.ExchangeIdentity(context.Background(), "code-1", "http://127.0.0.1:10000/auth/github/callback")
	if err != nil {
		t.Fatalf("ExchangeIdentity() error = %v", err)
	}
	if identity.Provider != authProviderGitHub || identity.Subject != "424242" || identity.Login != "playwright-operator" {
		t.Fatalf("ExchangeIdentity() = %+v, want configured test identity", identity)
	}
}

func TestGitHubAuthProviderRejectsProviderEndpointQueryFragmentAndUserinfo(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"https://github.com/login/oauth?via=test",
		"https://token@example.com/login/oauth",
		"https://api.github.com#fragment",
	}
	for _, raw := range testCases {
		raw := raw
		t.Run(raw, func(t *testing.T) {
			t.Parallel()

			_, err := NewGitHubAuthProvider(GitHubAuthProviderConfig{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				OAuthBaseURL: raw,
			})
			if err == nil {
				t.Fatalf("NewGitHubAuthProvider(%q) error = nil, want validation error", raw)
			}
		})
	}
}

func TestGitHubAuthProviderFetchUserReturnsStatusErrorBeforeDecode(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	provider, err := NewGitHubAuthProvider(GitHubAuthProviderConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		APIBaseURL:   server.URL,
	})
	if err != nil {
		t.Fatalf("NewGitHubAuthProvider() error = %v", err)
	}

	_, err = provider.fetchUser(context.Background(), "token-1")
	if err == nil {
		t.Fatal("fetchUser() error = nil, want status error")
	}
	if !strings.Contains(err.Error(), "502 Bad Gateway") {
		t.Fatalf("fetchUser() error = %v, want HTTP status in message", err)
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("fetchUser() error = %v, want provider status error", err)
	}
}
