package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

const (
	workerOwnershipRetryInterval = 10 * time.Second
	// Keep this at least 60 seconds plus Render's configured maxShutdownDelaySeconds plus a 30-second buffer. If the Render setting changes from its 30-second default, update this value and the release readiness timeout together.
	workerOwnershipMaximumWait = 5 * time.Minute
)

// WorkerLoop keeps polling the queue and processes submissions one-by-one.
type WorkerLoop struct {
	worker                 *Worker
	workerID               string
	pollInterval           time.Duration
	ownershipRetryInterval time.Duration
	ownershipMaximumWait   time.Duration
	onError                func(error)
	ready                  atomic.Bool
}

// NewWorkerLoop constructs one in-process queue poller.
func NewWorkerLoop(worker *Worker, workerID string, pollInterval time.Duration, onError func(error)) (*WorkerLoop, error) {
	if worker == nil {
		return nil, fmt.Errorf("service: worker is required")
	}
	if strings.TrimSpace(workerID) == "" {
		return nil, fmt.Errorf("service: worker_id is required")
	}
	if pollInterval <= 0 {
		return nil, fmt.Errorf("service: poll interval must be positive")
	}
	return &WorkerLoop{
		worker:                 worker,
		workerID:               workerID,
		pollInterval:           pollInterval,
		ownershipRetryInterval: workerOwnershipRetryInterval,
		ownershipMaximumWait:   workerOwnershipMaximumWait,
		onError:                onError,
	}, nil
}

// Ready reports whether the worker owns the queue and completed its initial recovery.
func (l *WorkerLoop) Ready() bool {
	return l.ready.Load()
}

// Run keeps processing queued submissions until the context is canceled.
func (l *WorkerLoop) Run(ctx context.Context) error {
	l.ready.Store(false)
	defer l.ready.Store(false)

	if guard, ok := l.worker.queue.(workerProcessGuard); ok {
		release, err := acquireWorkerWithRetry(ctx, guard, l.workerID, l.ownershipRetryInterval, l.ownershipMaximumWait)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		defer release()
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
		l.ready.Store(false)
		if _, err := l.worker.queue.RecoverExpired(ctx, time.Now().UTC()); err != nil {
			if l.onError != nil {
				l.onError(fmt.Errorf("recover expired leases: %w", err))
			}
			timer.Reset(l.pollInterval)
			continue
		}
		l.ready.Store(true)

		_, err := l.worker.ProcessNext(ctx, l.workerID)
		switch {
		case err == nil:
			timer.Reset(0)
		case ctx.Err() != nil:
			return nil
		case errors.Is(err, ErrNoQueuedSubmission):
			timer.Reset(l.pollInterval)
		default:
			if l.onError != nil {
				l.onError(err)
			}
			timer.Reset(l.pollInterval)
		}
	}
}

func acquireWorkerWithRetry(ctx context.Context, guard workerProcessGuard, workerID string, retryInterval, maximumWait time.Duration) (func(), error) {
	if retryInterval <= 0 {
		return nil, fmt.Errorf("service: worker ownership retry interval must be positive")
	}
	if maximumWait <= 0 {
		return nil, fmt.Errorf("service: worker ownership maximum wait must be positive")
	}

	deadline := time.NewTimer(maximumWait)
	defer deadline.Stop()

	for {
		release, err := guard.AcquireWorker(ctx, workerID)
		if err == nil {
			return release, nil
		}
		if !errors.Is(err, ErrWorkerQueueOwned) {
			return nil, err
		}

		retry := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			if !retry.Stop() {
				<-retry.C
			}
			return nil, ctx.Err()
		case <-deadline.C:
			if !retry.Stop() {
				<-retry.C
			}
			return nil, fmt.Errorf("service: worker ownership handoff exceeded %s: %w", maximumWait, ErrWorkerOwnershipTimeout)
		case <-retry.C:
		}
	}
}
