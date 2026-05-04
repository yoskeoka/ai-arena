package echo

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

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
