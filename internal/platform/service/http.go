package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var allowedOperatorOrigins = map[string]struct{}{
	"https://staging.ai-arena.pages.dev": {},
	"https://ai-arena.pages.dev":         {},
}

// ArtifactAccessMetadata is derived, non-durable access info for one artifact.
type ArtifactAccessMetadata struct {
	Locator     string     `json:"locator"`
	DownloadURL string     `json:"download_url,omitempty"`
	Issuer      string     `json:"issuer,omitempty"`
	Status      string     `json:"status,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// MatchDetailResponse is the HTTP detail response.
type MatchDetailResponse struct {
	MatchDetail
	ArtifactAccess map[string]ArtifactAccessMetadata `json:"artifact_access,omitempty"`
}

// ArtifactAccessIssuer derives per-artifact access metadata from stable locators.
type ArtifactAccessIssuer interface {
	Issue(context.Context, MatchDetail) (map[string]ArtifactAccessMetadata, error)
}

// DirectArtifactAccessIssuer exposes direct URLs when the locator is already remotely fetchable.
type DirectArtifactAccessIssuer struct{}

// Issue derives access metadata for the known detail locators.
func (DirectArtifactAccessIssuer) Issue(_ context.Context, detail MatchDetail) (map[string]ArtifactAccessMetadata, error) {
	artifacts := map[string]string{}
	addArtifactPath(artifacts, "result-summary", detail.ResultSummaryPath)
	addArtifactPath(artifacts, "record", detail.RecordPath)
	if detail.ReplayInputs != nil {
		addArtifactPath(artifacts, "snapshot", detail.ReplayInputs.SnapshotPath)
		addArtifactPath(artifacts, "history", detail.ReplayInputs.HistoryPath)
		addArtifactPath(artifacts, "exported-snapshot", detail.ReplayInputs.ExportedSnapshotPath)
	}
	for playerID, path := range detail.PlayerStderrPaths {
		addArtifactPath(artifacts, "stderr:"+playerID, path)
	}

	if len(artifacts) == 0 {
		return nil, nil
	}

	metadata := make(map[string]ArtifactAccessMetadata, len(artifacts))
	for kind, locator := range artifacts {
		entry := ArtifactAccessMetadata{
			Locator: locator,
			Issuer:  "locator-only",
			Status:  "locator-only",
		}
		if isDirectDownloadURL(locator) {
			entry.DownloadURL = locator
			entry.Issuer = "direct"
			entry.Status = "delegated"
		}
		metadata[kind] = entry
	}
	return metadata, nil
}

// OperatorAPI exposes the remote operator-facing HTTP API.
type OperatorAPI struct {
	commands       *CommandService
	runs           *RunCommandService
	queries        *QueryService
	general        *GeneralSubmissionService
	requests       *MatchRequestService
	presets        PresetCatalog
	artifactAccess ArtifactAccessIssuer
}

// NewOperatorAPI constructs the HTTP adapter for operator routes.
func NewOperatorAPI(commands *CommandService, queries *QueryService, general *GeneralSubmissionService, requests *MatchRequestService, presets PresetCatalog, artifactAccess ArtifactAccessIssuer, rankings ...*RankingService) (*OperatorAPI, error) {
	if commands == nil {
		return nil, fmt.Errorf("service: command service is required")
	}
	if queries == nil {
		return nil, fmt.Errorf("service: query service is required")
	}
	if general == nil {
		return nil, fmt.Errorf("service: general submission service is required")
	}
	if requests == nil {
		return nil, fmt.Errorf("service: match request service is required")
	}
	var rankingService *RankingService
	if len(rankings) > 0 {
		rankingService = rankings[0]
	}
	runs, err := NewRunCommandService(commands, queries.queue, rankingService)
	if err != nil {
		return nil, err
	}
	if presets == nil {
		return nil, fmt.Errorf("service: preset catalog is required")
	}
	if artifactAccess == nil {
		artifactAccess = DirectArtifactAccessIssuer{}
	}
	return &OperatorAPI{
		commands:       commands,
		runs:           runs,
		queries:        queries,
		general:        general,
		requests:       requests,
		presets:        presets,
		artifactAccess: artifactAccess,
	}, nil
}

// Handler builds one HTTP handler tree for the operator API.
func (a *OperatorAPI) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.handleHealthz)
	mux.HandleFunc("/api/v1/game-registrations", a.handleGameRegistrations)
	mux.HandleFunc("/api/v1/ai-submissions", a.handleAISubmissions)
	mux.HandleFunc("/api/v1/match-requests", a.handleMatchRequests)
	mux.HandleFunc("POST /api/v1/preset-matches", a.handlePresetMatches)
	mux.HandleFunc("POST /api/v1/runs/{run_id}/retry", a.handleRunRetry)
	mux.HandleFunc("POST /api/v1/runs/{run_id}/rerun", a.handleRunRerun)
	mux.HandleFunc("POST /api/v1/runs/{run_id}/promote", a.handleRunPromote)
	mux.HandleFunc("GET /api/v1/matches/active", a.handleActiveMatches)
	mux.HandleFunc("GET /api/v1/matches/completed", a.handleCompletedMatches)
	mux.HandleFunc("GET /api/v1/runs/{run_id}", a.handleMatchDetail)
	return withOperatorCORS(mux)
}

func (a *OperatorAPI) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *OperatorAPI) handleGameRegistrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.general.ListGames(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		req, err := decodeJSON[GameRegistrationRequest](r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		record, err := a.general.RegisterGame(r.Context(), req)
		if err != nil {
			writeError(w, statusCodeForServiceError(err), err)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	default:
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
	}
}

func (a *OperatorAPI) handleAISubmissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.general.ListAIs(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		req, err := decodeJSON[AISubmissionRequest](r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		record, err := a.general.RegisterAI(r.Context(), req)
		if err != nil {
			writeError(w, statusCodeForServiceError(err), err)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	default:
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
	}
}

func (a *OperatorAPI) handlePresetMatches(w http.ResponseWriter, r *http.Request) {
	req, err := decodeJSON[PresetMatchRequest](r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	submission, err := a.presets.Build(r.Context(), req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrPresetNotFound) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	_, record, err := a.requests.CreatePreset(r.Context(), req.PresetID, submission)
	if err != nil {
		writeError(w, statusCodeForServiceError(err), err)
		return
	}
	item, _, err := buildResultListItem(r.Context(), record, a.queries.reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *OperatorAPI) handleMatchRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.requests.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		req, err := decodeJSON[MatchRequestCreateRequest](r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		record, _, err := a.requests.Create(r.Context(), req)
		if err != nil {
			writeError(w, statusCodeForServiceError(err), err)
			return
		}
		writeJSON(w, http.StatusCreated, record)
	default:
		writeError(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s is not allowed", r.Method))
	}
}

func (a *OperatorAPI) handleActiveMatches(w http.ResponseWriter, r *http.Request) {
	a.handleMatchList(w, r, map[LifecycleState]struct{}{
		StateQueued:     {},
		StateLeased:     {},
		StateRunning:    {},
		StatePersisting: {},
	})
}

func (a *OperatorAPI) handleRunRetry(w http.ResponseWriter, r *http.Request) {
	record, err := a.runs.Retry(r.Context(), r.PathValue("run_id"))
	if err != nil {
		writeError(w, statusCodeForServiceError(err), err)
		return
	}
	item, _, err := buildResultListItem(r.Context(), record, a.queries.reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *OperatorAPI) handleRunRerun(w http.ResponseWriter, r *http.Request) {
	record, err := a.runs.Rerun(r.Context(), r.PathValue("run_id"))
	if err != nil {
		writeError(w, statusCodeForServiceError(err), err)
		return
	}
	item, _, err := buildResultListItem(r.Context(), record, a.queries.reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *OperatorAPI) handleRunPromote(w http.ResponseWriter, r *http.Request) {
	record, err := a.runs.Promote(r.Context(), r.PathValue("run_id"))
	if err != nil {
		writeError(w, statusCodeForServiceError(err), err)
		return
	}
	item, _, err := buildResultListItem(r.Context(), record, a.queries.reader)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *OperatorAPI) handleCompletedMatches(w http.ResponseWriter, r *http.Request) {
	a.handleMatchList(w, r, map[LifecycleState]struct{}{
		StateCompleted: {},
		StateFailed:    {},
		StateCanceled:  {},
	})
}

func (a *OperatorAPI) handleMatchList(w http.ResponseWriter, r *http.Request, allowed map[LifecycleState]struct{}) {
	items, err := a.queries.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	filtered := make([]ResultListItem, 0, len(items))
	for _, item := range items {
		if _, ok := allowed[item.LifecycleState]; !ok {
			continue
		}
		filtered = append(filtered, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": filtered})
}

func (a *OperatorAPI) handleMatchDetail(w http.ResponseWriter, r *http.Request) {
	runID := strings.TrimSpace(r.PathValue("run_id"))
	if runID == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("service: run_id is required"))
		return
	}
	detail, err := a.queries.Get(r.Context(), runID)
	if err != nil {
		writeError(w, statusCodeForServiceError(err), err)
		return
	}
	artifactAccess, err := a.artifactAccess.Issue(r.Context(), detail)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, MatchDetailResponse{
		MatchDetail:    detail,
		ArtifactAccess: artifactAccess,
	})
}

func decodeJSON[T any](r *http.Request) (T, error) {
	defer r.Body.Close()

	var value T
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&value); err != nil {
		return value, fmt.Errorf("decode request body: %w", err)
	}
	return value, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func statusCodeForServiceError(err error) int {
	switch {
	case errors.Is(err, ErrQueueRecordNotFound), errors.Is(err, ErrPresetNotFound), errors.Is(err, ErrGameRegistrationNotFound), errors.Is(err, ErrAISubmissionNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrConflict):
		return http.StatusConflict
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func withOperatorCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		applyOperatorCORSHeaders(w, r)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func applyOperatorCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return
	}
	if _, ok := allowedOperatorOrigins[origin]; !ok {
		return
	}
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func addArtifactPath(artifacts map[string]string, kind, path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	artifacts[kind] = path
}

func isDirectDownloadURL(locator string) bool {
	parsed, err := url.Parse(locator)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
