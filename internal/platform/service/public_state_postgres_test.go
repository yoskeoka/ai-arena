package service

import (
	"context"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
)

func TestPostgresPublicStateStoreSharesMonotonicStateAcrossInstances(t *testing.T) {
	ctx := context.Background()
	dsn := postgresTestDSN(t)
	first, err := NewPostgresPublicStateStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresPublicStateStore(first) error = %v", err)
	}
	defer first.Close()
	if _, err := first.pool.Exec(ctx, "DELETE FROM public_match_states WHERE match_id = $1", "public-state-postgres-test"); err != nil {
		t.Fatalf("clear public state: %v", err)
	}
	t.Cleanup(func() {
		_, _ = first.pool.Exec(context.Background(), "DELETE FROM public_match_states WHERE match_id = $1", "public-state-postgres-test")
	})

	firstState, err := first.Publish(ctx, "public-state-postgres-test", "run-1", game.ExportedSnapshot{Turn: 1, PublicState: []byte(`{"turn":1}`)})
	if err != nil {
		t.Fatalf("Publish(first) error = %v", err)
	}
	if firstState.Version != 1 {
		t.Fatalf("first version = %d, want 1", firstState.Version)
	}
	second, err := NewPostgresPublicStateStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresPublicStateStore(second) error = %v", err)
	}
	defer second.Close()
	secondState, err := second.Publish(ctx, "public-state-postgres-test", "run-1", game.ExportedSnapshot{Turn: 2, PublicState: []byte(`{"turn":2}`)})
	if err != nil {
		t.Fatalf("Publish(second) error = %v", err)
	}
	if secondState.Version != 2 {
		t.Fatalf("second version = %d, want 2", secondState.Version)
	}
	latest, found, err := first.Latest(ctx, "public-state-postgres-test", "run-1")
	if err != nil || !found {
		t.Fatalf("Latest() = (%+v, %v, %v), want found state", latest, found, err)
	}
	if latest.Version != 2 || latest.Snapshot.Turn != 2 {
		t.Fatalf("latest = %+v, want version 2 turn 2", latest)
	}
}
