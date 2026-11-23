package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
)

func loadGameMasterBundleDescriptor(path, selectedRuleset string, stderrLimitBytes int) (registry.GameDescriptor, catalog.GameMetadata, error) {
	bundle, err := readArtifactBundle(path)
	if err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, err
	}
	if bundle.Manifest.ArtifactKind != "game" {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("game master bundle: artifact kind %q is not game", bundle.Manifest.ArtifactKind)
	}
	if selectedRuleset == "" {
		if len(bundle.Manifest.Rulesets) != 1 {
			return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("game master bundle: --ruleset is required when the bundle does not declare exactly one ruleset")
		}
		selectedRuleset = bundle.Manifest.Rulesets[0].RulesetVersion
	}
	rulesetIndex := 0
	for rulesetIndex < len(bundle.Manifest.Rulesets) && bundle.Manifest.Rulesets[rulesetIndex].RulesetVersion != selectedRuleset {
		rulesetIndex++
	}
	if rulesetIndex == len(bundle.Manifest.Rulesets) {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("game master bundle: ruleset %q is not declared", selectedRuleset)
	}
	ruleset := bundle.Manifest.Rulesets[rulesetIndex]
	meta := catalog.GameMetadata{
		GameID:         bundle.Manifest.GameID,
		GameVersion:    bundle.Manifest.GameVersion,
		RulesetVersion: ruleset.RulesetVersion,
	}
	if err := catalog.ValidateMetadata(meta); err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("game master bundle metadata invalid: %w", err)
	}
	major, err := catalog.MajorVersion(meta.GameVersion)
	if err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, err
	}
	build := func(spec registry.BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
		if spec.GameVersion != meta.GameVersion || spec.Ruleset != meta.RulesetVersion {
			return nil, fmt.Errorf("game master bundle: build game version and ruleset must match the selected bundle")
		}
		if ruleset.PlayerCount > 0 && ruleset.PlayerCount != len(spec.Players) {
			return nil, fmt.Errorf("game master bundle: ruleset %q requires %d players, got %d", ruleset.RulesetVersion, ruleset.PlayerCount, len(spec.Players))
		}
		dir, modulePath, err := materializeArtifactBundle(bundle, "ai-arena-game-bundle-")
		if err != nil {
			return nil, err
		}
		ctx := spec.Context
		if ctx == nil {
			ctx = context.Background()
		}
		session, err := gamemaster.StartWASMWASI(gamemaster.WASIConfig{
			Context:          ctx,
			ExpectedMetadata: meta,
			ModulePath:       modulePath,
			Dir:              dir,
			Args:             append([]string(nil), bundle.Manifest.Runtime.Args...),
			Players:          append([]game.Player(nil), spec.Players...),
			RNGSeed:          spec.RNGSeed,
			ResumeSnapshot:   snapshot,
			MemoryLimitPages: bundle.Manifest.Runtime.MemoryLimitPages,
			StderrLimitBytes: stderrLimitBytes,
		})
		if err != nil {
			_ = removeBundleDirectory(dir)
			return nil, err
		}
		return &cleanupGameMasterSession{Session: session, dir: dir}, nil
	}

	builderID := fmt.Sprintf("bundle:%s", bundle.Digest)
	return registry.GameDescriptor{
		RegistryKey: registry.RegistryKey{GameID: meta.GameID, GameVersionMajor: major},
		GameID:      meta.GameID,
		GameVersion: meta.GameVersion,
		ArtifactID:  bundle.Digest,
		BuilderID:   builderID,
		BuildMode:   registry.BuildModeWASMWASI,
		BuildConstraints: registry.BuildConstraints{
			SupportedRulesets: []string{meta.RulesetVersion},
		},
		BuildSession: func(spec registry.BuildSpec) (gamemaster.Session, error) {
			return build(spec, nil)
		},
		BuildSessionFromSnapshot: func(spec registry.BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
			return build(spec, &snapshot)
		},
		SnapshotFromHistory: func(registry.BuildSpec, []match.Event, int) (game.Snapshot, error) {
			return game.Snapshot{}, fmt.Errorf("game master bundle: history replay reconstruction is not supported")
		},
	}, meta, nil
}

func removeBundleDirectory(dir string) error {
	if filepath.Clean(dir) == "." || dir == "" {
		return nil
	}
	return os.RemoveAll(dir)
}
