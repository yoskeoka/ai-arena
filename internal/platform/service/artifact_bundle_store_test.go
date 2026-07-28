package service

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yoskeoka/ai-arena/artifactbundle"
)

func TestFilesystemBundleStoreRoundTripAndMaterialize(t *testing.T) {
	data := bundleFixture(t)
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewFilesystemBundleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), bundle); err != nil {
		t.Fatal(err)
	}
	path, err := store.Materialize(context.Background(), bundle.Digest, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "module.wasm" {
		t.Fatalf("path = %q", path)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestS3BundleStoreDoesNotOverwriteExistingDigest(t *testing.T) {
	data := bundleFixture(t)
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	artifactStore, shutdown := newTestS3ArtifactStore(t)
	defer shutdown()
	store, err := NewS3BundleStore(artifactStore)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, bundle); err != nil {
		t.Fatalf("first Put() error = %v", err)
	}
	if err := store.Put(ctx, bundle); err != nil {
		t.Fatalf("idempotent Put() error = %v", err)
	}
	conflicting := bundle
	conflicting.Bytes = []byte("different bytes")
	if err := store.Put(ctx, conflicting); err == nil {
		t.Fatal("Put accepted conflicting bytes for an existing digest")
	}
	stored, err := store.Read(ctx, bundle.Digest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, bundle.Bytes) {
		t.Fatal("Put overwrote the existing bundle bytes")
	}
}

func bundleFixture(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	entry, _ := writer.Create("manifest.json")
	_, _ = entry.Write([]byte(`{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"test","game_version":"2.0.0","runtime":{"kind":"wasm-wasi","module":"module.wasm"}}`))
	entry, _ = writer.Create("module.wasm")
	_, _ = entry.Write([]byte{0, 97, 115, 109, 1, 0, 0, 0})
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
