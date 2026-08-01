package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yoskeoka/ai-arena/artifactbundle"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func readArtifactBundle(path string) (artifactbundle.Bundle, error) {
	// #nosec G304 -- the runner path is explicitly supplied by the caller.
	data, err := os.ReadFile(path)
	if err != nil {
		return artifactbundle.Bundle{}, fmt.Errorf("read artifact bundle %s: %w", path, err)
	}
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		return artifactbundle.Bundle{}, fmt.Errorf("read artifact bundle %s: %w", path, err)
	}
	return bundle, nil
}

func materializeArtifactBundle(bundle artifactbundle.Bundle, prefix string) (string, string, error) {
	dir, err := os.MkdirTemp("", prefix)
	if err != nil {
		return "", "", err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}

	manifestPath := filepath.Join(dir, "manifest.json")
	moduleName := bundle.Manifest.Runtime.Module
	modulePath := filepath.Join(dir, moduleName)
	moduleBytes := artifactbundle.ModuleBytes(bundle.Bytes, moduleName)
	if moduleBytes == nil {
		cleanup()
		return "", "", fmt.Errorf("artifact bundle %s: module %q is missing", bundle.Digest, moduleName)
	}
	if err := os.WriteFile(manifestPath, artifactbundle.ManifestJSON(bundle.Manifest), 0o600); err != nil {
		cleanup()
		return "", "", fmt.Errorf("materialize artifact bundle %s manifest: %w", bundle.Digest, err)
	}
	if err := os.WriteFile(modulePath, moduleBytes, 0o600); err != nil {
		cleanup()
		return "", "", fmt.Errorf("materialize artifact bundle %s module: %w", bundle.Digest, err)
	}
	return dir, modulePath, nil
}

type cleanupGameMasterSession struct {
	gamemaster.Session
	dir string
}

func (s *cleanupGameMasterSession) Shutdown(ctx context.Context) error {
	shutdownErr := s.Session.Shutdown(ctx)
	removeErr := os.RemoveAll(s.dir)
	return errors.Join(shutdownErr, removeErr)
}

type cleanupPlayerSession struct {
	*session.Session
	dir string
}

func (s *cleanupPlayerSession) Close(ctx context.Context) error {
	closeErr := s.Session.Close(ctx)
	removeErr := os.RemoveAll(s.dir)
	return errors.Join(closeErr, removeErr)
}

var _ match.PlayerSession = (*cleanupPlayerSession)(nil)
