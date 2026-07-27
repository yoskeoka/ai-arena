package registry

import (
	"context"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

// BundleMaterializer supplies a worker-private WASM module for an immutable digest.
type BundleMaterializer interface {
	Materialize(context.Context, string, string) (string, error)
}

// WASIResolver resolves admitted artifact-backed game releases.
type WASIResolver struct{ materializer BundleMaterializer }

func NewWASIResolver(materializer BundleMaterializer) (*WASIResolver, error) {
	if materializer == nil {
		return nil, fmt.Errorf("registry: bundle materializer is required")
	}
	return &WASIResolver{materializer: materializer}, nil
}

func (r *WASIResolver) Resolve(ctx context.Context, record DescriptorRecord) (GameDescriptor, error) {
	if err := validateDescriptorRecord(record); err != nil {
		return GameDescriptor{}, err
	}
	if record.BuildMode != BuildModeWASMWASI || record.ArtifactID == "" || record.GameVersion == "" {
		return GameDescriptor{}, fmt.Errorf("registry: descriptor is not an admitted WASI artifact release")
	}
	build := func(spec BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
		dir, err := os.MkdirTemp("", "ai-arena-game-")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(dir)
		module, err := r.materializer.Materialize(ctx, record.ArtifactID, dir)
		if err != nil {
			return nil, err
		}
		return gamemaster.StartWASMWASI(gamemaster.WASIConfig{ExpectedMetadata: catalog.GameMetadata{GameID: record.GameID, GameVersion: record.GameVersion, RulesetVersion: spec.Ruleset}, ModulePath: module, Args: append([]string(nil), record.RuntimeArgs...), Players: append([]game.Player(nil), spec.Players...), RNGSeed: spec.RNGSeed, ResumeSnapshot: snapshot, MemoryLimitPages: record.MemoryLimitPages, StderrLimitBytes: 4096})
	}
	return GameDescriptor{RegistryKey: record.RegistryKey, GameID: record.GameID, GameVersion: record.GameVersion, ArtifactID: record.ArtifactID, BuilderID: record.BuilderID, BuildMode: record.BuildMode, BuildConstraints: copyBuildConstraints(record.BuildConstraints), BuildSession: func(spec BuildSpec) (gamemaster.Session, error) { return build(spec, nil) }, BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
		return build(spec, &snapshot)
	}, SnapshotFromHistory: func(BuildSpec, []match.Event, int) (game.Snapshot, error) {
		return game.Snapshot{}, fmt.Errorf("registry: WASI replay reconstruction is not implemented")
	}}, nil
}
