package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestOperatorAPIPresetLifecycle(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	queries, err := NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
				{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	createReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/preset-matches", bytes.NewBufferString(`{"preset_id":"echo-reference"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/preset-matches status = %d, body = %s", createResp.Code, createResp.Body.String())
	}
	var created ResultListItem
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(created) error = %v", err)
	}
	if created.LifecycleState != StateQueued {
		t.Fatalf("created.LifecycleState = %q, want %q", created.LifecycleState, StateQueued)
	}

	gameRegistrations, err := general.ListGames(context.Background())
	if err != nil {
		t.Fatalf("general.ListGames() error = %v", err)
	}
	if len(gameRegistrations) != 1 || gameRegistrations[0].RegistrationID != "echo-count-v2" {
		t.Fatalf("game registrations = %+v, want materialized echo-count-v2", gameRegistrations)
	}
	if gameRegistrations[0].Source != SourcePreset || gameRegistrations[0].SourceID != "echo-reference" {
		t.Fatalf("game registration source = %+v, want preset echo-reference", gameRegistrations[0])
	}
	registeredAIs, err := general.ListAIs(context.Background())
	if err != nil {
		t.Fatalf("general.ListAIs() error = %v", err)
	}
	if len(registeredAIs) != 2 {
		t.Fatalf("len(registeredAIs) = %d, want 2", len(registeredAIs))
	}
	if registeredAIs[0].Source != SourcePreset || registeredAIs[0].SourceID != "echo-reference" {
		t.Fatalf("registered AI source = %+v, want preset echo-reference", registeredAIs[0])
	}

	activeResp := httptest.NewRecorder()
	handler.ServeHTTP(activeResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/active", nil))
	if activeResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/matches/active status = %d, body = %s", activeResp.Code, activeResp.Body.String())
	}
	var active struct {
		Items []ResultListItem `json:"items"`
	}
	if err := json.Unmarshal(activeResp.Body.Bytes(), &active); err != nil {
		t.Fatalf("json.Unmarshal(active) error = %v", err)
	}
	if len(active.Items) != 1 {
		t.Fatalf("len(active.Items) = %d, want 1", len(active.Items))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loop, err := NewWorkerLoop(newTestWorker(t, store, 0), "worker-http", 5*time.Millisecond, nil)
	if err != nil {
		t.Fatalf("NewWorkerLoop() error = %v", err)
	}
	done := make(chan error, 1)
	go func() {
		done <- loop.Run(ctx)
	}()

	var completedItem ResultListItem
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		completedResp := httptest.NewRecorder()
		handler.ServeHTTP(completedResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/completed", nil))
		if completedResp.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/matches/completed status = %d, body = %s", completedResp.Code, completedResp.Body.String())
		}
		var completed struct {
			Items []ResultListItem `json:"items"`
		}
		if err := json.Unmarshal(completedResp.Body.Bytes(), &completed); err != nil {
			t.Fatalf("json.Unmarshal(completed) error = %v", err)
		}
		if len(completed.Items) == 1 {
			completedItem = completed.Items[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if completedItem.RunID == "" {
		t.Fatal("completed match did not appear before timeout")
	}
	if completedItem.TerminalStatus == nil || *completedItem.TerminalStatus != contract.StatusCompleted {
		t.Fatalf("completedItem.TerminalStatus = %v, want completed", completedItem.TerminalStatus)
	}

	detailResp := httptest.NewRecorder()
	handler.ServeHTTP(detailResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/runs/"+created.RunID, nil))
	if detailResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/runs/{run_id} status = %d, body = %s", detailResp.Code, detailResp.Body.String())
	}
	var detail MatchDetailResponse
	if err := json.Unmarshal(detailResp.Body.Bytes(), &detail); err != nil {
		t.Fatalf("json.Unmarshal(detail) error = %v", err)
	}
	if detail.ResultSummary == nil {
		t.Fatal("detail.ResultSummary = nil, want compact summary")
	}
	if len(detail.ArtifactAccess) == 0 {
		t.Fatal("detail.ArtifactAccess = empty, want derived metadata")
	}
	if detail.ArtifactAccess["result-summary"].Status != "locator-only" {
		t.Fatalf("result-summary access status = %q, want locator-only", detail.ArtifactAccess["result-summary"].Status)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("WorkerLoop.Run() error = %v", err)
	}
}

func TestOperatorAPIRejectsUnknownPreset(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/preset-matches", bytes.NewBufferString(`{"preset_id":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s, want %d", resp.Code, resp.Body.String(), http.StatusNotFound)
	}
}

