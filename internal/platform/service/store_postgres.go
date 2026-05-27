package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresQueueStoreSchema = `
CREATE TABLE IF NOT EXISTS service_queue_records (
    submission_id TEXT PRIMARY KEY,
    queue_order BIGSERIAL NOT NULL UNIQUE,
    match_id TEXT NOT NULL,
    game_id TEXT NOT NULL,
    game_version TEXT NOT NULL,
    ruleset_version TEXT NOT NULL,
    players_json JSONB NOT NULL,
    output_dir TEXT NOT NULL,
    attempt_count INTEGER NOT NULL,
    state TEXT NOT NULL,
    worker_id TEXT,
    terminal_json JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

// PostgresQueueStore keeps queue state in PostgreSQL for cross-process durability.
type PostgresQueueStore struct {
	pool *pgxpool.Pool
}

// NewPostgresQueueStore constructs a durable PostgreSQL-backed queue store.
func NewPostgresQueueStore(ctx context.Context, dsn string) (*PostgresQueueStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("service: postgres dsn is required")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("service: open postgres queue store: %w", err)
	}
	if _, err := pool.Exec(ctx, postgresQueueStoreSchema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("service: init postgres queue store schema: %w", err)
	}

	return &PostgresQueueStore{pool: pool}, nil
}

// Close releases PostgreSQL connections held by the queue store.
func (s *PostgresQueueStore) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

// Enqueue appends one admitted submission in queued state.
func (s *PostgresQueueStore) Enqueue(ctx context.Context, submission MatchSubmission) (QueueRecord, error) {
	playersJSON, err := json.Marshal(submission.Players)
	if err != nil {
		return QueueRecord{}, fmt.Errorf("service: marshal submitted players: %w", err)
	}

	const query = `
INSERT INTO service_queue_records (
    submission_id,
    match_id,
    game_id,
    game_version,
    ruleset_version,
    players_json,
    output_dir,
    attempt_count,
    state
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);`
	_, err = s.pool.Exec(
		ctx,
		query,
		submission.SubmissionID,
		submission.MatchID,
		submission.Game.GameID,
		submission.Game.GameVersion,
		submission.Game.RulesetVersion,
		playersJSON,
		submission.OutputDir,
		submission.AttemptCount,
		StateQueued,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return QueueRecord{}, fmt.Errorf("service: submission_id %q already exists", submission.SubmissionID)
		}
		return QueueRecord{}, fmt.Errorf("service: enqueue submission: %w", err)
	}

	return QueueRecord{
		Submission: cloneMatchSubmission(submission),
		State:      StateQueued,
	}, nil
}

// Claim moves the next queued record to leased for the supplied worker id.
func (s *PostgresQueueStore) Claim(ctx context.Context, workerID string) (QueueRecord, error) {
	if strings.TrimSpace(workerID) == "" {
		return QueueRecord{}, fmt.Errorf("service: worker_id is required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return QueueRecord{}, fmt.Errorf("service: begin claim tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const query = `
