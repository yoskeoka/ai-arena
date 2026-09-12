package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
)

// PostgresPublicStateStore keeps latest exported state durable across processes.
type PostgresPublicStateStore struct {
	pool *pgxpool.Pool
}

// NewPostgresPublicStateStore opens the durable public-state backend.
func NewPostgresPublicStateStore(ctx context.Context, dsn string) (*PostgresPublicStateStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("service: postgres dsn is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("service: open postgres public state store: %w", err)
	}
	var marker int
	if err := pool.QueryRow(ctx, "SELECT 1 FROM public_match_states LIMIT 1").Scan(&marker); err != nil && err != pgx.ErrNoRows {
		pool.Close()
		return nil, fmt.Errorf("service: postgres public state schema is not applied: %w", err)
	}
	return &PostgresPublicStateStore{pool: pool}, nil
}

// Close releases database connections.
func (s *PostgresPublicStateStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

// Publish atomically increments a run-local version and stores the safe snapshot.
func (s *PostgresPublicStateStore) Publish(ctx context.Context, matchID, runID string, snapshot game.ExportedSnapshot) (PublicState, error) {
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(runID) == "" {
		return PublicState{}, fmt.Errorf("service: public state match_id and run_id are required")
	}
	snapshot.MatchID = matchID
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return PublicState{}, fmt.Errorf("service: encode exported snapshot: %w", err)
	}
	var version int64
	err = s.pool.QueryRow(ctx, `
INSERT INTO public_match_states (match_id, run_id, version, exported_snapshot_json)
VALUES ($1, $2, 1, $3)
ON CONFLICT (match_id, run_id) DO UPDATE
SET version = public_match_states.version + 1,
    exported_snapshot_json = EXCLUDED.exported_snapshot_json,
    updated_at = NOW()
RETURNING version`, matchID, runID, payload).Scan(&version)
	if err != nil {
		return PublicState{}, fmt.Errorf("service: publish public state: %w", err)
	}
	return PublicState{MatchID: matchID, RunID: runID, Version: version, Snapshot: cloneExportedSnapshot(snapshot)}, nil
}

// Latest retrieves the latest durable state for one selected run.
func (s *PostgresPublicStateStore) Latest(ctx context.Context, matchID, runID string) (PublicState, bool, error) {
	var version int64
	var payload []byte
	err := s.pool.QueryRow(ctx, `SELECT version, exported_snapshot_json FROM public_match_states WHERE match_id = $1 AND run_id = $2`, matchID, runID).Scan(&version, &payload)
	if err != nil {
		if err == pgx.ErrNoRows {
			return PublicState{}, false, nil
		}
		return PublicState{}, false, fmt.Errorf("service: read public state: %w", err)
	}
	var snapshot game.ExportedSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return PublicState{}, false, fmt.Errorf("service: decode public state: %w", err)
	}
	return PublicState{MatchID: matchID, RunID: runID, Version: version, Snapshot: snapshot}, true, nil
}
