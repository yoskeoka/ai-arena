// Package authtest contains repo-owned auth fixtures for local and CI verification seams.
package authtest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

// GitHubOAuthTestUser describes one repo-owned OAuth identity for local and CI verification.
type GitHubOAuthTestUser struct {
	UserID          string
	NumericID       int64
	Login           string
	Email           string
	Role            string
	SeedAccount     bool
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
		SeedAccount:     true,
		DisplayLabel:    "Spectator User 01",
		DisplayHintLine: "spectator-user01 (participant role)",
	},
	{
		UserID:          "developer-user01",
		NumericID:       710002,
		Login:           "developer-user01",
		Email:           "developer-user01@example.com",
		Role:            "developer",
		SeedAccount:     true,
		DisplayLabel:    "Developer User 01",
		DisplayHintLine: "developer-user01 (developer role)",
	},
	{
		UserID:          "operator-user01",
		NumericID:       710003,
		Login:           "operator-user01",
		Email:           "operator-user01@example.com",
		Role:            "operator",
		SeedAccount:     true,
		DisplayLabel:    "Operator User 01",
		DisplayHintLine: "operator-user01 (operator role)",
	},
	{
		UserID:          "operator-signup-user01",
		NumericID:       710004,
		Login:           "operator-signup-user01",
		Email:           "operator-signup-user01@example.com",
		Role:            "operator",
		SeedAccount:     false,
		DisplayLabel:    "Operator Signup User 01",
		DisplayHintLine: "operator-signup-user01 (signup-only operator role)",
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

// SeedGitHubOAuthTestUsers seeds the canonical auth-store users into the auth store if they do not already exist.
func SeedGitHubOAuthTestUsers(ctx context.Context, store *service.PostgresAuthStore, now time.Time) error {
	for _, user := range GitHubOAuthTestUsers() {
		if !user.SeedAccount {
			continue
		}
		if err := seedGitHubOAuthTestUser(ctx, store, user, now); err != nil {
			return err
		}
	}
	return nil
}

// SeedGitHubOAuthTestUsersFromEnv seeds the canonical test users when a Postgres DSN is available.
func SeedGitHubOAuthTestUsersFromEnv(ctx context.Context, postgresDSN string, now time.Time) error {
	if strings.TrimSpace(postgresDSN) == "" {
		postgresDSN = strings.TrimSpace(os.Getenv("ARENA_SERVICE_POSTGRES_DSN"))
	}
	if strings.TrimSpace(postgresDSN) == "" {
		return nil
	}
	store, err := service.NewPostgresAuthStore(ctx, postgresDSN)
	if err != nil {
		return err
	}
	defer store.Close()
	return SeedGitHubOAuthTestUsers(ctx, store, now)
}

func seedGitHubOAuthTestUser(ctx context.Context, store *service.PostgresAuthStore, user GitHubOAuthTestUser, now time.Time) error {
	identity := service.AuthIdentity{
		Provider: "github",
		Subject:  user.ProviderSubject(),
		Login:    user.Login,
		Email:    user.Email,
	}
	_, err := store.ResolveIdentityLogin(ctx, identity, "", now)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, service.ErrSignupInviteRequired):
		return err
	}
	invite, err := store.CreateSignupInvite(ctx, user.Role, now.Add(24*time.Hour))
	if err != nil {
		return err
	}
	_, err = store.ResolveIdentityLogin(ctx, identity, invite.InviteToken, now)
	return err
}
