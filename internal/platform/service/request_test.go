package service

import (
	"context"
	"errors"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestMatchRequestServiceCreatesQueuedSubmission(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	general := newTestGeneralSubmissionService(t)
	requests, err := NewMatchRequestService(general, commands, store, nil)
	if err != nil {
		t.Fatalf("NewMatchRequestService() error = %v", err)
	}

	game, err := general.RegisterGame(context.Background(), GameRegistrationRequest{
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
	})
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}
	ai1, err := general.RegisterAI(context.Background(), AISubmissionRequest{
		GameRegistrationID: game.RegistrationID,
		ArtifactRef:        repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		DisplayName:        "Echo 1",
	})
	if err != nil {
		t.Fatalf("RegisterAI(ai1) error = %v", err)
	}
	ai2, err := general.RegisterAI(context.Background(), AISubmissionRequest{
		GameRegistrationID: game.RegistrationID,
		ArtifactRef:        repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
		DisplayName:        "Echo 2",
	})
	if err != nil {
		t.Fatalf("RegisterAI(ai2) error = %v", err)
	}

	item, record, err := requests.Create(context.Background(), MatchRequestCreateRequest{
		GameRegistrationID: game.RegistrationID,
		Participants: []MatchRequestParticipant{
			{PlayerID: "p1", AISubmissionID: ai1.AISubmissionID},
			{PlayerID: "p2", AISubmissionID: ai2.AISubmissionID},
		},
		OutputDir: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.LifecycleState != StateQueued {
		t.Fatalf("item.LifecycleState = %q, want %q", item.LifecycleState, StateQueued)
	}
	if item.ScheduledSubmissionID != record.Submission.SubmissionID {
		t.Fatalf("item.ScheduledSubmissionID = %q, want %q", item.ScheduledSubmissionID, record.Submission.SubmissionID)
	}

	items, err := requests.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Participants[0].AISubmissionID != ai1.AISubmissionID {
		t.Fatalf("items[0].Participants[0].AISubmissionID = %q, want %q", items[0].Participants[0].AISubmissionID, ai1.AISubmissionID)
	}
}

func TestMatchRequestServiceRejectsMismatchedAIRegistration(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	general := newTestGeneralSubmissionService(t)
	requests, err := NewMatchRequestService(general, commands, store, nil)
	if err != nil {
		t.Fatalf("NewMatchRequestService() error = %v", err)
	}

	game1, err := general.RegisterGame(context.Background(), GameRegistrationRequest{
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
	})
	if err != nil {
		t.Fatalf("RegisterGame(game1) error = %v", err)
	}
	game2, err := general.RegisterGame(context.Background(), GameRegistrationRequest{
		Game: contract.GameMetadata{
			GameID:         "janken",
			GameVersion:    "2.1.0",
			RulesetVersion: "regular",
		},
	})
	if err != nil {
		t.Fatalf("RegisterGame(game2) error = %v", err)
	}
	ai, err := general.RegisterAI(context.Background(), AISubmissionRequest{
		GameRegistrationID: game2.RegistrationID,
		ArtifactRef:        repoJoin(t, "testdata/ai/janken/janken-rock-ai"),
	})
	if err != nil {
		t.Fatalf("RegisterAI() error = %v", err)
	}

	if _, _, err := requests.Create(context.Background(), MatchRequestCreateRequest{
		GameRegistrationID: game1.RegistrationID,
		Participants: []MatchRequestParticipant{
			{PlayerID: "p1", AISubmissionID: ai.AISubmissionID},
		},
		OutputDir: t.TempDir(),
	}); err == nil {
		t.Fatal("Create() error = nil, want mismatch error")
	}
}

func TestMatchRequestServiceRollsBackQueuedSubmissionWhenSaveFails(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	general := newTestGeneralSubmissionService(t)
	requests, err := NewMatchRequestService(general, commands, store, failingMatchRequestStore{err: errors.New("boom")})
	if err != nil {
		t.Fatalf("NewMatchRequestService() error = %v", err)
	}

	game, err := general.RegisterGame(context.Background(), GameRegistrationRequest{
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
	})
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}
	ai, err := general.RegisterAI(context.Background(), AISubmissionRequest{
		GameRegistrationID: game.RegistrationID,
		ArtifactRef:        repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
	})
	if err != nil {
		t.Fatalf("RegisterAI() error = %v", err)
	}

	_, _, err = requests.Create(context.Background(), MatchRequestCreateRequest{
		GameRegistrationID: game.RegistrationID,
		Participants: []MatchRequestParticipant{
			{PlayerID: "p1", AISubmissionID: ai.AISubmissionID},
		},
		OutputDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Create() error = nil, want store failure")
	}
	records, listErr := store.List(context.Background())
	if listErr != nil {
		t.Fatalf("store.List() error = %v", listErr)
	}
	if len(records) != 1 {
		t.Fatalf("len(records) = %d, want 1", len(records))
	}
	if records[0].State != StateCanceled {
		t.Fatalf("records[0].State = %q, want %q", records[0].State, StateCanceled)
	}
}

func TestInMemoryMatchRequestStoreClonesParticipants(t *testing.T) {
	store := NewInMemoryMatchRequestStore()
	item := MatchRequest{
		RequestID:             "req-1",
		GameRegistrationID:    "echo-count-v2",
		Game:                  contract.GameMetadata{GameID: "echo-count", GameVersion: "2.0.0", RulesetVersion: "phase2-simultaneous-2turn"},
		Participants:          []MatchRequestParticipant{{PlayerID: "p1", AISubmissionID: "ai-1"}},
		OutputDir:             "out",
		ScheduledSubmissionID: "sub-1",
		ScheduledMatchID:      "match-1",
		LifecycleState:        StateQueued,
	}

	if err := store.Save(context.Background(), item); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	item.Participants[0].PlayerID = "mutated"

	items, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if items[0].Participants[0].PlayerID != "p1" {
		t.Fatalf("items[0].Participants[0].PlayerID = %q, want %q", items[0].Participants[0].PlayerID, "p1")
	}

	items[0].Participants[0].PlayerID = "external-mutation"
	again, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List(second) error = %v", err)
	}
	if again[0].Participants[0].PlayerID != "p1" {
		t.Fatalf("again[0].Participants[0].PlayerID = %q, want %q", again[0].Participants[0].PlayerID, "p1")
	}
}

type failingMatchRequestStore struct {
	err error
}

func (s failingMatchRequestStore) Save(context.Context, MatchRequest) error {
	return s.err
}

func (s failingMatchRequestStore) List(context.Context) ([]MatchRequest, error) {
	return nil, s.err
}
