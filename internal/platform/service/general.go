package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/registry"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

var (
	// ErrGameRegistrationNotFound reports that no registered game exists for the requested id.
	ErrGameRegistrationNotFound = errors.New("service: game registration not found")
	// ErrAISubmissionNotFound reports that no admitted AI submission exists for the requested id.
	ErrAISubmissionNotFound = errors.New("service: ai submission not found")
)

// RegistrationSource marks how one general-lane entity entered the system.
type RegistrationSource string

const (
	// SourceManual means the operator registered the entity directly.
	SourceManual RegistrationSource = "manual"
	// SourcePreset means the entity was materialized from one preset definition.
	SourcePreset RegistrationSource = "preset"
)

// ValidationState captures the current admission state of one AI submission.
type ValidationState string

const (
	// ValidationReady means synchronous admission succeeded.
	ValidationReady ValidationState = "ready"
)

// RegisteredGame is the operator-facing metadata view for one admitted game.
type RegisteredGame struct {
	RegistrationID    string                `json:"registration_id"`
	Game              contract.GameMetadata `json:"game"`
	BuildMode         registry.BuildMode    `json:"build_mode"`
	BuilderID         string                `json:"builder_id"`
	SupportedRulesets []string              `json:"supported_rulesets"`
	Source            RegistrationSource    `json:"source,omitempty"`
	SourceID          string                `json:"source_id,omitempty"`
}

// RegisteredAI is one admitted AI artifact identity for the general operator lane.
type RegisteredAI struct {
	AISubmissionID     string                `json:"ai_submission_id"`
	GameRegistrationID string                `json:"game_registration_id"`
	Game               contract.GameMetadata `json:"game"`
	ArtifactRef        string                `json:"artifact_ref"`
	DisplayName        string                `json:"display_name"`
	RuntimeKind        runtime.Kind          `json:"runtime_kind"`
	AIID               string                `json:"ai_id"`
	ValidationState    ValidationState       `json:"validation_state"`
	Source             RegistrationSource    `json:"source,omitempty"`
	SourceID           string                `json:"source_id,omitempty"`
}

// GameRegistrationRequest registers one operator-visible game metadata view.
type GameRegistrationRequest struct {
	RegistrationID string                `json:"registration_id,omitempty"`
	Game           contract.GameMetadata `json:"game"`
}

// AISubmissionRequest registers one admitted AI artifact for a game registration.
type AISubmissionRequest struct {
	AISubmissionID     string `json:"ai_submission_id,omitempty"`
	GameRegistrationID string `json:"game_registration_id"`
	ArtifactRef        string `json:"artifact_ref"`
	DisplayName        string `json:"display_name,omitempty"`
}

// GameRegistrationStore persists operator-facing registered game views.
type GameRegistrationStore interface {
	Save(context.Context, RegisteredGame) error
	Get(context.Context, string) (RegisteredGame, error)
	List(context.Context) ([]RegisteredGame, error)
}

// AISubmissionStore persists admitted AI artifact identities.
type AISubmissionStore interface {
	Save(context.Context, RegisteredAI) error
	Get(context.Context, string) (RegisteredAI, error)
	List(context.Context) ([]RegisteredAI, error)
}

// GeneralSubmissionService validates and exposes general-lane registration entities.
type GeneralSubmissionService struct {
	baseDir             string
	registry            *registry.Registry
	games               GameRegistrationStore
	submissions         AISubmissionStore
	newAISubmissionIDFn func() string
}

// NewGeneralSubmissionService constructs the general-lane registration service.
func NewGeneralSubmissionService(baseDir string, reg *registry.Registry, games GameRegistrationStore, submissions AISubmissionStore) (*GeneralSubmissionService, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("service: base_dir is required")
	}
	if reg == nil {
		reg = registry.Default()
	}
	if games == nil {
		games = NewInMemoryGameRegistrationStore()
	}
	if submissions == nil {
		submissions = NewInMemoryAISubmissionStore()
	}
	return &GeneralSubmissionService{
		baseDir:             baseDir,
		registry:            reg,
		games:               games,
		submissions:         submissions,
		newAISubmissionIDFn: func() string { return "ai-" + uuid.NewString() },
	}, nil
}

// RegisterGame validates and stores one operator-facing registered game view.
func (s *GeneralSubmissionService) RegisterGame(ctx context.Context, req GameRegistrationRequest) (RegisteredGame, error) {
	if err := catalog.ValidateMetadata(catalog.GameMetadata(req.Game)); err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	descriptor, err := s.registry.LookupVersion(ctx, req.Game.GameID, req.Game.GameVersion)
	if err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	if !slicesContain(descriptor.BuildConstraints.SupportedRulesets, req.Game.RulesetVersion) {
		return RegisteredGame{}, fmt.Errorf("%w: service: ruleset %q is not supported for game %q version %q", ErrBadRequest, req.Game.RulesetVersion, req.Game.GameID, req.Game.GameVersion)
	}

	registrationID := strings.TrimSpace(req.RegistrationID)
	if registrationID == "" {
		registrationID = defaultGameRegistrationID(req.Game)
	}
	record := RegisteredGame{
		RegistrationID:    registrationID,
		Game:              req.Game,
		BuildMode:         descriptor.BuildMode,
		BuilderID:         descriptor.BuilderID,
		SupportedRulesets: append([]string(nil), descriptor.BuildConstraints.SupportedRulesets...),
		Source:            SourceManual,
	}
	if err := s.games.Save(ctx, record); err != nil {
		return RegisteredGame{}, wrapConflict(err)
	}
	return record, nil
}

