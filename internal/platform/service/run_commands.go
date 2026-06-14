package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// RunCommandService orchestrates retry, rerun, and official-run correction.
type RunCommandService struct {
	commands *CommandService
	queue    QueueStore
	rankings *RankingService
	newRunID func() string
}

// NewRunCommandService constructs the run follow-up command layer.
func NewRunCommandService(commands *CommandService, queue QueueStore, rankings *RankingService) (*RunCommandService, error) {
	if commands == nil {
		return nil, fmt.Errorf("service: command service is required")
	}
	if queue == nil {
		return nil, fmt.Errorf("service: queue store is required")
	}
	return &RunCommandService{
		commands: commands,
		queue:    queue,
		rankings: rankings,
		newRunID: func() string { return "run-" + uuid.NewString() },
	}, nil
}

// Retry appends one new run from a failed run.
func (s *RunCommandService) Retry(ctx context.Context, runID string) (QueueRecord, error) {
	record, err := s.queue.Get(ctx, strings.TrimSpace(runID))
	if err != nil {
		return QueueRecord{}, err
	}
	if record.State != StateFailed {
		return QueueRecord{}, fmt.Errorf("%w: service: only failed runs can be retried", ErrConflict)
	}
	next := cloneMatchSubmission(record.Submission)
	next.RunID = s.newRunID()
	next.AttemptCount++
	next.ParentRunID = record.Submission.RunID
	next.RunKind = RunKindRetry
	next.Official = false
	return s.commands.Submit(ctx, next)
}

// Rerun appends one new candidate run from a completed run.
func (s *RunCommandService) Rerun(ctx context.Context, runID string) (QueueRecord, error) {
	record, err := s.queue.Get(ctx, strings.TrimSpace(runID))
	if err != nil {
		return QueueRecord{}, err
	}
	if record.State != StateCompleted {
		return QueueRecord{}, fmt.Errorf("%w: service: only completed runs can be rerun", ErrConflict)
	}
	next := cloneMatchSubmission(record.Submission)
	next.RunID = s.newRunID()
	next.AttemptCount++
	next.ParentRunID = record.Submission.RunID
	next.RunKind = RunKindRerun
	next.Official = false
	return s.commands.Submit(ctx, next)
}

// Promote marks one completed run as the official run for its logical match.
func (s *RunCommandService) Promote(ctx context.Context, runID string) (QueueRecord, error) {
	target, err := s.queue.Get(ctx, strings.TrimSpace(runID))
	if err != nil {
		return QueueRecord{}, err
	}
	if target.State != StateCompleted {
		return QueueRecord{}, fmt.Errorf("%w: service: only completed runs can be promoted", ErrConflict)
	}

	records, err := s.queue.List(ctx)
	if err != nil {
		return QueueRecord{}, err
	}
	for _, record := range records {
		if record.Submission.MatchID != target.Submission.MatchID {
			continue
		}
		next := cloneQueueRecord(record)
		next.Submission.Official = record.Submission.RunID == target.Submission.RunID
		if next.Submission.Official == record.Submission.Official {
			continue
		}
		if err := s.queue.Update(ctx, next); err != nil {
			return QueueRecord{}, err
		}
		if next.Submission.RunID == target.Submission.RunID {
			target = next
		}
	}

	if s.rankings != nil {
		if err := s.rankings.RefreshCompletedRun(ctx, target); err != nil {
			return QueueRecord{}, err
		}
	}
	return cloneQueueRecord(target), nil
}
