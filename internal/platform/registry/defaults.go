package registry

import (
	"encoding/json"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

func mustDefaultRegistry() *Registry {
	store, err := NewInMemoryStore(
		DescriptorRecord{
			RegistryKey: RegistryKey{GameID: echo.GameID, GameVersionMajor: 2},
			GameID:      echo.GameID,
			BuildMode:   BuildModeInProcess,
			BuilderID:   echo.BuilderIDInProcess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: echo.SupportedRulesets(),
			},
		},
		DescriptorRecord{
			RegistryKey: RegistryKey{GameID: echo.SubprocessGameID, GameVersionMajor: 2},
			GameID:      echo.SubprocessGameID,
			BuildMode:   BuildModeLocalSubprocess,
			BuilderID:   echo.BuilderIDLocalSubprocess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: echo.SupportedRulesets(),
			},
		},
		DescriptorRecord{
			RegistryKey: RegistryKey{GameID: janken.GameID, GameVersionMajor: 2},
			GameID:      janken.GameID,
			BuildMode:   BuildModeInProcess,
			BuilderID:   janken.BuilderIDInProcess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: janken.SupportedRulesets(),
			},
		},
	)
	if err != nil {
		panicDefaultRegistry(err)
	}

	resolver, err := NewStaticResolver(map[string]DescriptorBuilder{
		echo.BuilderIDInProcess: {
			BuildMode: BuildModeInProcess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: echo.SupportedRulesets(),
			},
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
		echo.BuilderIDLocalSubprocess: {
			BuildMode: BuildModeLocalSubprocess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: echo.SupportedRulesets(),
			},
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
		janken.BuilderIDInProcess: {
			BuildMode: BuildModeInProcess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: janken.SupportedRulesets(),
			},
			BuildSession: func(spec BuildSpec) (gamemaster.Session, error) {
				return buildJankenInProcessSession(spec, nil)
			},
			BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				return buildJankenInProcessSession(spec, &snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return janken.SnapshotFromHistory(spec.GameVersion, spec.Ruleset, append([]game.Player(nil), spec.Players...), events, targetTurn)
			},
		},
	})
	if err != nil {
		panicDefaultRegistry(err)
	}

	registry, err := New(store, resolver)
	if err != nil {
		panicDefaultRegistry(err)
	}
	return registry
}

func panicDefaultRegistry(err error) {
	payload, _ := json.Marshal(err.Error())
	panic(fmt.Sprintf("build default registry: %s", payload))
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

func buildJankenInProcessSession(spec BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
	cfg := janken.Config{
		GameVersion: spec.GameVersion,
		Ruleset:     spec.Ruleset,
		Players:     append([]game.Player(nil), spec.Players...),
	}
	var (
		master game.Master
		err    error
	)
	if snapshot == nil {
		master, err = janken.New(cfg)
	} else {
		master, err = janken.NewFromSnapshot(cfg, *snapshot)
	}
	if err != nil {
		return nil, err
	}
	return gamemaster.NewInProcessSession(master), nil
}