// RegisterAI validates and stores one admitted AI artifact identity.
func (s *GeneralSubmissionService) RegisterAI(ctx context.Context, req AISubmissionRequest) (RegisteredAI, error) {
	registrationID := strings.TrimSpace(req.GameRegistrationID)
	if registrationID == "" {
		return RegisteredAI{}, fmt.Errorf("%w: service: game_registration_id is required", ErrBadRequest)
	}
	game, err := s.games.Get(ctx, registrationID)
	if err != nil {
		if errors.Is(err, ErrGameRegistrationNotFound) {
			return RegisteredAI{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
		}
		return RegisteredAI{}, err
	}

	loaded, err := validateRegisteredArtifact(s.baseDir, game.Game, req.ArtifactRef)
	if err != nil {
		return RegisteredAI{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	aiSubmissionID := strings.TrimSpace(req.AISubmissionID)
	if aiSubmissionID == "" {
		aiSubmissionID = s.newAISubmissionIDFn()
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = loaded.AIID
	}
	record := RegisteredAI{
		AISubmissionID:     aiSubmissionID,
		GameRegistrationID: game.RegistrationID,
		Game:               game.Game,
		ArtifactRef:        strings.TrimSpace(req.ArtifactRef),
		DisplayName:        displayName,
		RuntimeKind:        loaded.Runtime.Kind,
		AIID:               loaded.AIID,
		ValidationState:    ValidationReady,
		Source:             SourceManual,
	}
	if err := s.submissions.Save(ctx, record); err != nil {
		return RegisteredAI{}, wrapConflict(err)
	}
	return record, nil
}

// ListGames returns the known game registrations in insertion order.
func (s *GeneralSubmissionService) ListGames(ctx context.Context) ([]RegisteredGame, error) {
	return s.games.List(ctx)
}

// ListAIs returns the known AI submissions in insertion order.
func (s *GeneralSubmissionService) ListAIs(ctx context.Context) ([]RegisteredAI, error) {
	return s.submissions.List(ctx)
}

// MaterializePreset converts one preset submission into general-lane identities.
func (s *GeneralSubmissionService) MaterializePreset(ctx context.Context, presetID string, submission MatchSubmission) (RegisteredGame, []RegisteredAI, error) {
	game, err := s.materializePresetGame(ctx, presetID, submission.Game)
	if err != nil {
		return RegisteredGame{}, nil, err
	}
	items := make([]RegisteredAI, 0, len(submission.Players))
	for _, player := range submission.Players {
		item, err := s.materializePresetAI(ctx, presetID, game, player)
		if err != nil {
			return RegisteredGame{}, nil, err
		}
		items = append(items, item)
	}
	return game, items, nil
}

func (s *GeneralSubmissionService) materializePresetGame(ctx context.Context, presetID string, game contract.GameMetadata) (RegisteredGame, error) {
	record, err := s.RegisterGame(ctx, GameRegistrationRequest{
		RegistrationID: defaultGameRegistrationID(game),
		Game:           game,
	})
	if err == nil {
		record.Source = SourcePreset
		record.SourceID = presetID
		_ = s.games.Save(ctx, record)
		return record, nil
	}
	if !errors.Is(err, ErrConflict) {
		return RegisteredGame{}, err
	}
	record, getErr := s.games.Get(ctx, defaultGameRegistrationID(game))
	if getErr != nil {
		return RegisteredGame{}, getErr
	}
	return record, nil
}

func (s *GeneralSubmissionService) materializePresetAI(ctx context.Context, presetID string, game RegisteredGame, player SubmittedPlayer) (RegisteredAI, error) {
	record, err := s.RegisterAI(ctx, AISubmissionRequest{
		AISubmissionID:     defaultPresetAISubmissionID(presetID, player.PlayerID),
		GameRegistrationID: game.RegistrationID,
		ArtifactRef:        player.ArtifactRef,
		DisplayName:        player.PlayerID,
	})
	if err == nil {
		record.Source = SourcePreset
		record.SourceID = presetID
		_ = s.submissions.Save(ctx, record)
		return record, nil
	}
	if !errors.Is(err, ErrConflict) {
		return RegisteredAI{}, err
	}
	record, getErr := s.submissions.Get(ctx, defaultPresetAISubmissionID(presetID, player.PlayerID))
	if getErr != nil {
		return RegisteredAI{}, getErr
	}
	return record, nil
}

func defaultGameRegistrationID(game contract.GameMetadata) string {
	major, err := catalog.MajorVersion(game.GameVersion)
	if err != nil {
		return strings.TrimSpace(game.GameID)
	}
	return fmt.Sprintf("%s-v%d", strings.TrimSpace(game.GameID), major)
}

func defaultPresetAISubmissionID(presetID, playerID string) string {
	return fmt.Sprintf("preset-%s-%s", strings.TrimSpace(presetID), strings.TrimSpace(playerID))
}

func wrapConflict(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrConflict) {
		return err
	}
	if strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("%w: %w", ErrConflict, err)
	}
	return err
}

// InMemoryGameRegistrationStore keeps general game registrations inside one process.
type InMemoryGameRegistrationStore struct {
	mu    sync.Mutex
	order []string
	items map[string]RegisteredGame
}

// NewInMemoryGameRegistrationStore constructs one in-memory game registration store.
func NewInMemoryGameRegistrationStore() *InMemoryGameRegistrationStore {
	return &InMemoryGameRegistrationStore{items: make(map[string]RegisteredGame)}
}

// Save inserts or idempotently reuses one game registration.
func (s *InMemoryGameRegistrationStore) Save(_ context.Context, record RegisteredGame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(record.RegistrationID) == "" {
		return fmt.Errorf("service: registration_id is required")
	}
	if existing, ok := s.items[record.RegistrationID]; ok {
		if !sameRegisteredGame(existing, record) {
			return fmt.Errorf("service: registration_id %q already exists", record.RegistrationID)
		}
		s.items[record.RegistrationID] = cloneRegisteredGame(record)
		return nil
	}
	s.order = append(s.order, record.RegistrationID)
	s.items[record.RegistrationID] = cloneRegisteredGame(record)
	return nil
}

// Get returns one registered game by id.
func (s *InMemoryGameRegistrationStore) Get(_ context.Context, registrationID string) (RegisteredGame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.items[registrationID]
	if !ok {
		return RegisteredGame{}, ErrGameRegistrationNotFound
	}
	return cloneRegisteredGame(record), nil
}

// List returns registered games in insertion order.
func (s *InMemoryGameRegistrationStore) List(_ context.Context) ([]RegisteredGame, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]RegisteredGame, 0, len(s.order))
	for _, id := range s.order {
		items = append(items, cloneRegisteredGame(s.items[id]))
	}
	return items, nil
}

