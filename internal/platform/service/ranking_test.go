package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

func TestRankingServiceApplyCompletedAndGet(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store, err := NewLocalRankingSnapshotStore(baseDir)
	if err != nil {
		t.Fatalf("NewLocalRankingSnapshotStore() error = %v", err)
	}
	rankings, err := NewRankingService(store, nil)
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}

	submission := rankingTestSubmission("sub-1", "match-1", []SubmittedPlayer{
		{PlayerID: "alpha", ArtifactRef: "artifact://alpha"},
		{PlayerID: "beta", ArtifactRef: "artifact://beta"},
	})
	summary := rankingTestSummary("match-1", []game.Placement{
		{PlayerID: "alpha", Place: 1},
		{PlayerID: "beta", Place: 2},
	})

	if err := rankings.ApplyCompleted(context.Background(), submission, summary); err != nil {
		t.Fatalf("ApplyCompleted() error = %v", err)
	}
	if err := rankings.ApplyCompleted(context.Background(), submission, summary); err != nil {
		t.Fatalf("ApplyCompleted() duplicate error = %v", err)
	}

	stored, err := rankings.Get(context.Background(), RankingScope{
		GameID:         "echo",
		GameVersion:    "v1",
		RulesetVersion: "default",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Snapshot.CompletedMatches != 1 {
		t.Fatalf("CompletedMatches = %d, want 1", stored.Snapshot.CompletedMatches)
	}
	if len(stored.Snapshot.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(stored.Snapshot.Entries))
	}
	if stored.Snapshot.Entries[0].CompetitorRef != "artifact://alpha" || stored.Snapshot.Entries[0].FirstPlaces != 1 {
		t.Fatalf("alpha entry = %+v, want first place counted", stored.Snapshot.Entries[0])
	}
	if stored.Snapshot.Entries[1].PlacementCounts[2] != 1 {
		t.Fatalf("beta placement_counts = %+v, want place 2 counted once", stored.Snapshot.Entries[1].PlacementCounts)
	}
}

func TestRankingServiceRecomputeAndVerify(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store, err := NewLocalRankingSnapshotStore(baseDir)
	if err != nil {
		t.Fatalf("NewLocalRankingSnapshotStore() error = %v", err)
	}
	queue := NewInMemoryQueueStore()
	rankings, err := NewRankingService(store, queue)
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}

	firstSubmission := rankingTestSubmission("sub-1", "match-1", []SubmittedPlayer{
		{PlayerID: "alpha", ArtifactRef: "artifact://alpha"},
		{PlayerID: "beta", ArtifactRef: "artifact://beta"},
	})
	firstSummaryPath := writeRankingSummaryFile(t, baseDir, "match-1", rankingTestSummary("match-1", []game.Placement{
		{PlayerID: "alpha", Place: 1},
		{PlayerID: "beta", Place: 2},
	}))
	seedCompletedRecord(t, queue, firstSubmission, firstSummaryPath)
	if err := rankings.ApplyCompleted(context.Background(), firstSubmission, rankingTestSummary("match-1", []game.Placement{
		{PlayerID: "alpha", Place: 1},
		{PlayerID: "beta", Place: 2},
	})); err != nil {
		t.Fatalf("ApplyCompleted(first) error = %v", err)
	}

	secondSubmission := rankingTestSubmission("sub-2", "match-2", []SubmittedPlayer{
		{PlayerID: "alpha", ArtifactRef: "artifact://alpha"},
		{PlayerID: "beta", ArtifactRef: "artifact://beta"},
	})
	secondSummaryPath := writeRankingSummaryFile(t, baseDir, "match-2", rankingTestSummary("match-2", []game.Placement{
		{PlayerID: "beta", Place: 1},
		{PlayerID: "alpha", Place: 2},
	}))
	seedCompletedRecord(t, queue, secondSubmission, secondSummaryPath)
	if err := rankings.ApplyCompleted(context.Background(), secondSubmission, rankingTestSummary("match-2", []game.Placement{
		{PlayerID: "beta", Place: 1},
		{PlayerID: "alpha", Place: 2},
	})); err != nil {
		t.Fatalf("ApplyCompleted(second) error = %v", err)
	}

	scope := RankingScope{GameID: "echo", GameVersion: "v1", RulesetVersion: "default"}
	recomputed, err := rankings.Recompute(context.Background(), scope)
	if err != nil {
		t.Fatalf("Recompute() error = %v", err)
	}
	if recomputed.CompletedMatches != 2 {
		t.Fatalf("Recompute().CompletedMatches = %d, want 2", recomputed.CompletedMatches)
	}

	verification, err := rankings.Verify(context.Background(), scope)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !verification.StoredSnapshotOK {
		t.Fatalf("Verify().StoredSnapshotOK = false, want true; payload = %+v", verification)
	}
}

