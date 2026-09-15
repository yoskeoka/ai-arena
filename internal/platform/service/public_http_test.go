package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

func TestPublicMatchUsesPinnedParticipantOrderAndCompletionTime(t *testing.T) {
	t.Parallel()
	completedAt := time.Date(2026, time.September, 15, 1, 2, 3, 0, time.UTC)
	record := QueueRecord{Submission: MatchSubmission{MatchID: "match-1", RunID: "run-1", Players: []SubmittedPlayer{
		{PlayerID: "player-z", BotName: "second admitted", AISubmissionID: "revision-2"},
		{PlayerID: "player-a", BotName: "first admitted", AISubmissionID: "revision-1"},
	}}, State: StateCompleted, CompletedAt: &completedAt}
	match := publicMatchFromRecord(record)
	if match.CompletedAt == nil || !match.CompletedAt.Equal(completedAt) {
		t.Fatalf("CompletedAt = %v, want %v", match.CompletedAt, completedAt)
	}
	if len(match.Participants) != 2 || match.Participants[0].PlayerID != "player-z" || match.Participants[1].PlayerID != "player-a" {
		t.Fatalf("Participants = %#v, want submitted order", match.Participants)
	}
	body, err := json.Marshal(match)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, forbidden := range []string{"bot_id", "artifact_ref", "artifact_id", "output_dir"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("public payload leaked %q: %s", forbidden, body)
		}
	}

	record.Submission.Players[1].BotName = ""
	if got := publicMatchFromRecord(record).Participants; got != nil {
		t.Fatalf("Participants = %#v, want omitted incomplete legacy provenance", got)
	}
}

func TestPublicMatchListFiltersSortsAndPaginatesStably(t *testing.T) {
	t.Parallel()
	completedEarly := time.Date(2026, time.September, 14, 1, 0, 0, 0, time.UTC)
	completedLate := time.Date(2026, time.September, 15, 1, 0, 0, 0, time.UTC)
	queue := &publicListQueueStore{records: []QueueRecord{
		publicListRecord("match-z", "reversi", "1.1.0", "standard", &completedLate),
		publicListRecord("match-a", "reversi", "1.2.0", "xot", &completedLate),
		publicListRecord("match-b", "reversi", "1.0.0", "standard", &completedEarly),
		publicListRecord("match-null", "reversi", "1.0.0", "standard", nil),
		publicListRecord("match-other-major", "reversi", "2.0.0", "standard", &completedLate),
		publicListRecord("match-other-game", "janken", "1.0.0", "classic", &completedLate),
	}}
	queries, err := NewPublicQueryService(queue, NewInMemoryPublicStateStore(), NewDefaultArtifactReader(nil))
	if err != nil {
		t.Fatalf("NewPublicQueryService() error = %v", err)
	}

	page, err := queries.List(context.Background(), PublicMatchListOptions{GameID: "reversi", GameVersionMajor: 1, Page: 1, Limit: 2, SortOrder: "desc"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := page.Pagination, (PublicMatchPagination{Page: 1, Limit: 2, Total: 4, TotalPages: 2}); got != want {
		t.Fatalf("Pagination = %#v, want %#v", got, want)
	}
	if got, want := page.AvailableRulesetVersions, []string{"standard", "xot"}; !slices.Equal(got, want) {
		t.Fatalf("AvailableRulesetVersions = %#v, want %#v", got, want)
	}
	if got, want := publicMatchIDs(page.Items), []string{"match-a", "match-z"}; !slices.Equal(got, want) {
		t.Fatalf("page 1 IDs = %#v, want %#v", got, want)
	}

	page, err = queries.List(context.Background(), PublicMatchListOptions{GameID: "reversi", GameVersionMajor: 1, RulesetVersion: "standard", Page: 1, Limit: 20, SortOrder: "asc"})
	if err != nil {
		t.Fatalf("List(filtered) error = %v", err)
	}
	if got, want := publicMatchIDs(page.Items), []string{"match-b", "match-z", "match-null"}; !slices.Equal(got, want) {
		t.Fatalf("filtered IDs = %#v, want %#v", got, want)
	}
	if got, want := page.AvailableRulesetVersions, []string{"standard", "xot"}; !slices.Equal(got, want) {
		t.Fatalf("ruleset metadata = %#v, want scope-wide %#v", got, want)
	}
}

func TestPublicAPIListRejectsInvalidQueryAndPreservesDefaults(t *testing.T) {
	t.Parallel()
	queue := &publicListQueueStore{records: []QueueRecord{publicListRecord("match-1", "reversi", "1.0.0", "standard", nil)}}
	queries, err := NewPublicQueryService(queue, NewInMemoryPublicStateStore(), NewDefaultArtifactReader(nil))
	if err != nil {
		t.Fatalf("NewPublicQueryService() error = %v", err)
	}
	api, err := NewPublicAPI(queries)
	if err != nil {
		t.Fatalf("NewPublicAPI() error = %v", err)
	}
	for _, query := range []string{"?page=0", "?limit=101", "?game_version_major=zero", "?game_version_major=2147483648", "?sort=match_id", "?sort_order=sideways", "?unknown=value", "?page=1&page=2"} {
		response := httptest.NewRecorder()
		api.Handler().ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1-alpha/public/matches"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("GET %s status = %d, want %d", query, response.Code, http.StatusBadRequest)
		}
	}
	response := httptest.NewRecorder()
	api.Handler().ServeHTTP(response, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1-alpha/public/matches", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("default list status = %d, body = %s", response.Code, response.Body.String())
	}
	var body PublicMatchList
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got, want := body.Pagination, (PublicMatchPagination{Page: 1, Limit: 20, Total: 1, TotalPages: 1}); got != want {
		t.Fatalf("defaults pagination = %#v, want %#v", got, want)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

type publicListQueueStore struct {
	*InMemoryQueueStore
	records []QueueRecord
}

func (s *publicListQueueStore) List(context.Context) ([]QueueRecord, error) {
	return append([]QueueRecord(nil), s.records...), nil
}

func publicListRecord(matchID, gameID, gameVersion, rulesetVersion string, completedAt *time.Time) QueueRecord {
	return QueueRecord{Submission: MatchSubmission{MatchID: matchID, RunID: "run-" + matchID, Game: contract.GameMetadata{GameID: gameID, GameVersion: gameVersion, RulesetVersion: rulesetVersion}}, State: StateCompleted, CompletedAt: completedAt}
}

func publicMatchIDs(items []PublicMatch) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.MatchID)
	}
	return ids
}