// InMemoryAISubmissionStore keeps admitted AI identities inside one process.
type InMemoryAISubmissionStore struct {
	mu    sync.Mutex
	order []string
	items map[string]RegisteredAI
}

// NewInMemoryAISubmissionStore constructs one in-memory AI submission store.
func NewInMemoryAISubmissionStore() *InMemoryAISubmissionStore {
	return &InMemoryAISubmissionStore{items: make(map[string]RegisteredAI)}
}

// Save inserts or idempotently reuses one AI submission.
func (s *InMemoryAISubmissionStore) Save(_ context.Context, record RegisteredAI) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(record.AISubmissionID) == "" {
		return fmt.Errorf("service: ai_submission_id is required")
	}
	if existing, ok := s.items[record.AISubmissionID]; ok {
		if !sameRegisteredAI(existing, record) {
			return fmt.Errorf("service: ai_submission_id %q already exists", record.AISubmissionID)
		}
		s.items[record.AISubmissionID] = cloneRegisteredAI(record)
		return nil
	}
	s.order = append(s.order, record.AISubmissionID)
	s.items[record.AISubmissionID] = cloneRegisteredAI(record)
	return nil
}

// Get returns one admitted AI submission by id.
func (s *InMemoryAISubmissionStore) Get(_ context.Context, aiSubmissionID string) (RegisteredAI, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, ok := s.items[aiSubmissionID]
	if !ok {
		return RegisteredAI{}, ErrAISubmissionNotFound
	}
	return cloneRegisteredAI(record), nil
}

// List returns admitted AI submissions in insertion order.
func (s *InMemoryAISubmissionStore) List(_ context.Context) ([]RegisteredAI, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]RegisteredAI, 0, len(s.order))
	for _, id := range s.order {
		items = append(items, cloneRegisteredAI(s.items[id]))
	}
	return items, nil
}

func sameRegisteredGame(a, b RegisteredGame) bool {
	return a.RegistrationID == b.RegistrationID &&
		a.Game == b.Game &&
		a.BuildMode == b.BuildMode &&
		a.BuilderID == b.BuilderID &&
		slices.Equal(a.SupportedRulesets, b.SupportedRulesets) &&
		a.Source == b.Source &&
		a.SourceID == b.SourceID
}

func sameRegisteredAI(a, b RegisteredAI) bool {
	return a.AISubmissionID == b.AISubmissionID &&
		a.GameRegistrationID == b.GameRegistrationID &&
		a.Game == b.Game &&
		a.ArtifactRef == b.ArtifactRef &&
		a.DisplayName == b.DisplayName &&
		a.RuntimeKind == b.RuntimeKind &&
		a.AIID == b.AIID &&
		a.ValidationState == b.ValidationState &&
		a.Source == b.Source &&
		a.SourceID == b.SourceID
}

func cloneRegisteredGame(record RegisteredGame) RegisteredGame {
	record.SupportedRulesets = append([]string(nil), record.SupportedRulesets...)
	return record
}

func cloneRegisteredAI(record RegisteredAI) RegisteredAI {
	return record
}
