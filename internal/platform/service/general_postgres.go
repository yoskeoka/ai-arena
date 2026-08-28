package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresGameRegistrationStore persists a game release and its stable ruleset scope.
type PostgresGameRegistrationStore struct{ pool *pgxpool.Pool }

func NewPostgresGameRegistrationStore(ctx context.Context, dsn string) (*PostgresGameRegistrationStore, error) {
	pool, err := pgxpool.New(ctx, strings.TrimSpace(dsn))
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresGameRegistrationStore{pool: pool}, nil
}
func (s *PostgresGameRegistrationStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *PostgresGameRegistrationStore) Save(ctx context.Context, r RegisteredGame) error {
	if r.RegistrationID == "" || r.ArtifactID == "" || r.PlayerCount < 1 || r.MaxActiveBotsPerOwner < 1 {
		return fmt.Errorf("%w: activated game artifact and ruleset limits are required", ErrBadRequest)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var releaseID string
	err = tx.QueryRow(ctx, `SELECT release_id::text FROM game_releases WHERE artifact_id=$1`, r.ArtifactID).Scan(&releaseID)
	if errors.Is(err, pgx.ErrNoRows) {
		releaseID = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO game_releases(release_id,game_id,game_version,artifact_id) VALUES($1,$2,$3,$4)`, releaseID, r.Game.GameID, r.Game.GameVersion, r.ArtifactID)
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO competition_scopes(scope_id,game_id,game_version_major,ruleset_version,active_release_id,player_count,max_active_bots_per_owner) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(scope_id) DO UPDATE SET active_release_id=EXCLUDED.active_release_id,player_count=EXCLUDED.player_count,max_active_bots_per_owner=EXCLUDED.max_active_bots_per_owner`, r.RegistrationID, r.Game.GameID, majorVersion(r.Game.GameVersion), r.Game.RulesetVersion, releaseID, r.PlayerCount, r.MaxActiveBotsPerOwner)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *PostgresGameRegistrationStore) Get(ctx context.Context, id string) (RegisteredGame, error) {
	var r RegisteredGame
	err := s.pool.QueryRow(ctx, `SELECT s.scope_id,s.game_id,r.game_version,s.ruleset_version,r.artifact_id,s.player_count,s.max_active_bots_per_owner FROM competition_scopes s JOIN game_releases r ON r.release_id=s.active_release_id WHERE s.scope_id=$1`, id).Scan(&r.RegistrationID, &r.Game.GameID, &r.Game.GameVersion, &r.Game.RulesetVersion, &r.ArtifactID, &r.PlayerCount, &r.MaxActiveBotsPerOwner)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrGameRegistrationNotFound
	}
	return r, err
}
func (s *PostgresGameRegistrationStore) List(ctx context.Context) ([]RegisteredGame, error) {
	rows, err := s.pool.Query(ctx, `SELECT s.scope_id,s.game_id,r.game_version,s.ruleset_version,r.artifact_id,s.player_count,s.max_active_bots_per_owner FROM competition_scopes s JOIN game_releases r ON r.release_id=s.active_release_id ORDER BY s.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RegisteredGame
	for rows.Next() {
		var r RegisteredGame
		if err := rows.Scan(&r.RegistrationID, &r.Game.GameID, &r.Game.GameVersion, &r.Game.RulesetVersion, &r.ArtifactID, &r.PlayerCount, &r.MaxActiveBotsPerOwner); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func majorVersion(v string) int { var n int; _, _ = fmt.Sscanf(v, "%d.", &n); return n }

var _ GameRegistrationStore = (*PostgresGameRegistrationStore)(nil)
