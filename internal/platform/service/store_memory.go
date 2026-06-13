package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrQueueRecordNotFound reports that no queue record exists for the requested submission id.
	ErrQueueRecordNotFound = errors.New("service: queue record not found")
	// ErrNoQueuedSubmission reports that no queued record is available to claim.
	ErrNoQueuedSubmission = errors.New("service: no queued submission available")
)

// InMemoryQueueStore keeps queue state inside one process for the initial service skeleton.
type InMemoryQueueStore struct {
	mu      sync.Mutex
	all     []string
	order   []string
	records map[string]QueueRecord
}

// NewInMemoryQueueStore constructs the initial replaceable queue backend.
func NewInMemoryQueueStore() *InMemoryQueueStore {
	return &InMemoryQueueStore{
		records: make(map[string]QueueRecord),
	}
}

// Enqueue appends one admitted submission in queued state.
func (s *InMemoryQueueStore) Enqueue(_ context.Context, submission MatchSubmission) (QueueRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(submission.RunID) == "" {
		return QueueRecord{}, fmt.Errorf("service: run_id is required")
	}
	if _, exists := s.records[submission.RunID]; exists {
		return QueueRecord{}, fmt.Errorf("service: run_id %q already exists", submission.RunID)
	}
	record := QueueRecord{
		Submission: cloneMatchSubmission(submission),
		State:      StateQueued,
	}
	s.records[submission.RunID] = record
	s.all = append(s.all, submission.RunID)
	s.order = append(s.order, submission.RunID)
	return cloneQueueRecord(record), nil
}

// Claim moves the next queued record to leased for the supplied worker id.
func (s *InMemoryQueueStore) Claim(_ context.Context, workerID string) (QueueRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(workerID) == "" {
		return QueueRecord{}, fmt.Errorf("service: worker_id is required")
	}
	for len(s.order) > 0 {
		runID := s.order[0]
		s.order = s.order[1:]

		record, ok := s.records[runID]
		if !ok || record.State != StateQueued {
			continue
		}
		record.State = StateLeased
		record.Lease = &WorkerLease{WorkerID: workerID}
		s.records[runID] = record
		return cloneQueueRecord(record), nil
	}
	return QueueRecord{}, ErrNoQueuedSubmission
}

// Update replaces one existing record after validating the lifecycle transition.
func (s *InMemoryQueueStore) Update(_ context.Context, next QueueRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.records[next.Submission.RunID]
	if !ok {
		return ErrQueueRecordNotFound
	}
	if err := ValidateTransition(current.State, next.State); err != nil {
		return err
	}
	s.records[next.Submission.RunID] = cloneQueueRecord(next)
	return nil
}

// CancelQueued moves one queued record into canceled.
func (s *InMemoryQueueStore) CancelQueued(_ context.Context, runID string) (QueueRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[runID]
	if !ok {
		return QueueRecord{}, ErrQueueRecordNotFound
	}
	if record.State != StateQueued {
		return QueueRecord{}, fmt.Errorf("service: only queued submissions can be canceled")
	}
	if err := ValidateTransition(record.State, StateCanceled); err != nil {
		return QueueRecord{}, err
	}
	record.State = StateCanceled
	record.Lease = nil
	record.Terminal = nil
	s.records[runID] = record
	s.removeFromOrder(runID)
	return cloneQueueRecord(record), nil
}

// Get returns one existing queue record by run id.
func (s *InMemoryQueueStore) Get(_ context.Context, runID string) (QueueRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.records[runID]
	if !ok {
		return QueueRecord{}, ErrQueueRecordNotFound
	}
	return cloneQueueRecord(record), nil
}

// List returns queue records in submission insertion order.
func (s *InMemoryQueueStore) List(_ context.Context) ([]QueueRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records := make([]QueueRecord, 0, len(s.all))
	for _, submissionID := range s.all {
		record, ok := s.records[submissionID]
		if !ok {
			continue
		}
		records = append(records, cloneQueueRecord(record))
	}
	return records, nil
}

func (s *InMemoryQueueStore) removeFromOrder(submissionID string) {
	filtered := s.order[:0]
	for _, queuedID := range s.order {
		if queuedID == submissionID {
			continue
		}
		filtered = append(filtered, queuedID)
	}
	s.order = filtered
}

func cloneQueueRecord(record QueueRecord) QueueRecord {
	record.Submission = cloneMatchSubmission(record.Submission)
	if record.Lease != nil {
		lease := *record.Lease
		record.Lease = &lease
	}
	if record.Terminal != nil {
		terminal := *record.Terminal
		if terminal.PlayerStderrPaths != nil {
			terminal.PlayerStderrPaths = make(map[string]string, len(record.Terminal.PlayerStderrPaths))
			for playerID, path := range record.Terminal.PlayerStderrPaths {
				terminal.PlayerStderrPaths[playerID] = path
			}
		}
		record.Terminal = &terminal
	}
	return record
}

func cloneMatchSubmission(submission MatchSubmission) MatchSubmission {
	submission.Players = append([]SubmittedPlayer(nil), submission.Players...)
	return submission
}
