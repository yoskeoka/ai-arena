package echo

import (
	"context"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestSimultaneousScoringAndPlacements(t *testing.T) {
	master := newTestMaster(t, game.Simultaneous)

	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}
	if step.Mode != game.Simultaneous || len(step.Requests) != 2 {
		t.Fatalf("unexpected step: %+v", step)
	}

	outcomes := []game.ActionOutcome{
		master.NormalizeAction(step.Requests[0], game.ActionOutcome{PlayerID: "p1", Outcome: session.OutcomeAccepted, Action: raw(`{"echo":1}`)}),
		master.NormalizeAction(step.Requests[1], game.ActionOutcome{PlayerID: "p2", Outcome: session.OutcomeAccepted, Action: raw(`{"echo":1}`)}),
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
	master := newTestMaster(t, game.Sequential)

	step1, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep 1: %v", err)
	}
	if got := step1.Requests[0].PlayerID; got != "p1" {
		t.Fatalf("first player = %q, want p1", got)
	}
	if err := master.ApplyStep(context.Background(), *step1, []game.ActionOutcome{
		master.NormalizeAction(step1.Requests[0], game.ActionOutcome{PlayerID: "p1", Outcome: session.OutcomeAccepted, Action: raw(`{"echo":1}`)}),
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
	master := newTestMaster(t, game.Simultaneous)

	step, err := master.NextStep(context.Background())
	if err != nil {
		t.Fatalf("NextStep: %v", err)
	}
	outcome := master.NormalizeAction(step.Requests[0], game.ActionOutcome{
		PlayerID: "p1",
		Outcome:  session.OutcomeAccepted,
		Action:   raw(`{"echo":999}`),
	})
	if outcome.Outcome != session.OutcomeNoAction {
		t.Fatalf("outcome = %q, want no_action", outcome.Outcome)
	}
	if outcome.FailureReason != "invalid-illegal-action" {
		t.Fatalf("failure reason = %q, want invalid-illegal-action", outcome.FailureReason)
	}
}

func newTestMaster(t *testing.T, mode game.DecisionMode) *Master {
	t.Helper()

	master, err := New(Config{
		Mode:  mode,
		Turns: 2,
		Players: []game.Player{
			{PlayerID: "p1", AIID: "bot-a"},
			{PlayerID: "p2", AIID: "bot-b"},
		},
		Deadline: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return master
}

func raw(s string) []byte {
	return []byte(s)
}
