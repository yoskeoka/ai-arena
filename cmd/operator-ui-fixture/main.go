// Command operator-ui-fixture starts a deterministic local backend for operator UI browser verification.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

func main() {
	listenAddr := flag.String("listen-addr", "127.0.0.1:10000", "HTTP listen address")
	flag.Parse()

	backend, err := newFixtureBackend(*listenAddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("operator-ui fixture listening on http://%s", *listenAddr)
	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           backend.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

type fixtureBackend struct {
	api         http.Handler
	downloadURL string
	resultBytes []byte
}

func newFixtureBackend(listenAddr string) (*fixtureBackend, error) {
	outputDir, err := os.MkdirTemp("", "ai-arena-operator-ui-fixture-")
	if err != nil {
		return nil, fmt.Errorf("create fixture output dir: %w", err)
	}

	store := service.NewInMemoryQueueStore()
	commands, err := service.NewCommandService(store, staticValidator{})
	if err != nil {
		return nil, err
	}
	queries, err := service.NewQueryService(store)
	if err != nil {
		return nil, err
	}

	presets, err := service.NewStaticPresetCatalog([]service.MatchPresetDefinition{
		{
			PresetID:  "echo-reference",
			Game:      fixtureGame(),
			Players:   fixturePlayers(),
			OutputDir: filepath.Join(outputDir, "preset-queue"),
		},
	})
	if err != nil {
		return nil, err
	}

	summaryPath, summaryBytes, err := writeResultSummary(outputDir)
	if err != nil {
		return nil, err
	}
	if err := seedActiveRecord(store, filepath.Join(outputDir, "active")); err != nil {
		return nil, err
	}
	if err := seedCompletedRecord(store, summaryPath); err != nil {
		return nil, err
	}

	downloadURL := fmt.Sprintf("http://%s/fixture-artifacts/result-summary.json", listenAddr)
	api, err := service.NewOperatorAPI(commands, queries, presets, fixtureArtifactAccessIssuer{
		downloadURL: downloadURL,
	})
	if err != nil {
		return nil, err
	}
	return &fixtureBackend{
		api:         api.Handler(),
		downloadURL: downloadURL,
		resultBytes: summaryBytes,
	}, nil
}

func (b *fixtureBackend) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", b.api)
	mux.Handle("/api/", b.api)
	mux.HandleFunc("/fixture-artifacts/result-summary.json", b.handleResultSummaryDownload)
	return mux
}

func (b *fixtureBackend) handleResultSummaryDownload(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b.resultBytes)
}

type staticValidator struct{}

func (staticValidator) Validate(_ context.Context, submission service.MatchSubmission) error {
	return service.ValidateSubmission(submission)
}

type fixtureArtifactAccessIssuer struct {
	downloadURL string
}

func (i fixtureArtifactAccessIssuer) Issue(ctx context.Context, detail service.MatchDetail) (map[string]service.ArtifactAccessMetadata, error) {
	metadata, err := service.DirectArtifactAccessIssuer{}.Issue(ctx, detail)
	if err != nil {
		return nil, err
	}
	entry, ok := metadata["result-summary"]
	if !ok {
		return metadata, nil
	}
	entry.DownloadURL = i.downloadURL
	entry.Issuer = "fixture-direct"
	entry.Status = "delegated"
	expiresAt := time.Date(2026, time.June, 3, 18, 30, 0, 0, time.UTC)
	entry.ExpiresAt = &expiresAt
	metadata["result-summary"] = entry
	return metadata, nil
}

func seedActiveRecord(store *service.InMemoryQueueStore, outputDir string) error {
	_, err := store.Enqueue(context.Background(), service.MatchSubmission{
		SubmissionID: "sub-active-queued",
		MatchID:      "match-active-queued",
		Game:         fixtureGame(),
		Players:      fixturePlayers(),
		OutputDir:    outputDir,
		AttemptCount: 1,
	})
	return err
}

func seedCompletedRecord(store *service.InMemoryQueueStore, summaryPath string) error {
	ctx := context.Background()
	submission := service.MatchSubmission{
		SubmissionID: "sub-completed-local",
		MatchID:      "match-completed-local",
		Game:         fixtureGame(),
		Players:      fixturePlayers(),
		OutputDir:    "fixture-output/operator-ui",
		AttemptCount: 1,
	}
	if _, err := store.Enqueue(ctx, submission); err != nil {
		return err
	}
	record, err := store.Claim(ctx, "fixture-worker")
	if err != nil {
		return err
	}
	record.State = service.StateRunning
	if err := store.Update(ctx, record); err != nil {
		return err
	}
	record.State = service.StatePersisting
	if err := store.Update(ctx, record); err != nil {
		return err
	}
	record.State = service.StateCompleted
	record.Terminal = &service.TerminalArtifacts{
		MatchDir:          "https://fixture.local/artifacts/match-completed-local",
		RecordPath:        "https://fixture.local/artifacts/match-completed-local/record.json",
		ResultSummaryPath: summaryPath,
		PlayerStderrPaths: map[string]string{
			"p1": "https://fixture.local/artifacts/match-completed-local/stderr-p1.log",
			"p2": "https://fixture.local/artifacts/match-completed-local/stderr-p2.log",
		},
		MatchStatus: game.StatusCompleted,
	}
	return store.Update(ctx, record)
}

func writeResultSummary(outputDir string) (string, []byte, error) {
	summary := artifacts.ResultSummary{
		MatchID:        "match-completed-local",
		GameID:         fixtureGame().GameID,
		GameVersion:    fixtureGame().GameVersion,
		RulesetVersion: fixtureGame().RulesetVersion,
		Status:         game.StatusCompleted,
		Turn:           3,
		Placements: []game.Placement{
			{PlayerID: "p1", Place: 1},
			{PlayerID: "p2", Place: 2},
		},
		ArtifactPaths: artifacts.PathRefs{
			Record:           "record.json",
			Snapshot:         "snapshot.json",
			History:          "history.json",
			ExportedSnapshot: "exported-snapshot.json",
		},
	}
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("marshal result summary: %w", err)
	}
	path := filepath.Join(outputDir, "result-summary.json")
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", nil, fmt.Errorf("write result summary: %w", err)
	}
	return path, append(data, '\n'), nil
}

func fixtureGame() contract.GameMetadata {
	return contract.GameMetadata{
		GameID:         "echo-count",
		GameVersion:    "2.0.0",
		RulesetVersion: "phase2-simultaneous-3turn",
	}
}

func fixturePlayers() []service.SubmittedPlayer {
	return []service.SubmittedPlayer{
		{PlayerID: "p1", ArtifactRef: "file:///fixture/ai/echo/p1"},
		{PlayerID: "p2", ArtifactRef: "file:///fixture/ai/echo/p2"},
	}
}
