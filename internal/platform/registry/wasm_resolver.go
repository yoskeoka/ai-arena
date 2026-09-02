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

// NewWASIResolver constructs a resolver that materializes admitted WASI game releases through the provided bundle materializer.
func NewWASIResolver(materializer BundleMaterializer) (*WASIResolver, error) {
	if materializer == nil {
		return nil, fmt.Errorf("registry: bundle materializer is required")
	}
	return &WASIResolver{materializer: materializer}, nil
}

// Resolve converts an admitted WASI descriptor into a game descriptor whose sessions use the immutable artifact and clean up private files on shutdown.
func (r *WASIResolver) Resolve(_ context.Context, record DescriptorRecord) (GameDescriptor, error) {
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
		sessionCtx := spec.Context
		if sessionCtx == nil {
			sessionCtx = context.Background()
		}
		module, err := r.materializer.Materialize(sessionCtx, record.ArtifactID, dir)
		if err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		session, err := gamemaster.StartWASMWASI(gamemaster.WASIConfig{Context: sessionCtx, ExpectedMetadata: catalog.GameMetadata{GameID: record.GameID, GameVersion: record.GameVersion, RulesetVersion: spec.Ruleset}, ModulePath: module, Dir: dir, Args: append([]string(nil), record.RuntimeArgs...), Players: append([]game.Player(nil), spec.Players...), RNGSeed: spec.RNGSeed, ResumeSnapshot: snapshot, MemoryLimitPages: record.MemoryLimitPages, StderrLimitBytes: 4096})
		if err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		return &cleanupSession{Session: session, cleanup: func() { _ = os.RemoveAll(dir) }}, nil
	}
	return GameDescriptor{RegistryKey: record.RegistryKey, GameID: record.GameID, GameVersion: record.GameVersion, ArtifactID: record.ArtifactID, BuilderID: record.BuilderID, BuildMode: record.BuildMode, BuildConstraints: copyBuildConstraints(record.BuildConstraints), BuildSession: func(spec BuildSpec) (gamemaster.Session, error) { return build(spec, nil) }, BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
		return build(spec, &snapshot)
	}, SnapshotFromHistory: func(BuildSpec, []match.Event, int) (game.Snapshot, error) {
		return game.Snapshot{}, fmt.Errorf("registry: WASI replay reconstruction is not implemented")
	}}, nil
}

type cleanupSession struct {
	gamemaster.Session
	cleanup func()
}

func (s *cleanupSession) Shutdown(ctx context.Context) error {
	err := s.Session.Shutdown(ctx)
	s.cleanup()
	return err
}

// NewWASIOverlay preserves built-in descriptors while adding a writable WASI
// artifact admission registry for the service process.
func NewWASIOverlay(materializer BundleMaterializer) (*Registry, error) {
	wasi, err := NewWASIResolver(materializer)
	if err != nil {
		return nil, err
	}
	base, ok := defaultRegistry.store.(*InMemoryStore)
	if !ok {
		return nil, fmt.Errorf("registry: default store is not cloneable")
	}
	store, err := NewInMemoryStore()
	if err != nil {
		return nil, err
	}
	for _, releases := range base.records {
		for _, record := range releases {
			if err := store.Register(record); err != nil {
				return nil, err
			}
		}
	}
	return New(store, modeResolver{fallback: defaultRegistry.resolver, wasi: wasi})
}

type modeResolver struct {
	fallback DescriptorResolver
	wasi     DescriptorResolver
}

func (r modeResolver) Resolve(ctx context.Context, record DescriptorRecord) (GameDescriptor, error) {
	if record.BuildMode == BuildModeWASMWASI {
		return r.wasi.Resolve(ctx, record)
	}
	return r.fallback.Resolve(ctx, record)
}
