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
	api, err := NewOperatorAPI(commands, queries, presets, DirectArtifactAccessIssuer{})
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
	if completedItem.SubmissionID == "" {
		t.Fatal("completed match did not appear before timeout")
	}
	if completedItem.TerminalStatus == nil || *completedItem.TerminalStatus != contract.StatusCompleted {
		t.Fatalf("completedItem.TerminalStatus = %v, want completed", completedItem.TerminalStatus)
	}

	detailResp := httptest.NewRecorder()
	handler.ServeHTTP(detailResp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/matches/"+created.SubmissionID, nil))
	if detailResp.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/matches/{submission_id} status = %d, body = %s", detailResp.Code, detailResp.Body.String())
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
	api, err := NewOperatorAPI(commands, queries, presets, DirectArtifactAccessIssuer{})
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
	api, err := NewOperatorAPI(commands, queries, presets, DirectArtifactAccessIssuer{})
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
	api, err := NewOperatorAPI(commands, queries, presets, DirectArtifactAccessIssuer{})
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

func TestStatusCodeForServiceError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "not found", err: ErrQueueRecordNotFound, want: http.StatusNotFound},
		{name: "bad request", err: fmt.Errorf("%w: %w", ErrBadRequest, errors.New("service: output_dir is required")), want: http.StatusBadRequest},
		{name: "conflict", err: fmt.Errorf("%w: %w", ErrConflict, errors.New("service: submission_id already exists")), want: http.StatusConflict},
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
