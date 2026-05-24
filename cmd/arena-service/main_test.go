package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

func TestRunResolvesRelativeOutputDirAgainstBaseDir(t *testing.T) {
	baseDir := repoRoot(t)

	input := `{
  "submission_id": "sub-1",
  "match_id": "match-1",
  "game": {
    "game_id": "janken",
    "game_version": "2.1.0",
    "ruleset_version": "regular"
  },
  "players": [
    {
      "player_id": "p1",
      "artifact_ref": "./testdata/ai/janken/janken-rock-ai"
    }
  ],
  "output_dir": "arena-service-output",
  "attempt_count": 1
}`

	submissionPath := filepath.Join(t.TempDir(), "submission.json")
	if err := os.WriteFile(submissionPath, []byte(input), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"submit", "--submission", submissionPath, "--base-dir", baseDir}, &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}

	var record struct {
		Submission struct {
			OutputDir string `json:"output_dir"`
		} `json:"submission"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	want := filepath.Join(baseDir, "arena-service-output")
	if record.Submission.OutputDir != want {
		t.Fatalf("output_dir = %q, want %q", record.Submission.OutputDir, want)
	}
}

func TestRunOncePersistsTerminalArtifacts(t *testing.T) {
	baseDir := repoRoot(t)
	outputDir := t.TempDir()
	submission := service.MatchSubmission{
		SubmissionID: "sub-echo-1",
		MatchID:      "match-echo-1",
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
		Players: []service.SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
		},
		OutputDir:    outputDir,
		AttemptCount: 1,
	}
	submissionPath := writeSubmissionFile(t, submission)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"run-once", "--submission", submissionPath, "--base-dir", baseDir}, &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}

	var record struct {
		State    string `json:"state"`
		Terminal struct {
			MatchDir          string            `json:"match_dir"`
			RecordPath        string            `json:"record_path"`
			ResultSummaryPath string            `json:"result_summary_path"`
			PlayerStderrPaths map[string]string `json:"player_stderr_paths"`
			MatchStatus       string            `json:"match_status"`
		} `json:"terminal"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if record.State != "completed" {
		t.Fatalf("state = %q, want completed", record.State)
	}
	if record.Terminal.MatchStatus != "completed" {
		t.Fatalf("match_status = %q, want completed", record.Terminal.MatchStatus)
	}
	assertPathExists(t, record.Terminal.MatchDir)
	assertPathExists(t, record.Terminal.RecordPath)
	assertPathExists(t, record.Terminal.ResultSummaryPath)
	if len(record.Terminal.PlayerStderrPaths) != 2 {
		t.Fatalf("len(player_stderr_paths) = %d, want 2", len(record.Terminal.PlayerStderrPaths))
	}
	for _, path := range record.Terminal.PlayerStderrPaths {
		assertPathExists(t, path)
	}
	assertPathExists(t, filepath.Join(record.Terminal.MatchDir, "structured-log.ndjson"))
}

func TestSubmitRejectsIncompatibleArtifactWithoutArtifacts(t *testing.T) {
	baseDir := repoRoot(t)
	outputDir := filepath.Join(t.TempDir(), "rejected-output")
	submissionPath := writeSubmissionFile(t, service.MatchSubmission{
		SubmissionID: "sub-bad-1",
		MatchID:      "match-bad-1",
		Game: contract.GameMetadata{
			GameID:         "janken",
			GameVersion:    "2.1.0",
			RulesetVersion: "regular",
		},
		Players: []service.SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai")},
		},
		OutputDir:    outputDir,
		AttemptCount: 1,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"submit", "--submission", submissionPath, "--base-dir", baseDir}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() returned nil error")
	}
	if _, statErr := os.Stat(filepath.Join(outputDir, "match-bad-1")); !os.IsNotExist(statErr) {
		t.Fatalf("match artifact directory should not exist, stat error = %v", statErr)
	}
}

func TestSubmitCancelKeepsQueuedOnlyCancelArtifactFree(t *testing.T) {
	baseDir := repoRoot(t)
	outputDir := filepath.Join(t.TempDir(), "canceled-output")
	submissionPath := writeSubmissionFile(t, service.MatchSubmission{
		SubmissionID: "sub-cancel-1",
		MatchID:      "match-cancel-1",
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
		Players: []service.SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
		},
		OutputDir:    outputDir,
		AttemptCount: 1,
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"submit-cancel", "--submission", submissionPath, "--base-dir", baseDir}, &stdout, &stderr); err != nil {
		t.Fatalf("run() error = %v, stderr = %s", err, stderr.String())
	}

	var record struct {
		State    string           `json:"state"`
		Terminal *json.RawMessage `json:"terminal"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if record.State != "canceled" {
		t.Fatalf("state = %q, want canceled", record.State)
	}
	if record.Terminal != nil {
		t.Fatalf("terminal = %s, want nil", string(*record.Terminal))
	}
	if _, statErr := os.Stat(filepath.Join(outputDir, "match-cancel-1")); !os.IsNotExist(statErr) {
		t.Fatalf("match artifact directory should not exist, stat error = %v", statErr)
	}
}

func writeSubmissionFile(t *testing.T, submission service.MatchSubmission) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "submission.json")
	data, err := json.MarshalIndent(submission, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent() error = %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	return path
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := goRuntime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func repoJoin(t *testing.T, rel string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), filepath.Clean(rel))
}

func assertPathExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path %q to exist: %v", path, err)
	}
}
