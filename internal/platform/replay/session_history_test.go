package replay

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestSnapshotFromSessionHistoryReplaysTurnBoundary(t *testing.T) {
	mockSession := &scriptedSession{
		meta: catalog.GameMetadata{
			GameID:         "test",
			GameVersion:    "1.0.0",
			RulesetVersion: "regular",
		},
		steps: []*game.DecisionStep{
			{
				Turn: 1,
				Mode: game.Simultaneous,
				Requests: []game.DecisionRequest{
					{PlayerID: "p1"},
					{PlayerID: "p2"},
				},
			},
			{
				Turn: 2,
				Mode: game.Sequential,
				Requests: []game.DecisionRequest{
					{PlayerID: "p1"},
				},
			},
			{
				Turn: 3,
				Mode: game.Sequential,
				Requests: []game.DecisionRequest{
					{PlayerID: "p2"},
				},
			},
		},
	}

	snapshot, err := SnapshotFromSessionHistory(func() (gamemaster.Session, error) {
		return mockSession, nil
	}, []match.Event{
		actionEvent(1, "turn_result", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted}),
		actionEvent(1, "turn_timeout", "p2", game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusNoAction, FailureReason: session.ReasonTimeout}),
		actionEvent(2, "turn_result", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted}),
	}, 2)
	if err != nil {
		t.Fatalf("SnapshotFromSessionHistory: %v", err)
	}
	if snapshot.Turn != 2 {
		t.Fatalf("snapshot.Turn = %d, want 2", snapshot.Turn)
	}
	if len(mockSession.applied) != 2 {
		t.Fatalf("len(applied) = %d, want 2", len(mockSession.applied))
	}
	if got := mockSession.applied[0][1].FailureReason; got != session.ReasonTimeout {
		t.Fatalf("applied timeout reason = %q, want %q", got, session.ReasonTimeout)
	}
}

func TestSnapshotFromSessionHistoryDetectsHistoryMismatch(t *testing.T) {
	_, err := SnapshotFromSessionHistory(func() (gamemaster.Session, error) {
		return &scriptedSession{
			meta: catalog.GameMetadata{
				GameID:         "test",
				GameVersion:    "1.0.0",
				RulesetVersion: "regular",
			},
			steps: []*game.DecisionStep{
				{
					Turn: 1,
					Mode: game.Sequential,
					Requests: []game.DecisionRequest{
						{PlayerID: "p1"},
					},
				},
			},
		}, nil
	}, []match.Event{
		actionEvent(1, "turn_result", "p2", game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusAccepted}),
	}, 1)
	if err == nil || !strings.Contains(err.Error(), "history mismatch") {
		t.Fatalf("SnapshotFromSessionHistory error = %v, want history mismatch", err)
	}
}

func actionEvent(turn int, kind, playerID string, payload any) match.Event {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return match.Event{Turn: turn, Kind: kind, PlayerID: playerID, Payload: raw}
}

type scriptedSession struct {
	meta        catalog.GameMetadata
	steps       []*game.DecisionStep
	stepIndex   int
	applied     [][]game.ActionStatus
	snapshot    game.Snapshot
	initialized bool
}

func (s *scriptedSession) Metadata() catalog.GameMetadata {
	return s.meta
}

func (s *scriptedSession) InitializeMatch(context.Context) (game.InitState, error) {
	s.initialized = true
	return game.InitState{}, nil
}

func (s *scriptedSession) NextDecisionStep(context.Context) (*game.DecisionStep, error) {
	if s.stepIndex >= len(s.steps) {
		return nil, nil
	}
	step := s.steps[s.stepIndex]
	s.stepIndex++
	return step, nil
}

func (s *scriptedSession) NormalizeAction(context.Context, game.DecisionRequest, game.ActionStatus) (game.ActionStatus, error) {
	panic("NormalizeAction should not be called during history replay")
}

func (s *scriptedSession) ApplyDecisionResults(_ context.Context, step game.DecisionStep, actionStatuses []game.ActionStatus) error {
	s.applied = append(s.applied, append([]game.ActionStatus(nil), actionStatuses...))
	s.snapshot.Turn = step.Turn
	return nil
}

func (s *scriptedSession) CurrentSnapshot(context.Context) (game.Snapshot, error) {
	return s.snapshot, nil
}

func (s *scriptedSession) CurrentExportedSnapshot(context.Context) (game.ExportedSnapshot, error) {
	return game.ExportedSnapshot{}, nil
}

func (s *scriptedSession) CurrentResult(context.Context) (game.MatchResult, error) {
	return game.MatchResult{}, nil
}

func (s *scriptedSession) Shutdown(context.Context) error {
	return nil
}
