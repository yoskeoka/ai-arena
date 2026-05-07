package runtime

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

type Kind string

const (
	KindLocalSubprocess Kind = "local-subprocess"
	KindWASMWASI        Kind = "wasm-wasi"

	DefaultWASMMemoryLimitPages uint32 = 64
)

type Config struct {
	Kind             Kind
	Command          []string
	ModulePath       string
	Args             []string
	Dir              string
	Env              []string
	StderrLimitBytes int
	MemoryLimitPages uint32
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

type adapterImpl interface {
	Send(protocol.Request) error
	Incoming() <-chan Message
	StderrSnapshot() StderrSnapshot
	Close(context.Context) error
}

type Adapter struct {
	impl adapterImpl
}

const stdinCloseGracePeriod = 50 * time.Millisecond

func Start(ctx context.Context, cfg Config) (*Adapter, error) {
	switch cfg.Kind {
	case "", KindLocalSubprocess:
		impl, err := startLocalSubprocess(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return &Adapter{impl: impl}, nil
	case KindWASMWASI:
		impl, err := startWASMWASI(ctx, cfg)
		if err != nil {
			return nil, err
		}
		return &Adapter{impl: impl}, nil
	default:
		return nil, errors.New("runtime: unsupported kind")
	}
}

func (a *Adapter) Send(req protocol.Request) error {
	return a.impl.Send(req)
}

func (a *Adapter) Incoming() <-chan Message {
	return a.impl.Incoming()
}

func (a *Adapter) StderrSnapshot() StderrSnapshot {
	return a.impl.StderrSnapshot()
}

func (a *Adapter) Close(ctx context.Context) error {
	return a.impl.Close(ctx)
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