func TestOperatorAPIAllowsConfiguredCORSOrigins(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	getReq := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/active", nil)
	getReq.Header.Set("Origin", "https://staging.ai-arena.pages.dev")
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)
	if got := getResp.Header().Get("Access-Control-Allow-Origin"); got != "https://staging.ai-arena.pages.dev" {
		t.Fatalf("GET allow-origin = %q, want staging Pages origin", got)
	}
	if got := getResp.Header().Get("Vary"); got != "Origin" {
		t.Fatalf("GET vary = %q, want Origin", got)
	}

	optionsReq := httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/api/v1/preset-matches", nil)
	optionsReq.Header.Set("Origin", "https://ai-arena.pages.dev")
	optionsResp := httptest.NewRecorder()
	handler.ServeHTTP(optionsResp, optionsReq)
	if optionsResp.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS /api/v1/preset-matches status = %d, want %d", optionsResp.Code, http.StatusNoContent)
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Origin"); got != "https://ai-arena.pages.dev" {
		t.Fatalf("OPTIONS allow-origin = %q, want production Pages origin", got)
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("OPTIONS allow-methods = %q, want %q", got, "GET, POST, OPTIONS")
	}
	if got := optionsResp.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Fatalf("OPTIONS allow-headers = %q, want Content-Type", got)
	}
}

func TestOperatorAPIDoesNotAllowUnknownCORSOrigin(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/active", nil)
	req.Header.Set("Origin", "https://example.com")
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q, want empty", got)
	}
}

func TestOperatorAPIGeneralRegistrationRoutesRejectUnsupportedMethod(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}

	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, "/api/v1/game-registrations", nil)
	resp := httptest.NewRecorder()
	api.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, body = %s, want %d", resp.Code, resp.Body.String(), http.StatusMethodNotAllowed)
	}
}

