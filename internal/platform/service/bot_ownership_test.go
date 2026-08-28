package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestBotOwnershipRevisionDoesNotConsumeAnotherSlot(t *testing.T) {
	store := NewInMemoryBotOwnershipStore()
	scope := CompetitionScope{ScopeID: "reversi-v1-regular", MaxActiveBotsPerOwner: 1}
	bot, first, err := store.CreateOrRevise(context.Background(), BotRevisionRequest{Scope: scope, OwnerAccountID: "a", BotName: " Alpha Bot ", ArtifactID: "a1"})
	if err != nil {
		t.Fatal(err)
	}
	updated, second, err := store.CreateOrRevise(context.Background(), BotRevisionRequest{Scope: scope, OwnerAccountID: "a", BotID: bot.BotID, ArtifactID: "a2"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.BotID != bot.BotID || updated.ActiveRevisionID != second.AISubmissionID || first.AISubmissionID == second.AISubmissionID {
		t.Fatalf("revision did not retain bot identity: %+v", updated)
	}
	_, _, err = store.CreateOrRevise(context.Background(), BotRevisionRequest{Scope: scope, OwnerAccountID: "a", BotName: "second", ArtifactID: "a3"})
	if !errors.Is(err, ErrBotQuotaExceeded) {
		t.Fatalf("quota error = %v", err)
	}
}

func TestBotOwnershipConcurrentCreateEnforcesBoundary(t *testing.T) {
	store := NewInMemoryBotOwnershipStore()
	scope := CompetitionScope{ScopeID: "scope", MaxActiveBotsPerOwner: 3}
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _, err := store.CreateOrRevise(context.Background(), BotRevisionRequest{Scope: scope, OwnerAccountID: "owner", BotName: string(rune('a' + i)), ArtifactID: string(rune('a' + i))})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ErrBotQuotaExceeded) {
			t.Fatalf("create error = %v", err)
		}
	}
	if success != 3 {
		t.Fatalf("successful creates = %d, want 3", success)
	}
}

func TestBotOwnershipQuotaIsPerOwnerAndRetirementPreservesIdentity(t *testing.T) {
	store := NewInMemoryBotOwnershipStore()
	scope := CompetitionScope{ScopeID: "reversi-v1-regular", MaxActiveBotsPerOwner: 1}
	bot, _, err := store.CreateOrRevise(context.Background(), BotRevisionRequest{Scope: scope, OwnerAccountID: "a", BotName: "Alpha", ArtifactID: "a1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateOrRevise(context.Background(), BotRevisionRequest{Scope: scope, OwnerAccountID: "b", BotName: " alpha ", ArtifactID: "b1"}); err != nil {
		t.Fatal(err)
	}
	retired, err := store.Retire(context.Background(), "a", bot.BotID)
	if err != nil || retired.LifecycleState != BotRetired {
		t.Fatalf("Retire() = %+v, %v", retired, err)
	}
	active, _ := store.ListByOwner(context.Background(), "a", scope.ScopeID, false)
	all, _ := store.ListByOwner(context.Background(), "a", scope.ScopeID, true)
	if len(active) != 0 || len(all) != 1 || all[0].BotID != bot.BotID {
		t.Fatalf("lists = active:%+v all:%+v", active, all)
	}
}
