package service

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryQueueStoreRecoversExpiredInFlightLease(t *testing.T) {
	store := NewInMemoryQueueStore()
	submission := testSubmission(repoJoin(t, "testdata/ai/janken/janken-rock-ai"))
	submission.RunID = "recovery-run"
	submission.MatchID = "recovery-match"
	if _, err := store.Enqueue(context.Background(), submission); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	record, err := store.Claim(context.Background(), "abandoned-worker")
	if err != nil {
		t.Fatalf("Claim() error = %v", err)
	}
	record.State = StateRunning
	record.Lease.Deadline = time.Now().UTC().Add(-time.Second)
	if err := store.Update(context.Background(), record); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	recovered, err := store.RecoverExpired(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("RecoverExpired() error = %v", err)
	}
	if recovered != 1 {
		t.Fatalf("RecoverExpired() = %d, want 1", recovered)
	}
	loaded, err := store.Get(context.Background(), submission.RunID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if loaded.State != StateQueued || loaded.Lease != nil {
		t.Fatalf("recovered record = %+v, want queued record without lease", loaded)
	}
}
