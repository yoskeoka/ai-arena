package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CommandService orchestrates submission admission and queued-only cancellation.
type CommandService struct {
	queue     QueueStore
	validator AdmissionValidator
}

// NewCommandService constructs the service command layer for 0049.
func NewCommandService(queue QueueStore, validator AdmissionValidator) (*CommandService, error) {
	if queue == nil {
		return nil, fmt.Errorf("service: queue store is required")
	}
	if validator == nil {
		return nil, fmt.Errorf("service: admission validator is required")
	}
	return &CommandService{
		queue:     queue,
		validator: validator,
	}, nil
}

// Submit validates a match submission and enqueues it when admission passes.
func (s *CommandService) Submit(ctx context.Context, submission MatchSubmission) (QueueRecord, error) {
	if err := s.validator.Validate(ctx, submission); err != nil {
		return QueueRecord{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	record, err := s.queue.Enqueue(ctx, submission)
	if err == nil {
		return record, nil
	}
	if strings.Contains(err.Error(), "already exists") {
		return QueueRecord{}, fmt.Errorf("%w: %w", ErrConflict, err)
	}
	if errors.Is(err, ErrBadRequest) || errors.Is(err, ErrConflict) {
		return QueueRecord{}, err
	}
	return QueueRecord{}, err
}

// Cancel transitions one queued submission into canceled.
func (s *CommandService) Cancel(ctx context.Context, submissionID string) (QueueRecord, error) {
	return s.queue.CancelQueued(ctx, submissionID)
}