WITH next_record AS (
    SELECT submission_id
    FROM service_queue_records
    WHERE state = $1
    ORDER BY queue_order
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE service_queue_records AS records
SET state = $2, worker_id = $3, updated_at = NOW()
FROM next_record
WHERE records.submission_id = next_record.submission_id
RETURNING
    records.submission_id,
    records.match_id,
    records.game_id,
    records.game_version,
    records.ruleset_version,
    records.players_json,
    records.output_dir,
    records.attempt_count,
    records.state,
    records.worker_id,
    records.terminal_json;`
	record, err := scanQueueRecord(tx.QueryRow(ctx, query, StateQueued, StateLeased, workerID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return QueueRecord{}, ErrNoQueuedSubmission
		}
		return QueueRecord{}, fmt.Errorf("service: claim queued submission: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return QueueRecord{}, fmt.Errorf("service: commit claim tx: %w", err)
	}
	return record, nil
}

// Update replaces one existing record after validating the lifecycle transition.
func (s *PostgresQueueStore) Update(ctx context.Context, next QueueRecord) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("service: begin update tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	current, err := s.loadRecordTx(ctx, tx, next.Submission.SubmissionID, true)
	if err != nil {
		return err
	}
	if err := ValidateTransition(current.State, next.State); err != nil {
		return err
	}

	playersJSON, err := json.Marshal(next.Submission.Players)
	if err != nil {
		return fmt.Errorf("service: marshal submitted players: %w", err)
	}
	var terminalJSON []byte
	if next.Terminal != nil {
		terminalJSON, err = json.Marshal(next.Terminal)
		if err != nil {
			return fmt.Errorf("service: marshal terminal artifacts: %w", err)
		}
	}

	const query = `
UPDATE service_queue_records
SET
    match_id = $2,
    game_id = $3,
    game_version = $4,
    ruleset_version = $5,
    players_json = $6,
    output_dir = $7,
    attempt_count = $8,
    state = $9,
    worker_id = $10,
    terminal_json = $11,
    updated_at = NOW()
WHERE submission_id = $1;`
	if _, err := tx.Exec(
		ctx,
		query,
		next.Submission.SubmissionID,
		next.Submission.MatchID,
		next.Submission.Game.GameID,
		next.Submission.Game.GameVersion,
		next.Submission.Game.RulesetVersion,
		playersJSON,
		next.Submission.OutputDir,
		next.Submission.AttemptCount,
		next.State,
		workerIDFromLease(next.Lease),
		jsonOrNil(terminalJSON),
	); err != nil {
		return fmt.Errorf("service: update queue record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("service: commit update tx: %w", err)
	}
	return nil
}

// CancelQueued moves one queued record into canceled.
func (s *PostgresQueueStore) CancelQueued(ctx context.Context, submissionID string) (QueueRecord, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return QueueRecord{}, fmt.Errorf("service: begin cancel tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	record, err := s.loadRecordTx(ctx, tx, submissionID, true)
	if err != nil {
		return QueueRecord{}, err
	}
	if record.State != StateQueued {
		return QueueRecord{}, fmt.Errorf("service: only queued submissions can be canceled")
	}
	if err := ValidateTransition(record.State, StateCanceled); err != nil {
		return QueueRecord{}, err
	}

	record.State = StateCanceled
	record.Lease = nil
	record.Terminal = nil

	const query = `
UPDATE service_queue_records
SET state = $2, worker_id = NULL, terminal_json = NULL, updated_at = NOW()
WHERE submission_id = $1;`
	if _, err := tx.Exec(ctx, query, submissionID, StateCanceled); err != nil {
		return QueueRecord{}, fmt.Errorf("service: cancel queue record: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return QueueRecord{}, fmt.Errorf("service: commit cancel tx: %w", err)
	}
	return cloneQueueRecord(record), nil
}

func (s *PostgresQueueStore) loadRecord(ctx context.Context, submissionID string) (QueueRecord, error) {
	return s.loadRecordTx(ctx, s.pool, submissionID, false)
}

func (s *PostgresQueueStore) loadRecordTx(ctx context.Context, querier queueRecordQuerier, submissionID string, forUpdate bool) (QueueRecord, error) {
	query := `
SELECT
    submission_id,
    match_id,
    game_id,
    game_version,
    ruleset_version,
    players_json,
    output_dir,
    attempt_count,
    state,
    worker_id,
    terminal_json
FROM service_queue_records
WHERE submission_id = $1`
	if forUpdate {
		query += " FOR UPDATE"
	}

	record, err := scanQueueRecord(querier.QueryRow(ctx, query, submissionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return QueueRecord{}, ErrQueueRecordNotFound
		}
		return QueueRecord{}, fmt.Errorf("service: load queue record: %w", err)
	}
	return record, nil
}

type queueRecordQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func scanQueueRecord(row pgx.Row) (QueueRecord, error) {
	var (
		submissionID   string
		matchID        string
		gameID         string
		gameVersion    string
		rulesetVersion string
		playersJSON    []byte
		outputDir      string
		attemptCount   int
		state          string
		workerID       *string
		terminalJSON   []byte
	)
	if err := row.Scan(
		&submissionID,
		&matchID,
		&gameID,
		&gameVersion,
		&rulesetVersion,
		&playersJSON,
		&outputDir,
		&attemptCount,
		&state,
		&workerID,
		&terminalJSON,
	); err != nil {
		return QueueRecord{}, err
	}

	var players []SubmittedPlayer
	if err := json.Unmarshal(playersJSON, &players); err != nil {
		return QueueRecord{}, fmt.Errorf("service: decode submitted players: %w", err)
	}

	record := QueueRecord{
		Submission: MatchSubmission{
			SubmissionID: submissionID,
			MatchID:      matchID,
			OutputDir:    outputDir,
			AttemptCount: attemptCount,
			Players:      players,
		},
		State: LifecycleState(state),
	}
	record.Submission.Game.GameID = gameID
	record.Submission.Game.GameVersion = gameVersion
	record.Submission.Game.RulesetVersion = rulesetVersion

	if workerID != nil && strings.TrimSpace(*workerID) != "" {
		record.Lease = &WorkerLease{WorkerID: *workerID}
	}
	if len(terminalJSON) > 0 {
		var terminal TerminalArtifacts
		if err := json.Unmarshal(terminalJSON, &terminal); err != nil {
			return QueueRecord{}, fmt.Errorf("service: decode terminal artifacts: %w", err)
		}
		record.Terminal = &terminal
	}

	return cloneQueueRecord(record), nil
}

func workerIDFromLease(lease *WorkerLease) any {
	if lease == nil || strings.TrimSpace(lease.WorkerID) == "" {
		return nil
	}
	return lease.WorkerID
}

func jsonOrNil(data []byte) any {
	if len(data) == 0 {
		return nil
	}
	return data
}
