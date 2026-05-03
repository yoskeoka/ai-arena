package match

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func TestRunnerBuildsCompletedRecordAcrossSimultaneousAndSequentialSteps(t *testing.T) {
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
		steps: []*game.DecisionStep{
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
				},
			},
			{
				Turn: 2,
				Mode: game.Sequential,
				Requests: []game.DecisionRequest{
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

	if record.Status != string(game.StatusCompleted) {
		t.Fatalf("record.Status = %q, want completed", record.Status)
	}
	if len(master.applied) != 3 {
		t.Fatalf("ApplyStep call count = %d, want 3", len(master.applied))
	}
	if got := record.Snapshot.PerPlayer["p2"].LastOutcome.FailureReason; got != session.ReasonTimeout {
		t.Fatalf("p2 failure reason = %q, want timeout", got)
	}
	if !hasEventKind(record.EventLog, "late_response_ignored") {
		t.Fatalf("event log missing late_response_ignored: %+v", record.EventLog)
	}
	if !hasEventKind(record.EventLog, "session_shutdown_completed") {
		t.Fatalf("event log missing session_shutdown_completed: %+v", record.EventLog)
	}
	if record.ExportedSnapshot.MatchID != "match-001" {
		t.Fatalf("exported snapshot match id = %q, want match-001", record.ExportedSnapshot.MatchID)
	}
}

func TestRunnerReturnsFailedRecordForInitFailure(t *testing.T) {
	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{
		metadata: baseMetadata(),
	}
	sessions := map[string]PlayerSession{
		"p1": &fakeSession{
			initResult: session.Result{Outcome: session.OutcomeNoAction, FailureReason: session.ReasonRuntimeStop},
		},
	}

	record, err := NewRunner("match-002", players, master, sessions).Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want init failure")
	}
	if record.Status != string(game.StatusFailed) {
		t.Fatalf("record.Status = %q, want failed", record.Status)
	}
	if !hasEventKind(record.EventLog, "runtime_exited") {
		t.Fatalf("event log missing runtime_exited: %+v", record.EventLog)
	}
	if !hasEventKind(record.EventLog, "match_failed") {
		t.Fatalf("event log missing match_failed: %+v", record.EventLog)
	}
}

func TestRunnerReturnsCanceledRecordForInitCancellation(t *testing.T) {
	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{
		metadata: baseMetadata(),
		initErr:  context.Canceled,
	}
	sessions := map[string]PlayerSession{
		"p1": &fakeSession{},
	}

	record, err := NewRunner("match-002-canceled", players, master, sessions).Run(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
	if record.Status != string(game.StatusCanceled) {
		t.Fatalf("record.Status = %q, want canceled", record.Status)
	}
	if !hasEventKind(record.EventLog, "match_canceled") {
		t.Fatalf("event log missing match_canceled: %+v", record.EventLog)
	}
}

func TestRunnerReturnsCanceledRecordWhenContextCanceled(t *testing.T) {
	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{
		metadata:         baseMetadata(),
		cancelOnNextStep: true,
	}
	sessions := map[string]PlayerSession{
		"p1": &fakeSession{},
	}

	record, err := NewRunner("match-003", players, master, sessions).Run(context.Background())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context.Canceled", err)
	}
	if record.Status != string(game.StatusCanceled) {
		t.Fatalf("record.Status = %q, want canceled", record.Status)
	}
	if !hasEventKind(record.EventLog, "match_canceled") {
		t.Fatalf("event log missing match_canceled: %+v", record.EventLog)
	}
}

func TestRunnerLogsGameOverAndShutdownFailures(t *testing.T) {
	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{
		metadata: baseMetadata(),
	}
	sessions := map[string]PlayerSession{
		"p1": &fakeSession{
			gameOverErr: errors.New("game_over failed"),
			closeErr:    errors.New("close failed"),
		},
	}

	record, err := NewRunner("match-004", players, master, sessions).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasEventKind(record.EventLog, "session_shutdown_failed") {
		t.Fatalf("event log missing session_shutdown_failed: %+v", record.EventLog)
	}
	if !hasEventKind(record.EventLog, "runtime_exited") {
		t.Fatalf("event log missing runtime_exited: %+v", record.EventLog)
	}
}

type fakeMaster struct {
	metadata         catalog.GameMetadata
	steps            []*game.DecisionStep
	index            int
	applied          [][]game.ActionOutcome
	initErr          error
	cancelOnNextStep bool
}

func (f *fakeMaster) Metadata() catalog.GameMetadata {
	return f.metadata
}

func (f *fakeMaster) Init(context.Context) (game.InitState, error) {
	if f.initErr != nil {
		return game.InitState{}, f.initErr
	}
	return game.InitState{
		PerPlayer: map[string]json.RawMessage{
			"p1": raw(`{"ready":true}`),
			"p2": raw(`{"ready":true}`),
		},
	}, nil
}

func (f *fakeMaster) NextStep(context.Context) (*game.DecisionStep, error) {
	if f.cancelOnNextStep {
		return nil, context.Canceled
	}
	if f.index >= len(f.steps) {
		return nil, nil
	}
	step := f.steps[f.index]
	f.index++
	return step, nil
}

func (f *fakeMaster) NormalizeAction(_ game.DecisionRequest, outcome game.ActionOutcome) game.ActionOutcome {
	return outcome
}

func (f *fakeMaster) ApplyStep(_ context.Context, _ game.DecisionStep, outcomes []game.ActionOutcome) error {
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
	gameOverErr error
	closeErr    error
}

func (f *fakeSession) Init(context.Context, session.Request) session.Result {
	if f.initResult.Outcome == "" {
		return session.Result{Outcome: session.OutcomeAccepted, Payload: raw(`{"ready":true}`)}
	}
	return f.initResult
}

func (f *fakeSession) Turn(context.Context, session.Request) session.Result {
	if f.turnIndex >= len(f.turnResults) {
		return session.Result{Outcome: session.OutcomeAccepted, Payload: raw(`{"action":"noop"}`)}
	}
	result := f.turnResults[f.turnIndex]
	f.turnIndex++
	return result
}

func (f *fakeSession) GameOver(context.Context, any) error {
	return f.gameOverErr
}

func (f *fakeSession) Close(context.Context) error {
	return f.closeErr
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

func baseMetadata() catalog.GameMetadata {
	return catalog.GameMetadata{
		GameID:         "echo-count",
		GameVersion:    "2.0.0",
		RulesetVersion: "phase2",
		TurnMode:       "simultaneous",
	}
}
