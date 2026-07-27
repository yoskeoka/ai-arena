package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ai-arena/artifactbundle"
)

// BundleStore persists validated immutable bundle bytes by their digest.
type BundleStore interface {
	Put(context.Context, artifactbundle.Bundle) error
	Read(context.Context, string) ([]byte, error)
	Materialize(context.Context, string, string) (string, error)
}

// FilesystemBundleStore is the local content-addressed implementation.
type FilesystemBundleStore struct{ root string }

func NewFilesystemBundleStore(root string) (*FilesystemBundleStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("service: bundle store root is required")
	}
	return &FilesystemBundleStore{root: root}, nil
}

func (s *FilesystemBundleStore) Put(_ context.Context, bundle artifactbundle.Bundle) error {
	path, err := s.pathFor(bundle.Digest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if string(existing) != string(bundle.Bytes) {
			return fmt.Errorf("service: digest collision for %s", bundle.Digest)
		}
		return nil
	}
	return os.WriteFile(path, bundle.Bytes, 0o600)
}

func (s *FilesystemBundleStore) Read(_ context.Context, digest string) ([]byte, error) {
	path, err := s.pathFor(digest)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

// Materialize extracts a verified bundle into a worker-private directory.
func (s *FilesystemBundleStore) Materialize(ctx context.Context, digest, destination string) (string, error) {
	data, err := s.Read(ctx, digest)
	if err != nil {
		return "", err
	}
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		return "", err
	}
	if bundle.Digest != digest {
		return "", fmt.Errorf("service: stored bundle digest mismatch")
	}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return "", err
	}
	// The fixed layout has already rejected every non-regular entry.
	for _, item := range []struct {
		name string
		data []byte
	}{{"manifest.json", artifactbundle.ManifestJSON(bundle.Manifest)}, {bundle.Manifest.Runtime.Module, artifactbundle.ModuleBytes(data, bundle.Manifest.Runtime.Module)}} {
		if item.data == nil {
			return "", fmt.Errorf("service: bundle module missing")
		}
		if err := os.WriteFile(filepath.Join(destination, item.name), item.data, 0o600); err != nil {
			return "", err
		}
	}
	return filepath.Join(destination, bundle.Manifest.Runtime.Module), nil
}

func (s *FilesystemBundleStore) pathFor(digest string) (string, error) {
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		return "", fmt.Errorf("service: invalid artifact digest %q", digest)
	}
	return filepath.Join(s.root, "sha256", strings.TrimPrefix(digest, "sha256:")+".zip"), nil
}
