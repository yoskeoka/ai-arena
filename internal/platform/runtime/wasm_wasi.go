package runtime

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	wazerosys "github.com/tetratelabs/wazero/sys"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

type wasmWASIAdapter struct {
	stdin            *io.PipeWriter
	stdinReader      *io.PipeReader
	stdout           *io.PipeReader
	stdoutWriter     *io.PipeWriter
	incoming         chan Message
	done             chan error
	stderr           *captureBuffer
	cancel           context.CancelFunc
	closeRuntime     func()
	mu               sync.Mutex
	shutdownExpected bool
}

func startWASMWASI(parent context.Context, cfg Config) (*wasmWASIAdapter, error) {
	if cfg.ModulePath == "" {
		return nil, errors.New("runtime: module path is required")
	}

	wasmBytes, err := os.ReadFile(resolvePath(cfg.Dir, cfg.ModulePath))
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(parent)
	runtimeCfg := wazero.NewRuntimeConfig().WithCloseOnContextDone(true)
	runtimeCfg = runtimeCfg.WithMemoryLimitPages(effectiveWASMMemoryLimitPages(cfg))

	rt := wazero.NewRuntimeWithConfig(ctx, runtimeCfg)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		cancel()
		_ = rt.Close(ctx)
		return nil, err
	}
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		cancel()
		_ = rt.Close(ctx)
		return nil, err
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderr := newCaptureBuffer(cfg.StderrLimitBytes)
	adapter := &wasmWASIAdapter{
		stdin:        stdinWriter,
		stdinReader:  stdinReader,
		stdout:       stdoutReader,
		stdoutWriter: stdoutWriter,
		incoming:     make(chan Message, 16),
		done:         make(chan error, 1),
		stderr:       stderr,
		cancel:       cancel,
		closeRuntime: func() { _ = rt.Close(ctx) },
	}

	moduleCfg := wazero.NewModuleConfig().
		WithStdin(stdinReader).
		WithStdout(stdoutWriter).
		WithStderr(stderr).
		WithSysNanotime().
		WithSysWalltime().
		WithArgs(defaultWASMArgs(cfg)...)
	for _, envKV := range cfg.Env {
		key, value, ok := splitEnv(envKV)
		if !ok {
			cancel()
			adapter.closeRuntime()
			return nil, errors.New("runtime: invalid env entry")
		}
		moduleCfg = moduleCfg.WithEnv(key, value)
	}

	stdoutDone := make(chan struct{})
	go readStdout(stdoutReader, adapter.incoming, stdoutDone)
	go func() {
		_, err := rt.InstantiateModule(ctx, compiled, moduleCfg)
		_ = stdoutWriter.Close()
		_ = stdinReader.Close()
		<-stdoutDone
		adapter.done <- adapter.normalizeExit(err)
		close(adapter.done)
		close(adapter.incoming)
		adapter.closeRuntime()
	}()

	return adapter, nil
}

func (a *wasmWASIAdapter) Send(req protocol.Request) error {
	return protocol.NewEncoder(a.stdin).Encode(req)
}

func (a *wasmWASIAdapter) Incoming() <-chan Message {
	return a.incoming
}

func (a *wasmWASIAdapter) StderrSnapshot() StderrSnapshot {
	return a.stderr.snapshot()
}

func (a *wasmWASIAdapter) Close(ctx context.Context) error {
	select {
	case err := <-a.done:
		return err
	default:
	}

	a.mu.Lock()
	a.shutdownExpected = true
	a.mu.Unlock()

	_ = a.stdin.Close()

	select {
	case err := <-a.done:
		return err
	case <-time.After(stdinCloseGracePeriod):
	}

	a.cancel()

	select {
	case err := <-a.done:
		return err
	case <-ctx.Done():
		a.cancel()
		return ctx.Err()
	}
}

func (a *wasmWASIAdapter) normalizeExit(err error) error {
	if err == nil {
		return nil
	}

	var exitErr *wazerosys.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 0 {
		return nil
	}

	a.mu.Lock()
	shutdownExpected := a.shutdownExpected
	a.mu.Unlock()
	if shutdownExpected {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if errors.As(err, &exitErr) && exitErr.ExitCode() == wazerosys.ExitCodeContextCanceled {
			return nil
		}
	}
	return err
}

func effectiveWASMMemoryLimitPages(cfg Config) uint32 {
	if cfg.MemoryLimitPages > 0 {
		return cfg.MemoryLimitPages
	}
	return DefaultWASMMemoryLimitPages
}

func defaultWASMArgs(cfg Config) []string {
	if len(cfg.Args) > 0 {
		return append([]string(nil), cfg.Args...)
	}
	return []string{cfg.ModulePath}
}

func splitEnv(entry string) (string, string, bool) {
	for i := 0; i < len(entry); i++ {
		if entry[i] == '=' {
			return entry[:i], entry[i+1:], entry[:i] != ""
		}
	}
	return "", "", false
}

func resolvePath(dir, path string) string {
	if filepath.IsAbs(path) || dir == "" {
		return path
	}
	return filepath.Join(dir, path)
}

func readStdout(stdout io.Reader, incoming chan<- Message, done chan<- struct{}) {
	defer close(done)

	dec := protocol.NewDecoder(stdout)
	for {
		resp, err := dec.DecodeResponse()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrClosed) {
				return
			}
			incoming <- Message{Err: err}
			return
		}
		respCopy := resp
		incoming <- Message{Response: &respCopy}
	}
}
