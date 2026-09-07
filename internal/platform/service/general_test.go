package service

import (
	"context"
	"testing"

	"github.com/yoskeoka/ai-arena/artifactbundle"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
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
	if game.RegistrationID != "echo-count-v2-phase2-simultaneous-2turn" {
		t.Fatalf("game.RegistrationID = %q, want scope id", game.RegistrationID)
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

func TestGeneralSubmissionServiceRegistersAdmittedArtifactWithoutBuiltInGame(t *testing.T) {
	ctx := context.Background()
	gameBytes := buildWASIBundle(t, "./cmd/janken-gamemaster", `{"schema_version":"arena-bundle/v1","artifact_kind":"game","game_id":"admitted-game","game_version":"1.2.3","rulesets":[{"ruleset_version":"standard","player_count":2,"max_active_bots_per_owner":3}],"runtime":{"kind":"wasm-wasi","module":"module.wasm","memory_limit_pages":1024}}`)
	bundle, err := artifactbundle.Read(gameBytes)
	if err != nil {
		t.Fatal(err)
	}
	bundles := newFilesystemTestBundleStore(t)
	reg, err := registry.NewWASIOverlay(bundles)
	if err != nil {
		t.Fatal(err)
	}
	admission, err := NewArtifactAdmissionService(bundles, reg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admission.RegisterGameBundle(ctx, gameBytes); err != nil {
		t.Fatal(err)
	}
	service, err := NewGeneralSubmissionService(repoRoot(t), reg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	service.WithBundleStore(bundles)

	registered, err := service.RegisterGame(ctx, GameRegistrationRequest{ArtifactID: bundle.Digest, RulesetVersion: "standard"})
	if err != nil {
		t.Fatalf("RegisterGame() error = %v", err)
	}
	if registered.Game.GameID != bundle.Manifest.GameID || registered.Game.GameVersion != bundle.Manifest.GameVersion || registered.ArtifactID != bundle.Digest {
		t.Fatalf("registered game = %+v, want manifest identity and digest", registered)
	}
	if registered.PlayerCount != 2 || registered.MaxActiveBotsPerOwner != 3 {
		t.Fatalf("registered limits = %d/%d, want 2/3", registered.PlayerCount, registered.MaxActiveBotsPerOwner)
	}

	for _, req := range []GameRegistrationRequest{
		{ArtifactID: "sha256:missing", RulesetVersion: "standard"},
		{ArtifactID: bundle.Digest, RulesetVersion: "missing"},
		{ArtifactID: bundle.Digest, Game: contract.GameMetadata{GameID: "different-game", RulesetVersion: "standard"}},
	} {
		if _, err := service.RegisterGame(ctx, req); err == nil {
			t.Fatalf("RegisterGame(%+v) returned nil error", req)
		}
	}
	registeredGames, err := service.ListGames(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(registeredGames) != 1 {
		t.Fatalf("registered game count = %d, want 1", len(registeredGames))
	}
}
