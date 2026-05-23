package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

const errGameMasterManifestFreshRunOnly = "game master manifest overlay supports fresh run only"

type gameMasterManifest struct {
	Metadata catalog.GameMetadata    `json:"metadata"`
	Runtime  catalog.RuntimeManifest `json:"runtime"`
}

func loadGameMasterManifestDescriptor(path string, stderrLimitBytes int) (registry.GameDescriptor, catalog.GameMetadata, error) {
	var manifest gameMasterManifest
	if err := readJSON(path, &manifest); err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("read game master manifest %s: %w", path, err)
	}
	if err := catalog.ValidateMetadata(manifest.Metadata); err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("game master manifest metadata invalid: %w", err)
	}
	runtimeCfg, err := resolveGameMasterRuntime(path, manifest.Runtime)
	if err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("game master manifest runtime invalid: %w", err)
	}

	builderID := fmt.Sprintf("manifest:%s", filepath.Clean(path))
	unsupportedFreshOnly := errors.New(errGameMasterManifestFreshRunOnly)
	return registry.GameDescriptor{
		RegistryKey: registry.RegistryKey{
			GameID:           manifest.Metadata.GameID,
			GameVersionMajor: mustGameVersionMajor(manifest.Metadata.GameVersion),
		},
		GameID:    manifest.Metadata.GameID,
		BuilderID: builderID,
		BuildMode: registry.BuildModeLocalSubprocess,
		BuildConstraints: registry.BuildConstraints{
			SupportedRulesets: []string{manifest.Metadata.RulesetVersion},
		},
		BuildSession: func(spec registry.BuildSpec) (gamemaster.Session, error) {
			if err := catalog.Compatible(manifest.Metadata, catalog.GameMetadata{
				GameID:         manifest.Metadata.GameID,
				GameVersion:    spec.GameVersion,
				RulesetVersion: spec.Ruleset,
			}); err != nil {
				return nil, fmt.Errorf("game master manifest metadata incompatible: %w", err)
			}
			return gamemaster.StartLocalSubprocess(gamemaster.LocalSubprocessConfig{
				ExpectedMetadata: manifest.Metadata,
				Command:          append([]string(nil), runtimeCfg.Command...),
				Dir:              runtimeCfg.Dir,
				Players:          append([]game.Player(nil), spec.Players...),
				RNGSeed:          spec.RNGSeed,
				StderrLimitBytes: stderrLimitBytes,
			})
		},
		BuildSessionFromSnapshot: func(registry.BuildSpec, game.Snapshot) (gamemaster.Session, error) {
			return nil, unsupportedFreshOnly
		},
		SnapshotFromHistory: func(registry.BuildSpec, []match.Event, int) (game.Snapshot, error) {
			return game.Snapshot{}, unsupportedFreshOnly
		},
	}, manifest.Metadata, nil
}

func resolveGameMasterRuntime(manifestPath string, manifest catalog.RuntimeManifest) (runtime.Config, error) {
	if manifest.Kind != "" && manifest.Kind != runtime.KindLocalSubprocess {
		return runtime.Config{}, fmt.Errorf("%w: runtime.kind %q is unsupported for game master manifests", catalog.ErrInvalidMetadata, manifest.Kind)
	}
	if len(manifest.Command) == 0 {
		return runtime.Config{}, fmt.Errorf("%w: runtime.command is required", catalog.ErrInvalidMetadata)
	}

	command := append([]string(nil), manifest.Command...)
	if isManifestRelativeCommand(command[0]) {
		command[0] = filepath.Join(filepath.Dir(manifestPath), command[0])
	}
	return runtime.Config{
		Kind:    runtime.KindLocalSubprocess,
		Command: command,
		Dir:     filepath.Dir(manifestPath),
	}, nil
}

func isManifestRelativeCommand(command string) bool {
	if filepath.IsAbs(command) {
		return false
	}
	return strings.Contains(command, "/") || strings.Contains(command, string(filepath.Separator))
}

func mustGameVersionMajor(version string) int {
	major, err := catalog.MajorVersion(version)
	if err != nil {
		panic(err)
	}
	return major
}
