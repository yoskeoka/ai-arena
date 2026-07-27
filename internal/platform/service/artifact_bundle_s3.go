package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ai-arena/artifactbundle"
)

// S3BundleStore implements the immutable bundle contract through an S3/R2 object store.
type S3BundleStore struct{ store *S3ArtifactStore }

func NewS3BundleStore(store *S3ArtifactStore) (*S3BundleStore, error) {
	if store == nil {
		return nil, fmt.Errorf("service: S3 artifact store is required")
	}
	return &S3BundleStore{store: store}, nil
}

func (s *S3BundleStore) Put(ctx context.Context, bundle artifactbundle.Bundle) error {
	key, err := bundleObjectKey(bundle.Digest)
	if err != nil {
		return err
	}
	_, err = s.store.PutBytes(ctx, key, bundle.Bytes, "application/zip")
	return err
}

func (s *S3BundleStore) Read(ctx context.Context, digest string) ([]byte, error) {
	key, err := bundleObjectKey(digest)
	if err != nil {
		return nil, err
	}
	return s.store.ReadLocator(ctx, s.store.ObjectLocator(key))
}

func (s *S3BundleStore) Materialize(ctx context.Context, digest, destination string) (string, error) {
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
	module := artifactbundle.ModuleBytes(data, bundle.Manifest.Runtime.Module)
	if module == nil {
		return "", fmt.Errorf("service: bundle module missing")
	}
	path := filepath.Join(destination, bundle.Manifest.Runtime.Module)
	// #nosec G703 -- module name is validated as a single root entry by artifactbundle.Read.
	if err := os.WriteFile(path, module, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func bundleObjectKey(digest string) (string, error) {
	if !strings.HasPrefix(digest, "sha256:") || len(digest) != len("sha256:")+64 {
		return "", fmt.Errorf("service: invalid artifact digest %q", digest)
	}
	return "bundles/sha256/" + strings.TrimPrefix(digest, "sha256:") + ".zip", nil
}
