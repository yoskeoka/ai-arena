package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/match"
)

// PublicState is one atomically published exported snapshot for a selected run.
type PublicState struct {
	MatchID  string
	RunID    string
	Version  int64
	Snapshot game.ExportedSnapshot
}

// PublicStateStore persists the latest exported snapshot for each run.
type PublicStateStore interface {
	Publish(context.Context, string, string, game.ExportedSnapshot) (PublicState, error)
	Latest(context.Context, string, string) (PublicState, bool, error)
}

// InMemoryPublicStateStore is the local development and test public-state lane.
type InMemoryPublicStateStore struct {
	mu     sync.Mutex
	states map[string]PublicState
}

// NewInMemoryPublicStateStore constructs an empty atomic state publisher.
func NewInMemoryPublicStateStore() *InMemoryPublicStateStore {
	return &InMemoryPublicStateStore{states: make(map[string]PublicState)}
}

// Publish replaces a run's latest state and increments its scoped version.
func (s *InMemoryPublicStateStore) Publish(_ context.Context, matchID, runID string, snapshot game.ExportedSnapshot) (PublicState, error) {
	if matchID == "" || runID == "" {
		return PublicState{}, fmt.Errorf("service: public state match_id and run_id are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := matchID + "\x00" + runID
	state := PublicState{MatchID: matchID, RunID: runID, Version: s.states[key].Version + 1, Snapshot: cloneExportedSnapshot(snapshot)}
	state.Snapshot.MatchID = matchID
	s.states[key] = state
	return clonePublicState(state), nil
}

// Latest returns the newest atomically published state for one run.
func (s *InMemoryPublicStateStore) Latest(_ context.Context, matchID, runID string) (PublicState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.states[matchID+"\x00"+runID]
	return clonePublicState(state), ok, nil
}

func clonePublicState(state PublicState) PublicState {
	state.Snapshot = cloneExportedSnapshot(state.Snapshot)
	return state
}

func cloneExportedSnapshot(snapshot game.ExportedSnapshot) game.ExportedSnapshot {
	snapshot.PublicState = append(json.RawMessage(nil), snapshot.PublicState...)
	snapshot.Players = append([]game.ExportedPlayerSnapshot(nil), snapshot.Players...)
	return snapshot
}

// PublicStatePublisher adapts runner snapshot notifications to a state store.
type PublicStatePublisher struct {
	store PublicStateStore
	match string
	run   string
}

// NewPublicStatePublisher constructs a publisher bound to one logical run.
func NewPublicStatePublisher(store PublicStateStore, matchID, runID string) *PublicStatePublisher {
	return &PublicStatePublisher{store: store, match: matchID, run: runID}
}

// OnExportedSnapshot implements match.ExportedSnapshotObserver.
func (p *PublicStatePublisher) OnExportedSnapshot(snapshot game.ExportedSnapshot) {
	if p == nil || p.store == nil {
		return
	}
	_, _ = p.store.Publish(context.Background(), p.match, p.run, snapshot)
}

// OnEvent intentionally ignores private runner events.
func (*PublicStatePublisher) OnEvent(match.Event) {}

// OnRecordBuilt intentionally ignores the private terminal record.
func (*PublicStatePublisher) OnRecordBuilt(match.Record) {}

// PublicMatch is the anonymous response identity for a selected logical match.
type PublicMatch struct {
	MatchID        string                `json:"match_id"`
	SelectedRunID  string                `json:"selected_run_id"`
	Game           contract.GameMetadata `json:"game"`
	LifecycleState LifecycleState        `json:"lifecycle_state"`
}

// PublicReplayMetadata is safe terminal metadata for an opaque public replay.
type PublicReplayMetadata struct {
	Availability string `json:"availability"`
	Format       string `json:"format,omitempty"`
	Version      string `json:"version,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	Digest       string `json:"digest,omitempty"`
}

// PublicMatchDetail adds replay availability to a match identity.
type PublicMatchDetail struct {
	PublicMatch
	Replay PublicReplayMetadata `json:"replay"`
}

// PublicStateResponse is the polling response; payload is only exported state.
type PublicStateResponse struct {
	PublicMatch
	Availability string          `json:"availability"`
	StateVersion int64           `json:"state_version,omitempty"`
	Turn         int             `json:"turn,omitempty"`
	PublicState  json.RawMessage `json:"public_state,omitempty"`
	RetryAfterMS int             `json:"retry_after_ms"`
}

// PublicReplayResponse is the dedicated bounded replay resource.
type PublicReplayResponse struct {
	Availability string          `json:"availability"`
	Format       string          `json:"format,omitempty"`
	Version      string          `json:"version,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
}

// PublicQueryService selects public runs and exposes only safe state and replay data.
type PublicQueryService struct {
	queue  QueueStore
	states PublicStateStore
	reader ArtifactReader
}

// NewPublicQueryService constructs the spectator read-side adapter.
func NewPublicQueryService(queue QueueStore, states PublicStateStore, reader ArtifactReader) (*PublicQueryService, error) {
	if queue == nil || states == nil || reader == nil {
		return nil, fmt.Errorf("service: public query queue, state store, and reader are required")
	}
	return &PublicQueryService{queue: queue, states: states, reader: reader}, nil
}

// List returns one discoverable selected run per logical match.
func (s *PublicQueryService) List(ctx context.Context) ([]PublicMatch, error) {
	records, err := s.queue.List(ctx)
	if err != nil {
		return nil, err
	}
	selected := selectedPublicRecords(records)
	items := make([]PublicMatch, 0, len(selected))
	for _, record := range selected {
		items = append(items, publicMatchFromRecord(record))
	}
	return items, nil
}

// Get returns the selected public run and replay availability for a match.
func (s *PublicQueryService) Get(ctx context.Context, matchID string) (PublicMatchDetail, bool, error) {
	record, ok, err := s.selected(ctx, matchID)
	if err != nil || !ok {
		return PublicMatchDetail{}, ok, err
	}
	return PublicMatchDetail{PublicMatch: publicMatchFromRecord(record), Replay: replayMetadata(record)}, true, nil
}

// State returns the latest published exported state or a documented unavailable response.
func (s *PublicQueryService) State(ctx context.Context, matchID string) (PublicStateResponse, bool, error) {
	record, ok, err := s.selected(ctx, matchID)
	if err != nil || !ok {
		return PublicStateResponse{}, ok, err
	}
	response := PublicStateResponse{PublicMatch: publicMatchFromRecord(record), Availability: "state_unavailable", RetryAfterMS: 1000}
	state, found, err := s.states.Latest(ctx, record.Submission.MatchID, record.Submission.RunID)
	if err != nil {
		return PublicStateResponse{}, false, err
	}
	if !found {
		return response, true, nil
	}
	response.Availability = "available"
	response.StateVersion = state.Version
	response.Turn = state.Snapshot.Turn
	response.PublicState = append(json.RawMessage(nil), state.Snapshot.PublicState...)
	if isPublicTerminal(record.State) {
		response.RetryAfterMS = 0
	}
	return response, true, nil
}

// Replay reads one bounded, game-produced public artifact and normalizes all failures.
func (s *PublicQueryService) Replay(ctx context.Context, matchID string) (PublicReplayResponse, bool, error) {
	record, ok, err := s.selected(ctx, matchID)
	if err != nil || !ok {
		return PublicReplayResponse{}, ok, err
	}
	metadata := replayMetadata(record)
	response := PublicReplayResponse{Availability: metadata.Availability, Format: metadata.Format, Version: metadata.Version}
	if metadata.Availability != "available" || record.Terminal == nil {
		return response, true, nil
	}
	body, err := s.reader.Read(ctx, record.Terminal.PublicReplayPath)
	if err != nil || len(body) > maxPublicReplayBytes || !json.Valid(body) {
		return PublicReplayResponse{Availability: "replay_unavailable"}, true, nil
	}
	response.Payload = append(json.RawMessage(nil), body...)
	return response, true, nil
}

func (s *PublicQueryService) selected(ctx context.Context, matchID string) (QueueRecord, bool, error) {
	records, err := s.queue.List(ctx)
	if err != nil {
		return QueueRecord{}, false, err
	}
	for _, record := range selectedPublicRecords(records) {
		if record.Submission.MatchID == matchID {
			return record, true, nil
		}
	}
	return QueueRecord{}, false, nil
}

func selectedPublicRecords(records []QueueRecord) []QueueRecord {
	selected := make(map[string]QueueRecord)
	for _, record := range records {
		if !isPublicDiscoverable(record.State) {
			continue
		}
		current, exists := selected[record.Submission.MatchID]
		if !exists || (record.State == StateCompleted && record.Submission.Official) || !(current.State == StateCompleted && current.Submission.Official) {
			selected[record.Submission.MatchID] = record
		}
	}
	result := make([]QueueRecord, 0, len(selected))
	for _, record := range selected {
		result = append(result, record)
	}
	return result
}

func isPublicDiscoverable(state LifecycleState) bool {
	switch state {
	case StateRunning, StatePersisting, StateCompleted, StateFailed, StateCanceled:
		return true
	default:
		return false
	}
}

func isPublicTerminal(state LifecycleState) bool {
	return state == StateCompleted || state == StateFailed || state == StateCanceled
}

func publicMatchFromRecord(record QueueRecord) PublicMatch {
	return PublicMatch{MatchID: record.Submission.MatchID, SelectedRunID: record.Submission.RunID, Game: record.Submission.Game, LifecycleState: record.State}
}

func replayMetadata(record QueueRecord) PublicReplayMetadata {
	if record.State != StateCompleted || record.Terminal == nil || record.Terminal.PublicReplayPath == "" || record.Terminal.PublicReplayFormat == "" || record.Terminal.PublicReplayVersion == "" || record.Terminal.PublicReplaySize > maxPublicReplayBytes {
		return PublicReplayMetadata{Availability: "replay_unavailable"}
	}
	return PublicReplayMetadata{Availability: "available", Format: record.Terminal.PublicReplayFormat, Version: record.Terminal.PublicReplayVersion, SizeBytes: record.Terminal.PublicReplaySize, Digest: record.Terminal.PublicReplayDigest}
}
