package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultArtifactReaderReadBoundedUnderRejectsPathsOutsideOutputDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	inside := filepath.Join(root, "match-1", "public-replay.json")
	if err := os.MkdirAll(filepath.Dir(inside), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(inside, []byte(`{"moves":["d3"]}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	outside := filepath.Join(t.TempDir(), "private.json")
	if err := os.WriteFile(outside, []byte(`{"private":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	reader := NewDefaultArtifactReader(nil)
	data, err := reader.ReadBoundedUnder(context.Background(), inside, root, 1024)
	if err != nil {
		t.Fatalf("ReadBoundedUnder() inside output directory error = %v", err)
	}
	if string(data) != `{"moves":["d3"]}` {
		t.Fatalf("ReadBoundedUnder() = %q", data)
	}
	if _, err := reader.ReadBoundedUnder(context.Background(), outside, root, 1024); err == nil {
		t.Fatal("ReadBoundedUnder() outside output directory unexpectedly succeeded")
	}
}

func TestDefaultArtifactReaderReadBoundedUnderRejectsEscapingSymlink(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "private.json")
	if err := os.WriteFile(outside, []byte(`{"private":true}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	link := filepath.Join(root, "public-replay.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	reader := NewDefaultArtifactReader(nil)
	if _, err := reader.ReadBoundedUnder(context.Background(), link, root, 1024); err == nil {
		t.Fatal("ReadBoundedUnder() through escaping symlink unexpectedly succeeded")
	}
}
