// Package authtest contains repo-owned auth fixtures for local and CI verification seams.
package authtest

import (
	"fmt"
	"strings"
)

// GitHubOAuthTestUser describes one repo-owned OAuth identity for local and CI verification.
type GitHubOAuthTestUser struct {
	UserID          string
	NumericID       int64
	Login           string
	Email           string
	Role            string
	DisplayLabel    string
	DisplayHintLine string
}

var githubOAuthTestUsers = []GitHubOAuthTestUser{
	{
		UserID:          "spectator-user01",
		NumericID:       710001,
		Login:           "spectator-user01",
		Email:           "spectator-user01@example.com",
		Role:            "participant",
		DisplayLabel:    "Spectator User 01",
		DisplayHintLine: "spectator-user01 (participant role)",
	},
	{
		UserID:          "developer-user01",
		NumericID:       710002,
		Login:           "developer-user01",
		Email:           "developer-user01@example.com",
		Role:            "developer",
		DisplayLabel:    "Developer User 01",
		DisplayHintLine: "developer-user01 (developer role)",
	},
	{
		UserID:          "operator-user01",
		NumericID:       710003,
		Login:           "operator-user01",
		Email:           "operator-user01@example.com",
		Role:            "operator",
		DisplayLabel:    "Operator User 01",
		DisplayHintLine: "operator-user01 (operator role)",
	},
}

// DefaultGitHubOAuthTestUserID is the default login target for auth-enabled operator verification.
const DefaultGitHubOAuthTestUserID = "operator-user01"

// GitHubOAuthTestUsers returns the canonical repo-owned test users.
func GitHubOAuthTestUsers() []GitHubOAuthTestUser {
	users := make([]GitHubOAuthTestUser, len(githubOAuthTestUsers))
	copy(users, githubOAuthTestUsers)
	return users
}

// LookupGitHubOAuthTestUser resolves one canonical test user by user-facing identifier.
func LookupGitHubOAuthTestUser(userID string) (GitHubOAuthTestUser, bool) {
	normalized := strings.TrimSpace(strings.ToLower(userID))
	for _, user := range githubOAuthTestUsers {
		if normalized == strings.ToLower(user.UserID) {
			return user, true
		}
	}
	return GitHubOAuthTestUser{}, false
}

// ProviderSubject returns the normalized provider subject persisted in auth storage.
func (u GitHubOAuthTestUser) ProviderSubject() string {
	return fmt.Sprintf("%d", u.NumericID)
}
