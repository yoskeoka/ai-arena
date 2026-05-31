package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/replay"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

type gameMasterManifest struct {
	Metadata catalog.GameMetadata    `json:"metadata"`
	Runtime  catalog.RuntimeManifest `json:"runtime"`
}

func loadGameMasterManifestDescriptor(path string, stderrLimitBytes int) (registry.GameDescriptor, catalog.GameMetadata, error) {
	var manifest gameMasterManifest
	// #nosec G304 -- the operator explicitly chooses the local manifest path.
	data, err := os.ReadFile(path)
	if err != nil {
		return registry.GameDescriptor{}, catalog.GameMetadata{}, fmt.Errorf("read game master manifest %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
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
		BuildSessionFromSnapshot: func(spec registry.BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
			if err := catalog.Compatible(manifest.Metadata, catalog.GameMetadata{
				GameID:         snapshot.GameID,
				GameVersion:    snapshot.GameVersion,
				RulesetVersion: snapshot.RulesetVersion,
			}); err != nil {
				return nil, fmt.Errorf("game master manifest metadata incompatible: %w", err)
			}
			return gamemaster.StartLocalSubprocess(gamemaster.LocalSubprocessConfig{
				ExpectedMetadata: manifest.Metadata,
				Command:          append([]string(nil), runtimeCfg.Command...),
				Dir:              runtimeCfg.Dir,
				Players:          append([]game.Player(nil), spec.Players...),
				RNGSeed:          spec.RNGSeed,
				ResumeSnapshot:   &snapshot,
				StderrLimitBytes: stderrLimitBytes,
			})
		},
		SnapshotFromHistory: func(spec registry.BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
			return replay.SnapshotFromSessionHistory(func() (gamemaster.Session, error) {
				return gamemaster.StartLocalSubprocess(gamemaster.LocalSubprocessConfig{
					ExpectedMetadata: manifest.Metadata,
					Command:          append([]string(nil), runtimeCfg.Command...),
					Dir:              runtimeCfg.Dir,
					Players:          append([]game.Player(nil), spec.Players...),
					RNGSeed:          spec.RNGSeed,
					StderrLimitBytes: stderrLimitBytes,
				})
			}, events, targetTurn)
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

	manifestDir, err := filepath.Abs(filepath.Dir(manifestPath))
	if err != nil {
		return runtime.Config{}, fmt.Errorf("resolve manifest directory: %w", err)
	}
	command := append([]string(nil), manifest.Command...)
	command[0], err = normalizeManifestRuntimeCommand(manifestDir, command[0])
	if err != nil {
		return runtime.Config{}, err
	}
	return runtime.Config{
		Kind:    runtime.KindLocalSubprocess,
		Command: command,
		Dir:     manifestDir,
	}, nil
}

func normalizeManifestRuntimeCommand(manifestDir, command string) (string, error) {
	if !isManifestRelativeCommand(command) {
		return command, nil
	}
	resolved, err := filepath.Abs(filepath.Join(manifestDir, command))
	if err != nil {
		return "", fmt.Errorf("resolve runtime.command[0]: %w", err)
	}
	return resolved, nil
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
