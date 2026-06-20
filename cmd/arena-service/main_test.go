package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/service"
)

func TestRunResolvesRelativeOutputDirAgainstBaseDir(t *testing.T) {
	baseDir := repoRoot(t)

	input := `{
  "run_id": "run-1",
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
  "attempt_count": 1,
  "run_kind": "initial"
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

func TestResolvingPresetCatalogResolvesRelativeOutputDirAgainstBaseDir(t *testing.T) {
	baseDir := repoRoot(t)
	presets, err := service.NewStaticPresetCatalog([]service.MatchPresetDefinition{
		{
			PresetID: "echo-reference",
			Game: contract.GameMetadata{
				GameID:         "echo-count",
				GameVersion:    "2.0.0",
				RulesetVersion: "phase2-simultaneous-2turn",
			},
			Players: []service.SubmittedPlayer{
				{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
				{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			},
			OutputDir: "arena-service-output",
		},
	})
	if err != nil {
		t.Fatalf("NewStaticPresetCatalog() error = %v", err)
	}

	submission, err := resolvingPresetCatalog{
		baseDir: baseDir,
		opaque:  false,
		next:    presets,
	}.Build(context.Background(), service.PresetMatchRequest{PresetID: "echo-reference"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	want := filepath.Join(baseDir, "arena-service-output")
	if submission.OutputDir != want {
		t.Fatalf("output_dir = %q, want %q", submission.OutputDir, want)
	}
}

func TestResolveOutputDirKeepsOpaquePrefixForR2Lane(t *testing.T) {
	submission := service.MatchSubmission{OutputDir: "arena-service-output"}
	resolveOutputDir("/tmp/base", true, &submission)
	if submission.OutputDir != "arena-service-output" {
		t.Fatalf("output_dir = %q, want opaque prefix preserved", submission.OutputDir)
	}
}

func TestRunWithoutSubcommandShowsTopLevelUsage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(nil, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() returned nil error")
	}
	want := "usage: arena-service <submit|run-once|submit-cancel|list|get|read|serve|ranking-get|ranking-recompute|ranking-verify|signup-invite-create> ..."
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRunListGetAndReadUseQuerySurface(t *testing.T) {
	app := newSharedTestCLIApp(t)

	completed := service.MatchSubmission{
		RunID:   "run-list-completed",
		MatchID: "match-list-completed",
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
		Players: []service.SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
		},
		OutputDir:    t.TempDir(),
		AttemptCount: 1,
		RunKind:      service.RunKindInitial,
	}
	if _, err := app.runOnce(context.Background(), completed, "worker-1"); err != nil {
		t.Fatalf("runOnce(completed) error = %v", err)
	}

	queued := service.MatchSubmission{
		RunID:   "run-list-queued",
		MatchID: "match-list-queued",
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
		Players: []service.SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
		},
		OutputDir:    filepath.Join(t.TempDir(), "queued-output"),
		AttemptCount: 1,
		RunKind:      service.RunKindInitial,
	}
	if _, err := app.commands.Submit(context.Background(), queued); err != nil {
		t.Fatalf("Submit(queued) error = %v", err)
	}

	factory := func(string, time.Duration, string, artifactRuntimeConfig) (*cliApp, error) {
		return app, nil
	}

	var listOut bytes.Buffer
	var stderr bytes.Buffer
	if err := runWithFactory([]string{"list"}, &listOut, &stderr, factory); err != nil {
		t.Fatalf("runWithFactory(list) error = %v, stderr = %s", err, stderr.String())
	}
	var items []service.ResultListItem
	if err := json.Unmarshal(listOut.Bytes(), &items); err != nil {
		t.Fatalf("json.Unmarshal(list) error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	var getOut bytes.Buffer
	if err := runWithFactory([]string{"get", "--run-id", completed.RunID}, &getOut, &stderr, factory); err != nil {
		t.Fatalf("runWithFactory(get) error = %v, stderr = %s", err, stderr.String())
	}
	var detail service.MatchDetail
	if err := json.Unmarshal(getOut.Bytes(), &detail); err != nil {
		t.Fatalf("json.Unmarshal(get) error = %v", err)
	}
	if detail.ResultSummary == nil {
		t.Fatal("detail.ResultSummary = nil, want compact summary")
	}
	if detail.ReplayInputs == nil {
		t.Fatal("detail.ReplayInputs = nil, want replay/resume/audit locators")
	}
	if !detail.ReplayInputs.Verification.Checked || !detail.ReplayInputs.Verification.Consistent {
		t.Fatalf("detail.ReplayInputs.Verification = %+v, want checked consistent replay inputs", detail.ReplayInputs.Verification)
	}

	var readOut bytes.Buffer
	if err := runWithFactory([]string{"read", "--run-id", completed.RunID, "--artifact", "result-summary"}, &readOut, &stderr, factory); err != nil {
		t.Fatalf("runWithFactory(read) error = %v, stderr = %s", err, stderr.String())
	}
	var summary artifacts.ResultSummary
	if err := json.Unmarshal(readOut.Bytes(), &summary); err != nil {
		t.Fatalf("json.Unmarshal(read) error = %v", err)
	}
	if summary.MatchID != completed.MatchID {
		t.Fatalf("summary.MatchID = %q, want %q", summary.MatchID, completed.MatchID)
	}

	for _, artifactKind := range []string{"snapshot", "history", "exported-snapshot"} {
		readOut.Reset()
		if err := runWithFactory([]string{"read", "--run-id", completed.RunID, "--artifact", artifactKind}, &readOut, &stderr, factory); err != nil {
			t.Fatalf("runWithFactory(read %s) error = %v, stderr = %s", artifactKind, err, stderr.String())
		}
		if readOut.Len() == 0 {
			t.Fatalf("read %s returned empty output", artifactKind)
		}
	}
}

func TestRunOncePersistsTerminalArtifacts(t *testing.T) {
	baseDir := repoRoot(t)
	outputDir := t.TempDir()
	submission := service.MatchSubmission{
		RunID:   "run-echo-1",
		MatchID: "match-echo-1",
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
		RunKind:      service.RunKindInitial,
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

func TestRunOnceFailureStillPrintsTerminalRecord(t *testing.T) {
	baseDir := repoRoot(t)
	outputDir := t.TempDir()
	submission := service.MatchSubmission{
		RunID:   "run-echo-timeout-1",
		MatchID: "match-echo-timeout-1",
		Game: contract.GameMetadata{
			GameID:         "echo-count",
			GameVersion:    "2.0.0",
			RulesetVersion: "phase2-simultaneous-2turn",
		},
		Players: []service.SubmittedPlayer{
			{PlayerID: "p1", ArtifactRef: repoJoin(t, "testdata/ai/echo/echo-ai-2turn")},
			{PlayerID: "p2", ArtifactRef: repoJoin(t, "testdata/ai/echo/timeout-ai")},
		},
		OutputDir:    outputDir,
		AttemptCount: 1,
		RunKind:      service.RunKindInitial,
	}
	submissionPath := writeSubmissionFile(t, submission)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{
		"run-once",
		"--submission", submissionPath,
		"--base-dir", baseDir,
		"--match-timeout", "10ms",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("run() returned nil error")
	}

	var record struct {
		State    string `json:"state"`
		Terminal struct {
			MatchDir          string            `json:"match_dir"`
			RecordPath        string            `json:"record_path"`
			ResultSummaryPath string            `json:"result_summary_path"`
			PlayerStderrPaths map[string]string `json:"player_stderr_paths"`
			MatchStatus       string            `json:"match_status"`
			Error             string            `json:"error"`
		} `json:"terminal"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &record); jsonErr != nil {
		t.Fatalf("json.Unmarshal() error = %v", jsonErr)
	}
	if record.State != "completed" {
		t.Fatalf("state = %q, want completed", record.State)
	}
	if record.Terminal.MatchStatus != "canceled" {
		t.Fatalf("match_status = %q, want canceled", record.Terminal.MatchStatus)
	}
	assertPathExists(t, record.Terminal.MatchDir)
	assertPathExists(t, record.Terminal.RecordPath)
	assertPathExists(t, record.Terminal.ResultSummaryPath)
	if len(record.Terminal.PlayerStderrPaths) == 0 {
		t.Fatal("player_stderr_paths should not be empty")
	}
}

