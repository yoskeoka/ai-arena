package registry

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

type BuildMode string

const (
	BuildModeInProcess             BuildMode = "in-process"
	BuildModeLocalSubprocess       BuildMode = "local-subprocess"
	BuildModeFutureExternalAdapter BuildMode = "future-external-adapter"
)

type RegistryKey struct {
	GameID           string
	GameVersionMajor int
}

type BuildSpec struct {
	GameVersion string
	Ruleset     string
	Players     []game.Player
}

type GameDescriptor struct {
	RegistryKey         RegistryKey
	GameID              string
	BuildMode           BuildMode
	BuildFresh          func(BuildSpec) (game.Master, error)
	BuildFromSnapshot   func(BuildSpec, game.Snapshot) (game.Master, error)
	SnapshotFromHistory func(BuildSpec, []match.Event, int) (game.Snapshot, error)
}

type Registry struct {
	descriptors map[RegistryKey]GameDescriptor
}

func New(descriptors ...GameDescriptor) (*Registry, error) {
	r := &Registry{descriptors: make(map[RegistryKey]GameDescriptor, len(descriptors))}
	for _, descriptor := range descriptors {
		if err := r.Register(descriptor); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) Register(descriptor GameDescriptor) error {
	if descriptor.GameID == "" {
		return fmt.Errorf("registry: game_id is required")
	}
	if descriptor.RegistryKey.GameID == "" {
		return fmt.Errorf("registry: registry key game_id is required")
	}
	if descriptor.RegistryKey.GameID != descriptor.GameID {
		return fmt.Errorf("registry: descriptor game_id %q does not match key %q", descriptor.GameID, descriptor.RegistryKey.GameID)
	}
	if descriptor.RegistryKey.GameVersionMajor <= 0 {
		return fmt.Errorf("registry: registry key game_version major must be positive")
	}
	if descriptor.BuildFresh == nil {
		return fmt.Errorf("registry: BuildFresh is required")
	}
	if descriptor.BuildFromSnapshot == nil {
		return fmt.Errorf("registry: BuildFromSnapshot is required")
	}
	if descriptor.SnapshotFromHistory == nil {
		return fmt.Errorf("registry: SnapshotFromHistory is required")
	}
	if _, exists := r.descriptors[descriptor.RegistryKey]; exists {
		return fmt.Errorf("registry: duplicate descriptor for %s@%d", descriptor.GameID, descriptor.RegistryKey.GameVersionMajor)
	}
	r.descriptors[descriptor.RegistryKey] = descriptor
	return nil
}

func (r *Registry) Lookup(gameID, gameVersion string) (GameDescriptor, error) {
	major, err := majorVersion(gameVersion)
	if err != nil {
		return GameDescriptor{}, fmt.Errorf("registry: invalid game version %q", gameVersion)
	}
	key := RegistryKey{GameID: gameID, GameVersionMajor: major}
	descriptor, ok := r.descriptors[key]
	if !ok {
		return GameDescriptor{}, fmt.Errorf("unsupported game %q", gameID)
	}
	return descriptor, nil
}

func Lookup(gameID, gameVersion string) (GameDescriptor, error) {
	return defaultRegistry.Lookup(gameID, gameVersion)
}

func majorVersion(version string) (int, error) {
	parts := strings.Split(version, ".")
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("invalid semver")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid semver")
	}
	return major, nil
}

func mustDefaultRegistry() *Registry {
	descriptors := []GameDescriptor{
		{
			RegistryKey: RegistryKey{GameID: echo.GameID, GameVersionMajor: 2},
			GameID:      echo.GameID,
			BuildMode:   BuildModeInProcess,
			BuildFresh: func(spec BuildSpec) (game.Master, error) {
				return echo.New(echo.Config{
					GameVersion: spec.GameVersion,
					Ruleset:     spec.Ruleset,
					Players:     append([]game.Player(nil), spec.Players...),
				})
			},
			BuildFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (game.Master, error) {
				return echo.NewFromSnapshot(echo.Config{
					GameVersion: spec.GameVersion,
					Ruleset:     spec.Ruleset,
					Players:     append([]game.Player(nil), spec.Players...),
				}, snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return echo.SnapshotFromHistory(spec.GameVersion, spec.Ruleset, append([]game.Player(nil), spec.Players...), events, targetTurn)
			},
		},
		{
			RegistryKey: RegistryKey{GameID: janken.GameID, GameVersionMajor: 2},
			GameID:      janken.GameID,
			BuildMode:   BuildModeInProcess,
			BuildFresh: func(spec BuildSpec) (game.Master, error) {
				return janken.New(janken.Config{
					GameVersion: spec.GameVersion,
					Ruleset:     spec.Ruleset,
					Players:     append([]game.Player(nil), spec.Players...),
				})
			},
			BuildFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (game.Master, error) {
				return janken.NewFromSnapshot(janken.Config{
					GameVersion: spec.GameVersion,
					Ruleset:     spec.Ruleset,
					Players:     append([]game.Player(nil), spec.Players...),
				}, snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return janken.SnapshotFromHistory(spec.GameVersion, spec.Ruleset, append([]game.Player(nil), spec.Players...), events, targetTurn)
			},
		},
	}
	r, err := New(descriptors...)
	if err != nil {
		payload, _ := json.Marshal(err.Error())
		panic(fmt.Sprintf("build default registry: %s", payload))
	}
	return r
}

var defaultRegistry = mustDefaultRegistry()
