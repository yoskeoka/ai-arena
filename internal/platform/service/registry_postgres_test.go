package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func TestPostgresDescriptorStorePersistsExactRuntimeMetadata(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()
	first, err := NewPostgresDescriptorStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := first.pool.Exec(ctx, "TRUNCATE game_releases CASCADE"); err != nil {
		first.Close()
		t.Fatal(err)
	}
	record := registry.DescriptorRecord{RegistryKey: registry.RegistryKey{GameID: "durable-game", GameVersionMajor: 2}, GameID: "durable-game", GameVersion: "2.4.0", ArtifactID: "sha256:durable", BuildMode: registry.BuildModeWASMWASI, BuilderID: "artifact/sha256:durable", RuntimeArgs: []string{"--safe", "value"}, MemoryLimitPages: 17, BuildConstraints: registry.BuildConstraints{SupportedRulesets: []string{"regular"}}}
	if err := first.Register(ctx, record); err != nil {
		first.Close()
		t.Fatal(err)
	}
	if err := first.Register(ctx, record); err != nil {
		first.Close()
		t.Fatalf("idempotent Register: %v", err)
	}
	first.Close()

	second, err := NewPostgresDescriptorStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(second.Close)
	byKey, err := second.Lookup(ctx, record.RegistryKey)
	if err != nil {
		t.Fatal(err)
	}
	byArtifact, err := second.LookupArtifact(ctx, record.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	if !sameDescriptorRecord(byKey, record) || !sameDescriptorRecord(byArtifact, record) {
		t.Fatalf("durable records = %#v / %#v, want %#v", byKey, byArtifact, record)
	}
	if _, err := second.LookupArtifact(ctx, "sha256:missing"); !errors.Is(err, registry.ErrRecordNotFound) {
		t.Fatalf("missing artifact error = %v", err)
	}
}

func TestPostgresDescriptorStoreRepairsLegacyRuntimeMetadataOnlyForSameArtifact(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()
	store, err := NewPostgresDescriptorStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if _, err := store.pool.Exec(ctx, "TRUNCATE game_releases CASCADE"); err != nil {
		t.Fatal(err)
	}
	record := registry.DescriptorRecord{RegistryKey: registry.RegistryKey{GameID: "legacy-game", GameVersionMajor: 1}, GameID: "legacy-game", GameVersion: "1.0.0", ArtifactID: "sha256:legacy", BuildMode: registry.BuildModeWASMWASI, BuilderID: "artifact/sha256:legacy", RuntimeArgs: []string{"--safe"}, MemoryLimitPages: 9, BuildConstraints: registry.BuildConstraints{SupportedRulesets: []string{"regular"}}}
	seedLegacyDescriptor(t, ctx, store, record)
	if _, err := store.Lookup(ctx, record.RegistryKey); err == nil {
		t.Fatal("Lookup() returned nil for incomplete legacy descriptor")
	}
	if _, err := store.LookupArtifact(ctx, record.ArtifactID); err == nil {
		t.Fatal("LookupArtifact() returned nil for incomplete legacy descriptor")
	}
	if err := store.Register(ctx, record); err != nil {
		t.Fatalf("Register() repair error = %v", err)
	}
	byKey, err := store.Lookup(ctx, record.RegistryKey)
	if err != nil {
		t.Fatal(err)
	}
	byArtifact, err := store.LookupArtifact(ctx, record.ArtifactID)
	if err != nil {
		t.Fatal(err)
	}
	if !sameDescriptorRecord(byKey, record) || !sameDescriptorRecord(byArtifact, record) {
		t.Fatalf("repaired records = %#v / %#v, want %#v", byKey, byArtifact, record)
	}
	if err := store.Register(ctx, record); err != nil {
		t.Fatalf("idempotent Register() after repair error = %v", err)
	}
}

func TestPostgresDescriptorStoreDoesNotRepairImmutableMismatch(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()
	store, err := NewPostgresDescriptorStore(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if _, err := store.pool.Exec(ctx, "TRUNCATE game_releases CASCADE"); err != nil {
		t.Fatal(err)
	}
	record := registry.DescriptorRecord{RegistryKey: registry.RegistryKey{GameID: "legacy-game", GameVersionMajor: 1}, GameID: "legacy-game", GameVersion: "1.0.0", ArtifactID: "sha256:legacy", BuildMode: registry.BuildModeWASMWASI, BuilderID: "artifact/sha256:legacy", RuntimeArgs: []string{"--safe"}, MemoryLimitPages: 9, BuildConstraints: registry.BuildConstraints{SupportedRulesets: []string{"regular"}}}
	seedLegacyDescriptor(t, ctx, store, record)
	for _, name := range []string{"game", "version", "build mode", "builder", "rulesets"} {
		t.Run(name, func(t *testing.T) {
			candidate := record
			switch name {
			case "game":
				candidate.GameID = "other-game"
				candidate.RegistryKey.GameID = candidate.GameID
			case "version":
				candidate.GameVersion = "1.0.1"
			case "build mode":
				candidate.BuildMode = "other"
			case "builder":
				candidate.BuilderID = "other-builder"
			case "rulesets":
				candidate.BuildConstraints.SupportedRulesets = []string{"other"}
			}
			if err := store.Register(ctx, candidate); err == nil {
				t.Fatalf("Register(%s mismatch) returned nil", name)
			}
			var args *[]byte
			var memory *int32
			if err := store.pool.QueryRow(ctx, `SELECT runtime_args,memory_limit_pages FROM game_releases WHERE artifact_id=$1`, record.ArtifactID).Scan(&args, &memory); err != nil {
				t.Fatal(err)
			}
			if args != nil || memory != nil {
				t.Fatalf("mismatch %s repaired runtime metadata", name)
			}
		})
	}
}

func seedLegacyDescriptor(t *testing.T, ctx context.Context, store *PostgresDescriptorStore, record registry.DescriptorRecord) {
	t.Helper()
	if _, err := store.pool.Exec(ctx, `ALTER TABLE game_releases ALTER COLUMN runtime_args DROP NOT NULL, ALTER COLUMN memory_limit_pages DROP NOT NULL`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := store.pool.Exec(ctx, `DELETE FROM game_releases WHERE artifact_id=$1`, record.ArtifactID); err != nil {
			t.Error(err)
		}
		if _, err := store.pool.Exec(ctx, `ALTER TABLE game_releases ALTER COLUMN runtime_args SET NOT NULL, ALTER COLUMN memory_limit_pages SET NOT NULL`); err != nil {
			t.Error(err)
		}
	})
	if _, err := store.pool.Exec(ctx, `INSERT INTO game_releases(release_id,game_id,game_version,artifact_id,build_mode,builder_id,supported_rulesets,runtime_args,memory_limit_pages,source,source_id) VALUES($1,$2,$3,$4,$5,$6,$7,NULL,NULL,'manual','')`, uuid.New(), record.GameID, record.GameVersion, record.ArtifactID, record.BuildMode, record.BuilderID, []byte(`["regular"]`)); err != nil {
		t.Fatal(err)
	}
}