func TestPublicAPIStateIsAnonymousVersionedAndExportedOnly(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	queue := NewInMemoryQueueStore()
	record, err := queue.Enqueue(ctx, publicTestSubmission("run-1", "match-1"))
	if err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	record.State = StateLeased
	if err := queue.Update(ctx, record); err != nil {
		t.Fatalf("Update() leased error = %v", err)
	}
	record.State = StateRunning
	if err := queue.Update(ctx, record); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	states := NewInMemoryPublicStateStore()
	_, err = states.Publish(ctx, "match-1", "run-1", game.ExportedSnapshot{Turn: 3, PublicState: []byte(`{"board":["black"]}`)})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	queries, err := NewPublicQueryService(queue, states, NewDefaultArtifactReader(nil))
	if err != nil {
		t.Fatalf("NewPublicQueryService() error = %v", err)
	}
	api, err := NewPublicAPI(queries)
	if err != nil {
		t.Fatalf("NewPublicAPI() error = %v", err)
	}

	response := httptest.NewRecorder()
	request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1-alpha/public/matches/match-1/state", nil)
	request.Header.Set("Origin", "https://viewer.example")
	api.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	body := response.Body.String()
	for _, want := range []string{`"selected_run_id": "run-1"`, `"state_version": 1`, `"board": [`} {
		if !strings.Contains(body, want) {
			t.Fatalf("body %q does not contain %q", body, want)
		}
	}
	for _, forbidden := range []string{"record_path", "artifact_access", "output_dir", "credential"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body leaked %q: %s", forbidden, body)
		}
	}
}

