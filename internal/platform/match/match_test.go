package match

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
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
				{Status: session.StatusAccepted, Payload: raw(`{"action":"a"}`)},
				{
					Status:                 session.StatusAccepted,
					Payload:                raw(`{"action":"c"}`),
					IgnoredLateResponseIDs: []string{"turn-1-p1"},
				},
			},
		},
		"p2": &fakeSession{
			turnResults: []session.Result{
				{Status: session.StatusAccepted, Payload: raw(`{"action":"b"}`)},
				{Status: session.StatusNoAction, FailureReason: session.ReasonTimeout},
			},
		},
	}

	record, err := NewRunner("match-001", players, master, sessions).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if record.Status != game.StatusCompleted {
		t.Fatalf("record.Status = %q, want completed", record.Status)
	}
	if len(master.applied) != 3 {
		t.Fatalf("ApplyStep call count = %d, want 3", len(master.applied))
	}
	if got := record.Snapshot.PerPlayer["p2"].LastActionStatus.FailureReason; got != session.ReasonTimeout {
		t.Fatalf("p2 failure reason = %q, want timeout", got)
	}
	if !hasEventKind(record.EventLog, "late_response_ignored") {
		t.Fatalf("event log missing late_response_ignored: %+v", record.EventLog)
	}
	if !hasEventKind(record.EventLog, "session_shutdown_completed") {
		t.Fatalf("event log missing session_shutdown_completed: %+v", record.EventLog)
	}
	if got := sessions["p1"].(*fakeSession).initRequests[0].Deadline; got != defaultInitAckDeadline {
		t.Fatalf("init deadline = %s, want %s", got, defaultInitAckDeadline)
	}
	if record.ExportedSnapshot.MatchID != "match-001" {
		t.Fatalf("exported snapshot match id = %q, want match-001", record.ExportedSnapshot.MatchID)
	}
}

func TestRunnerUsesConfiguredInitAckDeadline(t *testing.T) {
	t.Setenv(initAckTimeoutEnv, "2200ms")

	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{metadata: baseMetadata()}
	sess := &fakeSession{}

	record, err := NewRunner("match-init-timeout-env", players, master, map[string]PlayerSession{"p1": sess}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if record.Status != game.StatusCompleted {
		t.Fatalf("record.Status = %q, want completed", record.Status)
	}
	if got := sess.initRequests[0].Deadline; got != 2200*time.Millisecond {
		t.Fatalf("init deadline = %s, want 2.2s", got)
	}
}

func TestRunnerFallsBackToDefaultInitAckDeadlineForInvalidOverride(t *testing.T) {
	t.Setenv(initAckTimeoutEnv, "invalid")

	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{metadata: baseMetadata()}
	sess := &fakeSession{}

	record, err := NewRunner("match-init-timeout-invalid", players, master, map[string]PlayerSession{"p1": sess}).Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if record.Status != game.StatusCompleted {
		t.Fatalf("record.Status = %q, want completed", record.Status)
	}
	if got := sess.initRequests[0].Deadline; got != defaultInitAckDeadline {
		t.Fatalf("init deadline = %s, want %s", got, defaultInitAckDeadline)
	}
}

func TestRunnerReturnsFailedRecordForInitFailure(t *testing.T) {
	players := []game.Player{{PlayerID: "p1", AIID: "bot-a"}}
	master := &fakeMaster{
		metadata: baseMetadata(),
	}
	sessions := map[string]PlayerSession{
		"p1": &fakeSession{
			initResult: session.Result{Status: session.StatusNoAction, FailureReason: session.ReasonRuntimeStop},
		},
	}

	record, err := NewRunner("match-002", players, master, sessions).Run(context.Background())
	if err == nil {
		t.Fatal("Run returned nil error, want init failure")
	}
	if record.Status != game.StatusFailed {
		t.Fatalf("record.Status = %q, want failed", record.Status)
	}
	if !hasEventKind(record.EventLog, "runtime_exited") {
		t.Fatalf("event log missing runtime_exited: %+v", record.EventLog)
	}
	if !hasEventKind(record.EventLog, "session_initialization_failed") {
		t.Fatalf("event log missing session_initialization_failed: %+v", record.EventLog)
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
	if record.Status != game.StatusCanceled {
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
	if record.Status != game.StatusCanceled {
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
			gameOverResult: session.Result{Status: session.StatusNoAction, FailureReason: session.ReasonTimeout},
			closeErr:       errors.New("close failed"),
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
	applied          [][]game.ActionStatus
	initErr          error
	cancelOnNextStep bool
}

func (f *fakeMaster) Metadata() catalog.GameMetadata {
	return f.metadata
}

func (f *fakeMaster) InitializeMatch(context.Context) (game.InitState, error) {
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

func (f *fakeMaster) NextDecisionStep(context.Context) (*game.DecisionStep, error) {
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

func (f *fakeMaster) NormalizeAction(_ context.Context, _ game.DecisionRequest, actionStatus game.ActionStatus) (game.ActionStatus, error) {
	return actionStatus, nil
}

func (f *fakeMaster) ApplyDecisionResults(_ context.Context, _ game.DecisionStep, actionStatuses []game.ActionStatus) error {
	copied := make([]game.ActionStatus, len(actionStatuses))
	copy(copied, actionStatuses)
	f.applied = append(f.applied, copied)
	return nil
}

func (f *fakeMaster) CurrentSnapshot(context.Context) (game.Snapshot, error) {
	return game.Snapshot{
		Turn:      2,
		Status:    game.StatusRunning,
		GameState: raw(`{"phase":"done"}`),
		PerPlayer: map[string]game.PlayerSnapshot{
			"p1": {VisibleState: raw(`{"visible":"current"}`)},
			"p2": {VisibleState: raw(`{"visible":"current"}`)},
		},
	}, nil
}

func (f *fakeMaster) CurrentExportedSnapshot(context.Context) (game.ExportedSnapshot, error) {
	return game.ExportedSnapshot{
		Turn:        2,
		Status:      game.StatusRunning,
		PublicState: raw(`{"public":"done"}`),
	}, nil
}

func (f *fakeMaster) CurrentResult(context.Context) (game.MatchResult, error) {
	return game.MatchResult{
		Placements: []game.Placement{
			{PlayerID: "p1", Place: 1},
			{PlayerID: "p2", Place: 2},
		},
	}, nil
}

func (f *fakeMaster) Shutdown(context.Context) error {
	return nil
}

type fakeSession struct {
	initResult     session.Result
	initRequests   []session.Request
	turnResults    []session.Result
	turnIndex      int
	gameOverResult session.Result
	closeErr       error
}

func (f *fakeSession) Init(_ context.Context, req session.Request) session.Result {
	f.initRequests = append(f.initRequests, req)
	if f.initResult.Status == "" {
		return session.Result{Status: session.StatusAccepted, Payload: raw(`{"ready":true}`)}
	}
	return f.initResult
}

func (f *fakeSession) Turn(context.Context, session.Request) session.Result {
	if f.turnIndex >= len(f.turnResults) {
		return session.Result{Status: session.StatusAccepted, Payload: raw(`{"action":"noop"}`)}
	}
	result := f.turnResults[f.turnIndex]
	f.turnIndex++
	return result
}

func (f *fakeSession) GameOver(context.Context, session.Request) session.Result {
	if f.gameOverResult.Status == "" {
		return session.Result{Status: session.StatusAccepted, Payload: raw(`{"ack":true}`)}
	}
	return f.gameOverResult
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
	}
}

var _ gamemaster.Session = (*fakeMaster)(nil)