func TestSubmitRejectsIncompatibleArtifactWithoutArtifacts(t *testing.T) {
	baseDir := repoRoot(t)
	outputDir := filepath.Join(t.TempDir(), "rejected-output")
	submissionPath := writeSubmissionFile(t, service.MatchSubmission{
		RunID:   "run-bad-1",
		MatchID: "match-bad-1",
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
		RunKind:      service.RunKindInitial,
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
		RunID:   "run-cancel-1",
		MatchID: "match-cancel-1",
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
		RunKind:      service.RunKindInitial,
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

func newSharedTestCLIApp(t *testing.T) *cliApp {
	t.Helper()

	baseDir := repoRoot(t)
	dryRun, err := service.NewLocalDryRunChecker(baseDir)
	if err != nil {
		t.Fatalf("NewLocalDryRunChecker() error = %v", err)
	}
	validator, err := service.NewDefaultAdmissionValidator(nil, dryRun)
	if err != nil {
		t.Fatalf("NewDefaultAdmissionValidator() error = %v", err)
	}
	store := service.NewInMemoryQueueStore()
	commands, err := service.NewCommandService(store, validator)
	if err != nil {
		t.Fatalf("NewCommandService() error = %v", err)
	}
	queries, err := service.NewQueryService(store)
	if err != nil {
		t.Fatalf("NewQueryService() error = %v", err)
	}
	reader := service.NewDefaultArtifactReader(nil)
	rankingStore, err := service.NewLocalRankingSnapshotStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalRankingSnapshotStore() error = %v", err)
	}
	rankings, err := service.NewRankingService(rankingStore, store, reader)
	if err != nil {
		t.Fatalf("NewRankingService() error = %v", err)
	}
	return &cliApp{
		commands:       commands,
		queries:        queries,
		queue:          store,
		reader:         reader,
		rankings:       rankings,
		artifactAccess: service.DirectArtifactAccessIssuer{},
		persister:      service.LocalTerminalPersister{},
		baseDir:        baseDir,
		closeFn:        func() {},
	}
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
