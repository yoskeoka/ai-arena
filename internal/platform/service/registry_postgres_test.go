package service

import (
	"context"
	"errors"
	"testing"

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
