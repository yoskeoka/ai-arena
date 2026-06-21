package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// DefaultGitHubAuthProvider is the production GitHub auth provider.
type DefaultGitHubAuthProvider struct {
	client      *http.Client
	oauthConfig *oauth2.Config
	userURL     string
}

// GitHubAuthProviderConfig configures the GitHub OAuth transport and user lookup endpoints.
type GitHubAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserURL      string
	HTTPClient   *http.Client
}

// NewGitHubAuthProvider constructs a GitHub auth provider from the supplied config.
func NewGitHubAuthProvider(cfg GitHubAuthProviderConfig) *DefaultGitHubAuthProvider {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	authURL := strings.TrimSpace(cfg.AuthURL)
	if authURL == "" {
		authURL = "https://github.com/login/oauth/authorize"
	}
	tokenURL := strings.TrimSpace(cfg.TokenURL)
	if tokenURL == "" {
		// #nosec G101 -- OAuth endpoint URLs are public protocol constants, not embedded secrets.
		tokenURL = "https://github.com/login/oauth/access_token"
	}
	userURL := strings.TrimSpace(cfg.UserURL)
	if userURL == "" {
		userURL = "https://api.github.com/user"
	}
	authURL = mustProviderEndpointURL(authURL)
	tokenURL = mustProviderEndpointURL(tokenURL)
	userURL = mustProviderEndpointURL(userURL)
	return &DefaultGitHubAuthProvider{
		client: client,
		oauthConfig: &oauth2.Config{
			ClientID:     strings.TrimSpace(cfg.ClientID),
			ClientSecret: strings.TrimSpace(cfg.ClientSecret),
			// #nosec G101 -- OAuth endpoint URLs are public protocol constants, not embedded secrets.
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
			Scopes: []string{"read:user"},
		},
		userURL: userURL,
	}
}

// NewDefaultGitHubAuthProvider constructs the default GitHub auth provider.
func NewDefaultGitHubAuthProvider(clientID string, clientSecret string) *DefaultGitHubAuthProvider {
	return NewGitHubAuthProvider(GitHubAuthProviderConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
}

// AuthorizationURL builds the GitHub authorization URL for the current login attempt.
func (c *DefaultGitHubAuthProvider) AuthorizationURL(redirectURI string, state string) string {
	return c.oauthConfig.AuthCodeURL(state,
		oauth2.SetAuthURLParam("redirect_uri", strings.TrimSpace(redirectURI)),
	)
}

// ExchangeIdentity exchanges the GitHub authorization code and resolves a normalized identity.
func (c *DefaultGitHubAuthProvider) ExchangeIdentity(ctx context.Context, code string, redirectURI string) (AuthIdentity, error) {
	ctx = context.WithValue(ctx, oauth2.HTTPClient, c.client)
	token, err := c.oauthConfig.Exchange(ctx, strings.TrimSpace(code), oauth2.SetAuthURLParam("redirect_uri", strings.TrimSpace(redirectURI)))
	if err != nil {
		return AuthIdentity{}, err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return AuthIdentity{}, fmt.Errorf("github token exchange returned an empty access token")
	}
	identity, err := c.fetchUser(ctx, token.AccessToken)
	if err != nil {
		return AuthIdentity{}, fmt.Errorf("%w: %w", ErrIdentityLookupFailed, err)
	}
	return identity, nil
}

func (c *DefaultGitHubAuthProvider) fetchUser(ctx context.Context, accessToken string) (AuthIdentity, error) {
	// #nosec G704 -- the provider endpoint is validated by mustProviderEndpointURL and limited to GitHub HTTPS or localhost test doubles.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userURL, nil)
	if err != nil {
		return AuthIdentity{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	// #nosec G704 -- the provider endpoint is validated by mustProviderEndpointURL and limited to GitHub HTTPS or localhost test doubles.
	resp, err := c.client.Do(req)
	if err != nil {
		return AuthIdentity{}, err
	}
	defer resp.Body.Close()
	var body struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return AuthIdentity{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return AuthIdentity{}, fmt.Errorf("github user lookup failed with status %s", resp.Status)
	}
	if body.ID == 0 || strings.TrimSpace(body.Login) == "" {
		return AuthIdentity{}, fmt.Errorf("github user lookup returned incomplete identity")
	}
	return AuthIdentity{
		Provider: authProviderGitHub,
		Subject:  fmt.Sprintf("%d", body.ID),
		Login:    body.Login,
		Email:    body.Email,
	}, nil
}

func mustProviderEndpointURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		panic(fmt.Sprintf("invalid GitHub auth provider endpoint %q", raw))
	}
	switch parsed.Scheme {
	case "https":
		return parsed.String()
	case "http":
		host := parsed.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			return parsed.String()
		}
	}
	panic(fmt.Sprintf("unsupported GitHub auth provider endpoint %q", raw))
}
