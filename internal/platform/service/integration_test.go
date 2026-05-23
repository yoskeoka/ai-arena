package service

import (
	"context"
	"errors"
	"path/filepath"
	goRuntime "runtime"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
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

func newTestCommandService(t *testing.T) *CommandService {
	t.Helper()

	dryRun, err := NewLocalDryRunChecker(repoRoot(t))
	if err != nil {
		t.Fatalf("NewLocalDryRunChecker() error = %v", err)
	}
	validator, err := NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		t.Fatalf("NewDefaultAdmissionValidator() error = %v", err)
	}
	commands, err := NewCommandService(NewInMemoryQueueStore(), validator)
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	return commands
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
