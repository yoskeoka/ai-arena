package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// WorkerLoop keeps polling the queue and processes submissions one-by-one.
type WorkerLoop struct {
	worker       *Worker
	workerID     string
	pollInterval time.Duration
	onError      func(error)
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
		worker:       worker,
		workerID:     workerID,
		pollInterval: pollInterval,
		onError:      onError,
	}, nil
}

// Run keeps processing queued submissions until the context is canceled.
func (l *WorkerLoop) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}

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
