package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"golang.org/x/mod/semver"
)

// PostgresDescriptorStore is the durable external registry tier.
type PostgresDescriptorStore struct{ pool *pgxpool.Pool }

// NewPostgresDescriptorStore opens a PostgreSQL-backed durable descriptor store.
func NewPostgresDescriptorStore(ctx context.Context, dsn string) (*PostgresDescriptorStore, error) {
	pool, err := pgxpool.New(ctx, strings.TrimSpace(dsn))
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresDescriptorStore{pool: pool}, nil
}

// Close releases the PostgreSQL connection pool used by the descriptor store.
func (s *PostgresDescriptorStore) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

// Register durably records an admitted descriptor and verifies its immutable metadata.
func (s *PostgresDescriptorStore) Register(ctx context.Context, record registry.DescriptorRecord) error {
	if err := registry.ValidateDescriptorRecord(record); err != nil {
		return err
	}
	rulesets, err := json.Marshal(record.BuildConstraints.SupportedRulesets)
	if err != nil {
		return err
	}
	args, err := json.Marshal(record.RuntimeArgs)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO game_releases(release_id,game_id,game_version,artifact_id,build_mode,builder_id,supported_rulesets,runtime_args,memory_limit_pages,source,source_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,'manual','') ON CONFLICT (artifact_id) DO NOTHING`, uuid.New(), record.GameID, record.GameVersion, record.ArtifactID, record.BuildMode, record.BuilderID, rulesets, args, record.MemoryLimitPages); err != nil {
		return err
	}
	existing, err := scanStoredDescriptor(tx.QueryRow(ctx, `SELECT game_id,game_version,artifact_id,build_mode,builder_id,supported_rulesets,runtime_args,memory_limit_pages FROM game_releases WHERE artifact_id=$1 FOR UPDATE`, record.ArtifactID))
	if err != nil {
		return err
	}
	if !sameImmutableDescriptor(existing.record, record) {
		return fmt.Errorf("registry: conflicting descriptor for artifact %q", record.ArtifactID)
	}
	if existing.runtimeArgs == nil && existing.memoryLimitPages == nil {
		if _, err := tx.Exec(ctx, `UPDATE game_releases SET runtime_args=$1,memory_limit_pages=$2 WHERE artifact_id=$3`, args, record.MemoryLimitPages, record.ArtifactID); err != nil {
			return err
		}
	} else {
		complete, err := existing.descriptorRecord()
		if err != nil {
			return err
		}
		if !sameDescriptorRecord(complete, record) {
			return fmt.Errorf("registry: conflicting descriptor for artifact %q", record.ArtifactID)
		}
	}
	return tx.Commit(ctx)
}

// Lookup resolves the latest descriptor for a game and semver major from PostgreSQL.
func (s *PostgresDescriptorStore) Lookup(ctx context.Context, key registry.RegistryKey) (registry.DescriptorRecord, error) {
	if key.GameID == "" || key.GameVersionMajor <= 0 {
		return registry.DescriptorRecord{}, fmt.Errorf("registry: invalid lookup key")
	}
	rows, err := s.pool.Query(ctx, `SELECT game_id,game_version,artifact_id,build_mode,builder_id,supported_rulesets,runtime_args,memory_limit_pages FROM game_releases WHERE game_id=$1`, key.GameID)
	if err != nil {
		return registry.DescriptorRecord{}, err
	}
	defer rows.Close()
	var records []registry.DescriptorRecord
	for rows.Next() {
		record, scanErr := scanDescriptor(rows)
		if scanErr != nil {
			return registry.DescriptorRecord{}, scanErr
		}
		if record.RegistryKey.GameVersionMajor == key.GameVersionMajor {
			records = append(records, record)
		}
	}
	if err := rows.Err(); err != nil {
		return registry.DescriptorRecord{}, err
	}
	if len(records) == 0 {
		return registry.DescriptorRecord{}, fmt.Errorf("%w: game %q major %d", registry.ErrRecordNotFound, key.GameID, key.GameVersionMajor)
	}
	latest := records[0]
	for _, record := range records[1:] {
		if semver.Compare(normalizeVersion(record.GameVersion), normalizeVersion(latest.GameVersion)) > 0 {
			latest = record
		}
	}
	return latest, nil
}

// LookupArtifact resolves an exact admitted descriptor by its immutable artifact identity.
func (s *PostgresDescriptorStore) LookupArtifact(ctx context.Context, artifactID string) (registry.DescriptorRecord, error) {
	return s.lookupArtifact(ctx, artifactID)
}
func (s *PostgresDescriptorStore) lookupArtifact(ctx context.Context, artifactID string) (registry.DescriptorRecord, error) {
	record, err := scanDescriptor(s.pool.QueryRow(ctx, `SELECT game_id,game_version,artifact_id,build_mode,builder_id,supported_rulesets,runtime_args,memory_limit_pages FROM game_releases WHERE artifact_id=$1`, artifactID))
	if errors.Is(err, pgx.ErrNoRows) {
		return registry.DescriptorRecord{}, fmt.Errorf("%w: unsupported artifact %q", registry.ErrRecordNotFound, artifactID)
	}
	return record, err
}

type descriptorRow interface{ Scan(...any) error }

type storedDescriptor struct {
	record           registry.DescriptorRecord
	runtimeArgs      *[]byte
	memoryLimitPages *int32
}

func scanStoredDescriptor(row descriptorRow) (storedDescriptor, error) {
	var stored storedDescriptor
	var record registry.DescriptorRecord
	var rulesets []byte
	var args *[]byte
	var memory *int32
	if err := row.Scan(&record.GameID, &record.GameVersion, &record.ArtifactID, &record.BuildMode, &record.BuilderID, &rulesets, &args, &memory); err != nil {
		return stored, err
	}
	major, err := catalog.MajorVersion(record.GameVersion)
	if err != nil {
		return stored, fmt.Errorf("registry: invalid durable descriptor: %w", err)
	}
	record.RegistryKey = registry.RegistryKey{GameID: record.GameID, GameVersionMajor: major}
	if err := json.Unmarshal(rulesets, &record.BuildConstraints.SupportedRulesets); err != nil {
		return stored, err
	}
	return storedDescriptor{record: record, runtimeArgs: args, memoryLimitPages: memory}, nil
}

func (stored storedDescriptor) descriptorRecord() (registry.DescriptorRecord, error) {
	record := stored.record
	if stored.runtimeArgs == nil || stored.memoryLimitPages == nil {
		return record, fmt.Errorf("registry: incomplete durable descriptor")
	}
	if *stored.memoryLimitPages < 0 {
		return record, fmt.Errorf("registry: invalid durable memory page limit")
	}
	record.MemoryLimitPages = uint32(*stored.memoryLimitPages)
	if err := json.Unmarshal(*stored.runtimeArgs, &record.RuntimeArgs); err != nil {
		return record, err
	}
	if err := registry.ValidateDescriptorRecord(record); err != nil {
		return record, fmt.Errorf("registry: invalid durable descriptor: %w", err)
	}
	return record, nil
}

func scanDescriptor(row descriptorRow) (registry.DescriptorRecord, error) {
	stored, err := scanStoredDescriptor(row)
	if err != nil {
		return registry.DescriptorRecord{}, err
	}
	return stored.descriptorRecord()
}

func normalizeVersion(v string) string { return "v" + strings.TrimPrefix(v, "v") }
func sameDescriptorRecord(a, b registry.DescriptorRecord) bool {
	return sameImmutableDescriptor(a, b) && a.MemoryLimitPages == b.MemoryLimitPages && strings.Join(a.RuntimeArgs, "\x00") == strings.Join(b.RuntimeArgs, "\x00")
}

func sameImmutableDescriptor(a, b registry.DescriptorRecord) bool {
	return a.RegistryKey == b.RegistryKey && a.GameID == b.GameID && a.GameVersion == b.GameVersion && a.ArtifactID == b.ArtifactID && a.BuildMode == b.BuildMode && a.BuilderID == b.BuilderID && strings.Join(a.BuildConstraints.SupportedRulesets, "\x00") == strings.Join(b.BuildConstraints.SupportedRulesets, "\x00")
}

var _ registry.RegistryStore = (*PostgresDescriptorStore)(nil)
var _ registry.ArtifactRecordLookup = (*PostgresDescriptorStore)(nil)
var _ registry.RecordRegistrar = (*PostgresDescriptorStore)(nil)
