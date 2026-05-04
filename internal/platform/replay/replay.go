package replay

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func LoadRecord(path string) (match.Record, error) {
	// #nosec G304 -- the caller explicitly selects the local debug record input path.
	data, err := os.ReadFile(path)
	if err != nil {
		return match.Record{}, fmt.Errorf("read record input %s: %w", path, err)
	}
	var record match.Record
	if err := json.Unmarshal(data, &record); err != nil {
		return match.Record{}, fmt.Errorf("decode record input %s: %w", path, err)
	}
	return record, nil
}

func LoadSnapshot(path string) (game.Snapshot, error) {
	// #nosec G304 -- the caller explicitly selects the local debug snapshot input path.
	data, err := os.ReadFile(path)
	if err != nil {
		return game.Snapshot{}, fmt.Errorf("read snapshot input %s: %w", path, err)
	}
	var snapshot game.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return game.Snapshot{}, fmt.Errorf("decode snapshot input %s: %w", path, err)
	}
	return snapshot, nil
}

func LoadHistory(path string) ([]match.Event, error) {
	// #nosec G304 -- the caller explicitly selects the local debug history input path.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read history input %s: %w", path, err)
	}
	var events []match.Event
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("decode history input %s: %w", path, err)
	}
	return events, nil
}

func SnapshotFromHistory(meta catalog.GameMetadata, players []game.Player, events []match.Event, targetTurn int) (game.Snapshot, error) {
	switch meta.GameID {
	case echo.GameID:
		return echoSnapshotFromHistory(meta, players, events, targetTurn)
	default:
		return game.Snapshot{}, fmt.Errorf("unsupported history replay game %q", meta.GameID)
	}
}

func echoSnapshotFromHistory(meta catalog.GameMetadata, players []game.Player, events []match.Event, targetTurn int) (game.Snapshot, error) {
	maxTurns, err := echoRulesetTurns(meta.RulesetVersion)
	if err != nil {
		return game.Snapshot{}, err
	}
	if targetTurn < 0 || targetTurn > maxTurns {
		return game.Snapshot{}, fmt.Errorf("target turn %d out of range 0..%d", targetTurn, maxTurns)
	}

	score := make(map[string]int, len(players))
	perPlayer := make(map[string]game.PlayerSnapshot, len(players))
	for _, player := range players {
		score[player.PlayerID] = 0
		perPlayer[player.PlayerID] = game.PlayerSnapshot{
			LastActionStatus: game.ActionStatus{
				PlayerID:     player.PlayerID,
				ActionStatus: session.StatusNoAction,
			},
		}
	}

	for _, event := range events {
		if event.Turn == 0 || event.Turn > targetTurn {
			continue
		}
		switch event.Kind {
		case "turn_result", "turn_timeout", "protocol_error", "runtime_exited":
			var actionStatus game.ActionStatus
			if err := json.Unmarshal(event.Payload, &actionStatus); err != nil {
				continue
			}
			if _, ok := perPlayer[event.PlayerID]; !ok {
				return game.Snapshot{}, fmt.Errorf("history has unknown player %q", event.PlayerID)
			}
			if actionStatus.PlayerID == "" {
				actionStatus.PlayerID = event.PlayerID
			}
			playerState := perPlayer[event.PlayerID]
			playerState.LastActionStatus = actionStatus
			perPlayer[event.PlayerID] = playerState
			if actionStatus.ActionStatus == session.StatusAccepted {
				score[event.PlayerID]++
			}
		}
	}

	expected := targetTurn + 1
	if targetTurn >= maxTurns {
		expected = maxTurns
	}
	gameState, err := json.Marshal(map[string]any{
		"mode":     meta.TurnMode,
		"turn":     targetTurn,
		"expected": expected,
		"score":    score,
	})
	if err != nil {
		return game.Snapshot{}, fmt.Errorf("encode replay snapshot: %w", err)
	}

	return game.Snapshot{
		GameID:         meta.GameID,
		GameVersion:    meta.GameVersion,
		RulesetVersion: meta.RulesetVersion,
		Turn:           targetTurn,
		Status:         string(game.StatusRunning),
		GameState:      gameState,
		PerPlayer:      perPlayer,
	}, nil
}

func echoRulesetTurns(ruleset string) (int, error) {
	switch ruleset {
	case echo.RulesetSimultaneous3Turn, echo.RulesetSequential3Turn:
		return 3, nil
	case echo.RulesetSimultaneous2Turn:
		return 2, nil
	default:
		return 0, fmt.Errorf("unsupported echo ruleset %q", ruleset)
	}
}
