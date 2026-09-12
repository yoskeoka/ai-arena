package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

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
