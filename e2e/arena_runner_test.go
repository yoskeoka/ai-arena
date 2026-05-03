package e2e

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

type runnerLogRecord struct {
	MatchID string          `json:"match_id"`
	Seq     int             `json:"seq"`
	Kind    string          `json:"kind"`
	Turn    int             `json:"turn"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type arenaRunResult struct {
	Logs   []runnerLogRecord
	Record match.Record
	Stderr string
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
		if record.Status != "completed" {
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
		if record.Status != "completed" {
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
		status    string
		eventKind string
		reason    string
	}{
		{name: "timeout", player2: "./testdata/ai/echo/timeout-ai", status: "completed", eventKind: "turn_timeout", reason: "invalid-timeout"},
		{name: "invalid-action", player2: "./testdata/ai/echo/invalid-action-ai", status: "completed", eventKind: "turn_result", reason: "invalid-illegal-action"},
		{name: "bad-json", player2: "./testdata/ai/echo/bad-json-ai", status: "completed", eventKind: "protocol_error", reason: "invalid-protocol-malformed"},
		{name: "mismatched-id", player2: "./testdata/ai/echo/mismatched-id-ai", status: "completed", eventKind: "protocol_error", reason: "invalid-protocol-mismatched-id"},
		{name: "late-response", player2: "./testdata/ai/echo/late-response-ai", status: "completed", eventKind: "late_response_ignored", reason: "invalid-timeout"},
		{name: "init-timeout", player2: "./testdata/ai/echo/init-timeout-ai", status: "failed", eventKind: "match_failed", reason: "invalid-timeout"},
		{name: "shutdown-failure", player2: "./testdata/ai/echo/hung-after-game-over-ai", status: "completed", eventKind: "session_shutdown_failed", reason: ""},
		{name: "exit-after-init", player2: "./testdata/ai/echo/exit-after-init-ai", status: "completed", eventKind: "runtime_exited", reason: "runtime-stopped"},
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

	if result.Record.Status != "canceled" {
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
	if !hasLogKind(result.Logs, "match_started") {
		t.Fatalf("logs missing match_started: %+v", result.Logs)
	}
}

func runArena(t *testing.T, args ...string) arenaRunResult {
	t.Helper()

	recordPath := filepath.Join(t.TempDir(), "record.json")
	return runArenaWithPersistTarget(t, recordPath, args...)
}

func runArenaWithPersistTarget(t *testing.T, persistTarget string, args ...string) arenaRunResult {
	t.Helper()

	fullArgs := append([]string{"run", "./cmd/arena-runner"}, args...)
	fullArgs = append(fullArgs, "--persist-record", persistTarget)
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
	if persistTarget != "stdout" {
		data, err := os.ReadFile(persistTarget)
		if err != nil {
			t.Fatalf("read persisted record: %v", err)
		}
		if err := json.Unmarshal(data, &record); err != nil {
			t.Fatalf("decode persisted record: %v\nrecord=%s", err, data)
		}
	}

	return arenaRunResult{
		Logs:   logs,
		Record: record,
		Stderr: stderr.String(),
	}
}

func parseArenaOutput(stdout string) ([]runnerLogRecord, match.Record, error) {
	var (
		logs   []runnerLogRecord
		record match.Record
	)

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Bytes()
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(line, &envelope); err != nil {
			return nil, match.Record{}, err
		}
		if _, ok := envelope["event_log"]; ok {
			if err := json.Unmarshal(line, &record); err != nil {
				return nil, match.Record{}, err
			}
			continue
		}
		var logRecord runnerLogRecord
		if err := json.Unmarshal(line, &logRecord); err != nil {
			return nil, match.Record{}, err
		}
		logs = append(logs, logRecord)
	}
	if err := scanner.Err(); err != nil {
		return nil, match.Record{}, err
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

func hasFailureReason(events []match.Event, reason string) bool {
	for _, event := range events {
		if strings.Contains(string(event.Payload), reason) {
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
