package e2e

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

type runnerLogRecord struct {
	MatchID string          `json:"match_id"`
	Seq     int             `json:"seq"`
	Kind    string          `json:"kind"`
	Turn    int             `json:"turn"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type arenaRunResult struct {
	OutputDir string
	MatchDir  string
	Logs      []runnerLogRecord
	Record    match.Record
	Stderr    string
}

func TestArenaRunnerHappyPaths(t *testing.T) {
	t.Run("simultaneous", func(t *testing.T) {
		result := runArena(t,
			"--game", "echo-count",
			"--game-version", "2.0.0",
			"--ruleset", "phase2-simultaneous-3turn",
			"--match-id", "sim-happy",
			"--player", "p1=./testdata/ai/echo/echo-ai",
			"--player", "p2=./testdata/ai/echo/echo-ai",
		)
		record := result.Record
		if record.Status != contract.StatusCompleted {
			t.Fatalf("status = %q, want completed", record.Status)
		}
		if got := record.Result.Placements[0].Place; got != 1 {
			t.Fatalf("first place = %d, want 1", got)
		}
		if record.Snapshot.PerPlayer["p1"].StderrBytes == 0 {
			t.Fatal("expected stderr bytes for p1")
		}
		if !hasLogKind(result.Logs, "match_started") {
			t.Fatalf("logs missing match_started: %+v", result.Logs)
		}
		if !hasLogKind(result.Logs, "terminal_snapshot") || !hasLogKind(result.Logs, "terminal_exported_snapshot") || !hasLogKind(result.Logs, "terminal_summary") {
			t.Fatalf("logs missing terminal records: %+v", result.Logs)
		}
		if strings.TrimSpace(result.Stderr) != "" {
			t.Fatalf("stderr = %q, want empty", result.Stderr)
		}
	})

	t.Run("sequential", func(t *testing.T) {
		result := runArena(t,
			"--game", "echo-count",
			"--game-version", "2.0.0",
			"--ruleset", "phase2-sequential-3turn",
			"--match-id", "seq-happy",
			"--player", "p1=./testdata/ai/echo/echo-ai-sequential",
			"--player", "p2=./testdata/ai/echo/echo-ai-sequential",
		)
		record := result.Record
		if record.Status != contract.StatusCompleted {
			t.Fatalf("status = %q, want completed", record.Status)
		}
		if record.Snapshot.Turn != 3 {
			t.Fatalf("snapshot turn = %d, want 3", record.Snapshot.Turn)
		}
	})
}

func TestArenaRunnerPreflightMetadataMismatch(t *testing.T) {
	cmd := exec.CommandContext(newTestContext(t), "go", "run", "./cmd/arena-runner",
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/version3-ai",
	)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected metadata mismatch error")
	}
	if !strings.Contains(string(output), "metadata incompatible") {
		t.Fatalf("output = %s, want metadata incompatible", output)
	}
}

func TestArenaRunnerRejectsDuplicatePlayerIDs(t *testing.T) {
	cmd := exec.CommandContext(newTestContext(t), "go", "run", "./cmd/arena-runner",
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p1=./testdata/ai/echo/echo-ai",
	)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected duplicate player_id error")
	}
	if !strings.Contains(string(output), "duplicate player_id") {
		t.Fatalf("output = %s, want duplicate player_id", output)
	}
}

func TestArenaRunnerFailurePaths(t *testing.T) {
	cases := []struct {
		name      string
		player2   string
		status    contract.MatchStatus
		eventKind string
		reason    contract.FailureReason
	}{
		{name: "timeout", player2: "./testdata/ai/echo/timeout-ai", status: contract.StatusCompleted, eventKind: "turn_timeout", reason: contract.ReasonTimeout},
		{name: "invalid-action", player2: "./testdata/ai/echo/invalid-action-ai", status: contract.StatusCompleted, eventKind: "turn_result", reason: contract.ReasonIllegalAction},
		{name: "bad-json", player2: "./testdata/ai/echo/bad-json-ai", status: contract.StatusCompleted, eventKind: "protocol_error", reason: contract.ReasonMalformed},
		{name: "mismatched-id", player2: "./testdata/ai/echo/mismatched-id-ai", status: contract.StatusCompleted, eventKind: "protocol_error", reason: contract.ReasonMismatchedID},
		{name: "late-response", player2: "./testdata/ai/echo/late-response-ai", status: contract.StatusCompleted, eventKind: "late_response_ignored", reason: contract.ReasonTimeout},
		{name: "init-timeout", player2: "./testdata/ai/echo/init-timeout-ai", status: contract.StatusFailed, eventKind: "match_failed", reason: contract.ReasonTimeout},
		{name: "shutdown-failure", player2: "./testdata/ai/echo/hung-after-game-over-ai", status: contract.StatusCompleted, eventKind: "session_shutdown_failed", reason: ""},
		{name: "exit-after-init", player2: "./testdata/ai/echo/exit-after-init-ai", status: contract.StatusCompleted, eventKind: "runtime_exited", reason: contract.ReasonRuntimeStop},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runArena(t,
				"--game", "echo-count",
				"--game-version", "2.0.0",
				"--ruleset", "phase2-simultaneous-2turn",
				"--match-id", tc.name,
				"--player", "p1=./testdata/ai/echo/echo-ai-2turn",
				"--player", "p2="+tc.player2,
			)
			record := result.Record
			if record.Status != tc.status {
				t.Fatalf("status = %q, want %q", record.Status, tc.status)
			}
			if !hasEvent(record.EventLog, tc.eventKind) {
				t.Fatalf("event log missing %q", tc.eventKind)
			}
			if tc.reason != "" && !hasFailureReason(record.EventLog, tc.reason) && record.Snapshot.PerPlayer["p2"].LastActionStatus.FailureReason != tc.reason {
				t.Fatalf("missing failure reason %q", tc.reason)
			}
			if !hasLogKind(result.Logs, "terminal_summary") {
				t.Fatalf("logs missing terminal_summary: %+v", result.Logs)
			}
		})
	}
}

func TestArenaRunnerCanceledPath(t *testing.T) {
	result := runArena(t,
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-2turn",
		"--match-id", "canceled",
		"--match-timeout", "10ms",
		"--player", "p1=./testdata/ai/echo/echo-ai-2turn",
		"--player", "p2=./testdata/ai/echo/timeout-ai",
	)

	if result.Record.Status != contract.StatusCanceled {
		t.Fatalf("status = %q, want canceled", result.Record.Status)
	}
	if !hasEvent(result.Record.EventLog, "match_canceled") {
		t.Fatalf("event log missing match_canceled: %+v", result.Record.EventLog)
	}
	if !hasLogKind(result.Logs, "terminal_snapshot") || !hasLogKind(result.Logs, "terminal_summary") {
		t.Fatalf("logs missing canceled terminal records: %+v", result.Logs)
	}
	if !strings.Contains(result.Stderr, "context deadline exceeded") {
		t.Fatalf("stderr = %q, want context deadline exceeded", result.Stderr)
	}
}

func TestArenaRunnerCanPersistRecordToStdout(t *testing.T) {
	result := runArenaWithPersistTarget(t, "stdout",
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--match-id", "stdout-persist",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	)

	if result.Record.MatchID != "stdout-persist" {
		t.Fatalf("record match id = %q, want stdout-persist", result.Record.MatchID)
	}
	if _, err := os.Stat(filepath.Join(result.MatchDir, "record.json")); err != nil {
		t.Fatalf("standard record.json missing: %v", err)
	}
	if !hasLogKind(result.Logs, "match_started") {
		t.Fatalf("logs missing match_started: %+v", result.Logs)
	}
}

func TestArenaRunnerCanWriteStructuredLogToExtraFile(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "runner.ndjson")

	result := runArenaWithOptions(t, arenaRunOptions{
		LogTarget: logPath,
		Args: []string{
			"--game", "echo-count",
			"--game-version", "2.0.0",
			"--ruleset", "phase2-simultaneous-3turn",
			"--match-id", "file-log",
			"--player", "p1=./testdata/ai/echo/echo-ai",
			"--player", "p2=./testdata/ai/echo/echo-ai",
		},
	})

	if len(result.Logs) == 0 {
		t.Fatal("stdout logs missing")
	}
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	logs, _, err := parseArenaOutput(string(logData))
	if err != nil {
		t.Fatalf("parse log output: %v", err)
	}
	if !hasLogKind(logs, "terminal_summary") {
		t.Fatalf("file logs missing terminal_summary: %+v", logs)
	}
	standardLogData, err := os.ReadFile(filepath.Join(result.MatchDir, "structured-log.ndjson"))
	if err != nil {
		t.Fatalf("read standard structured log: %v", err)
	}
	if string(standardLogData) != string(logData) {
		t.Fatalf("standard structured log did not match extra log output")
	}
}

func TestArenaRunnerStartFromSnapshot(t *testing.T) {
	snapshotPath := filepath.Join(t.TempDir(), "snapshot.json")
	if err := os.WriteFile(snapshotPath, []byte(`{
		"game_id":"echo-count",
		"game_version":"2.0.0",
		"ruleset_version":"phase2-simultaneous-3turn",
		"turn":1,
		"status":"running",
		"game_state":{
			"mode":"simultaneous",
			"turn":1,
			"expected":2,
			"score":{"p1":1,"p2":1}
		},
		"per_player":{
			"p1":{"last_action_status":{"player_id":"p1","action_status":"accepted"}},
			"p2":{"last_action_status":{"player_id":"p2","action_status":"accepted"}}
		}
	}`), 0o644); err != nil {
		t.Fatalf("write snapshot input: %v", err)
	}
	exportedPath := filepath.Join(t.TempDir(), "exported.json")
	result := runArenaWithOptions(t, arenaRunOptions{
		ExportedTarget: exportedPath,
		Args: []string{
			"--game", "echo-count",
			"--game-version", "2.0.0",
			"--ruleset", "phase2-simultaneous-3turn",
			"--match-id", "snapshot-start",
			"--snapshot-input", snapshotPath,
			"--player", "p1=./testdata/ai/echo/echo-ai",
			"--player", "p2=./testdata/ai/echo/echo-ai",
		},
	})

	if result.Record.Status != "completed" {
		t.Fatalf("status = %q, want completed", result.Record.Status)
	}
	if result.Record.Snapshot.Turn != 3 {
		t.Fatalf("final snapshot turn = %d, want 3", result.Record.Snapshot.Turn)
	}
	var exported struct {
		Turn int `json:"turn"`
	}
	data, err := os.ReadFile(exportedPath)
	if err != nil {
		t.Fatalf("read exported snapshot output: %v", err)
	}
	if err := json.Unmarshal(data, &exported); err != nil {
		t.Fatalf("decode exported snapshot output: %v", err)
	}
	if exported.Turn != 1 {
		t.Fatalf("exported snapshot turn = %d, want 1", exported.Turn)
	}
	standardExportedData, err := os.ReadFile(filepath.Join(result.MatchDir, "exported-snapshot.json"))
	if err != nil {
		t.Fatalf("read standard exported snapshot: %v", err)
	}
	var standardExported struct {
		Turn int `json:"turn"`
	}
	if err := json.Unmarshal(standardExportedData, &standardExported); err != nil {
		t.Fatalf("decode standard exported snapshot: %v", err)
	}
	if standardExported.Turn != 3 {
		t.Fatalf("standard exported snapshot turn = %d, want 3", standardExported.Turn)
	}
}

func TestArenaRunnerResumeFromHistoryAndContinue(t *testing.T) {
	base := runArena(t,
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--match-id", "history-source",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	)
	historyPath := filepath.Join(t.TempDir(), "history.json")
	historyData, err := json.Marshal(base.Record.EventLog)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	if err := os.WriteFile(historyPath, historyData, 0o644); err != nil {
		t.Fatalf("write history input: %v", err)
	}

	result := runArenaWithOptions(t, arenaRunOptions{
		Args: []string{
			"--game", "echo-count",
			"--game-version", "2.0.0",
			"--ruleset", "phase2-simultaneous-3turn",
			"--match-id", "history-resume",
			"--history-input", historyPath,
			"--target-turn", "2",
			"--player", "p1=./testdata/ai/echo/echo-ai",
			"--player", "p2=./testdata/ai/echo/echo-ai",
		},
	})

	if result.Record.Status != "completed" {
		t.Fatalf("status = %q, want completed", result.Record.Status)
	}
	for _, event := range result.Record.EventLog {
		if event.Turn > 0 && event.Turn != 3 {
			t.Fatalf("resumed event turn = %d, want only turn 3 events in %+v", event.Turn, result.Record.EventLog)
		}
	}
	if result.Record.Snapshot.Turn != 3 {
		t.Fatalf("final snapshot turn = %d, want 3", result.Record.Snapshot.Turn)
	}
}

func TestArenaRunnerJankenHappyPath(t *testing.T) {
	result := runArena(t,
		"--game", "janken",
		"--game-version", janken.GameVersion,
		"--ruleset", janken.RulesetRegular,
		"--match-id", "janken-happy",
		"--player", "p1=./testdata/ai/janken/janken-cycle-ai",
		"--player", "p2=./testdata/ai/janken/janken-rock-ai",
	)

	if result.Record.Status != "completed" {
		t.Fatalf("status = %q, want completed", result.Record.Status)
	}
	if result.Record.Result.Placements[0].PlayerID != "p1" || result.Record.Result.Placements[0].Place != 1 {
		t.Fatalf("first placement = %+v, want p1 place 1", result.Record.Result.Placements[0])
	}

	var visible struct {
		Round         int `json:"round"`
		PublicHistory []struct {
			Actions map[string]string `json:"actions"`
		} `json:"public_history"`
	}
	if err := json.Unmarshal(result.Record.Snapshot.PerPlayer["p1"].VisibleState, &visible); err != nil {
		t.Fatalf("decode final visible state: %v", err)
	}
	if visible.Round != 5 {
		t.Fatalf("final visible round = %d, want 5", visible.Round)
	}
	if len(visible.PublicHistory) != 5 {
		t.Fatalf("len(public_history) = %d, want 5", len(visible.PublicHistory))
	}
	if got := visible.PublicHistory[4].Actions["p1"]; got != "paper" {
		t.Fatalf("round5 p1 action = %q, want paper", got)
	}
}

func TestArenaRunnerJankenResumeFromHistoryAndContinue(t *testing.T) {
	base := runArena(t,
		"--game", "janken",
		"--game-version", janken.GameVersion,
		"--ruleset", janken.RulesetRegular,
		"--match-id", "janken-history-source",
		"--player", "p1=./testdata/ai/janken/janken-cycle-ai",
		"--player", "p2=./testdata/ai/janken/janken-rock-ai",
	)
	historyPath := filepath.Join(t.TempDir(), "history.json")
	historyData, err := json.Marshal(base.Record.EventLog)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	if err := os.WriteFile(historyPath, historyData, 0o644); err != nil {
		t.Fatalf("write history input: %v", err)
	}

	result := runArenaWithOptions(t, arenaRunOptions{
		Args: []string{
			"--game", "janken",
			"--game-version", janken.GameVersion,
			"--ruleset", janken.RulesetRegular,
			"--match-id", "janken-history-resume",
			"--history-input", historyPath,
			"--target-turn", "3",
			"--player", "p1=./testdata/ai/janken/janken-cycle-ai",
			"--player", "p2=./testdata/ai/janken/janken-rock-ai",
		},
	})

	if result.Record.Status != "completed" {
		t.Fatalf("status = %q, want completed", result.Record.Status)
	}
	for _, event := range result.Record.EventLog {
		if event.Turn > 0 && event.Turn != 4 && event.Turn != 5 {
			t.Fatalf("resumed event turn = %d, want only turns 4-5 in %+v", event.Turn, result.Record.EventLog)
		}
	}
}

func TestArenaRunnerJankenTimeoutAndInvalidAffectPlacement(t *testing.T) {
	result := runArena(t,
		"--game", "janken",
		"--game-version", janken.GameVersion,
		"--ruleset", janken.RulesetRegular,
		"--match-id", "janken-failures",
		"--player", "p1=./testdata/ai/janken/janken-rock-ai",
		"--player", "p2=./testdata/ai/janken/janken-timeout-ai",
		"--player", "p3=./testdata/ai/janken/janken-invalid-ai",
	)

	if result.Record.Result.Placements[0].PlayerID != "p1" {
		t.Fatalf("winner = %q, want p1", result.Record.Result.Placements[0].PlayerID)
	}
	if got := result.Record.Snapshot.PerPlayer["p2"].LastActionStatus.FailureReason; got != "" {
		t.Fatalf("final p2 failure reason = %q, want empty after later accepted turns", got)
	}
	if !hasFailureReason(result.Record.EventLog, session.ReasonTimeout) {
		t.Fatalf("event log missing timeout failure: %+v", result.Record.EventLog)
	}
	if !hasFailureReason(result.Record.EventLog, contract.ReasonIllegalAction) {
		t.Fatalf("event log missing invalid action failure: %+v", result.Record.EventLog)
	}
}

func runArena(t *testing.T, args ...string) arenaRunResult {
	t.Helper()

	return runArenaWithOptions(t, arenaRunOptions{OutputDir: t.TempDir(), Args: args})
}

func runArenaWithPersistTarget(t *testing.T, persistTarget string, args ...string) arenaRunResult {
	t.Helper()

	return runArenaWithOptions(t, arenaRunOptions{PersistTarget: persistTarget, Args: args})
}

type arenaRunOptions struct {
	OutputDir      string
	PersistTarget  string
	LogTarget      string
	ExportedTarget string
	Args           []string
}

func runArenaWithOptions(t *testing.T, opts arenaRunOptions) arenaRunResult {
	t.Helper()

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = t.TempDir()
	}
	matchID := findFlagValue(opts.Args, "--match-id")
	if matchID == "" {
		t.Fatal("runArenaWithOptions requires --match-id")
	}
	matchDir := filepath.Join(outputDir, matchID)
	recordPath := filepath.Join(matchDir, "record.json")

	fullArgs := append([]string{"run", "./cmd/arena-runner"}, opts.Args...)
	fullArgs = append(fullArgs, "--output-dir", outputDir)
	if opts.PersistTarget != "" {
		fullArgs = append(fullArgs, "--persist-record", opts.PersistTarget)
	}
	if opts.LogTarget != "" {
		fullArgs = append(fullArgs, "--log-output", opts.LogTarget)
	}
	if opts.ExportedTarget != "" {
		fullArgs = append(fullArgs, "--exported-snapshot-output", opts.ExportedTarget)
	}
	cmd := exec.CommandContext(newTestContext(t), "go", fullArgs...)
	cmd.Dir = repoRoot(t)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("arena-runner failed: %v\nstderr=%s", err, stderr.String())
	}

	logs, record, err := parseArenaOutput(stdout.String())
	if err != nil {
		t.Fatalf("decode output: %v\nstdout=%s", err, stdout.String())
	}
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("read standard persisted record: %v", err)
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatalf("decode standard persisted record: %v\nrecord=%s", err, data)
	}

	return arenaRunResult{
		OutputDir: outputDir,
		MatchDir:  matchDir,
		Logs:      logs,
		Record:    record,
		Stderr:    stderr.String(),
	}
}

func TestArenaRunnerWritesStandardArtifactsToDefaultOutputDir(t *testing.T) {
	repo := repoRoot(t)
	matchID := "default-output-dir"
	artifactRoot := filepath.Join(repo, "arena-runner-output")
	matchDir := filepath.Join(artifactRoot, matchID)
	t.Cleanup(func() {
		_ = os.RemoveAll(matchDir)
	})

	cmd := exec.CommandContext(newTestContext(t), "go", "run", "./cmd/arena-runner",
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--match-id", matchID,
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	)
	cmd.Dir = repo
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("arena-runner failed: %v\nstderr=%s", err, stderr.String())
	}

	for _, name := range []string{"record.json", "structured-log.ndjson", "snapshot.json", "exported-snapshot.json", "history.json"} {
		if _, err := os.Stat(filepath.Join(matchDir, name)); err != nil {
			t.Fatalf("default artifact %s missing: %v", name, err)
		}
	}
}

func TestArenaRunnerRejectsEmptyOutputDir(t *testing.T) {
	cmd := exec.CommandContext(newTestContext(t), "go", "run", "./cmd/arena-runner",
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--match-id", "empty-output-dir",
		"--output-dir", "",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected empty output-dir error")
	}
	if !strings.Contains(string(output), "--output-dir must not be empty") {
		t.Fatalf("output = %s, want empty output-dir error", output)
	}
}

func TestArenaRunnerFailsBeforeSessionStartWhenOutputDirCannotBeCreated(t *testing.T) {
	base := t.TempDir()
	blockingPath := filepath.Join(base, "not-a-directory")
	if err := os.WriteFile(blockingPath, []byte("block"), 0o644); err != nil {
		t.Fatalf("write blocking path: %v", err)
	}

	cmd := exec.CommandContext(newTestContext(t), "go", "run", "./cmd/arena-runner",
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--match-id", "unwritable-output-dir",
		"--output-dir", blockingPath,
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	)
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected output-dir creation failure")
	}
	if !strings.Contains(string(output), "create artifact directory") {
		t.Fatalf("output = %s, want artifact directory error", output)
	}
	if strings.Contains(string(output), `"kind":"match_started"`) {
		t.Fatalf("output = %s, want fail-fast before session start", output)
	}
}

func TestArenaRunnerWritesDerivedArtifacts(t *testing.T) {
	result := runArena(t,
		"--game", "echo-count",
		"--game-version", "2.0.0",
		"--ruleset", "phase2-simultaneous-3turn",
		"--match-id", "derived-artifacts",
		"--player", "p1=./testdata/ai/echo/echo-ai",
		"--player", "p2=./testdata/ai/echo/echo-ai",
	)

	snapshotData, err := os.ReadFile(filepath.Join(result.MatchDir, "snapshot.json"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	recordSnapshotData, err := json.Marshal(result.Record.Snapshot)
	if err != nil {
		t.Fatalf("marshal record snapshot: %v", err)
	}
	if !jsonEqual(snapshotData, recordSnapshotData) {
		t.Fatal("snapshot.json did not match record snapshot")
	}

	exportedData, err := os.ReadFile(filepath.Join(result.MatchDir, "exported-snapshot.json"))
	if err != nil {
		t.Fatalf("read exported snapshot: %v", err)
	}
	recordExportedData, err := json.Marshal(result.Record.ExportedSnapshot)
	if err != nil {
		t.Fatalf("marshal record exported snapshot: %v", err)
	}
	if !jsonEqual(exportedData, recordExportedData) {
		t.Fatal("exported-snapshot.json did not match record exported_snapshot")
	}

	historyData, err := os.ReadFile(filepath.Join(result.MatchDir, "history.json"))
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	recordHistoryData, err := json.Marshal(result.Record.EventLog)
	if err != nil {
		t.Fatalf("marshal record history: %v", err)
	}
	if !jsonEqual(historyData, recordHistoryData) {
		t.Fatal("history.json did not match record event_log")
	}
}

func findFlagValue(args []string, name string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func jsonEqual(a, b []byte) bool {
	var left any
	if err := json.Unmarshal(a, &left); err != nil {
		return false
	}
	var right any
	if err := json.Unmarshal(b, &right); err != nil {
		return false
	}
	return strings.TrimSpace(string(mustJSON(left))) == strings.TrimSpace(string(mustJSON(right)))
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

func parseArenaOutput(stdout string) ([]runnerLogRecord, match.Record, error) {
	var (
		logs   []runnerLogRecord
		record match.Record
	)

	dec := json.NewDecoder(strings.NewReader(stdout))
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, match.Record{}, err
		}
		var envelope struct {
			EventLog json.RawMessage `json:"event_log"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return nil, match.Record{}, err
		}
		if envelope.EventLog != nil {
			if err := json.Unmarshal(raw, &record); err != nil {
				return nil, match.Record{}, err
			}
			continue
		}
		var logRecord runnerLogRecord
		if err := json.Unmarshal(raw, &logRecord); err != nil {
			return nil, match.Record{}, err
		}
		logs = append(logs, logRecord)
	}
	return logs, record, nil
}

func hasLogKind(logs []runnerLogRecord, kind string) bool {
	for _, record := range logs {
		if record.Kind == kind {
			return true
		}
	}
	return false
}

func hasEvent(events []match.Event, kind string) bool {
	for _, event := range events {
		if event.Kind == kind {
			return true
		}
	}
	return false
}

func hasFailureReason(events []match.Event, reason contract.FailureReason) bool {
	for _, event := range events {
		if strings.Contains(string(event.Payload), string(reason)) {
			return true
		}
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	return filepath.Clean("..")
}

func newTestContext(t *testing.T) context.Context {
	t.Helper()

	if deadline, ok := t.Deadline(); ok {
		timeout := time.Until(deadline) - time.Second
		if timeout > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			t.Cleanup(cancel)
			return ctx
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	return ctx
}