func TestPublicAPISelectedRunPreservesParticipantSequenceAcrossViews(t *testing.T) {
	ctx := context.Background()
	queue := NewInMemoryQueueStore()
	states := NewInMemoryPublicStateStore()
	first := publicTestSubmission("run-first", "match-participants")
	first.Players = []SubmittedPlayer{
		{PlayerID: "player-second", BotID: "private-bot-second", BotName: "Second submitted", AISubmissionID: "revision-second", ArtifactRef: "private-artifact-second"},
		{PlayerID: "player-first", BotID: "private-bot-first", BotName: "First submitted", AISubmissionID: "revision-first", ArtifactRef: "private-artifact-first"},
	}
	second := publicTestSubmission("run-promoted", "match-participants")
	second.Players = []SubmittedPlayer{
		{PlayerID: "player-new-first", BotID: "private-bot-new-first", BotName: "New first submitted", AISubmissionID: "revision-new-first", ArtifactRef: "private-artifact-new-first"},
		{PlayerID: "player-new-second", BotID: "private-bot-new-second", BotName: "New second submitted", AISubmissionID: "revision-new-second", ArtifactRef: "private-artifact-new-second"},
	}
	completePublicTestRecord(t, ctx, queue, first)
	completePublicTestRecord(t, ctx, queue, second)
	if _, err := queue.Promote(ctx, first.RunID); err != nil {
		t.Fatalf("Promote(first) error = %v", err)
	}
	if _, err := states.Publish(ctx, first.MatchID, first.RunID, game.ExportedSnapshot{Turn: 4, PublicState: []byte(`{"board":["first-run"]}`)}); err != nil {
		t.Fatalf("Publish(first) error = %v", err)
	}
	queries, err := NewPublicQueryService(queue, states, NewDefaultArtifactReader(nil))
	if err != nil {
		t.Fatalf("NewPublicQueryService() error = %v", err)
	}
	api, err := NewPublicAPI(queries)
	if err != nil {
		t.Fatalf("NewPublicAPI() error = %v", err)
	}

	assertSelectedPublicRun(t, ctx, api, "run-first", first.Players)
	if _, err := queue.Promote(ctx, second.RunID); err != nil {
		t.Fatalf("Promote(second) error = %v", err)
	}
	if _, err := states.Publish(ctx, second.MatchID, second.RunID, game.ExportedSnapshot{Turn: 5, PublicState: []byte(`{"board":["promoted-run"]}`)}); err != nil {
		t.Fatalf("Publish(second) error = %v", err)
	}
	assertSelectedPublicRun(t, ctx, api, "run-promoted", second.Players)
}

func completePublicTestRecord(t *testing.T, ctx context.Context, queue *InMemoryQueueStore, submission MatchSubmission) {
	t.Helper()
	record, err := queue.Enqueue(ctx, submission)
	if err != nil {
		t.Fatalf("Enqueue(%q) error = %v", submission.RunID, err)
	}
	for _, state := range []LifecycleState{StateLeased, StateRunning, StatePersisting, StateCompleted} {
		record.State = state
		if err := queue.Update(ctx, record); err != nil {
			t.Fatalf("Update(%q, %s) error = %v", submission.RunID, state, err)
		}
	}
}

