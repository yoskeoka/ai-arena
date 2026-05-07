package runtime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"testing"
	"time"

	wazerosys "github.com/tetratelabs/wazero/sys"
)

func TestStartStreamsAndCapturesStderr(t *testing.T) {
	adapter, err := Start(context.Background(), Config{
		Command:          helperCommand("runtime-echo"),
		Env:              []string{"GO_WANT_HELPER_PROCESS=runtime-echo"},
		StderrLimitBytes: 1024,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Close(closeCtx)
	}()

	msg := <-adapter.Incoming()
	if msg.Err != nil {
		t.Fatalf("incoming err: %v", msg.Err)
	}
	if msg.Response == nil || msg.Response.ID != "boot" {
		t.Fatalf("unexpected boot response: %+v", msg.Response)
	}

	deadline := time.Now().Add(time.Second)
	snapshot := adapter.StderrSnapshot()
	for !strings.Contains(snapshot.Output, "runtime stderr") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		snapshot = adapter.StderrSnapshot()
	}
	if !strings.Contains(snapshot.Output, "runtime stderr") {
		t.Fatalf("stderr output = %q, want runtime stderr", snapshot.Output)
	}
}

func TestCloseKillsProcessWhenShutdownDeadlineExpires(t *testing.T) {
	adapter, err := Start(context.Background(), Config{
		Command: []string{"/bin/sh", "-c", "tail -f /dev/null"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	closeCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := adapter.Close(closeCtx); err == nil {
		t.Fatal("Close returned nil, want timeout-related error")
	}
}

func TestCloseReturnsCrashExitWhenProcessAlreadyFailed(t *testing.T) {
	adapter, err := Start(context.Background(), Config{
		Command: []string{"/bin/sh", "-c", "exit 2"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := adapter.Close(closeCtx); err == nil {
		t.Fatal("Close returned nil, want crash exit error")
	}
}

func TestStartWASMWASIStreamsAndCapturesStderr(t *testing.T) {
	modulePath := buildWASMTestBot(t)

	adapter, err := Start(context.Background(), Config{
		Kind:             KindWASMWASI,
		ModulePath:       modulePath,
		Env:              []string{"BOT_MODE=boot-response"},
		StderrLimitBytes: 1024,
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Close(closeCtx)
	}()

	msg := <-adapter.Incoming()
	if msg.Err != nil {
		t.Fatalf("incoming err: %v", msg.Err)
	}
	if msg.Response == nil || msg.Response.ID != "boot" {
		t.Fatalf("unexpected boot response: %+v", msg.Response)
	}

	deadline := time.Now().Add(time.Second)
	snapshot := adapter.StderrSnapshot()
	for !strings.Contains(snapshot.Output, "runtime stderr") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
		snapshot = adapter.StderrSnapshot()
	}
	if !strings.Contains(snapshot.Output, "runtime stderr") {
		t.Fatalf("stderr output = %q, want runtime stderr", snapshot.Output)
	}
}

func TestStartWASMWASIReturnsMalformedOutput(t *testing.T) {
	modulePath := buildWASMTestBot(t)

	adapter, err := Start(context.Background(), Config{
		Kind:       KindWASMWASI,
		ModulePath: modulePath,
		Env:        []string{"BOT_MODE=bad-json"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = adapter.Close(closeCtx)
	}()

	msg := <-adapter.Incoming()
	if msg.Err == nil {
		t.Fatal("incoming err = nil, want malformed output error")
	}
}

func TestEffectiveWASMMemoryLimitPagesDefaultsWhenUnset(t *testing.T) {
	if got := effectiveWASMMemoryLimitPages(Config{}); got != DefaultWASMMemoryLimitPages {
		t.Fatalf("effectiveWASMMemoryLimitPages() = %d, want %d", got, DefaultWASMMemoryLimitPages)
	}
	if got := effectiveWASMMemoryLimitPages(Config{MemoryLimitPages: 12}); got != 12 {
		t.Fatalf("effectiveWASMMemoryLimitPages() = %d, want 12", got)
	}
}

func TestWASMWASINormalizeExitSuppressesCleanExit(t *testing.T) {
	adapter := &wasmWASIAdapter{}
	if err := adapter.normalizeExit(wazerosys.NewExitError(0)); err != nil {
		t.Fatalf("normalizeExit(clean) = %v, want nil", err)
	}
}

func TestWASMWASINormalizeExitSuppressesExpectedShutdownCancellation(t *testing.T) {
	adapter := &wasmWASIAdapter{shutdownExpected: true}
	if err := adapter.normalizeExit(context.Canceled); err != nil {
		t.Fatalf("normalizeExit(context.Canceled) = %v, want nil", err)
	}
	if err := adapter.normalizeExit(wazerosys.NewExitError(wazerosys.ExitCodeContextCanceled)); err != nil {
		t.Fatalf("normalizeExit(exit canceled) = %v, want nil", err)
	}
}

func TestWASMWASINormalizeExitKeepsUnexpectedExit(t *testing.T) {
	adapter := &wasmWASIAdapter{}
	err := adapter.normalizeExit(wazerosys.NewExitError(2))
	if !errors.Is(err, wazerosys.NewExitError(2)) {
		t.Fatalf("normalizeExit(unexpected) = %v, want exit error", err)
	}
}

func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("GO_WANT_HELPER_PROCESS")
	if mode == "" {
		return
	}

	switch mode {
	case "runtime-echo":
		fmt.Fprintln(os.Stderr, "runtime stderr")
		fmt.Println(`{"jsonrpc":"2.0","id":"boot","result":{"ready":true}}`)
	case "runtime-hang":
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt)
		<-ch
		time.Sleep(time.Second)
	case "session-bot":
		runSessionBot()
	default:
		panic("unknown helper mode: " + mode)
	}

	os.Exit(0)
}

func runSessionBot() {
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
			return
		}
	}
}

func helperCommand(_ string) []string {
	return []string{os.Args[0], "-test.run=TestHelperProcess"}
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
