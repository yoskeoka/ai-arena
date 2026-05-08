package registry

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yoskeoka/ai-arena/games/dungeon"
	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
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
		DescriptorRecord{
			RegistryKey: RegistryKey{GameID: dungeon.GameID, GameVersionMajor: 1},
			GameID:      dungeon.GameID,
			BuildMode:   BuildModeLocalSubprocess,
			BuilderID:   dungeon.BuilderIDSubprocess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: dungeon.SupportedRulesets(),
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
		dungeon.BuilderIDSubprocess: {
			BuildMode: BuildModeLocalSubprocess,
			BuildConstraints: BuildConstraints{
				SupportedRulesets: dungeon.SupportedRulesets(),
			},
			BuildSession: func(spec BuildSpec) (gamemaster.Session, error) {
				return buildDungeonLocalSubprocessSession(spec, nil)
			},
			BuildSessionFromSnapshot: func(spec BuildSpec, snapshot game.Snapshot) (gamemaster.Session, error) {
				return buildDungeonLocalSubprocessSession(spec, &snapshot)
			},
			SnapshotFromHistory: func(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
				return dungeonSnapshotFromHistory(spec, events, targetTurn)
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

func buildDungeonLocalSubprocessSession(spec BuildSpec, snapshot *game.Snapshot) (gamemaster.Session, error) {
	meta, _, err := dungeon.MetadataForSelection(spec.GameVersion, spec.Ruleset)
	if err != nil {
		return nil, err
	}
	rngSeed := spec.RNGSeed
	if snapshot != nil && strings.TrimSpace(rngSeed) == "" {
		rngSeed, err = dungeonSeedFromSnapshot(*snapshot)
		if err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(rngSeed) == "" {
		rngSeed = dungeon.DefaultRNGSeed
	}
	command := []string{
		"go", "run", "./cmd/dungeon-gamemaster",
		"--game-version", spec.GameVersion,
		"--ruleset", spec.Ruleset,
	}
	return gamemaster.StartLocalSubprocess(gamemaster.LocalSubprocessConfig{
		ExpectedMetadata: catalog.GameMetadata{
			GameID:         meta.GameID,
			GameVersion:    meta.GameVersion,
			RulesetVersion: meta.RulesetVersion,
		},
		Command:          command,
		Players:          append([]game.Player(nil), spec.Players...),
		RNGSeed:          rngSeed,
		ResumeSnapshot:   snapshot,
		StderrLimitBytes: 4096,
	})
}

func dungeonSnapshotFromHistory(spec BuildSpec, events []match.Event, targetTurn int) (game.Snapshot, error) {
	playerIDs := make([]string, 0, len(spec.Players))
	for _, player := range spec.Players {
		playerIDs = append(playerIDs, player.PlayerID)
	}
	world, err := dungeon.New(dungeon.Config{
		GameVersion: spec.GameVersion,
		Ruleset:     spec.Ruleset,
		PlayerIDs:   playerIDs,
		RNGSeed:     seedOrDefault(spec.RNGSeed),
	})
	if err != nil {
		return game.Snapshot{}, err
	}
	if targetTurn < 0 || targetTurn > world.Ruleset().MaxTurns {
		return game.Snapshot{}, fmt.Errorf("target turn %d out of range 0..%d", targetTurn, world.Ruleset().MaxTurns)
	}

	statusesByTurn := make(map[int]map[string]game.ActionStatus)
	for _, event := range events {
		if event.Turn == 0 || event.Turn > targetTurn {
			continue
		}
		switch event.Kind {
		case "turn_result", "turn_timeout", "protocol_error", "runtime_exited":
			var actionStatus game.ActionStatus
			if err := json.Unmarshal(event.Payload, &actionStatus); err != nil {
				return game.Snapshot{}, fmt.Errorf("decode history event payload seq=%d: %w", event.Seq, err)
			}
			if _, ok := statusesByTurn[event.Turn]; !ok {
				statusesByTurn[event.Turn] = make(map[string]game.ActionStatus)
			}
			if actionStatus.PlayerID == "" {
				actionStatus.PlayerID = event.PlayerID
			}
			statusesByTurn[event.Turn][event.PlayerID] = actionStatus
		}
	}

	lastAction := make(map[string]game.ActionStatus, len(spec.Players))
	for turn := 1; turn <= targetTurn && !world.Terminal(); turn++ {
		actions := make(map[string]dungeon.Action)
		for _, playerID := range world.PendingPlayerIDs() {
			status := game.ActionStatus{PlayerID: playerID, ActionStatus: contract.ActionNoAction}
			if perTurn, ok := statusesByTurn[turn]; ok {
				if replayed, ok := perTurn[playerID]; ok {
					status = replayed
				}
			}
			lastAction[playerID] = status
			if status.ActionStatus != contract.ActionAccepted {
				actions[playerID] = dungeon.Action{Action: "wait"}
				continue
			}
			action, err := dungeon.ParseAction(status.Action)
			if err != nil {
				return game.Snapshot{}, err
			}
			actions[playerID] = action
		}
		if err := world.Apply(actions); err != nil {
			return game.Snapshot{}, err
		}
	}

	full := world.FullState()
	perPlayer := make(map[string]game.PlayerSnapshot, len(spec.Players))
	for _, player := range spec.Players {
		visible, err := world.CurrentVisibleState(player.PlayerID)
		if err != nil {
			return game.Snapshot{}, err
		}
		status := lastAction[player.PlayerID]
		if status.PlayerID == "" {
			status = game.ActionStatus{PlayerID: player.PlayerID, ActionStatus: contract.ActionNoAction}
		}
		perPlayer[player.PlayerID] = game.PlayerSnapshot{
			VisibleState:     mustRawJSON(visible),
			LastActionStatus: status,
		}
	}
	return game.Snapshot{
		GameID:         dungeon.GameID,
		GameVersion:    spec.GameVersion,
		RulesetVersion: spec.Ruleset,
		Turn:           world.Turn(),
		Status:         snapshotStatusForDungeon(world),
		GameState:      mustRawJSON(full),
		PerPlayer:      perPlayer,
	}, nil
}

func dungeonSeedFromSnapshot(snapshot game.Snapshot) (string, error) {
	var state dungeon.FullState
	if err := json.Unmarshal(snapshot.GameState, &state); err != nil {
		return "", fmt.Errorf("decode dungeon snapshot game_state: %w", err)
	}
	return state.RNGSeed, nil
}

func seedOrDefault(seed string) string {
	if strings.TrimSpace(seed) == "" {
		return dungeon.DefaultRNGSeed
	}
	return seed
}

func snapshotStatusForDungeon(world *dungeon.Match) game.MatchStatus {
	if world.Terminal() {
		return game.StatusCompleted
	}
	return game.StatusRunning
}

func mustRawJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
