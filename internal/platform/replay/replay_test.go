package replay

import (
	"encoding/json"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestSnapshotFromHistoryBuildsTurnBoundarySnapshot(t *testing.T) {
	snapshot, err := SnapshotFromHistory(catalog.GameMetadata{
		GameID:         echo.GameID,
		GameVersion:    echo.GameVersion,
		RulesetVersion: echo.RulesetSimultaneous3Turn,
		TurnMode:       string(game.Simultaneous),
	}, []game.Player{
		{PlayerID: "p1"},
		{PlayerID: "p2"},
	}, []match.Event{
		event(1, "turn_result", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted}),
		event(1, "turn_result", "p2", game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusAccepted}),
		event(2, "turn_timeout", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusNoAction, FailureReason: session.ReasonTimeout}),
		event(2, "turn_result", "p2", game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusAccepted}),
		event(3, "turn_result", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted}),
	}, 2)
	if err != nil {
		t.Fatalf("SnapshotFromHistory: %v", err)
	}
	if snapshot.Turn != 2 {
		t.Fatalf("snapshot.Turn = %d, want 2", snapshot.Turn)
	}
	if got := snapshot.PerPlayer["p1"].LastActionStatus.FailureReason; got != session.ReasonTimeout {
		t.Fatalf("p1 failure reason = %q, want %q", got, session.ReasonTimeout)
	}
	var state struct {
		Turn     int            `json:"turn"`
		Expected int            `json:"expected"`
		Score    map[string]int `json:"score"`
	}
	if err := json.Unmarshal(snapshot.GameState, &state); err != nil {
		t.Fatalf("decode snapshot.GameState: %v", err)
	}
	if state.Turn != 2 {
		t.Fatalf("state.Turn = %d, want 2", state.Turn)
	}
	if state.Expected != 3 {
		t.Fatalf("state.Expected = %d, want 3", state.Expected)
	}
	if state.Score["p1"] != 1 || state.Score["p2"] != 2 {
		t.Fatalf("state.Score = %+v, want p1=1 p2=2", state.Score)
	}
	var visible struct {
		Turn     int            `json:"turn"`
		Expected int            `json:"expected"`
		Score    map[string]int `json:"score"`
	}
	if err := json.Unmarshal(snapshot.PerPlayer["p2"].VisibleState, &visible); err != nil {
		t.Fatalf("decode snapshot.PerPlayer[p2].VisibleState: %v", err)
	}
	if visible.Turn != 3 {
		t.Fatalf("visible.Turn = %d, want 3", visible.Turn)
	}
	if visible.Expected != 3 {
		t.Fatalf("visible.Expected = %d, want 3", visible.Expected)
	}
	if visible.Score["p1"] != 1 || visible.Score["p2"] != 2 {
		t.Fatalf("visible.Score = %+v, want p1=1 p2=2", visible.Score)
	}
}

func TestSnapshotFromHistoryBuildsJankenRoundBoundary(t *testing.T) {
	snapshot, err := SnapshotFromHistory(catalog.GameMetadata{
		GameID:         janken.GameID,
		GameVersion:    janken.GameVersion,
		RulesetVersion: janken.RulesetRegular,
		TurnMode:       string(game.Simultaneous),
	}, []game.Player{
		{PlayerID: "p1"},
		{PlayerID: "p2"},
	}, []match.Event{
		event(1, "turn_result", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted, Action: json.RawMessage(`{"action":"rock"}`)}),
		event(1, "turn_timeout", "p2", game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusNoAction, FailureReason: session.ReasonTimeout}),
		event(2, "turn_result", "p1", game.ActionStatus{PlayerID: "p1", ActionStatus: session.StatusAccepted, Action: json.RawMessage(`{"action":"paper"}`)}),
		event(2, "protocol_error", "p2", game.ActionStatus{PlayerID: "p2", ActionStatus: session.StatusNoAction, FailureReason: "invalid-illegal-action"}),
	}, 2)
	if err != nil {
		t.Fatalf("SnapshotFromHistory: %v", err)
	}

	if snapshot.Turn != 2 {
		t.Fatalf("snapshot.Turn = %d, want 2", snapshot.Turn)
	}
	var state struct {
		PublicHistory []json.RawMessage `json:"public_history"`
	}
	if err := json.Unmarshal(snapshot.PerPlayer["p1"].VisibleState, &state); err != nil {
		t.Fatalf("decode visible_state: %v", err)
	}
	if len(state.PublicHistory) != 2 {
		t.Fatalf("len(public_history) = %d, want 2", len(state.PublicHistory))
	}
	if got := snapshot.PerPlayer["p2"].LastActionStatus.FailureReason; got != "invalid-illegal-action" {
		t.Fatalf("p2 failure reason = %q, want invalid-illegal-action", got)
	}
}

func event(turn int, kind, playerID string, payload any) match.Event {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return match.Event{Turn: turn, Kind: kind, PlayerID: playerID, Payload: raw}
}
