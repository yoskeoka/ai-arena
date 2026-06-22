// Command github-oauth-test-seed seeds canonical auth test users into the Postgres auth store.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/authtest"
	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

func main() {
	var postgresDSN string
	flag.StringVar(&postgresDSN, "postgres-dsn", "", "PostgreSQL DSN for auth seed")
	flag.Parse()

	if err := run(postgresDSN); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(postgresDSN string) error {
	if strings.TrimSpace(postgresDSN) == "" {
		postgresDSN = strings.TrimSpace(os.Getenv("ARENA_SERVICE_POSTGRES_DSN"))
	}
	if strings.TrimSpace(postgresDSN) == "" {
		return fmt.Errorf("ARENA_SERVICE_POSTGRES_DSN is required")
	}

	store, err := service.NewPostgresAuthStore(context.Background(), postgresDSN)
	if err != nil {
		return err
	}
	defer store.Close()

	now := time.Now().UTC()
	for _, user := range authtest.GitHubOAuthTestUsers() {
		if err := seedUser(context.Background(), store, user, now); err != nil {
			return err
		}
	}
	return nil
}

func seedUser(ctx context.Context, store *service.PostgresAuthStore, user authtest.GitHubOAuthTestUser, now time.Time) error {
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
