package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultGitHubOAuthClient is the production HTTP client for GitHub OAuth exchanges.
type DefaultGitHubOAuthClient struct {
	client       *http.Client
	clientID     string
	clientSecret string
}

// NewDefaultGitHubOAuthClient constructs the default GitHub OAuth HTTP client.
func NewDefaultGitHubOAuthClient(clientID string, clientSecret string) *DefaultGitHubOAuthClient {
	return &DefaultGitHubOAuthClient{
		client:       &http.Client{Timeout: 10 * time.Second},
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
	}
}

// ExchangeCode exchanges a GitHub authorization code for an access token.
func (c *DefaultGitHubOAuthClient) ExchangeCode(ctx context.Context, code string, redirectURI string) (string, error) {
	payload := url.Values{}
	payload.Set("client_id", c.clientID)
	payload.Set("client_secret", c.clientSecret)
	payload.Set("code", strings.TrimSpace(code))
	payload.Set("redirect_uri", strings.TrimSpace(redirectURI))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewBufferString(payload.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if strings.TrimSpace(body.Error) != "" {
			return "", fmt.Errorf("github token exchange failed: %s", body.Error)
		}
		return "", fmt.Errorf("github token exchange failed with status %s", resp.Status)
	}
	if strings.TrimSpace(body.AccessToken) == "" {
		return "", fmt.Errorf("github token exchange returned an empty access token")
	}
	return body.AccessToken, nil
}

// FetchUser fetches the authenticated GitHub user profile for the access token.
func (c *DefaultGitHubOAuthClient) FetchUser(ctx context.Context, accessToken string) (GitHubUserProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return GitHubUserProfile{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	resp, err := c.client.Do(req)
	if err != nil {
		return GitHubUserProfile{}, err
	}
	defer resp.Body.Close()
	var body struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return GitHubUserProfile{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GitHubUserProfile{}, fmt.Errorf("github user lookup failed with status %s", resp.Status)
	}
	if body.ID == 0 || strings.TrimSpace(body.Login) == "" {
		return GitHubUserProfile{}, fmt.Errorf("github user lookup returned incomplete identity")
	}
	return GitHubUserProfile{
		Subject: fmt.Sprintf("%d", body.ID),
		Login:   body.Login,
		Email:   body.Email,
	}, nil
}