func TestWorkerProcessNextUpdatesRanking(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	queue := NewInMemoryQueueStore()
	rankingStore, err := NewLocalRankingSnapshotStore(baseDir)
	if err != nil {
		t.Fatalf("NewLocalRankingSnapshotStore() error = %v", err)
	}
	rankings, err := NewRankingService(rankingStore, queue)
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}

	submission := rankingTestSubmission("sub-1", "match-1", []SubmittedPlayer{
		{PlayerID: "alpha", ArtifactRef: "artifact://alpha"},
		{PlayerID: "beta", ArtifactRef: "artifact://beta"},
	})
	submission.OutputDir = filepath.Join(baseDir, "output")
	if _, err := queue.Enqueue(context.Background(), submission); err != nil {
		t.Fatalf("queue.Enqueue() error = %v", err)
	}

	worker, err := NewWorker(queue, staticRunnerInvoker{
		result: ExecutionResult{
			Record: rankingTestRecord(submission, []game.Placement{
				{PlayerID: "alpha", Place: 1},
				{PlayerID: "beta", Place: 2},
			}),
		},
	}, LocalTerminalPersister{}, rankings)
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}
	if _, err := worker.ProcessNext(context.Background(), "worker-1"); err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}

	stored, err := rankings.Get(context.Background(), RankingScope{
		GameID:         "echo",
		GameVersion:    "v1",
		RulesetVersion: "default",
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if stored.Snapshot.CompletedMatches != 1 {
		t.Fatalf("CompletedMatches = %d, want 1", stored.Snapshot.CompletedMatches)
	}
}

type staticRunnerInvoker struct {
	result ExecutionResult
	err    error
}

func (s staticRunnerInvoker) Run(_ context.Context, _ ExecutionRequest) (ExecutionResult, error) {
	return s.result, s.err
}

func rankingTestSubmission(submissionID, matchID string, players []SubmittedPlayer) MatchSubmission {
	return MatchSubmission{
		SubmissionID: submissionID,
		MatchID:      matchID,
		Game: contract.GameMetadata{
			GameID:         "echo",
			GameVersion:    "v1",
			RulesetVersion: "default",
		},
		Players:      players,
		OutputDir:    "unused",
		AttemptCount: 1,
	}
}

func rankingTestSummary(matchID string, placements []game.Placement) artifacts.ResultSummary {
	return artifacts.ResultSummary{
		MatchID:        matchID,
		GameID:         "echo",
		GameVersion:    "v1",
		RulesetVersion: "default",
		Status:         game.StatusCompleted,
		Turn:           5,
		Placements:     placements,
	}
}

func rankingTestRecord(submission MatchSubmission, placements []game.Placement) match.Record {
	return match.Record{
		MatchID: submission.MatchID,
		Game: contract.GameMetadata{
			GameID:         submission.Game.GameID,
			GameVersion:    submission.Game.GameVersion,
			RulesetVersion: submission.Game.RulesetVersion,
		},
		Players: []game.Player{
			{PlayerID: submission.Players[0].PlayerID, AIID: submission.Players[0].ArtifactRef},
			{PlayerID: submission.Players[1].PlayerID, AIID: submission.Players[1].ArtifactRef},
		},
		Status: game.StatusCompleted,
		Result: game.MatchResult{Placements: placements},
		Snapshot: game.Snapshot{
			MatchID:        submission.MatchID,
			GameID:         submission.Game.GameID,
			GameVersion:    submission.Game.GameVersion,
			RulesetVersion: submission.Game.RulesetVersion,
			Turn:           5,
			Status:         game.StatusCompleted,
			PerPlayer:      map[string]game.PlayerSnapshot{},
		},
		ExportedSnapshot: game.ExportedSnapshot{
			MatchID:        submission.MatchID,
			GameID:         submission.Game.GameID,
			GameVersion:    submission.Game.GameVersion,
			RulesetVersion: submission.Game.RulesetVersion,
			Turn:           5,
			Status:         game.StatusCompleted,
		},
	}
}

func writeRankingSummaryFile(t *testing.T, baseDir, matchID string, summary artifacts.ResultSummary) string {
	t.Helper()
	path := filepath.Join(baseDir, matchID, "result-summary.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func seedCompletedRecord(t *testing.T, queue *InMemoryQueueStore, submission MatchSubmission, summaryPath string) {
	t.Helper()
	ctx := context.Background()
	if _, err := queue.Enqueue(ctx, submission); err != nil {
		t.Fatalf("queue.Enqueue() error = %v", err)
	}
	record, err := queue.Claim(ctx, "worker-1")
	if err != nil {
		t.Fatalf("queue.Claim() error = %v", err)
	}
	record.State = StateRunning
	if err := queue.Update(ctx, record); err != nil {
		t.Fatalf("queue.Update(running) error = %v", err)
	}
	record.State = StatePersisting
	if err := queue.Update(ctx, record); err != nil {
		t.Fatalf("queue.Update(persisting) error = %v", err)
	}
	record.State = StateCompleted
	record.Terminal = &TerminalArtifacts{
		ResultSummaryPath: summaryPath,
		MatchStatus:       game.StatusCompleted,
	}
	if err := queue.Update(ctx, record); err != nil {
		t.Fatalf("queue.Update(completed) error = %v", err)
	}
}
