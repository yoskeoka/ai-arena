package match

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestRunnerHandlesSimultaneousAndSequentialAndBuildsRecord(t *testing.T) {
	players := []game.Player{
		{PlayerID: "p1", AIID: "bot-a"},
		{PlayerID: "p2", AIID: "bot-b"},
	}
	master := &fakeMaster{
		metadata: catalog.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2",
			TurnMode:       "simultaneous",
		},
		windows: []*game.DecisionWindow{
			{
				Turn: 1,
				Mode: game.Simultaneous,
				Requests: []game.DecisionRequest{
					{PlayerID: "p1", VisibleState: raw(`{"round":1}`), LegalActionHint: raw(`["a"]`), Deadline: time.Second},
					{PlayerID: "p2", VisibleState: raw(`{"round":1}`), LegalActionHint: raw(`["b"]`), Deadline: time.Second},
				},
			},
			{
				Turn: 2,
				Mode: game.Sequential,
				Requests: []game.DecisionRequest{
					{PlayerID: "p1", VisibleState: raw(`{"round":2}`), LegalActionHint: raw(`["c"]`), Deadline: time.Second},
					{PlayerID: "p2", VisibleState: raw(`{"round":2}`), LegalActionHint: raw(`["d"]`), Deadline: time.Second},
				},
			},
		},
	}
	sessions := map[string]PlayerSession{
		"p1": &fakeSession{
			turnResults: []session.Result{
				{Outcome: session.OutcomeAccepted, Payload: raw(`{"action":"a"}`)},
				{
					Outcome:                session.OutcomeAccepted,
					Payload:                raw(`{"action":"c"}`),
					IgnoredLateResponseIDs: []string{"turn-1-p1"},
				},
			},
		},
		"p2": &fakeSession{
			turnResults: []session.Result{
				{Outcome: session.OutcomeAccepted, Payload: raw(`{"action":"b"}`)},
				{Outcome: session.OutcomeNoAction, FailureReason: session.ReasonTimeout},
			},
		},
	}

	record, err := NewRunner("match-001", players, master, sessions).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if record.Status != "completed" {
		t.Fatalf("record.Status = %q, want completed", record.Status)
	}
	if len(master.applied) != 3 {
		t.Fatalf("ApplyDecision call count = %d, want 3", len(master.applied))
	}
	if got := record.Snapshot.PerPlayer["p2"].LastOutcome.FailureReason; got != session.ReasonTimeout {
		t.Fatalf("p2 failure reason = %q, want timeout", got)
	}
	if len(record.EventLog) == 0 {
		t.Fatal("event log is empty")
	}
	if record.ExportedSnapshot.MatchID != "match-001" {
		t.Fatalf("exported snapshot match id = %q", record.ExportedSnapshot.MatchID)
	}
	if !hasEventKind(record.EventLog, "late_response_ignored") {
		t.Fatalf("event log missing late_response_ignored: %+v", record.EventLog)
	}
}

type fakeMaster struct {
	metadata catalog.GameMetadata
	windows  []*game.DecisionWindow
	index    int
	applied  [][]game.ActionOutcome
}

func (f *fakeMaster) Metadata() catalog.GameMetadata {
	return f.metadata
}

func (f *fakeMaster) Init(context.Context) (game.InitState, error) {
	return game.InitState{
		PerPlayer: map[string]json.RawMessage{
			"p1": raw(`{"ready":true}`),
			"p2": raw(`{"ready":true}`),
		},
	}, nil
}

func (f *fakeMaster) NextDecision(context.Context) (*game.DecisionWindow, error) {
	if f.index >= len(f.windows) {
		return nil, nil
	}
	window := f.windows[f.index]
	f.index++
	return window, nil
}

func (f *fakeMaster) ApplyDecision(_ context.Context, _ game.DecisionWindow, outcomes []game.ActionOutcome) error {
	copied := make([]game.ActionOutcome, len(outcomes))
	copy(copied, outcomes)
	f.applied = append(f.applied, copied)
	return nil
}

func (f *fakeMaster) Snapshot() game.Snapshot {
	return game.Snapshot{
		Turn:      2,
		Status:    "running",
		GameState: raw(`{"phase":"done"}`),
	}
}

func (f *fakeMaster) ExportedSnapshot() game.ExportedSnapshot {
	return game.ExportedSnapshot{
		Turn:        2,
		Status:      "running",
		PublicState: raw(`{"public":"done"}`),
	}
}

func (f *fakeMaster) Result() game.MatchResult {
	return game.MatchResult{
		Placements: []game.Placement{
			{PlayerID: "p1", Place: 1},
			{PlayerID: "p2", Place: 2},
		},
	}
}

type fakeSession struct {
	initResult  session.Result
	turnResults []session.Result
	turnIndex   int
}

func (f *fakeSession) Init(context.Context, session.Request) session.Result {
	if f.initResult.Outcome == "" {
		return session.Result{Outcome: session.OutcomeAccepted, Payload: raw(`{"ready":true}`)}
	}
	return f.initResult
}

func (f *fakeSession) Turn(context.Context, session.Request) session.Result {
	result := f.turnResults[f.turnIndex]
	f.turnIndex++
	return result
}

func (f *fakeSession) GameOver(context.Context, any) error {
	return nil
}

func (f *fakeSession) StderrSnapshot() runtime.StderrSnapshot {
	return runtime.StderrSnapshot{BytesRead: 4}
}

func raw(s string) json.RawMessage {
	return json.RawMessage(s)
}

func hasEventKind(events []Event, kind string) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}
