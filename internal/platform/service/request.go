package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

// MatchRequestParticipant binds one operator-visible player id to one admitted AI submission.
type MatchRequestParticipant struct {
	PlayerID       string `json:"player_id"`
	AISubmissionID string `json:"ai_submission_id"`
}

// MatchRequest is one operator-created match request plus its scheduled queue submission.
type MatchRequest struct {
	RequestID             string                    `json:"request_id"`
	GameRegistrationID    string                    `json:"game_registration_id"`
	Game                  contract.GameMetadata     `json:"game"`
	Participants          []MatchRequestParticipant `json:"participants"`
	OutputDir             string                    `json:"output_dir"`
	Source                RegistrationSource        `json:"source,omitempty"`
	SourceID              string                    `json:"source_id,omitempty"`
	ScheduledSubmissionID string                    `json:"scheduled_submission_id"`
	ScheduledMatchID      string                    `json:"scheduled_match_id"`
	LifecycleState        LifecycleState            `json:"lifecycle_state"`
}

// MatchRequestCreateRequest is the operator-facing create payload for one general match request.
type MatchRequestCreateRequest struct {
	RequestID          string                    `json:"request_id,omitempty"`
	GameRegistrationID string                    `json:"game_registration_id"`
	Participants       []MatchRequestParticipant `json:"participants"`
	OutputDir          string                    `json:"output_dir"`
	MatchID            string                    `json:"match_id,omitempty"`
}

// MatchRequestStore persists accepted match requests.
type MatchRequestStore interface {
	Save(context.Context, MatchRequest) error
	List(context.Context) ([]MatchRequest, error)
}

// MatchRequestService validates general/preset match requests and schedules them into the queue.
type MatchRequestService struct {
	general           *GeneralSubmissionService
	commands          *CommandService
	queue             QueueStore
	store             MatchRequestStore
	newRequestIDFn    func() string
	newSubmissionIDFn func() string
	newMatchIDFn      func() string
}

// NewMatchRequestService constructs the minimal request+scheduling service.
func NewMatchRequestService(general *GeneralSubmissionService, commands *CommandService, queue QueueStore, store MatchRequestStore) (*MatchRequestService, error) {
	if general == nil {
		return nil, fmt.Errorf("service: general submission service is required")
	}
	if commands == nil {
		return nil, fmt.Errorf("service: command service is required")
	}
	if queue == nil {
		return nil, fmt.Errorf("service: queue store is required")
	}
	if store == nil {
		store = NewInMemoryMatchRequestStore()
	}
	return &MatchRequestService{
		general:           general,
		commands:          commands,
		queue:             queue,
		store:             store,
		newRequestIDFn:    func() string { return "req-" + uuid.NewString() },
		newSubmissionIDFn: func() string { return "sub-" + uuid.NewString() },
		newMatchIDFn:      func() string { return "match-" + uuid.NewString() },
	}, nil
}

// Create validates one manual match request and schedules it into the queue immediately.
func (s *MatchRequestService) Create(ctx context.Context, req MatchRequestCreateRequest) (MatchRequest, QueueRecord, error) {
	requestID := strings.TrimSpace(req.RequestID)
	if requestID == "" {
		requestID = s.newRequestIDFn()
	}
	gameRegistrationID := strings.TrimSpace(req.GameRegistrationID)
	if gameRegistrationID == "" {
		return MatchRequest{}, QueueRecord{}, fmt.Errorf("%w: service: game_registration_id is required", ErrBadRequest)
	}
	game, err := s.general.GetGame(ctx, gameRegistrationID)
	if err != nil {
		if errors.Is(err, ErrGameRegistrationNotFound) {
			return MatchRequest{}, QueueRecord{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
		}
		return MatchRequest{}, QueueRecord{}, err
	}

	players, err := s.resolveParticipants(ctx, gameRegistrationID, req.Participants)
	if err != nil {
		return MatchRequest{}, QueueRecord{}, err
	}
	matchID := strings.TrimSpace(req.MatchID)
	if matchID == "" {
		matchID = s.newMatchIDFn()
	}
	submission := MatchSubmission{
		SubmissionID: s.newSubmissionIDFn(),
		MatchID:      matchID,
		Game:         game.Game,
		Players:      players,
		OutputDir:    strings.TrimSpace(req.OutputDir),
		AttemptCount: 1,
	}
	record, err := s.commands.Submit(ctx, submission)
	if err != nil {
		return MatchRequest{}, QueueRecord{}, err
	}

	item := MatchRequest{
		RequestID:             requestID,
		GameRegistrationID:    gameRegistrationID,
		Game:                  game.Game,
		Participants:          cloneRequestParticipants(req.Participants),
		OutputDir:             submission.OutputDir,
		Source:                SourceManual,
		ScheduledSubmissionID: record.Submission.SubmissionID,
		ScheduledMatchID:      record.Submission.MatchID,
		LifecycleState:        record.State,
	}
	if err := s.store.Save(ctx, item); err != nil {
		return MatchRequest{}, QueueRecord{}, wrapConflict(err)
	}
	return item, record, nil
}

// CreatePreset materializes one preset into general identities, then schedules it like a general request.
func (s *MatchRequestService) CreatePreset(ctx context.Context, presetID string, submission MatchSubmission) (MatchRequest, QueueRecord, error) {
	game, items, err := s.general.MaterializePreset(ctx, presetID, submission)
	if err != nil {
		return MatchRequest{}, QueueRecord{}, err
	}
	participants := make([]MatchRequestParticipant, 0, len(submission.Players))
	for index, player := range submission.Players {
		participants = append(participants, MatchRequestParticipant{
			PlayerID:       player.PlayerID,
			AISubmissionID: items[index].AISubmissionID,
		})
	}

	record, err := s.commands.Submit(ctx, submission)
	if err != nil {
		return MatchRequest{}, QueueRecord{}, err
	}
	item := MatchRequest{
		RequestID:             "preset-" + presetID + "-" + record.Submission.SubmissionID,
		GameRegistrationID:    game.RegistrationID,
		Game:                  game.Game,
		Participants:          participants,
		OutputDir:             submission.OutputDir,
		Source:                SourcePreset,
		SourceID:              presetID,
		ScheduledSubmissionID: record.Submission.SubmissionID,
		ScheduledMatchID:      record.Submission.MatchID,
		LifecycleState:        record.State,
	}
	if err := s.store.Save(ctx, item); err != nil {
		return MatchRequest{}, QueueRecord{}, wrapConflict(err)
	}
	return item, record, nil
}

// List returns accepted match requests with current lifecycle derived from the queue when available.
func (s *MatchRequestService) List(ctx context.Context) ([]MatchRequest, error) {
	items, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		submissionID := strings.TrimSpace(items[index].ScheduledSubmissionID)
		if submissionID == "" {
			continue
		}
		record, getErr := s.queue.Get(ctx, submissionID)
		if getErr != nil {
			if errors.Is(getErr, ErrQueueRecordNotFound) {
				continue
			}
			return nil, getErr
		}
		items[index].LifecycleState = record.State
	}
	return items, nil
}

