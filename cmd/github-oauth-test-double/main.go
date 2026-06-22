// Command github-oauth-test-double starts a repo-owned mock GitHub OAuth server for local and CI verification.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/authtest"
)

func main() {
	listenAddr := flag.String("listen-addr", "127.0.0.1:10001", "listen address")
	postgresDSN := flag.String("postgres-dsn", "", "optional PostgreSQL DSN for seeding canonical auth test users")
	flag.Parse()

	logger := log.New(os.Stdout, "github-oauth-test-double: ", log.LstdFlags)
	if err := authtest.SeedGitHubOAuthTestUsersFromEnv(context.Background(), *postgresDSN, time.Now().UTC()); err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              strings.TrimSpace(*listenAddr),
		Handler:           newHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	logger.Printf("listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func newHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /authorize", handleAuthorizeForm)
	mux.HandleFunc("POST /authorize", handleAuthorizeSubmit)
	mux.HandleFunc("POST /token", handleAccessToken)
	mux.HandleFunc("POST /access_token", handleAccessToken)
	mux.HandleFunc("GET /user", handleUser)
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func handleAuthorizeForm(w http.ResponseWriter, r *http.Request) {
	selectedUserID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if selectedUserID == "" {
		selectedUserID = authtest.DefaultGitHubOAuthTestUserID
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	state := html.EscapeString(r.URL.Query().Get("state"))
	redirectURI := html.EscapeString(r.URL.Query().Get("redirect_uri"))
	clientID := html.EscapeString(r.URL.Query().Get("client_id"))
	selectedValue := html.EscapeString(selectedUserID)

	var hints strings.Builder
	for _, user := range authtest.GitHubOAuthTestUsers() {
		_, _ = fmt.Fprintf(&hints, "<li><code>%s</code> - %s</li>", html.EscapeString(user.UserID), html.EscapeString(user.DisplayHintLine))
	}

	_, _ = fmt.Fprintf(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>GitHub OAuth Test Double</title>
</head>
<body>
  <main>
    <h1>GitHub OAuth Test Double</h1>
    <p>Client ID: %s</p>
    <form method="post" action="/authorize">
      <input type="hidden" name="state" value="%s">
      <input type="hidden" name="redirect_uri" value="%s">
      <label for="user_id">User ID</label>
      <input id="user_id" name="user_id" type="text" value="%s" autocomplete="off">
      <button type="submit">Login</button>
    </form>
    <section>
      <h2>Available test users</h2>
      <ul>%s</ul>
    </section>
  </main>
</body>
</html>`, clientID, state, redirectURI, selectedValue, hints.String())
}

func handleAuthorizeSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	user, ok := authtest.LookupGitHubOAuthTestUser(r.FormValue("user_id"))
	if !ok {
		http.Error(w, "unknown user_id", http.StatusBadRequest)
		return
	}
	redirectURI := strings.TrimSpace(r.FormValue("redirect_uri"))
	redirect, err := validateRedirectURI(redirectURI)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := redirect.Query()
	query.Set("code", codeForUser(user.UserID))
	query.Set("state", strings.TrimSpace(r.FormValue("state")))
	redirect.RawQuery = query.Encode()
	// #nosec G710 -- validateRedirectURI restricts redirects to the local GitHub callback path on localhost/127.0.0.1.
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func handleAccessToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	userID, ok := userIDFromCode(r.FormValue("code"))
	if !ok {
		http.Error(w, "invalid code", http.StatusBadRequest)
		return
	}
	if _, ok := authtest.LookupGitHubOAuthTestUser(userID); !ok {
		http.Error(w, "unknown user_id", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": tokenForUser(userID),
		"token_type":   "bearer",
		"scope":        "read:user",
	})
}

func handleUser(w http.ResponseWriter, r *http.Request) {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authz, "Bearer ") {
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDFromToken(strings.TrimPrefix(authz, "Bearer "))
	if !ok {
		http.Error(w, "invalid bearer token", http.StatusUnauthorized)
		return
	}
	user, ok := authtest.LookupGitHubOAuthTestUser(userID)
	if !ok {
		http.Error(w, "unknown user_id", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":    user.NumericID,
		"login": user.Login,
		"email": user.Email,
	})
}

func validateRedirectURI(raw string) (*url.URL, error) {
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

func codeForUser(userID string) string {
	return "github-test-code:" + strings.TrimSpace(userID)
}

func userIDFromCode(code string) (string, bool) {
	return strings.CutPrefix(strings.TrimSpace(code), "github-test-code:")
}

func tokenForUser(userID string) string {
	return "github-test-token:" + strings.TrimSpace(userID)
}

func userIDFromToken(token string) (string, bool) {
	return strings.CutPrefix(strings.TrimSpace(token), "github-test-token:")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