func assertSelectedPublicRun(t *testing.T, ctx context.Context, api *PublicAPI, wantRunID string, wantPlayers []SubmittedPlayer) {
	t.Helper()
	paths := []string{
		"/api/v1-alpha/public/matches",
		"/api/v1-alpha/public/matches/match-participants",
		"/api/v1-alpha/public/matches/match-participants/state",
	}
	var wantCompletedAt string
	for _, path := range paths {
		response := httptest.NewRecorder()
		request := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		request.Header.Set("Origin", "https://viewer.example")
		api.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, body = %s", path, response.Code, response.Body.String())
		}
		if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Fatalf("GET %s Access-Control-Allow-Origin = %q", path, got)
		}
		body := response.Body.String()
		for _, forbidden := range []string{"bot_id", "artifact_ref", "artifact_id", "output_dir", "record_path", "credential"} {
			if strings.Contains(body, forbidden) {
				t.Fatalf("GET %s leaked %q: %s", path, forbidden, body)
			}
		}
		var payload map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatalf("GET %s decode error = %v", path, err)
		}
		if path == "/api/v1-alpha/public/matches" {
			items, ok := payload["items"].([]any)
			if !ok || len(items) != 1 {
				t.Fatalf("GET %s items = %#v", path, payload["items"])
			}
			payload, ok = items[0].(map[string]any)
			if !ok {
				t.Fatalf("GET %s item = %#v", path, items[0])
			}
		}
		if got := payload["selected_run_id"]; got != wantRunID {
			t.Fatalf("GET %s selected_run_id = %v, want %q", path, got, wantRunID)
		}
		completedAt, ok := payload["completed_at"].(string)
		if !ok || completedAt == "" {
			t.Fatalf("GET %s completed_at = %#v", path, payload["completed_at"])
		}
		if wantCompletedAt == "" {
			wantCompletedAt = completedAt
		} else if completedAt != wantCompletedAt {
			t.Fatalf("GET %s completed_at = %q, want immutable %q", path, completedAt, wantCompletedAt)
		}
		participants, ok := payload["participants"].([]any)
		if !ok || len(participants) != len(wantPlayers) {
			t.Fatalf("GET %s participants = %#v", path, payload["participants"])
		}
		for i, want := range wantPlayers {
			participant, ok := participants[i].(map[string]any)
			if !ok || participant["player_id"] != want.PlayerID || participant["display_name"] != want.BotName || participant["ai_submission_id"] != want.AISubmissionID {
				t.Fatalf("GET %s participant[%d] = %#v, want %#v", path, i, participants[i], want)
			}
		}
	}
}

func TestPublicAPIRejectsNonGETAndDoesNotExposeQueuedMatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	queue := NewInMemoryQueueStore()
	if _, err := queue.Enqueue(ctx, publicTestSubmission("run-queued", "match-queued")); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	queries, err := NewPublicQueryService(queue, NewInMemoryPublicStateStore(), NewDefaultArtifactReader(nil))
	if err != nil {
		t.Fatalf("NewPublicQueryService() error = %v", err)
	}
	api, err := NewPublicAPI(queries)
	if err != nil {
		t.Fatalf("NewPublicAPI() error = %v", err)
	}

	post := httptest.NewRecorder()
	api.Handler().ServeHTTP(post, httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1-alpha/public/matches", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d", post.Code)
	}
	notFound := httptest.NewRecorder()
	api.Handler().ServeHTTP(notFound, httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1-alpha/public/matches/match-queued", nil))
	if notFound.Code != http.StatusNotFound {
		t.Fatalf("queued status = %d", notFound.Code)
	}
}

func publicTestSubmission(runID, matchID string) MatchSubmission {
	return MatchSubmission{RunID: runID, MatchID: matchID, Game: contract.GameMetadata{GameID: "reversi", GameVersion: "1", RulesetVersion: "1"}, Players: []SubmittedPlayer{{PlayerID: "one", ArtifactRef: "test"}}, OutputDir: "test-output", AttemptCount: 1, RunKind: RunKindInitial}
}

func TestLocalTerminalPersisterStoresBoundedGameProducedPublicReplay(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"moves":["d3"]}`)
	submission := publicTestSubmission("run-replay", "match-replay")
	submission.OutputDir = t.TempDir()
	terminal, err := (LocalTerminalPersister{}).Persist(context.Background(), submission, ExecutionResult{Record: match.Record{
		MatchID:      "match-replay",
		Game:         contract.GameMetadata{GameID: "reversi", GameVersion: "1", RulesetVersion: "1"},
		PublicReplay: &game.PublicReplay{Format: "reversi-kifu", Version: "1", Payload: payload},
	}})
	if err != nil {
		t.Fatalf("Persist() error = %v", err)
	}
	if terminal.PublicReplayPath == "" || terminal.PublicReplayFormat != "reversi-kifu" || terminal.PublicReplayVersion != "1" || terminal.PublicReplaySize != int64(len(payload)) {
		t.Fatalf("public replay metadata = %#v", terminal)
	}
	got, err := os.ReadFile(terminal.PublicReplayPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}