func (s *MatchRequestService) resolveParticipants(ctx context.Context, gameRegistrationID string, participants []MatchRequestParticipant) ([]SubmittedPlayer, error) {
	if len(participants) == 0 {
		return nil, fmt.Errorf("%w: service: at least one participant is required", ErrBadRequest)
	}
	players := make([]SubmittedPlayer, 0, len(participants))
	seenPlayerIDs := make(map[string]struct{}, len(participants))
	for _, participant := range participants {
		playerID := strings.TrimSpace(participant.PlayerID)
		if playerID == "" {
			return nil, fmt.Errorf("%w: service: player_id is required", ErrBadRequest)
		}
		if _, exists := seenPlayerIDs[playerID]; exists {
			return nil, fmt.Errorf("%w: service: duplicate player_id %q", ErrBadRequest, playerID)
		}
		seenPlayerIDs[playerID] = struct{}{}
		aiSubmissionID := strings.TrimSpace(participant.AISubmissionID)
		if aiSubmissionID == "" {
			return nil, fmt.Errorf("%w: service: ai_submission_id is required for player %q", ErrBadRequest, playerID)
		}
		ai, err := s.general.GetAI(ctx, aiSubmissionID)
		if err != nil {
			if errors.Is(err, ErrAISubmissionNotFound) {
				return nil, fmt.Errorf("%w: %w", ErrBadRequest, err)
			}
			return nil, err
		}
		if ai.GameRegistrationID != gameRegistrationID {
			return nil, fmt.Errorf("%w: service: ai submission %q does not belong to game registration %q", ErrBadRequest, aiSubmissionID, gameRegistrationID)
		}
		players = append(players, SubmittedPlayer{
			PlayerID:    playerID,
			ArtifactRef: ai.ArtifactRef,
		})
	}
	return players, nil
}

func cloneRequestParticipants(items []MatchRequestParticipant) []MatchRequestParticipant {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]MatchRequestParticipant, len(items))
	copy(cloned, items)
	return cloned
}

// InMemoryMatchRequestStore keeps accepted match requests inside one process.
type InMemoryMatchRequestStore struct {
	mu    sync.Mutex
	order []string
	items map[string]MatchRequest
}

// NewInMemoryMatchRequestStore constructs one in-memory match request store.
func NewInMemoryMatchRequestStore() *InMemoryMatchRequestStore {
	return &InMemoryMatchRequestStore{items: make(map[string]MatchRequest)}
}

// Save persists one accepted request.
func (s *InMemoryMatchRequestStore) Save(_ context.Context, item MatchRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(item.RequestID) == "" {
		return fmt.Errorf("service: request_id is required")
	}
	if _, exists := s.items[item.RequestID]; exists {
		return fmt.Errorf("service: request_id %q already exists", item.RequestID)
	}
	s.items[item.RequestID] = item
	s.order = append(s.order, item.RequestID)
	return nil
}

// List returns accepted requests in insertion order.
func (s *InMemoryMatchRequestStore) List(_ context.Context) ([]MatchRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]MatchRequest, 0, len(s.order))
	for _, requestID := range s.order {
		items = append(items, s.items[requestID])
	}
	return items, nil
}
