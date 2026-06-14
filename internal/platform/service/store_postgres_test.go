package service

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestPostgresQueueStoreSharesQueueAcrossInstances(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()

	store1 := newTestPostgresQueueStore(t, ctx, dsn, true)
	submission1 := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission1.RunID = "run-pg-1"
	submission1.MatchID = "match-pg-1"
	submission2 := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission2.RunID = "run-pg-2"
	submission2.MatchID = "match-pg-2"

	if _, err := store1.Enqueue(ctx, submission1); err != nil {
		t.Fatalf("Enqueue(submission1) error = %v", err)
	}
	if _, err := store1.Enqueue(ctx, submission2); err != nil {
		t.Fatalf("Enqueue(submission2) error = %v", err)
	}
	store1.Close()

	store2 := newTestPostgresQueueStore(t, ctx, dsn, false)
	record, err := store2.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	if record.Submission.RunID != submission1.RunID {
		t.Fatalf("Claim() run_id = %q, want %q", record.Submission.RunID, submission1.RunID)
	}
	if record.Lease == nil || record.Lease.WorkerID != "worker-1" {
		t.Fatalf("Claim() lease = %+v, want worker-1", record.Lease)
	}
	record.State = StateRunning
	if err := store2.Update(ctx, record); err != nil {
		t.Fatalf("Update(running) error = %v", err)
	}
	record.State = StatePersisting
	if err := store2.Update(ctx, record); err != nil {
		t.Fatalf("Update(persisting) error = %v", err)
	}
	record.State = StateCompleted
	record.Terminal = &TerminalArtifacts{
		MatchDir:          "r2://matches/match-pg-1",
		RecordPath:        "r2://matches/match-pg-1/record.json",
		ResultSummaryPath: "r2://matches/match-pg-1/result-summary.json",
		PlayerStderrPaths: map[string]string{"p1": "r2://matches/match-pg-1/p1-stderr.log"},
		MatchStatus:       contract.StatusCompleted,
	}
	if err := store2.Update(ctx, record); err != nil {
		t.Fatalf("Update(completed) error = %v", err)
	}
	store2.Close()

	store3 := newTestPostgresQueueStore(t, ctx, dsn, false)
	loaded, err := store3.loadRecord(ctx, submission1.RunID)
	if err != nil {
		t.Fatalf("loadRecord() error = %v", err)
	}
	if loaded.State != StateCompleted {
		t.Fatalf("loaded.State = %q, want %q", loaded.State, StateCompleted)
	}
	if loaded.Terminal == nil {
		t.Fatal("loaded.Terminal = nil, want terminal summary")
	}
	if loaded.Terminal.RecordPath != "r2://matches/match-pg-1/record.json" {
		t.Fatalf("loaded.Terminal.RecordPath = %q, want durable record path", loaded.Terminal.RecordPath)
	}
	next, err := store3.Claim(ctx, "worker-2")
	if err != nil {
		t.Fatalf("Claim(second) error = %v", err)
	}
	if next.Submission.RunID != submission2.RunID {
		t.Fatalf("Claim(second) run_id = %q, want %q", next.Submission.RunID, submission2.RunID)
	}
}

func TestPostgresQueueStoreCancelQueued(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()
	store := newTestPostgresQueueStore(t, ctx, dsn, true)

	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission.RunID = "run-pg-cancel"
	submission.MatchID = "match-pg-cancel"

	if _, err := store.Enqueue(ctx, submission); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	record, err := store.CancelQueued(ctx, submission.RunID)
	if err != nil {
		t.Fatalf("CancelQueued() error = %v", err)
	}
	if record.State != StateCanceled {
		t.Fatalf("record.State = %q, want %q", record.State, StateCanceled)
	}
	if _, err := store.Claim(ctx, "worker-1"); err != ErrNoQueuedSubmission {
		t.Fatalf("Claim() error = %v, want %v", err, ErrNoQueuedSubmission)
	}
}

func TestPostgresQueueStoreListAndGet(t *testing.T) {
	dsn := postgresTestDSN(t)
	ctx := context.Background()
	store := newTestPostgresQueueStore(t, ctx, dsn, true)

	submission1 := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission1.RunID = "run-pg-list-1"
	submission1.MatchID = "match-pg-list-1"
	submission2 := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission2.RunID = "run-pg-list-2"
	submission2.MatchID = "match-pg-list-2"

	record1, err := store.Enqueue(ctx, submission1)
	if err != nil {
		t.Fatalf("Enqueue(submission1) error = %v", err)
	}
	record2, err := store.Enqueue(ctx, submission2)
	if err != nil {
		t.Fatalf("Enqueue(submission2) error = %v", err)
	}
	record2, err = store.CancelQueued(ctx, submission2.RunID)
	if err != nil {
		t.Fatalf("CancelQueued(submission2) error = %v", err)
	}

	records, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(records))
	}
	if records[0].Submission.RunID != record1.Submission.RunID {
		t.Fatalf("records[0].run_id = %q, want %q", records[0].Submission.RunID, record1.Submission.RunID)
	}
	if records[1].State != StateCanceled {
		t.Fatalf("records[1].State = %q, want %q", records[1].State, StateCanceled)
	}

	loaded, err := store.Get(ctx, submission1.RunID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.Submission.MatchID != submission1.MatchID {
		t.Fatalf("loaded.MatchID = %q, want %q", loaded.Submission.MatchID, submission1.MatchID)
	}
	if record2.Submission.RunID == "" {
		t.Fatal("canceled record should preserve run id")
	}
}

func TestPostgresAttemptCountRejectsOutOfRange(t *testing.T) {
	t.Parallel()

	if _, err := postgresAttemptCount(2147483648); err == nil {
		t.Fatal("postgresAttemptCount() error = nil, want out-of-range error")
	} else if !strings.Contains(err.Error(), "out of int32 range") {
		t.Fatalf("postgresAttemptCount() error = %v, want out-of-range detail", err)
	}
}

func newTestPostgresQueueStore(t *testing.T, ctx context.Context, dsn string, truncate bool) *PostgresQueueStore {
	t.Helper()

	store, err := NewPostgresQueueStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresQueueStore() error = %v", err)
	}
	if truncate {
		if _, err := store.pool.Exec(ctx, "TRUNCATE service_queue_records RESTART IDENTITY;"); err != nil {
			store.Close()
			t.Fatalf("truncate postgres queue store: %v", err)
		}
	}
	t.Cleanup(store.Close)
	return store
}

func postgresTestDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("AI_ARENA_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("AI_ARENA_PG_TEST_DSN is not set")
	}
	return dsn
}
