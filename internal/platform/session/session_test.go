package session

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

func TestSessionInitTurnTimeoutGameOverAndLateResponse(t *testing.T) {
	adapter, err := runtime.Start(context.Background(), runtime.Config{
		Command: []string{os.Args[0], "-test.run=TestHelperProcess"},
		Env:     []string{"GO_WANT_HELPER_PROCESS=session-bot"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Close(closeCtx)
	}()

	sess := New(adapter)
	var lateIDs []string
	sess.SetLateResponseHook(func(id string) {
		lateIDs = append(lateIDs, id)
	})

	initResult := sess.Init(context.Background(), Request{
		ID:       "init",
		Method:   "init",
		Params:   map[string]any{"match_id": "match-001"},
		Deadline: time.Second,
	})
	if initResult.Status != StatusAccepted {
		t.Fatalf("init status = %q, want accepted", initResult.Status)
	}

	timeoutResult := sess.Turn(context.Background(), Request{
		ID:       "turn-timeout",
		Method:   "turn",
		Params:   map[string]any{"turn": 1},
		Deadline: 50 * time.Millisecond,
	})
	if timeoutResult.FailureReason != ReasonTimeout {
		t.Fatalf("timeout reason = %q, want %q", timeoutResult.FailureReason, ReasonTimeout)
	}

	nextResult := sess.Turn(context.Background(), Request{
		ID:       "turn-next",
		Method:   "turn",
		Params:   map[string]any{"turn": 2},
		Deadline: time.Second,
	})
	if nextResult.Status != StatusAccepted {
		t.Fatalf("next status = %q, want accepted", nextResult.Status)
	}
	if len(nextResult.IgnoredLateResponseIDs) != 1 || nextResult.IgnoredLateResponseIDs[0] != "turn-timeout" {
		t.Fatalf("ignored late response ids = %v, want [turn-timeout]", nextResult.IgnoredLateResponseIDs)
	}

	deadline := time.Now().Add(time.Second)
	for len(lateIDs) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(lateIDs) != 1 || lateIDs[0] != "turn-timeout" {
		t.Fatalf("lateIDs = %v, want [turn-timeout]", lateIDs)
	}

	result := sess.GameOver(context.Background(), Request{
		ID:       "game-over",
		Method:   "game_over",
		Params:   map[string]any{"summary": "done", "shutdown_after_ms": 3000},
		Deadline: time.Second,
	})
	if result.Status != StatusAccepted {
		t.Fatalf("GameOver status = %q, want accepted", result.Status)
	}
}

func TestSessionWASMWASIInitTurnTimeoutGameOverAndLateResponse(t *testing.T) {
	modulePath := buildWASMTestBot(t)

	adapter, err := runtime.Start(context.Background(), runtime.Config{
		Kind:       runtime.KindWASMWASI,
		ModulePath: modulePath,
		Env:        []string{"BOT_MODE=session-bot"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Close(closeCtx)
	}()

	sess := New(adapter)
	var lateIDs []string
	sess.SetLateResponseHook(func(id string) {
		lateIDs = append(lateIDs, id)
	})

	initResult := sess.Init(context.Background(), Request{
		ID:       "init",
		Method:   "init",
		Params:   map[string]any{"match_id": "match-001"},
		Deadline: time.Second,
	})
	if initResult.Status != StatusAccepted {
		t.Fatalf("init status = %q, want accepted", initResult.Status)
	}

	timeoutResult := sess.Turn(context.Background(), Request{
		ID:       "turn-timeout",
		Method:   "turn",
		Params:   map[string]any{"turn": 1},
		Deadline: 50 * time.Millisecond,
	})
	if timeoutResult.FailureReason != ReasonTimeout {
		t.Fatalf("timeout reason = %q, want %q", timeoutResult.FailureReason, ReasonTimeout)
	}

	nextResult := sess.Turn(context.Background(), Request{
		ID:       "turn-next",
		Method:   "turn",
		Params:   map[string]any{"turn": 2},
		Deadline: time.Second,
	})
	if nextResult.Status != StatusAccepted {
		t.Fatalf("next status = %q, want accepted", nextResult.Status)
	}
	if len(nextResult.IgnoredLateResponseIDs) != 1 || nextResult.IgnoredLateResponseIDs[0] != "turn-timeout" {
		t.Fatalf("ignored late response ids = %v, want [turn-timeout]", nextResult.IgnoredLateResponseIDs)
	}

	deadline := time.Now().Add(time.Second)
	for len(lateIDs) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(lateIDs) != 1 || lateIDs[0] != "turn-timeout" {
		t.Fatalf("lateIDs = %v, want [turn-timeout]", lateIDs)
	}

	result := sess.GameOver(context.Background(), Request{
		ID:       "game-over",
		Method:   "game_over",
		Params:   map[string]any{"summary": "done", "shutdown_after_ms": 3000},
		Deadline: time.Second,
	})
	if result.Status != StatusAccepted {
		t.Fatalf("GameOver status = %q, want accepted", result.Status)
	}
}

func TestSessionWASMWASIRuntimeStoppedBeforeDeadline(t *testing.T) {
	modulePath := buildWASMTestBot(t)

	adapter, err := runtime.Start(context.Background(), runtime.Config{
		Kind:       runtime.KindWASMWASI,
		ModulePath: modulePath,
		Env:        []string{"BOT_MODE=exit-after-init"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Close(closeCtx)
	}()

	sess := New(adapter)
	initResult := sess.Init(context.Background(), Request{
		ID:       "init",
		Method:   "init",
		Params:   map[string]any{"match_id": "match-001"},
		Deadline: time.Second,
	})
	if initResult.Status != StatusAccepted {
		t.Fatalf("init status = %q, want accepted", initResult.Status)
	}

	result := sess.Turn(context.Background(), Request{
		ID:       "turn-next",
		Method:   "turn",
		Params:   map[string]any{"turn": 2},
		Deadline: time.Second,
	})
	if result.FailureReason != ReasonRuntimeStop {
		t.Fatalf("turn failure reason = %q, want %q", result.FailureReason, ReasonRuntimeStop)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "session-bot" {
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.Contains(line, `"id":"init"`):
			fmt.Println(`{"jsonrpc":"2.0","id":"init","result":{"ready":true}}`)
		case strings.Contains(line, `"id":"turn-timeout"`):
			time.Sleep(200 * time.Millisecond)
			fmt.Println(`{"jsonrpc":"2.0","id":"turn-timeout","result":{"action":"late"}}`)
		case strings.Contains(line, `"id":"turn-next"`):
			fmt.Println(`{"jsonrpc":"2.0","id":"turn-next","result":{"action":"paper"}}`)
		case strings.Contains(line, `"method":"game_over"`):
			fmt.Fprintln(os.Stderr, "game over received")
			fmt.Println(`{"jsonrpc":"2.0","id":"game-over","result":{"ack":true}}`)
			os.Exit(0)
		}
	}
}

func buildWASMTestBot(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-bot.wasm")
	buildCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	cmd := exec.CommandContext(buildCtx, "go", "build", "-o", outputPath, "./internal/platform/runtime/testdata/wasmtestbot")
	cmd.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("abs project root: %v", err)
	}
	cmd.Dir = projectRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build wasm test bot: %v\n%s", err, output)
	}
	return outputPath
}