func TestOperatorAPIGeneralRegistrationRoutes(t *testing.T) {
	commands := newTestCommandService(t)
	queries, err := NewQueryService(NewInMemoryQueueStore())
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, newTestMatchRequestService(t, general, commands, NewInMemoryQueueStore()), presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	gameReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/game-registrations", bytes.NewBufferString(`{"game":{"game_id":"echo-count","game_version":"2.0.0","ruleset_version":"phase2-simultaneous-2turn"}}`))
	gameReq.Header.Set("Content-Type", "application/json")
	gameResp := httptest.NewRecorder()
	handler.ServeHTTP(gameResp, gameReq)
	if gameResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/game-registrations status = %d, body = %s", gameResp.Code, gameResp.Body.String())
	}

	aiReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/ai-submissions", bytes.NewBufferString(fmt.Sprintf(`{"game_registration_id":"echo-count-v2","artifact_ref":%q}`, repoJoin(t, "testdata/ai/echo/echo-ai-2turn"))))
	aiReq.Header.Set("Content-Type", "application/json")
	aiResp := httptest.NewRecorder()
	handler.ServeHTTP(aiResp, aiReq)
	if aiResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/ai-submissions status = %d, body = %s", aiResp.Code, aiResp.Body.String())
	}

	var createdAI RegisteredAI
	if err := json.Unmarshal(aiResp.Body.Bytes(), &createdAI); err != nil {
		t.Fatalf("json.Unmarshal(createdAI) error = %v", err)
	}
	if createdAI.ValidationState != ValidationReady {
		t.Fatalf("createdAI.ValidationState = %q, want %q", createdAI.ValidationState, ValidationReady)
	}

	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/ai-submissions", nil))
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/ai-submissions status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Items []RegisteredAI `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(listed) error = %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("len(listed.Items) = %d, want 1", len(listed.Items))
	}
}

func TestOperatorAPIMatchRequestRoutes(t *testing.T) {
	store := NewInMemoryQueueStore()
	commands := newTestCommandServiceWithStore(t, store)
	queries, err := NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	general := newTestGeneralSubmissionService(t)
	requests := newTestMatchRequestService(t, general, commands, store)
	presets, err := NewStaticPresetCatalog([]MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: t.TempDir(),
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}
	api, err := NewOperatorAPI(commands, queries, general, requests, presets, DirectArtifactAccessIssuer{})
	if err != nil {
		t.Fatalf("NewOperatorAPI() error = %v", err)
	}
	handler := api.Handler()

	gameReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/game-registrations", bytes.NewBufferString(`{"game":{"game_id":"echo-count","game_version":"2.0.0","ruleset_version":"phase2-simultaneous-2turn"}}`))
	gameReq.Header.Set("Content-Type", "application/json")
	gameResp := httptest.NewRecorder()
	handler.ServeHTTP(gameResp, gameReq)
	if gameResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/game-registrations status = %d, body = %s", gameResp.Code, gameResp.Body.String())
	}

	aiPayloads := []string{
		fmt.Sprintf(`{"game_registration_id":"echo-count-v2","artifact_ref":%q,"display_name":"Echo 1"}`, repoJoin(t, "testdata/ai/echo/echo-ai-2turn")),
		fmt.Sprintf(`{"game_registration_id":"echo-count-v2","artifact_ref":%q,"display_name":"Echo 2"}`, repoJoin(t, "testdata/ai/echo/echo-ai-2turn")),
	}
	aiIDs := make([]string, 0, len(aiPayloads))
	for _, payload := range aiPayloads {
		aiReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/ai-submissions", bytes.NewBufferString(payload))
		aiReq.Header.Set("Content-Type", "application/json")
		aiResp := httptest.NewRecorder()
		handler.ServeHTTP(aiResp, aiReq)
		if aiResp.Code != http.StatusCreated {
			t.Fatalf("POST /api/v1/ai-submissions status = %d, body = %s", aiResp.Code, aiResp.Body.String())
		}
		var createdAI RegisteredAI
		if err := json.Unmarshal(aiResp.Body.Bytes(), &createdAI); err != nil {
			t.Fatalf("json.Unmarshal(createdAI) error = %v", err)
		}
		aiIDs = append(aiIDs, createdAI.AISubmissionID)
	}

	matchReq := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/match-requests", bytes.NewBufferString(fmt.Sprintf(`{"game_registration_id":"echo-count-v2","participants":[{"player_id":"p1","ai_submission_id":"%s"},{"player_id":"p2","ai_submission_id":"%s"}],"output_dir":%q}`, aiIDs[0], aiIDs[1], t.TempDir())))
	matchReq.Header.Set("Content-Type", "application/json")
	matchResp := httptest.NewRecorder()
	handler.ServeHTTP(matchResp, matchReq)
	if matchResp.Code != http.StatusCreated {
		t.Fatalf("POST /api/v1/match-requests status = %d, body = %s", matchResp.Code, matchResp.Body.String())
	}
	var created MatchRequest
	if err := json.Unmarshal(matchResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("json.Unmarshal(created) error = %v", err)
	}
	if created.LifecycleState != StateQueued {
		t.Fatalf("created.LifecycleState = %q, want %q", created.LifecycleState, StateQueued)
	}

	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/match-requests", nil))
	if listResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/match-requests status = %d, body = %s", listResp.Code, listResp.Body.String())
	}
	var listed struct {
		Items []MatchRequest `json:"items"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listed); err != nil {
		t.Fatalf("json.Unmarshal(listed) error = %v", err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("len(listed.Items) = %d, want 1", len(listed.Items))
	}
	if listed.Items[0].LatestRunID == "" {
		t.Fatal("listed.Items[0].LatestRunID = empty, want queued run id")
	}
}

func newTestGeneralSubmissionService(t *testing.T) *GeneralSubmissionService {
	t.Helper()

	general, err := NewGeneralSubmissionService(repoRoot(t), nil, nil, nil)
	if err != nil {
		t.Fatalf("NewGeneralSubmissionService() error = %v", err)
	}
	return general
}

func newTestMatchRequestService(t *testing.T, general *GeneralSubmissionService, commands *CommandService, queue QueueStore) *MatchRequestService {
	t.Helper()

	requests, err := NewMatchRequestService(general, commands, queue, nil)
	if err != nil {
		t.Fatalf("NewMatchRequestService() error = %v", err)
	}
	return requests
}

func TestStatusCodeForServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: ErrQueueRecordNotFound, want: http.StatusNotFound},
		{name: "bad request", err: fmt.Errorf("%w: %w", ErrBadRequest, errors.New("service: output_dir is required")), want: http.StatusBadRequest},
		{name: "conflict", err: fmt.Errorf("%w: %w", ErrConflict, errors.New("service: run_id already exists")), want: http.StatusConflict},
		{name: "internal prefixed error stays internal", err: errors.New("service: enqueue submission: connection reset"), want: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusCodeForServiceError(tt.err); got != tt.want {
				t.Fatalf("statusCodeForServiceError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}
