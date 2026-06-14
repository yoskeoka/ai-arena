package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	servicepostgressqlc "github.com/yoskeoka/ai-arena/internal/platform/service/postgres/sqlc"
)

const postgresQueueRecordPrimaryKey = "service_queue_records_pkey"

// PostgresQueueStore keeps queue state in PostgreSQL for cross-process durability.
type PostgresQueueStore struct {
	pool    *pgxpool.Pool
	queries *servicepostgressqlc.Queries
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
	if err := validatePostgresQueueStoreSchema(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresQueueStore{
		pool:    pool,
		queries: servicepostgressqlc.New(pool),
	}, nil
}

// Close releases PostgreSQL connections held by the queue store.
func (s *PostgresQueueStore) Close() {
	if s == nil || s.pool == nil {
		return
	}
	s.pool.Close()
}

// Enqueue appends one admitted run in queued state.
func (s *PostgresQueueStore) Enqueue(ctx context.Context, submission MatchSubmission) (QueueRecord, error) {
	if strings.TrimSpace(submission.RunID) == "" {
		return QueueRecord{}, fmt.Errorf("service: run_id is required")
	}

	playersJSON, err := json.Marshal(submission.Players)
	if err != nil {
		return QueueRecord{}, fmt.Errorf("service: marshal submitted players: %w", err)
	}
	attemptCount, err := postgresAttemptCount(submission.AttemptCount)
	if err != nil {
		return QueueRecord{}, err
	}

	err = s.queries.CreateQueueRecord(ctx, servicepostgressqlc.CreateQueueRecordParams{
		SubmissionID:   submission.RunID,
		MatchID:        submission.MatchID,
		ParentRunID:    textValue(submission.ParentRunID),
		RunKind:        string(submission.RunKind),
		Official:       submission.Official,
		GameID:         submission.Game.GameID,
		GameVersion:    submission.Game.GameVersion,
		RulesetVersion: submission.Game.RulesetVersion,
		PlayersJson:    playersJSON,
		OutputDir:      submission.OutputDir,
		AttemptCount:   attemptCount,
		State:          string(StateQueued),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == postgresQueueRecordPrimaryKey {
			return QueueRecord{}, fmt.Errorf("service: run_id %q already exists", submission.RunID)
		}
		return QueueRecord{}, fmt.Errorf("service: enqueue run: %w", err)
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

	row, err := s.queries.WithTx(tx).ClaimNextQueueRecord(ctx, servicepostgressqlc.ClaimNextQueueRecordParams{
		LeasedState: string(StateLeased),
		WorkerID:    textValue(workerID),
		QueuedState: string(StateQueued),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return QueueRecord{}, ErrNoQueuedSubmission
		}
		return QueueRecord{}, fmt.Errorf("service: claim queued run: %w", err)
	}
	record, err := queueRecordFromFields(
		row.SubmissionID,
		row.MatchID,
		row.ParentRunID,
		row.RunKind,
		row.Official,
		row.GameID,
		row.GameVersion,
		row.RulesetVersion,
		row.PlayersJson,
		row.OutputDir,
		row.AttemptCount,
		row.State,
		row.WorkerID,
		row.TerminalJson,
	)
	if err != nil {
		return QueueRecord{}, err
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

	current, err := s.loadRecordTx(ctx, tx, next.Submission.RunID, true)
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
	attemptCount, err := postgresAttemptCount(next.Submission.AttemptCount)
	if err != nil {
		return err
	}

	if err := s.queries.WithTx(tx).UpdateQueueRecord(ctx, servicepostgressqlc.UpdateQueueRecordParams{
		MatchID:        next.Submission.MatchID,
		ParentRunID:    textValue(next.Submission.ParentRunID),
		RunKind:        string(next.Submission.RunKind),
		Official:       next.Submission.Official,
		GameID:         next.Submission.Game.GameID,
		GameVersion:    next.Submission.Game.GameVersion,
		RulesetVersion: next.Submission.Game.RulesetVersion,
		PlayersJson:    playersJSON,
		OutputDir:      next.Submission.OutputDir,
		AttemptCount:   attemptCount,
		State:          string(next.State),
		WorkerID:       textValueFromLease(next.Lease),
		TerminalJson:   terminalJSON,
		SubmissionID:   next.Submission.RunID,
	}); err != nil {
		return fmt.Errorf("service: update queue record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("service: commit update tx: %w", err)
	}
	return nil
}

// CancelQueued moves one queued record into canceled.
func (s *PostgresQueueStore) CancelQueued(ctx context.Context, runID string) (QueueRecord, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return QueueRecord{}, fmt.Errorf("service: begin cancel tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	record, err := s.loadRecordTx(ctx, tx, runID, true)
	if err != nil {
		return QueueRecord{}, err
	}
	if record.State != StateQueued {
		return QueueRecord{}, fmt.Errorf("service: only queued runs can be canceled")
	}
	if err := ValidateTransition(record.State, StateCanceled); err != nil {
		return QueueRecord{}, err
	}

	record.State = StateCanceled
	record.Lease = nil
	record.Terminal = nil

	if err := s.queries.WithTx(tx).CancelQueueRecord(ctx, servicepostgressqlc.CancelQueueRecordParams{
		State:        string(StateCanceled),
		SubmissionID: runID,
	}); err != nil {
		return QueueRecord{}, fmt.Errorf("service: cancel queue record: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return QueueRecord{}, fmt.Errorf("service: commit cancel tx: %w", err)
	}
	return cloneQueueRecord(record), nil
}

// Get returns one existing queue record by run id.
func (s *PostgresQueueStore) Get(ctx context.Context, runID string) (QueueRecord, error) {
	return s.loadRecord(ctx, runID)
}

// List returns queue records in durable queue order.
func (s *PostgresQueueStore) List(ctx context.Context) ([]QueueRecord, error) {
	rows, err := s.queries.ListQueueRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf("service: list queue records: %w", err)
	}

	records := make([]QueueRecord, 0, len(rows))
	for _, row := range rows {
		record, err := queueRecordFromFields(
			row.SubmissionID,
			row.MatchID,
			row.ParentRunID,
			row.RunKind,
			row.Official,
			row.GameID,
			row.GameVersion,
			row.RulesetVersion,
			row.PlayersJson,
			row.OutputDir,
			row.AttemptCount,
			row.State,
			row.WorkerID,
			row.TerminalJson,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

func (s *PostgresQueueStore) loadRecord(ctx context.Context, runID string) (QueueRecord, error) {
	return s.loadRecordTx(ctx, nil, runID, false)
}

func (s *PostgresQueueStore) loadRecordTx(ctx context.Context, tx pgx.Tx, runID string, forUpdate bool) (QueueRecord, error) {
	queries := s.queries
	if tx != nil {
		queries = queries.WithTx(tx)
	}
	var (
		record QueueRecord
		err    error
	)
	if forUpdate {
		row, rowErr := queries.GetQueueRecordForUpdate(ctx, runID)
		if rowErr != nil {
			err = rowErr
		} else {
			record, err = queueRecordFromFields(
				row.SubmissionID,
				row.MatchID,
				row.ParentRunID,
				row.RunKind,
				row.Official,
				row.GameID,
				row.GameVersion,
				row.RulesetVersion,
				row.PlayersJson,
				row.OutputDir,
				row.AttemptCount,
				row.State,
				row.WorkerID,
				row.TerminalJson,
			)
		}
	} else {
		row, rowErr := queries.GetQueueRecord(ctx, runID)
		if rowErr != nil {
			err = rowErr
		} else {
			record, err = queueRecordFromFields(
				row.SubmissionID,
				row.MatchID,
				row.ParentRunID,
				row.RunKind,
				row.Official,
				row.GameID,
				row.GameVersion,
				row.RulesetVersion,
				row.PlayersJson,
				row.OutputDir,
				row.AttemptCount,
				row.State,
				row.WorkerID,
				row.TerminalJson,
			)
		}
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return QueueRecord{}, ErrQueueRecordNotFound
		}
		return QueueRecord{}, fmt.Errorf("service: load queue record: %w", err)
	}
	return record, nil
}

func queueRecordFromFields(
	runID string,
	matchID string,
	parentRunID pgtype.Text,
	runKind string,
	official bool,
	gameID string,
	gameVersion string,
	rulesetVersion string,
	playersJSON []byte,
	outputDir string,
	attemptCount int32,
	state string,
	workerID pgtype.Text,
	terminalJSON []byte,
) (QueueRecord, error) {
	var players []SubmittedPlayer
	if err := json.Unmarshal(playersJSON, &players); err != nil {
		return QueueRecord{}, fmt.Errorf("service: decode submitted players: %w", err)
	}

	record := QueueRecord{
		Submission: MatchSubmission{
			RunID:        runID,
			MatchID:      matchID,
			OutputDir:    outputDir,
			AttemptCount: int(attemptCount),
			Players:      players,
			RunKind:      RunKind(runKind),
			Official:     official,
		},
		State: LifecycleState(state),
	}
	if parentRunID.Valid {
		record.Submission.ParentRunID = parentRunID.String
	}
	record.Submission.Game.GameID = gameID
	record.Submission.Game.GameVersion = gameVersion
	record.Submission.Game.RulesetVersion = rulesetVersion

	if workerID.Valid && strings.TrimSpace(workerID.String) != "" {
		record.Lease = &WorkerLease{WorkerID: workerID.String}
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

func textValue(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textValueFromLease(lease *WorkerLease) pgtype.Text {
	if lease == nil {
		return pgtype.Text{}
	}
	return textValue(lease.WorkerID)
}

func validatePostgresQueueStoreSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var marker int
	if err := pool.QueryRow(ctx, "SELECT 1 FROM service_queue_records LIMIT 1").Scan(&marker); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("service: postgres queue store schema is not applied: %w", err)
	}
	return nil
}

func postgresAttemptCount(attemptCount int) (int32, error) {
	if attemptCount < -2147483648 || attemptCount > 2147483647 {
		return 0, fmt.Errorf("service: attempt_count %d is out of int32 range", attemptCount)
	}
	return int32(attemptCount), nil
}
