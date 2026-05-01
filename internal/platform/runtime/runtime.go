package runtime

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

type Config struct {
	Command          []string
	Dir              string
	Env              []string
	StderrLimitBytes int
}

type Message struct {
	Response *protocol.Response
	Err      error
}

type StderrSnapshot struct {
	Output    string
	BytesRead int
	Truncated bool
}

type Adapter struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	incoming chan Message
	done     chan error
	stderr   *captureBuffer
}

func Start(ctx context.Context, cfg Config) (*Adapter, error) {
	if len(cfg.Command) == 0 {
		return nil, errors.New("runtime: command is required")
	}

	cmd := exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	cmd.Dir = cfg.Dir
	cmd.Env = append(os.Environ(), cfg.Env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	stderr := newCaptureBuffer(cfg.StderrLimitBytes)
	adapter := &Adapter{
		cmd:      cmd,
		stdin:    stdin,
		incoming: make(chan Message, 16),
		done:     make(chan error, 1),
		stderr:   stderr,
	}

	go func() {
		_, _ = io.Copy(stderr, stderrPipe)
	}()

	go adapter.readStdout(stdout)
	go func() {
		adapter.done <- cmd.Wait()
		close(adapter.done)
		close(adapter.incoming)
	}()

	return adapter, nil
}

func (a *Adapter) Send(req protocol.Request) error {
	return protocol.NewEncoder(a.stdin).Encode(req)
}

func (a *Adapter) Incoming() <-chan Message {
	return a.incoming
}

func (a *Adapter) StderrSnapshot() StderrSnapshot {
	return a.stderr.snapshot()
}

func (a *Adapter) Close(ctx context.Context) error {
	if a.cmd.Process == nil {
		return nil
	}

	select {
	case err := <-a.done:
		return normalizeExit(err)
	default:
	}

	_ = a.stdin.Close()
	_ = a.cmd.Process.Signal(os.Interrupt)

	select {
	case err := <-a.done:
		return normalizeExit(err)
	case <-ctx.Done():
		_ = a.cmd.Process.Signal(syscall.SIGKILL)
		err := <-a.done
		if err == nil {
			return ctx.Err()
		}
		return errors.Join(ctx.Err(), normalizeExit(err))
	}
}

func (a *Adapter) readStdout(stdout io.Reader) {
	dec := protocol.NewDecoder(stdout)
	for {
		resp, err := dec.DecodeResponse()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			a.incoming <- Message{Err: err}
			continue
		}
		respCopy := resp
		a.incoming <- Message{Response: &respCopy}
	}
}

func normalizeExit(err error) error {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil
	}
	return err
}

type captureBuffer struct {
	limit     int
	totalRead int
	truncated bool
	mu        sync.Mutex
	buf       []byte
}

func newCaptureBuffer(limit int) *captureBuffer {
	return &captureBuffer{limit: limit}
}

func (b *captureBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.totalRead += len(p)
	if b.limit <= 0 {
		b.buf = append(b.buf, p...)
		return len(p), nil
	}

	remaining := b.limit - len(b.buf)
	if remaining > 0 {
		if len(p) <= remaining {
			b.buf = append(b.buf, p...)
		} else {
			b.buf = append(b.buf, p[:remaining]...)
			b.truncated = true
		}
	} else {
		b.truncated = true
	}

	return len(p), nil
}

func (b *captureBuffer) snapshot() StderrSnapshot {
	b.mu.Lock()
	defer b.mu.Unlock()

	return StderrSnapshot{
		Output:    string(b.buf),
		BytesRead: b.totalRead,
		Truncated: b.truncated,
	}
}
