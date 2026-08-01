package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/yoskeoka/ai-arena/artifactbundle"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func validatePlayerBundle(meta catalog.GameMetadata, bundle artifactbundle.Bundle) error {
	if bundle.Manifest.ArtifactKind != "ai" {
		return fmt.Errorf("artifact kind %q is not ai", bundle.Manifest.ArtifactKind)
	}
	if bundle.Manifest.GameID != meta.GameID {
		return fmt.Errorf("game_id mismatch: bundle %q, selected %q", bundle.Manifest.GameID, meta.GameID)
	}
	bundleMajor, err := catalog.MajorVersion(bundle.Manifest.GameVersion)
	if err != nil {
		return fmt.Errorf("AI bundle game version invalid: %w", err)
	}
	selectedMajor, err := catalog.MajorVersion(meta.GameVersion)
	if err != nil {
		return err
	}
	if bundleMajor != selectedMajor {
		return fmt.Errorf("game_version major mismatch: bundle %q, selected %q", bundle.Manifest.GameVersion, meta.GameVersion)
	}
	if strings.TrimSpace(bundle.Manifest.AIID) == "" {
		return fmt.Errorf("ai_id is required")
	}
	return nil
}

func loadPlayerBundleSession(ctx context.Context, meta catalog.GameMetadata, spec playerSpec, stderrLimitBytes int) (game.Player, match.PlayerSession, error) {
	if spec.Bundle == nil {
		return game.Player{}, nil, fmt.Errorf("%s: AI bundle is missing", spec.PlayerID)
	}
	if err := validatePlayerBundle(meta, *spec.Bundle); err != nil {
		return game.Player{}, nil, fmt.Errorf("%s: %w", spec.PlayerID, err)
	}
	dir, modulePath, err := materializeArtifactBundle(*spec.Bundle, "ai-arena-ai-bundle-")
	if err != nil {
		return game.Player{}, nil, fmt.Errorf("%s: %w", spec.PlayerID, err)
	}
	adapter, err := runtime.Start(ctx, runtime.Config{
		Kind:             runtime.KindWASMWASI,
		ModulePath:       modulePath,
		Dir:              dir,
		Args:             append([]string(nil), spec.Bundle.Manifest.Runtime.Args...),
		MemoryLimitPages: spec.Bundle.Manifest.Runtime.MemoryLimitPages,
		StderrLimitBytes: stderrLimitBytes,
	})
	if err != nil {
		_ = removeBundleDirectory(dir)
		return game.Player{}, nil, fmt.Errorf("%s: runtime start failed: %w", spec.PlayerID, err)
	}
	return game.Player{PlayerID: spec.PlayerID, AIID: spec.Bundle.Digest}, &cleanupPlayerSession{
		Session: session.New(adapter),
		dir:     dir,
	}, nil
}
