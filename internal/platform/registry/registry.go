package registry

import (
	"encoding/json"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

type BuildMode = gamemaster.Mode

const (
	BuildModeInProcess             BuildMode = gamemaster.ModeInProcess
	BuildModeLocalSubprocess       BuildMode = gamemaster.ModeLocalSubprocess
	BuildModeFutureExternalAdapter BuildMode = gamemaster.ModeFutureExternalAdapter
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
	RegistryKey              RegistryKey
	GameID                   string
	BuildMode                BuildMode
	BuildSession             func(BuildSpec) (gamemaster.Session, error)
	BuildSessionFromSnapshot func(BuildSpec, game.Snapshot) (gamemaster.Session, error)
	SnapshotFromHistory      func(BuildSpec, []match.Event, int) (game.Snapshot, error)
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
	switch descriptor.BuildMode {
	case BuildModeInProcess, BuildModeLocalSubprocess, BuildModeFutureExternalAdapter:
	default:
		if descriptor.BuildMode == "" {
			return fmt.Errorf("registry: BuildMode is required")
		}
		return fmt.Errorf("registry: unsupported BuildMode %q", descriptor.BuildMode)
	}
	if descriptor.BuildSession == nil {
		return fmt.Errorf("registry: BuildSession is required")
	}
	if descriptor.BuildSessionFromSnapshot == nil {
		return fmt.Errorf("registry: BuildSessionFromSnapshot is required")
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
	major, err := catalog.MajorVersion(gameVersion)
	if err != nil {
		return GameDescriptor{}, fmt.Errorf("registry: invalid game version %q: %w", gameVersion, err)
	}
	key := RegistryKey{GameID: gameID, GameVersionMajor: major}
	descriptor, ok := r.descriptors[key]
	if !ok {
		if r.hasGameID(gameID) {
			return GameDescriptor{}, fmt.Errorf("registry: unsupported game version major %d for game %q", major, gameID)
		}
		return GameDescriptor{}, fmt.Errorf("registry: unsupported game %q", gameID)
	}
	return descriptor, nil
}

func Lookup(gameID, gameVersion string) (GameDescriptor, error) {
	return defaultRegistry.Lookup(gameID, gameVersion)
}

func (r *Registry) hasGameID(gameID string) bool {
	for key := range r.descriptors {
		if key.GameID == gameID {
			return true
		}
	}
	return false
}

func mustDefaultRegistry() *Registry {
	descriptors := []GameDescriptor{
		{
			RegistryKey: RegistryKey{GameID: echo.GameID, GameVersionMajor: 2},
			GameID:      echo.GameID,
			BuildMode:   BuildModeInProcess,
			BuildSession: func(spec BuildSpec) (gamemaster.Session, error) {
				return buildEchoInProcessSession(spec, nil)
			},
			BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				return buildEchoInProcessSession(spec, &snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return echo.SnapshotFromHistory(spec.GameVersion, spec.Ruleset, append([]game.Player(nil), spec.Players...), events, targetTurn)
			},
		},
		{
			RegistryKey: RegistryKey{GameID: echo.SubprocessGameID, GameVersionMajor: 2},
			GameID:      echo.SubprocessGameID,
			BuildMode:   BuildModeLocalSubprocess,
			BuildSession: func(spec BuildSpec) (gamemaster.Session, error) {
				return buildEchoLocalSubprocessSession(spec, nil)
			},
			BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				return buildEchoLocalSubprocessSession(spec, &snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return echo.SnapshotFromHistoryWithGameID(echo.SubprocessGameID, spec.GameVersion, spec.Ruleset, append([]game.Player(nil), spec.Players...), events, targetTurn)
			},
		},
		{
			RegistryKey: RegistryKey{GameID: janken.GameID, GameVersionMajor: 2},
			GameID:      janken.GameID,
			BuildMode:   BuildModeInProcess,
			BuildSession: func(spec BuildSpec) (gamemaster.Session, error) {
				master, err := janken.New(janken.Config{
					GameVersion: spec.GameVersion,
					Ruleset:     spec.Ruleset,
					Players:     append([]game.Player(nil), spec.Players...),
				})
				if err != nil {
					return nil, err
				}
				return gamemaster.NewInProcessSession(master), nil
			},
			BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				master, err := janken.NewFromSnapshot(janken.Config{
					GameVersion: spec.GameVersion,
					Ruleset:     spec.Ruleset,
					Players:     append([]game.Player(nil), spec.Players...),
				}, snapshot)
				if err != nil {
					return nil, err
				}
				return gamemaster.NewInProcessSession(master), nil
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

func buildEchoInProcessSession(spec BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
	cfg := echo.Config{
		GameVersion: spec.GameVersion,
		Ruleset:     spec.Ruleset,
		Players:     append([]game.Player(nil), spec.Players...),
	}
	var (
		master game.Master
		err    error
	)
	if snapshot == nil {
		master, err = echo.New(cfg)
	} else {
		master, err = echo.NewFromSnapshot(cfg, *snapshot)
	}
	if err != nil {
		return nil, err
	}
	return gamemaster.NewInProcessSession(master), nil
}

func buildEchoLocalSubprocessSession(spec BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
	meta, _, _, _, err := echo.MetadataForSelectionWithGameID(echo.SubprocessGameID, spec.GameVersion, spec.Ruleset)
	if err != nil {
		return nil, err
	}
	command := []string{
		"go", "run", "./cmd/echo-count-gamemaster",
		"--game-id", echo.SubprocessGameID,
		"--game-version", spec.GameVersion,
		"--ruleset", spec.Ruleset,
	}
	return gamemaster.StartLocalSubprocess(gamemaster.LocalSubprocessConfig{
		ExpectedMetadata: meta,
		Command:          command,
		Players:          append([]game.Player(nil), spec.Players...),
		ResumeSnapshot:   snapshot,
		StderrLimitBytes: 4096,
	})
}
