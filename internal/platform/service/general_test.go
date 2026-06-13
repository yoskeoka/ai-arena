package service

import (
	"context"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestGeneralSubmissionServiceRegistersGameAndAI(t *testing.T) {
	service, err := NewGeneralSubmissionService(repoRoot(t), nil, nil, nil)
	if err != nil {
		t.Fatalf("NewGeneralSubmissionService() error = %v", err)
	}

	game, err := service.RegisterGame(context.Background(), GameRegistrationRequest{
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
	})
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}
	if game.RegistrationID != "echo-count-v2" {
		t.Fatalf("game.RegistrationID = %q, want %q", game.RegistrationID, "echo-count-v2")
	}

	ai, err := service.RegisterAI(context.Background(), AISubmissionRequest{
		GameRegistrationID: game.RegistrationID,
		ArtifactRef:        repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
	})
	if err != nil {
		t.Fatalf("RegisterAI() error = %v", err)
	}
	if ai.ValidationState != ValidationReady {
		t.Fatalf("ai.ValidationState = %q, want %q", ai.ValidationState, ValidationReady)
	}
	if ai.AIID == "" {
		t.Fatal("ai.AIID = empty, want loaded AI id")
	}
}

func TestGeneralSubmissionServiceRejectsUnknownGameRegistration(t *testing.T) {
	service, err := NewGeneralSubmissionService(repoRoot(t), nil, nil, nil)
	if err != nil {
		t.Fatalf("NewGeneralSubmissionService() error = %v", err)
	}

	_, err = service.RegisterAI(context.Background(), AISubmissionRequest{
		GameRegistrationID: "missing",
		ArtifactRef:        repoJoin(t, "testdata/ai/echo/echo-ai-2turn"),
	})
	if err == nil {
		t.Fatal("RegisterAI() returned nil error")
	}
}
