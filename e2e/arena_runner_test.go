package e2e

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

func TestArenaRunnerHappyPaths(t *testing.T) {
	t.Run("simultaneous", func(t *testing.T) {
		record, stderr := runArena(t,
			"--game", "echo-count",
			"--mode", "simultaneous",
			"--turns", "3",
			"--match-id", "sim-happy",
			"--player", "p1=./testdata/ai/echo/echo-ai",
			"--player", "p2=./testdata/ai/echo/echo-ai",
		)
		if record.Status != "completed" {
			t.Fatalf("status = %q, want completed", record.Status)
		}
		if got := record.Result.Placements[0].Place; got != 1 {
			t.Fatalf("first place = %d, want 1", got)
		}
		if record.Snapshot.PerPlayer["p1"].StderrBytes == 0 {
			t.Fatal("expected stderr bytes for p1")
		}
		if strings.TrimSpace(stderr) != "" {
			t.Fatalf("stderr = %q, want empty", stderr)
		}
	})

	t.Run("sequential", func(t *testing.T) {
		record, _ := runArena(t,
			"--game", "echo-count",
			"--mode", "sequential",
			"--turns", "3",
			"--match-id", "seq-happy",
			"--player", "p1=./testdata/ai/echo/echo-ai",
			"--player", "p2=./testdata/ai/echo/echo-ai",
		)
		if record.Status != "completed" {
			t.Fatalf("status = %q, want completed", record.Status)
		}
		if record.Snapshot.Turn != 3 {
			t.Fatalf("snapshot turn = %d, want 3", record.Snapshot.Turn)
		}
	})
}

func TestArenaRunnerPreflightMetadataMismatch(t *testing.T) {
	cmd := exec.CommandContext(context.Background(), "go", "run", "./cmd/arena-runner",
		"--game", "echo-count",
		"--mode", "simultaneous",
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
			record, _ := runArena(t,
				"--game", "echo-count",
				"--mode", "simultaneous",
				"--turns", "2",
				"--match-id", tc.name,
				"--player", "p1=./testdata/ai/echo/echo-ai",
				"--player", "p2="+tc.player2,
			)
			if record.Status != tc.status {
				t.Fatalf("status = %q, want %q", record.Status, tc.status)
			}
			if !hasEvent(record.EventLog, tc.eventKind) {
				t.Fatalf("event log missing %q", tc.eventKind)
			}
			if tc.reason != "" && !hasFailureReason(record.EventLog, tc.reason) && record.Snapshot.PerPlayer["p2"].LastOutcome.FailureReason != tc.reason {
				t.Fatalf("missing failure reason %q", tc.reason)
			}
		})
	}
}

func runArena(t *testing.T, args ...string) (match.Record, string) {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "go", append([]string{"run", "./cmd/arena-runner"}, args...)...)
	cmd.Dir = repoRoot(t)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("arena-runner failed: %v\nstderr=%s", err, stderr.String())
	}

	var record match.Record
	if err := json.Unmarshal([]byte(stdout.String()), &record); err != nil {
		t.Fatalf("decode record: %v\nstdout=%s", err, stdout.String())
	}
	return record, stderr.String()
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
