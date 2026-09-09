package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestAcquireWorkerWithRetryRetriesQueueOwnershipConflict(t *testing.T) {
	guard := &sequenceWorkerGuard{errs: []error{ErrWorkerQueueOwned, nil}}

	release, err := acquireWorkerWithRetry(context.Background(), guard, "worker-one", time.Millisecond, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("acquireWorkerWithRetry() error = %v", err)
	}
	if guard.calls != 2 {
		t.Fatalf("AcquireWorker() calls = %d, want 2", guard.calls)
	}
	if release == nil {
		t.Fatal("acquireWorkerWithRetry() release = nil")
	}
	release()
	if guard.releases != 1 {
		t.Fatalf("release calls = %d, want 1", guard.releases)
	}
}

func TestAcquireWorkerWithRetryFailsFastForNonOwnershipError(t *testing.T) {
	want := errors.New("database unavailable")
	guard := &sequenceWorkerGuard{errs: []error{want}}

	_, err := acquireWorkerWithRetry(context.Background(), guard, "worker-one", time.Millisecond, 100*time.Millisecond)
	if !errors.Is(err, want) {
		t.Fatalf("acquireWorkerWithRetry() error = %v, want %v", err, want)
	}
	if guard.calls != 1 {
		t.Fatalf("AcquireWorker() calls = %d, want 1", guard.calls)
	}
}

func TestAcquireWorkerWithRetryTimesOutAndStopsOnCancellation(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		guard := &sequenceWorkerGuard{errs: []error{ErrWorkerQueueOwned}, repeatLast: true}
		_, err := acquireWorkerWithRetry(context.Background(), guard, "worker-one", time.Millisecond, 5*time.Millisecond)
		if !errors.Is(err, ErrWorkerOwnershipTimeout) {
			t.Fatalf("acquireWorkerWithRetry() error = %v, want %v", err, ErrWorkerOwnershipTimeout)
		}
		if guard.calls < 2 {
			t.Fatalf("AcquireWorker() calls = %d, want retry before timeout", guard.calls)
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		guard := &sequenceWorkerGuard{errs: []error{ErrWorkerQueueOwned}, started: make(chan struct{}, 1)}
		done := make(chan error, 1)
		go func() {
			_, err := acquireWorkerWithRetry(ctx, guard, "worker-one", time.Hour, time.Hour)
			done <- err
		}()
		select {
		case <-guard.started:
		case <-time.After(time.Second):
			t.Fatal("AcquireWorker() was not called")
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("acquireWorkerWithRetry() error = %v, want context cancellation", err)
		}
	})
}

func TestWorkerLoopDoesNotMutateQueueBeforeOwnership(t *testing.T) {
	queue := &workerLoopQueueStore{
		InMemoryQueueStore: NewInMemoryQueueStore(),
		acquireStarted:     make(chan struct{}, 1),
	}
	worker, err := NewWorker(queue, stubRunnerInvoker{}, stubTerminalPersister{})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}
	loop, err := NewWorkerLoop(worker, "worker-one", time.Millisecond, nil)
	if err != nil {
		t.Fatalf("NewWorkerLoop() error = %v", err)
	}
	loop.ownershipRetryInterval = time.Hour
	loop.ownershipMaximumWait = time.Hour

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- loop.Run(ctx) }()
	select {
	case <-queue.acquireStarted:
	case <-time.After(time.Second):
		t.Fatal("worker ownership was not attempted")
	}
	if got := queue.recoverCalls.Load(); got != 0 {
		t.Fatalf("RecoverExpired() calls while ownership pending = %d, want 0", got)
	}
	if got := queue.claimCalls.Load(); got != 0 {
		t.Fatalf("Claim() calls while ownership pending = %d, want 0", got)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run() error after cancellation = %v", err)
	}
}

func TestWorkerLoopDefaultsToRenderHandoffDurations(t *testing.T) {
	worker, err := NewWorker(NewInMemoryQueueStore(), stubRunnerInvoker{}, stubTerminalPersister{})
	if err != nil {
		t.Fatalf("NewWorker() error = %v", err)
	}
	loop, err := NewWorkerLoop(worker, "worker-one", time.Second, nil)
	if err != nil {
		t.Fatalf("NewWorkerLoop() error = %v", err)
	}
	if loop.ownershipRetryInterval != 10*time.Second {
		t.Fatalf("ownership retry interval = %s, want 10s", loop.ownershipRetryInterval)
	}
	if loop.ownershipMaximumWait != 5*time.Minute {
		t.Fatalf("ownership maximum wait = %s, want 5m", loop.ownershipMaximumWait)
	}
}

type sequenceWorkerGuard struct {
	errs       []error
	calls      int
	releases   int
	started    chan struct{}
	repeatLast bool
}

func (g *sequenceWorkerGuard) AcquireWorker(context.Context, string) (func(), error) {
	g.calls++
	if g.started != nil {
		select {
		case g.started <- struct{}{}:
		default:
		}
	}
	index := g.calls - 1
	if index < len(g.errs) {
		if g.errs[index] != nil {
			return nil, g.errs[index]
		}
	} else if g.repeatLast && len(g.errs) > 0 && g.errs[len(g.errs)-1] != nil {
		return nil, g.errs[len(g.errs)-1]
	}
	return func() { g.releases++ }, nil
}

type workerLoopQueueStore struct {
	*InMemoryQueueStore
	acquireStarted chan struct{}
	recoverCalls   atomic.Int32
	claimCalls     atomic.Int32
}

func (s *workerLoopQueueStore) AcquireWorker(context.Context, string) (func(), error) {
	select {
	case s.acquireStarted <- struct{}{}:
	default:
	}
	return nil, ErrWorkerQueueOwned
}

func (s *workerLoopQueueStore) RecoverExpired(ctx context.Context, now time.Time) (int, error) {
	s.recoverCalls.Add(1)
	return s.InMemoryQueueStore.RecoverExpired(ctx, now)
}

func (s *workerLoopQueueStore) Claim(ctx context.Context, workerID string) (QueueRecord, error) {
	s.claimCalls.Add(1)
	return s.InMemoryQueueStore.Claim(ctx, workerID)
}
