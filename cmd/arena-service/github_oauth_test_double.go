package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	githubOAuthTestDoubleEnv = "ARENA_AUTH_GITHUB_TEST_DOUBLE"
	githubOAuthAuthorizePath = "/_internal/test-auth/github/authorize"
	githubOAuthTokenPath     = "/_internal/test-auth/github/token"
	githubOAuthUserPath      = "/_internal/test-auth/github/user"
)

func withGitHubOAuthTestDouble(next http.Handler, _ string) http.Handler {
	if strings.TrimSpace(os.Getenv(githubOAuthTestDoubleEnv)) == "" {
		return next
	}
	mux := http.NewServeMux()
	mux.HandleFunc(githubOAuthAuthorizePath, handleGitHubOAuthAuthorize)
	mux.HandleFunc(githubOAuthTokenPath, handleGitHubOAuthToken)
	mux.HandleFunc(githubOAuthUserPath, handleGitHubOAuthUser)
	mux.Handle("/", next)
	return mux
}

func ensureGitHubOAuthProviderEnv(rawPort string) {
	baseURL := absoluteLocalURL(rawPort)
	setenvDefault("ARENA_AUTH_GITHUB_PROVIDER_AUTH_URL", baseURL+githubOAuthAuthorizePath)
	setenvDefault("ARENA_AUTH_GITHUB_PROVIDER_TOKEN_URL", baseURL+githubOAuthTokenPath)
	setenvDefault("ARENA_AUTH_GITHUB_PROVIDER_USER_URL", baseURL+githubOAuthUserPath)
}

func absoluteLocalURL(rawPort string) string {
	trimmed := strings.TrimSpace(rawPort)
	if trimmed == "" {
		trimmed = "10000"
	}
	trimmed = strings.TrimPrefix(trimmed, ":")
	return "http://127.0.0.1:" + trimmed
}

func setenvDefault(key string, value string) {
	if strings.TrimSpace(os.Getenv(key)) != "" {
		return
	}
	_ = os.Setenv(key, value)
}

func handleGitHubOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		renderGitHubAuthorizeForm(w, r)
	case http.MethodPost:
		redirectURI := strings.TrimSpace(r.FormValue("redirect_uri"))
		if redirectURI == "" {
			http.Error(w, "redirect_uri is required", http.StatusBadRequest)
			return
		}
		redirect, err := validateGitHubOAuthRedirectURI(redirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		query := redirect.Query()
		query.Set("code", "playwright-github-code")
		query.Set("state", strings.TrimSpace(r.FormValue("state")))
		redirect.RawQuery = query.Encode()
		// #nosec G710 -- validateGitHubOAuthRedirectURI restricts redirects to the local GitHub callback path on localhost/127.0.0.1.
		http.Redirect(w, r, redirect.String(), http.StatusFound)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func renderGitHubAuthorizeForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	state := html.EscapeString(r.URL.Query().Get("state"))
	redirectURI := html.EscapeString(r.URL.Query().Get("redirect_uri"))
	clientID := html.EscapeString(r.URL.Query().Get("client_id"))
	login := html.EscapeString(authTestLogin())
	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>GitHub OAuth Test Double</title>
</head>
<body>
  <main>
    <h1>GitHub OAuth Test Double</h1>
    <p>Authorize the fixed tester account for Playwright regression coverage.</p>
    <p>Client ID: %s</p>
    <form method="post" action="%s">
      <input type="hidden" name="state" value="%s">
      <input type="hidden" name="redirect_uri" value="%s">
      <button type="submit">Continue as @%s</button>
    </form>
  </main>
</body>
</html>`, clientID, githubOAuthAuthorizePath, state, redirectURI, login)
}

func handleGitHubOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if strings.TrimSpace(r.FormValue("code")) == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}
	writeGitHubOAuthJSON(w, http.StatusOK, map[string]any{
		"access_token": "playwright-github-token",
		"token_type":   "bearer",
		"scope":        "read:user",
	})
}

func handleGitHubOAuthUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authz, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	writeGitHubOAuthJSON(w, http.StatusOK, map[string]any{
		"id":    authTestSubjectNumeric(),
		"login": authTestLogin(),
		"email": authTestEmail(),
	})
}

func authTestSubjectNumeric() int64 {
	if parsed, err := parseInt64(authTestSubject()); err == nil {
		return parsed
	}
	var value int64 = 424242
	if subject := strings.TrimSpace(os.Getenv("ARENA_AUTH_TEST_GITHUB_NUMERIC_ID")); subject != "" {
		if parsed, err := parseInt64(subject); err == nil {
			return parsed
		}
	}
	return value
}

func parseInt64(value string) (int64, error) {
	var parsed int64
	_, err := fmt.Sscan(value, &parsed)
	return parsed, err
}

func writeGitHubOAuthJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func validateGitHubOAuthRedirectURI(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("redirect_uri is invalid")
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("redirect_uri is invalid")
	}
	if parsed.Path != "/auth/github/callback" {
		return nil, fmt.Errorf("redirect_uri path is not allowed")
	}
	host := parsed.Hostname()
	if host != "localhost" && host != "127.0.0.1" {
		return nil, fmt.Errorf("redirect_uri host is not allowed")
	}
	return parsed, nil
}
