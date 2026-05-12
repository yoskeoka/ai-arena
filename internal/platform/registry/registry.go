package registry

import (
	"context"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

// BuildMode aliases the supported game-master hosting modes.
type BuildMode = gamemaster.Mode

const (
	// BuildModeInProcess runs the game master in-process.
	BuildModeInProcess BuildMode = gamemaster.ModeInProcess
	// BuildModeLocalSubprocess runs the game master as a child process.
	BuildModeLocalSubprocess BuildMode = gamemaster.ModeLocalSubprocess
	// BuildModeFutureExternalAdapter reserves an external adapter mode.
	BuildModeFutureExternalAdapter BuildMode = gamemaster.ModeFutureExternalAdapter
)

// RegistryKey identifies one game id and supported semver major.
type RegistryKey struct {
	GameID           string
	GameVersionMajor int
}

// BuildSpec captures the inputs required to build a game-master session.
type BuildSpec struct {
	GameVersion string
	Ruleset     string
	RNGSeed     string
	Players     []game.Player
}

// BuildConstraints describes ruleset-level constraints for a descriptor.
type BuildConstraints struct {
	SupportedRulesets []string
}

// DescriptorRecord is the stored metadata for one game descriptor entry.
type DescriptorRecord struct {
	RegistryKey      RegistryKey
	GameID           string
	BuildMode        BuildMode
	BuilderID        string
	BuildConstraints BuildConstraints
}

// GameDescriptor resolves build functions for one registry entry.
type GameDescriptor struct {
	RegistryKey              RegistryKey
	GameID                   string
	BuilderID                string
	BuildMode                BuildMode
	BuildConstraints         BuildConstraints
	BuildSession             func(BuildSpec) (gamemaster.Session, error)
	BuildSessionFromSnapshot func(BuildSpec, game.Snapshot) (gamemaster.Session, error)
	SnapshotFromHistory      func(BuildSpec, []match.Event, int) (game.Snapshot, error)
}

// RegistryStore looks up stored descriptor metadata by registry key.
type RegistryStore interface {
	Lookup(context.Context, RegistryKey) (DescriptorRecord, error)
}

// DescriptorResolver materializes a descriptor from stored metadata.
type DescriptorResolver interface {
	Resolve(context.Context, DescriptorRecord) (GameDescriptor, error)
}

// Registry coordinates descriptor lookup and resolution.
type Registry struct {
	store    RegistryStore
	resolver DescriptorResolver
}

// New constructs a registry from a store and resolver.
func New(store RegistryStore, resolver DescriptorResolver) (*Registry, error) {
	if store == nil {
		return nil, fmt.Errorf("registry: store is required")
	}
	if resolver == nil {
		return nil, fmt.Errorf("registry: resolver is required")
	}
	return &Registry{store: store, resolver: resolver}, nil
}

// Lookup resolves a descriptor by registry key.
func (r *Registry) Lookup(ctx context.Context, key RegistryKey) (GameDescriptor, error) {
	record, err := r.store.Lookup(ctx, key)
	if err != nil {
		return GameDescriptor{}, err
	}
	return r.resolver.Resolve(ctx, record)
}

// LookupVersion resolves a descriptor by game id and semver version string.
func (r *Registry) LookupVersion(ctx context.Context, gameID, gameVersion string) (GameDescriptor, error) {
	major, err := catalog.MajorVersion(gameVersion)
	if err != nil {
		return GameDescriptor{}, fmt.Errorf("registry: invalid game version %q: %w", gameVersion, err)
	}
	return r.Lookup(ctx, RegistryKey{
		GameID:           gameID,
		GameVersionMajor: major,
	})
}

// Default returns the process-wide default registry.
func Default() *Registry {
	return defaultRegistry
}

// Lookup resolves a descriptor through the default registry.
func Lookup(gameID, gameVersion string) (GameDescriptor, error) {
	return defaultRegistry.LookupVersion(context.Background(), gameID, gameVersion)
}

func validateRegistryKey(key RegistryKey) error {
	if key.GameID == "" {
		return fmt.Errorf("registry: registry key game_id is required")
	}
	if key.GameVersionMajor <= 0 {
		return fmt.Errorf("registry: registry key game_version major must be positive")
	}
	return nil
}

func validateBuildMode(mode BuildMode) error {
	switch mode {
	case BuildModeInProcess, BuildModeLocalSubprocess, BuildModeFutureExternalAdapter:
		return nil
	default:
		if mode == "" {
			return fmt.Errorf("registry: BuildMode is required")
		}
		return fmt.Errorf("registry: unsupported BuildMode %q", mode)
	}
}

func validateBuildConstraints(constraints BuildConstraints) error {
	if len(constraints.SupportedRulesets) == 0 {
		return fmt.Errorf("registry: at least one supported ruleset is required")
	}
	for _, ruleset := range constraints.SupportedRulesets {
		if ruleset == "" {
			return fmt.Errorf("registry: supported ruleset must not be empty")
		}
	}
	return nil
}

func copyBuildConstraints(constraints BuildConstraints) BuildConstraints {
	return BuildConstraints{
		SupportedRulesets: append([]string(nil), constraints.SupportedRulesets...),
	}
}

func sameRulesets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, value := range a {
		seen[value]++
	}
	for _, value := range b {
		count, ok := seen[value]
		if !ok {
			return false
		}
		if count == 1 {
			delete(seen, value)
			continue
		}
		seen[value] = count - 1
	}
	return len(seen) == 0
}
