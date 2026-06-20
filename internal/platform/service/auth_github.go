package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// DefaultGitHubAuthProvider is the production GitHub auth provider.
type DefaultGitHubAuthProvider struct {
	client      *http.Client
	oauthConfig *oauth2.Config
}

// NewDefaultGitHubAuthProvider constructs the default GitHub auth provider.
func NewDefaultGitHubAuthProvider(clientID string, clientSecret string) *DefaultGitHubAuthProvider {
	return &DefaultGitHubAuthProvider{
		client: &http.Client{Timeout: 10 * time.Second},
		oauthConfig: &oauth2.Config{
			ClientID:     strings.TrimSpace(clientID),
			ClientSecret: strings.TrimSpace(clientSecret),
			// #nosec G101 -- OAuth endpoint URLs are public protocol constants, not embedded secrets.
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
			Scopes: []string{"read:user"},
		},
	}
}

// AuthorizationURL builds the GitHub authorization URL for the current login attempt.
func (c *DefaultGitHubAuthProvider) AuthorizationURL(redirectURI string, state string) string {
	return c.oauthConfig.AuthCodeURL(state,
		oauth2.SetAuthURLParam("redirect_uri", strings.TrimSpace(redirectURI)),
	)
}

// ExchangeIdentity exchanges the GitHub authorization code and resolves a normalized identity.
func (c *DefaultGitHubAuthProvider) ExchangeIdentity(ctx context.Context, code string, redirectURI string) (AuthIdentity, error) {
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return AuthIdentity{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
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
