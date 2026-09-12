// Command local-oidc-test-provider starts the local/CI-only password OIDC provider.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/luikyv/go-oidc/pkg/goidc"
	"github.com/luikyv/go-oidc/pkg/provider"
	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

// #nosec G101 -- this fixed password is intentionally published by the local/CI-only test provider.
const testerPassword = "local-oidc-password"

var testers = map[string]struct{ name, email string }{
	"tester01": {"Tester One", "tester01@example.test"},
	"tester02": {"Tester Two", "tester02@example.test"},
}

func main() {
	listenAddr := flag.String("listen-addr", "127.0.0.1:10002", "listen address")
	issuer := flag.String("issuer", "http://127.0.0.1:10002", "OIDC issuer")
	postgresDSN := flag.String("postgres-dsn", "", "optional PostgreSQL DSN for idempotent tester account seeding")
	flag.Parse()
	if err := seedTesters(*postgresDSN); err != nil {
		log.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal(err)
	}
	jwks := goidc.JSONWebKeySet{Keys: []goidc.JSONWebKey{{KeyID: "local-oidc", Key: key, Algorithm: "RS256", Use: "sig"}}}
	op, err := provider.New(provider.Config{Issuer: *issuer, JWKS: func(context.Context) (goidc.JSONWebKeySet, error) { return jwks, nil }, IDTokenAlgs: []goidc.SignatureAlgorithm{goidc.SigAlgRS256}},
		provider.WithAuthCodeGrant(provider.AuthCodeGrantConfig{ResponseTypes: []goidc.ResponseType{goidc.ResponseTypeCode}}, provider.WithAuthPolicies(passwordPolicy())),
		provider.WithDCR(nil), provider.WithScopes(goidc.ScopeOpenID, goidc.ScopeProfile, goidc.ScopeEmail), provider.WithClaims(goidc.ClaimEmail, goidc.ClaimName, goidc.ClaimPreferredUsername),
		provider.WithIDTokenClaims(func(_ context.Context, grant *goidc.Grant) map[string]any { return grant.Store }),
	)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: *listenAddr, Handler: op.Handler(), ReadHeaderTimeout: 5 * time.Second}
	log.Printf("local OIDC provider listening on %s", *listenAddr)
	log.Fatal(server.ListenAndServe())
}

func seedTesters(postgresDSN string) error {
	if strings.TrimSpace(postgresDSN) == "" {
		return nil
	}
	store, err := service.NewPostgresAuthStore(context.Background(), postgresDSN)
	if err != nil {
		return err
	}
	defer store.Close()
	for userID, role := range map[string]string{"tester01": "operator", "tester02": "developer"} {
		user := testers[userID]
		identity := service.AuthIdentity{Provider: "local-oidc", Subject: userID, Login: userID, Email: user.email}
		if _, err := store.ResolveIdentityLogin(context.Background(), identity, "", time.Now().UTC()); err == nil {
			continue
		} else if !errors.Is(err, service.ErrSignupInviteRequired) {
			return err
		}
		invite, err := store.CreateSignupInvite(context.Background(), role, time.Now().UTC().Add(24*time.Hour))
		if err != nil {
			return err
		}
		_, err = store.ResolveIdentityLogin(context.Background(), identity, invite.InviteToken, time.Now().UTC())
		if err != nil {
			return err
		}
	}
	return nil
}

func passwordPolicy() goidc.AuthnPolicy {
	return goidc.NewPolicy("local-password", func(_ *http.Request, session *goidc.AuthnSession, _ *goidc.Client) bool {
		if session.Store == nil {
			session.Store = map[string]any{}
		}
		return true
	}, func(w http.ResponseWriter, r *http.Request, session *goidc.AuthnSession, _ *goidc.Client) (goidc.Status, error) {
		if r.Method != http.MethodPost {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = fmt.Fprintf(w, `<h1>Local OIDC sign in</h1><form method="post" action="/authorize/%s"><label>Username <input name="username"></label><label>Password <input name="password" type="password"></label><button>Login</button></form><p>tester01 and tester02; password: %s</p>`, html.EscapeString(session.ID), testerPassword)
			return goidc.StatusPending, nil
		}
		userID := strings.TrimSpace(r.FormValue("username"))
		user, ok := testers[userID]
		if !ok || r.FormValue("password") != testerPassword {
			return goidc.StatusFailure, fmt.Errorf("invalid local OIDC credentials")
		}
		session.Subject, session.Username = userID, userID
		session.GrantedScopes = session.Scopes
		session.Store[goidc.ClaimPreferredUsername] = userID
		session.Store[goidc.ClaimName] = user.name
		session.Store[goidc.ClaimEmail] = user.email
		return goidc.StatusSuccess, nil
	})
}
