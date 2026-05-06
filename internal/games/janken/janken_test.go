package janken

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestNormalizeActionRejectsInvalidPayload(t *testing.T) {
	master := newTestMaster(t)
	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}

	status := master.NormalizeAction(step.Requests[0], game.ActionStatus{
		PlayerID:     "p1",
		ActionStatus: session.StatusAccepted,
		Action:       json.RawMessage(`{"action":"lizard"}`),
	})
	if status.ActionStatus != session.StatusNoAction {
		t.Fatalf("status = %q, want no_action", status.ActionStatus)
	}
	if status.FailureReason != "invalid-illegal-action" {
		t.Fatalf("failure_reason = %q, want invalid-illegal-action", status.FailureReason)
	}
}

func TestApplyStepKeepsCurrentRoundHiddenUntilResolution(t *testing.T) {
	master := newTestMaster(t)
	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}

	assertVisibleStateRounds(t, master.VisibleState("p1"), 1, 0)

	statuses := []game.ActionStatus{
		acceptedAction("p1", "rock"),
		acceptedAction("p2", "scissors"),
	}
	if err := master.ApplyStep(context.Background(), *step, statuses); err != nil {
		t.Fatalf("ApplyStep: %v", err)
	}

	assertVisibleStateRounds(t, master.VisibleState("p1"), 2, 1)
	var state struct {
		PublicHistory []struct {
			Actions map[string]string `json:"actions"`
		} `json:"public_history"`
	}
	if err := json.Unmarshal(master.VisibleState("p2"), &state); err != nil {
		t.Fatalf("decode visible state: %v", err)
	}
	if got := state.PublicHistory[0].Actions["p1"]; got != "rock" {
		t.Fatalf("revealed p1 action = %q, want rock", got)
	}
}

func TestApplyStepCountsTimeoutInvalidAndTieBreak(t *testing.T) {
	master := newThreePlayerMaster(t)
	round1, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep round1: %v", err)
	}
	if err := master.ApplyStep(context.Background(), *round1, []game.ActionStatus{
		acceptedAction("p1", "rock"),
		timeoutAction("p2"),
		invalidAction("p3"),
	}); err != nil {
		t.Fatalf("ApplyStep round1: %v", err)
	}

	round2, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep round2: %v", err)
	}
	if err := master.ApplyStep(context.Background(), *round2, []game.ActionStatus{
		acceptedAction("p1", "rock"),
		acceptedAction("p2", "rock"),
		acceptedAction("p3", "rock"),
	}); err != nil {
		t.Fatalf("ApplyStep round2: %v", err)
	}

	round3, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep round3: %v", err)
	}
	if err := master.ApplyStep(context.Background(), *round3, []game.ActionStatus{
		acceptedAction("p1", "scissors"),
		acceptedAction("p2", "rock"),
		acceptedAction("p3", "paper"),
	}); err != nil {
		t.Fatalf("ApplyStep round3: %v", err)
	}

	for round := 4; round <= 5; round++ {
		step, err := master.NextStep(context.Background())
		if err != nil {
			t.Fatalf("NextStep round%d: %v", round, err)
		}
		if err := master.ApplyStep(context.Background(), *step, []game.ActionStatus{
			acceptedAction("p1", "rock"),
			acceptedAction("p2", "paper"),
			acceptedAction("p3", "paper"),
		}); err != nil {
			t.Fatalf("ApplyStep round%d: %v", round, err)
		}
	}

	result := master.Result()
	if result.Placements[0].PlayerID != "p3" || result.Placements[0].Place != 1 {
		t.Fatalf("first placement = %+v, want p3 place 1", result.Placements[0])
	}
	if result.Placements[1].PlayerID != "p2" || result.Placements[1].Place != 2 {
		t.Fatalf("second placement = %+v, want p2 place 2", result.Placements[1])
	}
	if result.Placements[2].PlayerID != "p1" || result.Placements[2].Place != 3 {
		t.Fatalf("third placement = %+v, want p1 place 3", result.Placements[2])
	}

	snapshot := master.Snapshot()
	var state snapshotState
	if err := json.Unmarshal(snapshot.GameState, &state); err != nil {
		t.Fatalf("decode snapshot game_state: %v", err)
	}
	if state.Scores["p2"].Timeouts != 1 {
		t.Fatalf("p2 timeouts = %d, want 1", state.Scores["p2"].Timeouts)
	}
	if state.Scores["p3"].InvalidActions != 1 {
		t.Fatalf("p3 invalid_actions = %d, want 1", state.Scores["p3"].InvalidActions)
	}
}

func TestNewFromSnapshotRestoresState(t *testing.T) {
	master := newTestMaster(t)
	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}
	if err := master.ApplyStep(context.Background(), *step, []game.ActionStatus{
		acceptedAction("p1", "paper"),
		acceptedAction("p2", "rock"),
	}); err != nil {
		t.Fatalf("ApplyStep: %v", err)
	}

	snapshot := master.Snapshot()
	snapshot.PerPlayer = map[string]game.PlayerSnapshot{
		"p1": {LastActionStatus: acceptedAction("p1", "paper")},
		"p2": {LastActionStatus: acceptedAction("p2", "rock")},
	}

	resumed, err := NewFromSnapshot(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetRegular,
		Players: []game.Player{
			{PlayerID: "p1"},
			{PlayerID: "p2"},
		},
	}, snapshot)
	if err != nil {
		t.Fatalf("NewFromSnapshot: %v", err)
	}

	assertVisibleStateRounds(t, resumed.VisibleState("p1"), 2, 1)
}

func newTestMaster(t *testing.T) *Master {
	t.Helper()
	master, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetRegular,
		Players: []game.Player{
			{PlayerID: "p1"},
			{PlayerID: "p2"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return master
}

func newThreePlayerMaster(t *testing.T) *Master {
	t.Helper()
	master, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetRegular,
		Players: []game.Player{
			{PlayerID: "p1"},
			{PlayerID: "p2"},
			{PlayerID: "p3"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return master
}

func acceptedAction(playerID, choice string) game.ActionStatus {
	return game.ActionStatus{
		PlayerID:     playerID,
		ActionStatus: session.StatusAccepted,
		Action:       json.RawMessage(`{"action":"` + choice + `"}`),
	}
}

func timeoutAction(playerID string) game.ActionStatus {
	return game.ActionStatus{
		PlayerID:      playerID,
		ActionStatus:  session.StatusNoAction,
		FailureReason: session.ReasonTimeout,
	}
}

func invalidAction(playerID string) game.ActionStatus {
	return game.ActionStatus{
		PlayerID:      playerID,
		ActionStatus:  session.StatusNoAction,
		FailureReason: "invalid-illegal-action",
	}
}

func assertVisibleStateRounds(t *testing.T, raw json.RawMessage, round, publicRounds int) {
	t.Helper()
	var state struct {
		Round         int               `json:"round"`
		PublicHistory []json.RawMessage `json:"public_history"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode visible state: %v", err)
	}
	if state.Round != round {
		t.Fatalf("state.Round = %d, want %d", state.Round, round)
	}
	if len(state.PublicHistory) != publicRounds {
		t.Fatalf("len(public_history) = %d, want %d", len(state.PublicHistory), publicRounds)
	}
}
