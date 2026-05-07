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
	DefaultMode              BuildMode
	SupportedModes           []BuildMode
	BuildSession             func(BuildMode, BuildSpec) (gamemaster.Session, error)
	BuildSessionFromSnapshot func(BuildMode, BuildSpec, game.Snapshot) (gamemaster.Session, error)
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
	switch descriptor.DefaultMode {
	case BuildModeInProcess, BuildModeLocalSubprocess, BuildModeFutureExternalAdapter:
	default:
		if descriptor.DefaultMode == "" {
			return fmt.Errorf("registry: DefaultMode is required")
		}
		return fmt.Errorf("registry: unsupported DefaultMode %q", descriptor.DefaultMode)
	}
	if len(descriptor.SupportedModes) == 0 {
		descriptor.SupportedModes = []BuildMode{descriptor.DefaultMode}
	}
	if err := validateSupportedModes(descriptor.DefaultMode, descriptor.SupportedModes); err != nil {
		return err
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

func validateSupportedModes(defaultMode BuildMode, modes []BuildMode) error {
	seen := make(map[BuildMode]struct{}, len(modes))
	hasDefault := false
	for _, mode := range modes {
		switch mode {
		case BuildModeInProcess, BuildModeLocalSubprocess, BuildModeFutureExternalAdapter:
		default:
			return fmt.Errorf("registry: unsupported mode %q", mode)
		}
		if _, exists := seen[mode]; exists {
			return fmt.Errorf("registry: duplicate supported mode %q", mode)
		}
		seen[mode] = struct{}{}
		if mode == defaultMode {
			hasDefault = true
		}
	}
	if !hasDefault {
		return fmt.Errorf("registry: DefaultMode %q must be present in SupportedModes", defaultMode)
	}
	return nil
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
			RegistryKey:    RegistryKey{GameID: echo.GameID, GameVersionMajor: 2},
			GameID:         echo.GameID,
			DefaultMode:    BuildModeInProcess,
			SupportedModes: []BuildMode{BuildModeInProcess, BuildModeLocalSubprocess},
			BuildSession: func(mode BuildMode, spec BuildSpec) (gamemaster.Session, error) {
				return buildEchoSession(mode, spec, nil)
			},
			BuildSessionFromSnapshot: func(mode BuildMode, spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				return buildEchoSession(mode, spec, &snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return echo.SnapshotFromHistory(spec.GameVersion, spec.Ruleset, append([]game.Player(nil), spec.Players...), events, targetTurn)
			},
		},
		{
			RegistryKey:    RegistryKey{GameID: janken.GameID, GameVersionMajor: 2},
			GameID:         janken.GameID,
			DefaultMode:    BuildModeInProcess,
			SupportedModes: []BuildMode{BuildModeInProcess},
			BuildSession: func(mode BuildMode, spec BuildSpec) (gamemaster.Session, error) {
				if mode != BuildModeInProcess {
					return nil, fmt.Errorf("registry: mode %q is unsupported for game %q", mode, janken.GameID)
				}
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
			BuildSessionFromSnapshot: func(mode BuildMode, spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				if mode != BuildModeInProcess {
					return nil, fmt.Errorf("registry: mode %q is unsupported for game %q", mode, janken.GameID)
				}
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

func buildEchoSession(mode BuildMode, spec BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
	switch mode {
	case BuildModeInProcess:
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
	case BuildModeLocalSubprocess:
		meta, _, _, _, err := echo.MetadataForSelection(spec.GameVersion, spec.Ruleset)
		if err != nil {
			return nil, err
		}
		command := []string{
			"go", "run", "./cmd/echo-count-gamemaster",
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
	default:
		return nil, fmt.Errorf("registry: mode %q is unsupported for game %q", mode, echo.GameID)
	}
}
