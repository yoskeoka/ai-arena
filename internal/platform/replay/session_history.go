package replay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

type sessionBuilder func() (gamemaster.Session, error)

// SnapshotFromSessionHistory rebuilds a snapshot by replaying persisted action statuses through a game-master session.
func SnapshotFromSessionHistory(build sessionBuilder, events []match.Event, targetTurn int) (game.Snapshot, error) {
	if targetTurn < 0 {
		return game.Snapshot{}, fmt.Errorf("target turn %d must be non-negative", targetTurn)
	}

	ctx := context.Background()
	session, err := build()
	if err != nil {
		return game.Snapshot{}, err
	}
	defer func() {
		_ = session.Shutdown(context.Background())
	}()

	if _, err := session.InitializeMatch(ctx); err != nil {
		return game.Snapshot{}, err
	}

	cursor := 0
	for {
		step, err := session.NextDecisionStep(ctx)
		if err != nil {
			return game.Snapshot{}, err
		}
		if step == nil {
			snapshot, err := session.CurrentSnapshot(ctx)
			if err != nil {
				return game.Snapshot{}, err
			}
			if snapshot.Turn < targetTurn {
				return game.Snapshot{}, fmt.Errorf("target turn %d exceeds terminal turn %d", targetTurn, snapshot.Turn)
			}
			return snapshot, nil
		}
		if step.Turn > targetTurn {
			return session.CurrentSnapshot(ctx)
		}

		actionStatuses := make([]game.ActionStatus, 0, len(step.Requests))
		for _, req := range step.Requests {
			status, nextCursor, err := consumeRecordedActionStatus(events, cursor, step.Turn, req.PlayerID)
			if err != nil {
				return game.Snapshot{}, err
			}
			cursor = nextCursor
			actionStatuses = append(actionStatuses, status)
		}
		if err := session.ApplyDecisionResults(ctx, *step, actionStatuses); err != nil {
			return game.Snapshot{}, err
		}
	}
}

func consumeRecordedActionStatus(events []match.Event, start int, turn int, playerID string) (game.ActionStatus, int, error) {
	for i := start; i < len(events); i++ {
		event := events[i]
		if !isRecordedActionStatusEvent(event.Kind) {
			continue
		}
		if event.Turn != turn {
			return game.ActionStatus{}, i, fmt.Errorf(
				"history mismatch: want turn %d player %q, got turn %d player %q kind %q",
				turn,
				playerID,
				event.Turn,
				event.PlayerID,
				event.Kind,
			)
		}
		if event.PlayerID != playerID {
			return game.ActionStatus{}, i, fmt.Errorf(
				"history mismatch: want turn %d player %q, got player %q kind %q",
				turn,
				playerID,
				event.PlayerID,
				event.Kind,
			)
		}
		var status game.ActionStatus
		if err := json.Unmarshal(event.Payload, &status); err != nil {
			return game.ActionStatus{}, i, fmt.Errorf(
				"decode history event payload seq=%d kind=%q turn=%d player_id=%q: %w",
				event.Seq,
				event.Kind,
				event.Turn,
				event.PlayerID,
				err,
			)
		}
		if status.PlayerID == "" {
			status.PlayerID = playerID
		}
		return status, i + 1, nil
	}
	return game.ActionStatus{}, len(events), fmt.Errorf("history mismatch: missing action status for turn %d player %q", turn, playerID)
}

func isRecordedActionStatusEvent(kind string) bool {
	switch kind {
	case "turn_result", "turn_timeout", "protocol_error", "runtime_exited":
		return true
	default:
		return false
	}
}
