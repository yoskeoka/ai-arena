package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

func TestCommandServiceSubmitSuccess(t *testing.T) {
	commands := newTestCommandService(t)

	record, err := commands.Submit(context.Background(), testSubmission("file://"+repoJoin(t, "testdata/ai/janken/janken-rock-ai")))
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if record.State != StateQueued {
		t.Fatalf("record.State = %q, want %q", record.State, StateQueued)
	}
	if record.Submission.Players[0].ArtifactRef == "" {
		t.Fatal("record submission artifact_ref should be preserved")
	}
	if record.Lease != nil {
		t.Fatalf("record.Lease = %+v, want nil", record.Lease)
	}
}

func TestCommandServiceRejectsIncompatibleArtifact(t *testing.T) {
	commands := newTestCommandService(t)

	_, err := commands.Submit(context.Background(), testSubmission(repoJoin(t, "testdata/ai/echo/echo-ai")))
	if err == nil {
		t.Fatal("Submit() returned nil error")
	}
}

func TestCommandServiceCancelQueued(t *testing.T) {
	commands := newTestCommandService(t)
	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))

	if _, err := commands.Submit(context.Background(), submission); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	record, err := commands.Cancel(context.Background(), submission.SubmissionID)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if record.State != StateCanceled {
		t.Fatalf("record.State = %q, want %q", record.State, StateCanceled)
	}
	if _, err := commands.Cancel(context.Background(), submission.SubmissionID); err == nil {
		t.Fatal("Cancel() returned nil error for terminal record")
	}
}

func TestCommandServiceRejectsUnsupportedRuleset(t *testing.T) {
	commands := newTestCommandService(t)
	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission.Game.RulesetVersion = "unsupported-ruleset"

	_, err := commands.Submit(context.Background(), submission)
	if err == nil {
		t.Fatal("Submit() returned nil error")
	}
	if !strings.Contains(err.Error(), "is not supported") {
		t.Fatalf("Submit() error = %v, want unsupported ruleset error", err)
	}
}

func TestInMemoryQueueStoreRejectsCancelAfterClaim(t *testing.T) {
	store := NewInMemoryQueueStore()
	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))

	if _, err := store.Enqueue(context.Background(), submission); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	if _, err := store.Claim(context.Background(), "worker-1"); err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if _, err := store.CancelQueued(context.Background(), submission.SubmissionID); err == nil {
		t.Fatal("CancelQueued() returned nil error")
	}
	if len(store.order) != 0 {
		t.Fatalf("len(store.order) = %d, want 0", len(store.order))
	}
}

func TestInMemoryQueueStoreReportsMissingRecord(t *testing.T) {
	store := NewInMemoryQueueStore()
	if _, err := store.CancelQueued(context.Background(), "missing"); !errors.Is(err, ErrQueueRecordNotFound) {
		t.Fatalf("CancelQueued() error = %v, want %v", err, ErrQueueRecordNotFound)
	}
}

func TestInMemoryQueueStoreCopiesSubmissionPlayers(t *testing.T) {
	store := NewInMemoryQueueStore()
	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))

	record, err := store.Enqueue(context.Background(), submission)
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	submission.Players[0].ArtifactRef = "mutated"
	if record.Submission.Players[0].ArtifactRef == "mutated" {
		t.Fatal("record submission changed after caller mutation")
	}
}

func TestEnsureCommandStartableSupportsGoRunFlags(t *testing.T) {
	baseDir := repoRoot(t)
	command := []string{"go", "run", "-mod=mod", "-tags", "integration", "./testdata/ai/janken/janken-rock-ai"}

	if err := ensureCommandStartable(baseDir, command); err != nil {
		t.Fatalf("ensureCommandStartable() error = %v", err)
	}
}

func TestWorkerProcessNextCompleted(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	submission := testEchoSubmission(t, t.TempDir(),
		"phase2-simultaneous-2turn",
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
	)

	if _, err := commands.Submit(context.Background(), submission); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	record, err := newTestWorker(t, store, 0).ProcessNext(context.Background(), "worker-1")
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if record.State != StateCompleted {
		t.Fatalf("record.State = %q, want %q", record.State, StateCompleted)
	}
	if record.Terminal == nil {
		t.Fatal("record.Terminal = nil, want persisted artifacts")
	}
	if record.Terminal.MatchStatus != contract.StatusCompleted {
		t.Fatalf("record.Terminal.MatchStatus = %q, want completed", record.Terminal.MatchStatus)
	}
	assertTerminalArtifacts(t, record)
	if _, err := os.Stat(filepath.Join(record.Terminal.MatchDir, "structured-log.ndjson")); err != nil {
		t.Fatalf("structured log artifact missing: %v", err)
	}
}

func TestWorkerProcessNextPersistsFailedRunnerRecord(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	submission := testEchoSubmission(t, t.TempDir(),
		"phase2-simultaneous-2turn",
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		repoJoin(t, "testdata/ai/echo/init-timeout-ai"),
	)

	if _, err := commands.Submit(context.Background(), submission); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	record, err := newTestWorker(t, store, 0).ProcessNext(context.Background(), "worker-1")
	if err == nil {
		t.Fatal("ProcessNext() returned nil error")
	}
	if record.State != StateCompleted {
		t.Fatalf("record.State = %q, want %q", record.State, StateCompleted)
	}
	if record.Terminal == nil {
		t.Fatal("record.Terminal = nil, want persisted artifacts")
	}
	if record.Terminal.MatchStatus != contract.StatusFailed {
		t.Fatalf("record.Terminal.MatchStatus = %q, want failed", record.Terminal.MatchStatus)
	}
	assertTerminalArtifacts(t, record)
}

