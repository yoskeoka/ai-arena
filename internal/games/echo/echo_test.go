package echo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestSimultaneousScoringAndPlacements(t *testing.T) {
	master := newTestMaster(t, RulesetSimultaneous3Turn)

	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}
	if step.Mode != game.Simultaneous || len(step.Requests) != 2 {
		t.Fatalf("unexpected step: %+v", step)
	}

	outcomes := []game.ActionStatus{
		master.NormalizeAction(step.Requests[0], game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted, Action: raw(`{"echo":1}`)}),
		master.NormalizeAction(step.Requests[1], game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusAccepted, Action: raw(`{"echo":1}`)}),
	}
	if err := master.ApplyStep(context.Background(), *step, outcomes); err != nil {
		t.Fatalf("ApplyStep: %v", err)
	}

	result := master.Result()
	if got := result.Placements[0].Place; got != 1 {
		t.Fatalf("first placement place = %d, want 1", got)
	}
}

func TestSequentialAdvancesPlayerOrder(t *testing.T) {
	master := newTestMaster(t, RulesetSequential3Turn)

	step1, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep 1: %v", err)
	}
	if got := step1.Requests[0].PlayerID; got != "p1" {
		t.Fatalf("first player = %q, want p1", got)
	}
	if err := master.ApplyStep(context.Background(), *step1, []game.ActionStatus{
		master.NormalizeAction(step1.Requests[0], game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted, Action: raw(`{"echo":1}`)}),
	}); err != nil {
		t.Fatalf("ApplyStep 1: %v", err)
	}

	step2, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep 2: %v", err)
	}
	if got := step2.Requests[0].PlayerID; got != "p2" {
		t.Fatalf("second player = %q, want p2", got)
	}
	if step2.Turn != 1 {
		t.Fatalf("second step turn = %d, want 1", step2.Turn)
	}
}

func TestNormalizeIllegalActionBecomesNoAction(t *testing.T) {
	master := newTestMaster(t, RulesetSimultaneous3Turn)

	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}
	actionStatus := master.NormalizeAction(step.Requests[0], game.ActionStatus{
		PlayerID:     "p1",
		ActionStatus: session.StatusAccepted,
		Action:       raw(`{"echo":999}`),
	})
	if actionStatus.ActionStatus != session.StatusNoAction {
		t.Fatalf("action status = %q, want no_action", actionStatus.ActionStatus)
	}
	if actionStatus.FailureReason != "invalid-illegal-action" {
		t.Fatalf("failure reason = %q, want invalid-illegal-action", actionStatus.FailureReason)
	}
}

func TestNewFromSnapshotRestoresNextTurnState(t *testing.T) {
	master, err := NewFromSnapshot(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSimultaneous3Turn,
		Players: []game.Player{
			{PlayerID: "p1"},
			{PlayerID: "p2"},
		},
	}, game.Snapshot{
		GameID:         GameID,
		GameVersion:    GameVersion,
		RulesetVersion: RulesetSimultaneous3Turn,
		Turn:           1,
		GameState: json.RawMessage(`{
			"mode":"simultaneous",
			"turn":1,
			"expected":2,
			"score":{"p1":1,"p2":1}
		}`),
		PerPlayer: map[string]game.PlayerSnapshot{
			"p1": {LastActionStatus: game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted}},
			"p2": {LastActionStatus: game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusAccepted}},
		},
	})
	if err != nil {
		t.Fatalf("NewFromSnapshot: %v", err)
	}

	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}
	if step == nil {
		t.Fatal("NextStep = nil, want turn 2 request")
	}
	if step.Turn != 2 {
		t.Fatalf("step.Turn = %d, want 2", step.Turn)
	}

	snapshot := master.Snapshot()
	if snapshot.Turn != 1 {
		t.Fatalf("snapshot.Turn = %d, want 1", snapshot.Turn)
	}
	exported := master.ExportedSnapshot()
	if len(exported.Players) != 2 {
		t.Fatalf("len(exported.Players) = %d, want 2", len(exported.Players))
	}
}

func newTestMaster(t *testing.T, ruleset string) *Master {
	t.Helper()

	master, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     ruleset,
		Players: []game.Player{
			{PlayerID: "p1", AIID: "bot-a"},
			{PlayerID: "p2", AIID: "bot-b"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return master
}

func raw(s string) []byte {
	return []byte(s)
}
