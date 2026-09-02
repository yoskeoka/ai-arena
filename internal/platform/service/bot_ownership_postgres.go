package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresBotOwnershipStore serializes owner/scope lifecycle changes durably.
type PostgresBotOwnershipStore struct{ pool *pgxpool.Pool }

// NewPostgresBotOwnershipStore opens and verifies a PostgreSQL-backed bot ownership store.
func NewPostgresBotOwnershipStore(ctx context.Context, dsn string) (*PostgresBotOwnershipStore, error) {
	pool, err := pgxpool.New(ctx, strings.TrimSpace(dsn))
	if err != nil {
		return nil, fmt.Errorf("service: open postgres bot store: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresBotOwnershipStore{pool: pool}, nil
}

// Close releases the PostgreSQL connection pool used by the bot ownership store.
func (s *PostgresBotOwnershipStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

// CreateOrRevise durably creates or revises an owned bot in a serializable transaction.
func (s *PostgresBotOwnershipStore) CreateOrRevise(ctx context.Context, req BotRevisionRequest) (OwnedBot, AISubmissionRevision, error) {
	if strings.TrimSpace(req.OwnerAccountID) == "" || strings.TrimSpace(req.Scope.ScopeID) == "" || strings.TrimSpace(req.ArtifactID) == "" {
		return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: owner, scope, and artifact are required", ErrBadRequest)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return OwnedBot{}, AISubmissionRevision{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var limit int
	if err := tx.QueryRow(ctx, `SELECT max_active_bots_per_owner FROM competition_scopes WHERE scope_id=$1 FOR UPDATE`, req.Scope.ScopeID).Scan(&limit); err != nil {
		return OwnedBot{}, AISubmissionRevision{}, mapBotStoreError(err)
	}
	bot := OwnedBot{}
	if id := strings.TrimSpace(req.BotID); id != "" {
		err = tx.QueryRow(ctx, `SELECT bot_id::text, owner_account_id::text, scope_id, bot_name, normalized_bot_name, lifecycle_state, COALESCE(active_submission_id::text,'') FROM ai_bots WHERE bot_id=$1 FOR UPDATE`, id).Scan(&bot.BotID, &bot.OwnerAccountID, &bot.ScopeID, &bot.BotName, &bot.NormalizedBotName, &bot.LifecycleState, &bot.ActiveRevisionID)
		if err != nil {
			return OwnedBot{}, AISubmissionRevision{}, mapBotStoreError(err)
		}
		if bot.OwnerAccountID != req.OwnerAccountID || bot.ScopeID != req.Scope.ScopeID || bot.LifecycleState != BotActive {
			return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: bot is not an active bot owned in this scope", ErrBadRequest)
		}
	} else {
		name := normalizeBotName(req.BotName)
		if name == "" {
			return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: bot name is required", ErrBadRequest)
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM ai_bots WHERE owner_account_id=$1 AND scope_id=$2 AND lifecycle_state='active'`, req.OwnerAccountID, req.Scope.ScopeID).Scan(&count); err != nil {
			return OwnedBot{}, AISubmissionRevision{}, err
		}
		if count >= limit {
			return OwnedBot{}, AISubmissionRevision{}, ErrBotQuotaExceeded
		}
		bot = OwnedBot{BotID: uuid.NewString(), OwnerAccountID: req.OwnerAccountID, ScopeID: req.Scope.ScopeID, BotName: strings.TrimSpace(req.BotName), NormalizedBotName: name, LifecycleState: BotActive}
		_, err = tx.Exec(ctx, `INSERT INTO ai_bots (bot_id,owner_account_id,scope_id,bot_name,normalized_bot_name,lifecycle_state) VALUES ($1,$2,$3,$4,$5,'active')`, bot.BotID, bot.OwnerAccountID, bot.ScopeID, bot.BotName, bot.NormalizedBotName)
		if err != nil {
			return OwnedBot{}, AISubmissionRevision{}, mapBotStoreError(err)
		}
	}
	revision := AISubmissionRevision{AISubmissionID: uuid.NewString(), BotID: bot.BotID, ArtifactID: req.ArtifactID, ValidationState: req.ValidationState}
	if revision.ValidationState == "" {
		revision.ValidationState = ValidationReady
	}
	err = tx.QueryRow(ctx, `INSERT INTO ai_submission_revisions (ai_submission_id,bot_id,artifact_id,runtime_kind,ai_id,validation_state) VALUES ($1,$2,$3,$4,$5,$6) RETURNING created_at`, revision.AISubmissionID, revision.BotID, revision.ArtifactID, req.RuntimeKind, req.AIID, revision.ValidationState).Scan(&revision.CreatedAt)
	if err != nil {
		return OwnedBot{}, AISubmissionRevision{}, mapBotStoreError(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE ai_bots SET active_submission_id=$1 WHERE bot_id=$2`, revision.AISubmissionID, bot.BotID); err != nil {
		return OwnedBot{}, AISubmissionRevision{}, err
	}
	bot.ActiveRevisionID = revision.AISubmissionID
	if err = tx.Commit(ctx); err != nil {
		return OwnedBot{}, AISubmissionRevision{}, mapBotStoreError(err)
	}
	return bot, revision, nil
}

// Retire marks an owned bot retired while preserving its stored identity and revision history.
func (s *PostgresBotOwnershipStore) Retire(ctx context.Context, owner, id string) (OwnedBot, error) {
	var b OwnedBot
	err := s.pool.QueryRow(ctx, `UPDATE ai_bots SET lifecycle_state='retired', retired_at=NOW() WHERE bot_id=$1 AND owner_account_id=$2 RETURNING bot_id::text,owner_account_id::text,scope_id,bot_name,normalized_bot_name,lifecycle_state,COALESCE(active_submission_id::text,'')`, id, owner).Scan(&b.BotID, &b.OwnerAccountID, &b.ScopeID, &b.BotName, &b.NormalizedBotName, &b.LifecycleState, &b.ActiveRevisionID)
	return b, mapBotStoreError(err)
}

// ListByOwner returns persisted bots for an owner and scope, optionally including retired bots.
func (s *PostgresBotOwnershipStore) ListByOwner(ctx context.Context, owner, scope string, retired bool) ([]OwnedBot, error) {
	query := `SELECT bot_id::text,owner_account_id::text,scope_id,bot_name,normalized_bot_name,lifecycle_state,COALESCE(active_submission_id::text,'') FROM ai_bots WHERE owner_account_id=$1 AND scope_id=$2`
	if !retired {
		query += " AND lifecycle_state='active'"
	}
	rows, err := s.pool.Query(ctx, query, owner, scope)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OwnedBot, 0)
	for rows.Next() {
		var b OwnedBot
		if err = rows.Scan(&b.BotID, &b.OwnerAccountID, &b.ScopeID, &b.BotName, &b.NormalizedBotName, &b.LifecycleState, &b.ActiveRevisionID); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// ListEligible returns the scope-wide, operator-only composition projection.
func (s *PostgresBotOwnershipStore) ListEligible(ctx context.Context, scopeID string) ([]EligibleBot, error) {
	rows, err := s.pool.Query(ctx, `SELECT b.bot_id::text, b.scope_id, b.bot_name, b.active_submission_id::text FROM ai_bots b JOIN ai_submission_revisions r ON r.ai_submission_id=b.active_submission_id WHERE b.scope_id=$1 AND b.lifecycle_state='active' AND r.validation_state='ready' ORDER BY b.bot_name, b.bot_id`, strings.TrimSpace(scopeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]EligibleBot, 0)
	for rows.Next() {
		var item EligibleBot
		if err = rows.Scan(&item.BotID, &item.ScopeID, &item.BotName, &item.ActiveRevisionID); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresBotOwnershipStore) ResolveEligible(ctx context.Context, scopeID string, ids []string) ([]ResolvedEligibleBot, error) {
	items := make([]ResolvedEligibleBot, 0, len(ids))
	for _, id := range ids {
		var item ResolvedEligibleBot
		err := s.pool.QueryRow(ctx, `SELECT b.bot_id::text,b.scope_id,b.bot_name,b.active_submission_id::text,r.artifact_id FROM ai_bots b JOIN ai_submission_revisions r ON r.ai_submission_id=b.active_submission_id WHERE b.bot_id=$1 AND b.scope_id=$2 AND b.lifecycle_state='active' AND r.validation_state='ready'`, strings.TrimSpace(id), strings.TrimSpace(scopeID)).Scan(&item.BotID, &item.ScopeID, &item.BotName, &item.ActiveRevisionID, &item.ArtifactID)
		if err != nil {
			return nil, mapBotStoreError(err)
		}
		items = append(items, item)
	}
	return items, nil
}
func mapBotStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBotNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", ErrConflict, pgErr.Message)
	}
	return err
}
