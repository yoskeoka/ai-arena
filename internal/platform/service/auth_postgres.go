package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PostgresAuthStore persists auth state in Postgres.
type PostgresAuthStore struct {
	pool *pgxpool.Pool
}

// SignupInviteRecord returns the raw invite token created for local/bootstrap flows.
type SignupInviteRecord struct {
	InviteToken string    `json:"invite_token"`
	Role        string    `json:"role"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// NewPostgresAuthStore opens the auth store and validates that auth tables exist.
func NewPostgresAuthStore(ctx context.Context, dsn string) (*PostgresAuthStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("service: postgres dsn is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("service: open postgres auth store: %w", err)
	}
	if err := validatePostgresAuthStoreSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresAuthStore{pool: pool}, nil
}

// Close releases the underlying Postgres connection pool.
func (s *PostgresAuthStore) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

// ResolveGitHubLogin loads an existing GitHub-linked principal or claims a signup invite for first login.
func (s *PostgresAuthStore) ResolveGitHubLogin(ctx context.Context, profile GitHubUserProfile, inviteToken string, now time.Time) (AuthPrincipal, error) {
	if strings.TrimSpace(profile.Subject) == "" || strings.TrimSpace(profile.Login) == "" {
		return AuthPrincipal{}, fmt.Errorf("service: github profile subject and login are required")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return AuthPrincipal{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	principal, err := lookupPrincipalByIdentity(ctx, tx, "github", profile.Subject)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return AuthPrincipal{}, err
		}
		return principal, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return AuthPrincipal{}, err
	}
	if strings.TrimSpace(inviteToken) == "" {
		return AuthPrincipal{}, ErrSignupInviteRequired
	}
	accountID, principal, err := createPrincipalFromInvite(ctx, tx, profile, inviteToken, now)
	if err != nil {
		return AuthPrincipal{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE signup_invites
		SET claimed_account_id = $1, claimed_at = $2
		WHERE invite_token_hash = $3
	`, accountID, now, tokenHash(inviteToken)); err != nil {
		return AuthPrincipal{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return AuthPrincipal{}, err
	}
	return principal, nil
}

// CreateSession creates a new persisted account session and returns the raw session token.
func (s *PostgresAuthStore) CreateSession(ctx context.Context, accountID string, expiresAt time.Time) (string, error) {
	token, err := randomToken(32)
	if err != nil {
		return "", err
	}
	sessionID := uuid.NewString()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO account_sessions (session_id, account_id, session_token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, sessionID, accountID, tokenHash(token), expiresAt)
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetSession resolves an active session token to its authenticated principal.
func (s *PostgresAuthStore) GetSession(ctx context.Context, sessionToken string, now time.Time) (AuthPrincipal, error) {
	return lookupPrincipalBySession(ctx, s.pool, tokenHash(sessionToken), now)
}

// DeleteSession removes a persisted session token from the auth store.
func (s *PostgresAuthStore) DeleteSession(ctx context.Context, sessionToken string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM account_sessions WHERE session_token_hash = $1`, tokenHash(sessionToken))
	return err
}

// CreateSignupInvite creates a new single-use signup invite for the requested role.
func (s *PostgresAuthStore) CreateSignupInvite(ctx context.Context, role string, expiresAt time.Time) (SignupInviteRecord, error) {
	role = strings.TrimSpace(role)
	if role == "" {
		return SignupInviteRecord{}, fmt.Errorf("service: invite role is required")
	}
	token, err := randomToken(24)
	if err != nil {
		return SignupInviteRecord{}, err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO signup_invites (invite_id, invite_token_hash, role, expires_at)
		VALUES ($1, $2, $3, $4)
	`, uuid.NewString(), tokenHash(token), role, expiresAt)
	if err != nil {
		return SignupInviteRecord{}, err
	}
	return SignupInviteRecord{
		InviteToken: token,
		Role:        role,
		ExpiresAt:   expiresAt,
	}, nil
}

func validatePostgresAuthStoreSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var missing int
	if err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_name IN ('accounts', 'account_identities', 'account_roles', 'account_sessions', 'signup_invites')
	`).Scan(&missing); err != nil {
		return err
	}
	if missing < 5 {
		return fmt.Errorf("service: auth tables are not available; run schema apply before starting auth-enabled service")
	}
	return nil
}

func lookupPrincipalBySession(ctx context.Context, db pgxQueryer, sessionTokenHash string, now time.Time) (AuthPrincipal, error) {
	var principal AuthPrincipal
	if err := db.QueryRow(ctx, `
		SELECT a.account_id, i.provider, i.provider_login, COALESCE(i.provider_email, '')
		FROM account_sessions s
		JOIN accounts a ON a.account_id = s.account_id
		JOIN account_identities i ON i.account_id = a.account_id
		WHERE s.session_token_hash = $1
		  AND s.expires_at > $2
		  AND i.provider = 'github'
	`, sessionTokenHash, now).Scan(&principal.AccountID, &principal.Provider, &principal.ProviderLogin, &principal.ProviderEmail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthPrincipal{}, ErrAuthenticationNeeded
		}
		return AuthPrincipal{}, err
	}
	roles, err := loadRoles(ctx, db, principal.AccountID)
	if err != nil {
		return AuthPrincipal{}, err
	}
	principal.Roles = roles
	return principal, nil
}

func lookupPrincipalByIdentity(ctx context.Context, db pgxQueryer, provider string, subject string) (AuthPrincipal, error) {
	var principal AuthPrincipal
	if err := db.QueryRow(ctx, `
		SELECT a.account_id, i.provider, i.provider_login, COALESCE(i.provider_email, '')
		FROM account_identities i
		JOIN accounts a ON a.account_id = i.account_id
		WHERE i.provider = $1 AND i.provider_subject = $2
	`, provider, subject).Scan(&principal.AccountID, &principal.Provider, &principal.ProviderLogin, &principal.ProviderEmail); err != nil {
		return AuthPrincipal{}, err
	}
	roles, err := loadRoles(ctx, db, principal.AccountID)
	if err != nil {
		return AuthPrincipal{}, err
	}
	principal.Roles = roles
	return principal, nil
}

func createPrincipalFromInvite(ctx context.Context, tx pgx.Tx, profile GitHubUserProfile, inviteToken string, now time.Time) (string, AuthPrincipal, error) {
	var role string
	var claimedAt *time.Time
	var expiresAt time.Time
	if err := tx.QueryRow(ctx, `
		SELECT role, claimed_at, expires_at
		FROM signup_invites
		WHERE invite_token_hash = $1
		FOR UPDATE
	`, tokenHash(inviteToken)).Scan(&role, &claimedAt, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", AuthPrincipal{}, ErrInvalidSignupInvite
		}
		return "", AuthPrincipal{}, err
	}
	if claimedAt != nil || !expiresAt.After(now) {
		return "", AuthPrincipal{}, ErrInvalidSignupInvite
	}
	accountID := uuid.NewString()
	if _, err := tx.Exec(ctx, `
		INSERT INTO accounts (account_id, created_at, updated_at)
		VALUES ($1, $2, $2)
	`, accountID, now); err != nil {
		return "", AuthPrincipal{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_identities (
			identity_id, account_id, provider, provider_subject, provider_login, provider_email, created_at, updated_at
		) VALUES ($1, $2, 'github', $3, $4, NULLIF($5, ''), $6, $6)
	`, uuid.NewString(), accountID, profile.Subject, profile.Login, profile.Email, now); err != nil {
		return "", AuthPrincipal{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO account_roles (account_id, role, created_at)
		VALUES ($1, $2, $3)
	`, accountID, role, now); err != nil {
		return "", AuthPrincipal{}, err
	}
	return accountID, AuthPrincipal{
		AccountID:     accountID,
		Provider:      "github",
		ProviderLogin: profile.Login,
		ProviderEmail: profile.Email,
		Roles:         []string{role},
	}, nil
}

func loadRoles(ctx context.Context, db pgxQueryer, accountID string) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT role FROM account_roles WHERE account_id = $1 ORDER BY role`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	roles := []string{}
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func tokenHash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