func TestWorkerProcessNextPersistsCanceledRunnerRecord(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	submission := testEchoSubmission(t, t.TempDir(),
		"phase2-simultaneous-2turn",
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		repoJoin(t, "testdata/ai/echo/timeout-ai"),
	)

	if _, err := commands.Submit(context.Background(), submission); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	record, err := newTestWorker(t, store, 10*time.Millisecond).ProcessNext(context.Background(), "worker-1")
	if err == nil {
		t.Fatal("ProcessNext() returned nil error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("ProcessNext() error = %v, want context deadline exceeded", err)
	}
	if record.State != StateCompleted {
		t.Fatalf("record.State = %q, want %q", record.State, StateCompleted)
	}
	if record.Terminal == nil {
		t.Fatal("record.Terminal = nil, want persisted artifacts")
	}
	if record.Terminal.MatchStatus != contract.StatusCanceled {
		t.Fatalf("record.Terminal.MatchStatus = %q, want canceled", record.Terminal.MatchStatus)
	}
	if !strings.Contains(record.Terminal.Error, "context deadline exceeded") {
		t.Fatalf("record.Terminal.Error = %q, want context deadline exceeded", record.Terminal.Error)
	}
	assertTerminalArtifacts(t, record)
}

func TestWorkerProcessNextFailsOnMismatchedMatchID(t *testing.T) {
	store := NewInMemoryQueueStore()
	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	if _, err := store.Enqueue(context.Background(), submission); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	worker, err := NewWorker(
		store,
		stubRunnerInvoker{
			result: ExecutionResult{
				Record: match.Record{MatchID: "other-match"},
			},
		},
		stubTerminalPersister{},
	)
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}

	record, err := worker.ProcessNext(context.Background(), "worker-1")
	if err == nil {
		t.Fatal("ProcessNext() returned nil error")
	}
	if !strings.Contains(err.Error(), "mismatched match_id") {
		t.Fatalf("ProcessNext() error = %v, want mismatched match_id", err)
	}
	if record.State != StateFailed {
		t.Fatalf("record.State = %q, want %q", record.State, StateFailed)
	}
}

func newTestCommandService(t *testing.T) *CommandService {
	return newTestCommandServiceWithStore(t, NewInMemoryQueueStore())
}

func newTestCommandServiceWithStore(t *testing.T, store QueueStore) *CommandService {
	t.Helper()

	dryRun, err := NewLocalDryRunChecker(repoRoot(t))
	if err != nil {
		t.Fatalf("NewLocalDryRunChecker() error = %v", err)
	}
	validator, err := NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		t.Fatalf("NewDefaultAdmissionValidator() error = %v", err)
	}
	commands, err := NewCommandService(store, validator)
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	return commands
}

func newTestWorker(t *testing.T, store QueueStore, timeout time.Duration) *Worker {
	t.Helper()

	invoker, err := NewLocalRunnerInvoker(repoRoot(t), nil, timeout)
	if err != nil {
		t.Fatalf("NewLocalRunnerInvoker() error = %v", err)
	}
	worker, err := NewWorker(store, invoker, LocalTerminalPersister{})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}
	return worker
}

func testSubmission(artifactRef string) MatchSubmission {
	return MatchSubmission{
		SubmissionID: "sub-1",
		MatchID:      "match-1",
		Game: contract.GameMetadata{
			GameID:         "janken",
			GameVersion:    "2.1.0",
			RulesetVersion: "regular",
		},
		Players: []SubmittedPlayer{
			{
				PlayerID:    "p1",
				ArtifactRef: artifactRef,
			},
		},
		OutputDir:    "arena-service-output",
		AttemptCount: 1,
	}
}

func testEchoSubmission(t *testing.T, outputDir, ruleset, player1, player2 string) MatchSubmission {
	t.Helper()

	return MatchSubmission{
		SubmissionID: "sub-echo-1",
		MatchID:      "match-echo-1",
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: ruleset,
		},
		Players: []SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: player1},
			{PlayerID: "p2", ArtifactRef: player2},
		},
		OutputDir:    outputDir,
		AttemptCount: 1,
	}
}

func assertTerminalArtifacts(t *testing.T, record QueueRecord) {
	t.Helper()

	if _, err := os.Stat(record.Terminal.RecordPath); err != nil {
		t.Fatalf("record artifact missing: %v", err)
	}
	if _, err := os.Stat(record.Terminal.ResultSummaryPath); err != nil {
		t.Fatalf("result summary artifact missing: %v", err)
	}
	if len(record.Terminal.PlayerStderrPaths) == 0 {
		t.Fatal("player stderr artifacts missing")
	}
	for playerID, path := range record.Terminal.PlayerStderrPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stderr artifact for %s missing: %v", playerID, err)
		}
	}
}

type stubRunnerInvoker struct {
	result ExecutionResult
	err    error
}

func (s stubRunnerInvoker) Run(context.Context, ExecutionRequest) (ExecutionResult, error) {
	return s.result, s.err
}

type stubTerminalPersister struct{}

func (stubTerminalPersister) Persist(context.Context, MatchSubmission, ExecutionResult) (TerminalArtifacts, error) {
	return TerminalArtifacts{}, nil
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := goRuntime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func repoJoin(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), filepath.Clean(rel))
}
