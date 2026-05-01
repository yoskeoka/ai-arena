package runtime

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"
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
			return
		}
	}
}

func helperCommand(_ string) []string {
	return []string{os.Args[0], "-test.run=TestHelperProcess"}
}
