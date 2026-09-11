package service

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/yoskeoka/ai-arena/artifactbundle"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func TestArtifactAdmissionRegistersGameRelease(t *testing.T) {
	store, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	registryStore, err := registry.NewInMemoryStore()
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := registry.NewWASIResolver(store)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := registry.New(registryStore, resolver)
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewArtifactAdmissionService(store, reg)
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.RegisterGameBundle(context.Background(), gameBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	if record.GameVersion != "2.1.0" || record.ArtifactID == "" {
		t.Fatalf("record = %+v", record)
	}
	repeated, err := service.RegisterGameBundle(context.Background(), gameBundle(t))
	if err != nil || repeated.ArtifactID != record.ArtifactID {
		t.Fatalf("repeat = %+v, %v", repeated, err)
	}
}

func TestArtifactAdmissionRepairsLegacyPostgresDescriptor(t *testing.T) {
	ctx := context.Background()
	durable, err := NewPostgresDescriptorStore(ctx, postgresTestDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(durable.Close)
	if _, err := durable.pool.Exec(ctx, "TRUNCATE game_releases CASCADE"); err != nil {
		t.Fatal(err)
	}
	data := gameBundle(t)
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	record := registry.DescriptorRecord{RegistryKey: registry.RegistryKey{GameID: bundle.Manifest.GameID, GameVersionMajor: 2}, GameID: bundle.Manifest.GameID, GameVersion: bundle.Manifest.GameVersion, ArtifactID: bundle.Digest, BuildMode: registry.BuildModeWASMWASI, BuilderID: "artifact/" + bundle.Digest, RuntimeArgs: append([]string{}, bundle.Manifest.Runtime.Args...), MemoryLimitPages: bundle.Manifest.Runtime.MemoryLimitPages, BuildConstraints: registry.BuildConstraints{SupportedRulesets: []string{"regular"}}}
	seedLegacyDescriptor(t, ctx, durable, record)
	bundles, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reg, err := registry.NewWASIOverlay(bundles, durable)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := NewArtifactAdmissionService(bundles, reg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admission.RegisterGameBundle(ctx, data); err != nil {
		t.Fatalf("RegisterGameBundle() repair error = %v", err)
	}
	if _, err := reg.LookupArtifact(ctx, bundle.Digest); err != nil {
		t.Fatalf("LookupArtifact() after repair error = %v", err)
	}
}

func gameBundle(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	entry, _ := writer.Create("manifest.json")
	_, _ = entry.Write([]byte(`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.1.0","rulesets":[{"ruleset_version":"regular"}],"runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`))
	entry, _ = writer.Create("module.wasm")
	_, _ = entry.Write([]byte{0, 97, 115, 109, 1, 0, 0, 0})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func aiBundle(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	entry, _ := writer.Create("manifest.json")
	_, _ = entry.Write([]byte(`{"schema_version":"arena-bundle/v1","artifact_kind":"ai","ai_id":"test-ai","game_id":"janken","game_version":"2.1.0","runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`))
	entry, _ = writer.Create("module.wasm")
	_, _ = entry.Write([]byte{0, 97, 115, 109, 1, 0, 0, 0})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
