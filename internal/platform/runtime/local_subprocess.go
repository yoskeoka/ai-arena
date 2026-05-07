package runtime

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

type localSubprocessAdapter struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	incoming chan Message
	done     chan error
	stderr   *captureBuffer
}

func startLocalSubprocess(ctx context.Context, cfg Config) (*localSubprocessAdapter, error) {
	if len(cfg.Command) == 0 {
		return nil, errors.New("runtime: command is required")
	}

	// #nosec G204 -- the platform executes a pre-tokenized command array, not a shell string.
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
	adapter := &localSubprocessAdapter{
		cmd:      cmd,
		stdin:    stdin,
		incoming: make(chan Message, 16),
		done:     make(chan error, 1),
		stderr:   stderr,
	}

	go func() {
		_, _ = io.Copy(stderr, stderrPipe)
	}()

	stdoutDone := make(chan struct{})
	go readStdout(stdout, adapter.incoming, stdoutDone)
	go func() {
		err := cmd.Wait()
		<-stdoutDone
		adapter.done <- err
		close(adapter.done)
		close(adapter.incoming)
	}()

	return adapter, nil
}

func (a *localSubprocessAdapter) Send(req protocol.Request) error {
	return protocol.NewEncoder(a.stdin).Encode(req)
}

func (a *localSubprocessAdapter) Incoming() <-chan Message {
	return a.incoming
}

func (a *localSubprocessAdapter) StderrSnapshot() StderrSnapshot {
	return a.stderr.snapshot()
}

func (a *localSubprocessAdapter) Close(ctx context.Context) error {
	if a.cmd.Process == nil {
		return nil
	}

	select {
	case err := <-a.done:
		return err
	default:
	}

	_ = a.stdin.Close()

	select {
	case err := <-a.done:
		return err
	case <-time.After(stdinCloseGracePeriod):
	}

	_ = a.cmd.Process.Signal(os.Interrupt)

	select {
	case err := <-a.done:
		return suppressExpectedShutdownExit(err, os.Interrupt)
	case <-ctx.Done():
		_ = a.cmd.Process.Signal(syscall.SIGKILL)
		err := <-a.done
		if err == nil {
			return ctx.Err()
		}
		return errors.Join(ctx.Err(), suppressExpectedShutdownExit(err, syscall.SIGKILL))
	}
}

func suppressExpectedShutdownExit(err error, expectedSignal os.Signal) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ProcessState != nil && exitErr.ProcessState.Sys() != nil {
		if status, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() && status.Signal() == expectedSignal {
			return nil
		}
	}
	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}
	return err
}
