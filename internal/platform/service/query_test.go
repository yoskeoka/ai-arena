package service

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestPostgresQueryServiceListAndGetAcrossLifecycleStates(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()
	store := newTestPostgresQueueStore(t, ctx, dsn, true)
	commands := newTestCommandServiceWithStore(t, store)
	query, err := NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}

	runningSubmission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	runningSubmission.SubmissionID = "sub-running"
	runningSubmission.MatchID = "match-running"
	if _, err := commands.Submit(ctx, runningSubmission); err != nil {
		t.Fatalf("Submit(running) error = %v", err)
	}
	runningRecord, err := store.Claim(ctx, "worker-running")
	if err != nil {
		t.Fatalf("Claim(running) error = %v", err)
	}
	runningRecord.State = StateRunning
	if err := store.Update(ctx, runningRecord); err != nil {
		t.Fatalf("Update(running) error = %v", err)
	}

	failedSubmission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	failedSubmission.SubmissionID = "sub-failed"
	failedSubmission.MatchID = "match-failed"
	if _, err := commands.Submit(ctx, failedSubmission); err != nil {
		t.Fatalf("Submit(failed) error = %v", err)
	}
	failedRecord, err := store.Claim(ctx, "worker-failed")
	if err != nil {
		t.Fatalf("Claim(failed) error = %v", err)
	}
	failedRecord.State = StateFailed
	if err := store.Update(ctx, failedRecord); err != nil {
		t.Fatalf("Update(failed) error = %v", err)
	}

	completedSubmission := testEchoSubmission(
		t,
		t.TempDir(),
		"phase2-simultaneous-2turn",
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
	)
	completedSubmission.SubmissionID = "sub-completed"
	completedSubmission.MatchID = "match-completed"
	if _, err := commands.Submit(ctx, completedSubmission); err != nil {
		t.Fatalf("Submit(completed) error = %v", err)
	}
	if _, err := newTestWorker(t, store, 0).ProcessNext(ctx, "worker-completed"); err != nil {
		t.Fatalf("ProcessNext(completed) error = %v", err)
	}

	queuedSubmission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	queuedSubmission.SubmissionID = "sub-queued"
	queuedSubmission.MatchID = "match-queued"
	if _, err := commands.Submit(ctx, queuedSubmission); err != nil {
		t.Fatalf("Submit(queued) error = %v", err)
	}

	canceledSubmission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	canceledSubmission.SubmissionID = "sub-canceled"
	canceledSubmission.MatchID = "match-canceled"
	if _, err := commands.Submit(ctx, canceledSubmission); err != nil {
		t.Fatalf("Submit(canceled) error = %v", err)
	}
	if _, err := commands.Cancel(ctx, canceledSubmission.SubmissionID); err != nil {
		t.Fatalf("Cancel(canceled) error = %v", err)
	}

	items, err := query.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 5 {
		t.Fatalf("len(items) = %d, want 5", len(items))
	}
	if items[0].LifecycleState != StateRunning {
		t.Fatalf("items[0].LifecycleState = %q, want %q", items[0].LifecycleState, StateRunning)
	}
	if items[1].LifecycleState != StateFailed {
		t.Fatalf("items[1].LifecycleState = %q, want %q", items[1].LifecycleState, StateFailed)
	}
	if items[2].LifecycleState != StateCompleted {
		t.Fatalf("items[2].LifecycleState = %q, want %q", items[2].LifecycleState, StateCompleted)
	}
	if items[2].TerminalStatus == nil || *items[2].TerminalStatus == "" {
		t.Fatal("completed list item should expose terminal_status")
	}
	if items[2].Turn == nil {
		t.Fatal("completed list item should expose compact result summary turn")
	}
	if items[3].LifecycleState != StateQueued {
		t.Fatalf("items[3].LifecycleState = %q, want %q", items[3].LifecycleState, StateQueued)
	}
	if items[4].LifecycleState != StateCanceled {
		t.Fatalf("items[4].LifecycleState = %q, want %q", items[4].LifecycleState, StateCanceled)
	}

	detail, err := query.Get(ctx, completedSubmission.SubmissionID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if detail.ResultSummary == nil {
		t.Fatal("detail.ResultSummary = nil, want decoded compact summary")
	}
	if detail.RecordPath == "" || detail.ResultSummaryPath == "" {
		t.Fatal("detail should expose persisted artifact locators")
	}
	if len(detail.PlayerStderrPaths) != 2 {
		t.Fatalf("len(detail.PlayerStderrPaths) = %d, want 2", len(detail.PlayerStderrPaths))
	}
	if detail.ReplayInputs == nil {
		t.Fatal("detail.ReplayInputs = nil, want replay/resume/audit locators")
	}
	if detail.ReplayInputs.RecordPath == "" || detail.ReplayInputs.SnapshotPath == "" || detail.ReplayInputs.HistoryPath == "" || detail.ReplayInputs.ExportedSnapshotPath == "" {
		t.Fatal("detail.ReplayInputs should expose record/snapshot/history/exported snapshot locators")
	}
	if !detail.ReplayInputs.Verification.Checked {
		t.Fatal("detail.ReplayInputs.Verification.Checked = false, want local verification")
	}
	if !detail.ReplayInputs.Verification.Consistent {
		t.Fatalf("detail.ReplayInputs.Verification.Consistent = false, issues = %v", detail.ReplayInputs.Verification.Issues)
	}
}

func TestQueryServiceGetReportsReplayInputVerificationIssues(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	query, err := NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}

	submission := testEchoSubmission(
		t,
		t.TempDir(),
		"phase2-simultaneous-2turn",
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
	)
	submission.SubmissionID = "sub-mismatch"
	submission.MatchID = "match-mismatch"
	if _, err := commands.Submit(ctx, submission); err != nil {
		t.Fatalf("Submit() error = %v", err)
	}
	if _, err := newTestWorker(t, store, 0).ProcessNext(ctx, "worker-1"); err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}

	detail, err := query.Get(ctx, submission.SubmissionID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if detail.ReplayInputs == nil {
		t.Fatal("detail.ReplayInputs = nil, want replay inputs")
	}
	if err := os.WriteFile(detail.ReplayInputs.SnapshotPath, []byte(`{"match_id":"tampered"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	detail, err = query.Get(ctx, submission.SubmissionID)
	if err != nil {
		t.Fatalf("Get() after tamper error = %v", err)
	}
	if !detail.ReplayInputs.Verification.Checked {
		t.Fatal("verification.Checked = false, want true")
	}
	if detail.ReplayInputs.Verification.Consistent {
		t.Fatal("verification.Consistent = true, want false after tampering snapshot")
	}
	if len(detail.ReplayInputs.Verification.Issues) == 0 {
		t.Fatal("verification.Issues = empty, want mismatch details")
	}
	if !strings.Contains(strings.Join(detail.ReplayInputs.Verification.Issues, "\n"), "snapshot artifact") {
		t.Fatalf("verification issues = %v, want snapshot artifact mismatch", detail.ReplayInputs.Verification.Issues)
	}
}
