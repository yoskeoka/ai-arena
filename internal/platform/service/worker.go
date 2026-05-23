package service

import (
	"context"
	"fmt"

	"github.com/yoskeoka/ai-arena/internal/platform/artifacts"
)

// Worker claims queued submissions, invokes the runner once, and persists terminal artifacts.
type Worker struct {
	queue     QueueStore
	runner    RunnerInvoker
	persister TerminalPersister
}

// NewWorker constructs the initial single-record worker orchestration.
func NewWorker(queue QueueStore, runner RunnerInvoker, persister TerminalPersister) (*Worker, error) {
	if queue == nil {
		return nil, fmt.Errorf("service: queue store is required")
	}
	if runner == nil {
		return nil, fmt.Errorf("service: runner invoker is required")
	}
	if persister == nil {
		return nil, fmt.Errorf("service: terminal persister is required")
	}
	return &Worker{
		queue:     queue,
		runner:    runner,
		persister: persister,
	}, nil
}

// ProcessNext claims the next queued submission for one worker and drives it to a terminal queue state.
func (w *Worker) ProcessNext(ctx context.Context, workerID string) (QueueRecord, error) {
	record, err := w.queue.Claim(ctx, workerID)
	if err != nil {
		return QueueRecord{}, err
	}

	record.State = StateRunning
	if err := w.queue.Update(ctx, record); err != nil {
		return QueueRecord{}, err
	}

	result, runErr := w.runner.Run(ctx, ExecutionRequest{Submission: record.Submission})
	if result.Record.MatchID == "" {
		record.State = StateFailed
		if updateErr := w.queue.Update(ctx, record); updateErr != nil {
			return QueueRecord{}, updateErr
		}
		if runErr != nil {
			return cloneQueueRecord(record), runErr
		}
		return cloneQueueRecord(record), fmt.Errorf("service: runner returned no terminal record")
	}

	record.State = StatePersisting
	if err := w.queue.Update(ctx, record); err != nil {
		return QueueRecord{}, err
	}

	terminal, err := w.persister.Persist(ctx, record.Submission, result)
	if err != nil {
		record.State = StateFailed
		if updateErr := w.queue.Update(ctx, record); updateErr != nil {
			return QueueRecord{}, updateErr
		}
		return cloneQueueRecord(record), err
	}
	terminal.MatchStatus = result.Record.Status
	terminal.Error = artifacts.TerminalError(result.Record)

	record.State = StateCompleted
	record.Terminal = &terminal
	if err := w.queue.Update(ctx, record); err != nil {
		return QueueRecord{}, err
	}
	if runErr != nil {
		return cloneQueueRecord(record), runErr
	}
	return cloneQueueRecord(record), nil
}
