package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/yoskeoka/ai-arena/artifactbundle"
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
)

// ValidationState captures the current admission state of one AI submission.
type ValidationState string

const (
	// ValidationReady means synchronous admission succeeded.
	ValidationReady ValidationState = "ready"
)

// RegisteredGame is the operator-facing metadata view for one admitted game.
type RegisteredGame struct {
	RegistrationID        string                `json:"registration_id"`
	Game                  contract.GameMetadata `json:"game"`
	ArtifactID            string                `json:"artifact_id,omitempty"`
	PlayerCount           int                   `json:"player_count,omitempty"`
	MaxActiveBotsPerOwner int                   `json:"max_active_bots_per_owner,omitempty"`
	BuildMode             registry.BuildMode    `json:"build_mode"`
	BuilderID             string                `json:"builder_id"`
	SupportedRulesets     []string              `json:"supported_rulesets"`
	Source                RegistrationSource    `json:"source,omitempty"`
	SourceID              string                `json:"source_id,omitempty"`
}

// RegisteredAI is one admitted AI artifact identity for the general operator lane.
type RegisteredAI struct {
	AISubmissionID     string                `json:"ai_submission_id"`
	GameRegistrationID string                `json:"game_registration_id"`
	Game               contract.GameMetadata `json:"game"`
	ArtifactRef        string                `json:"artifact_ref"`
	ArtifactID         string                `json:"artifact_id,omitempty"`
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
	ArtifactID     string                `json:"artifact_id,omitempty"`
	RulesetVersion string                `json:"ruleset_version,omitempty"`
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
	bundles             BundleStore
}

// WithBundleStore enables immutable AI bundle admission for this service.
func (s *GeneralSubmissionService) WithBundleStore(bundles BundleStore) *GeneralSubmissionService {
	s.bundles = bundles
	return s
}

// RegisterAIBundle admits an immutable WASI AI bundle for a registered game.
func (s *GeneralSubmissionService) RegisterAIBundle(ctx context.Context, req AISubmissionRequest, data []byte) (RegisteredAI, error) {
	if s.bundles == nil {
		return RegisteredAI{}, fmt.Errorf("service: AI bundle upload is not configured")
	}
	game, err := s.games.Get(ctx, strings.TrimSpace(req.GameRegistrationID))
	if err != nil {
		return RegisteredAI{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		return RegisteredAI{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	if bundle.Manifest.ArtifactKind != "ai" || bundle.Manifest.GameID != game.Game.GameID {
		return RegisteredAI{}, fmt.Errorf("%w: service: AI bundle is incompatible with game registration", ErrBadRequest)
	}
	major, err := catalog.MajorVersion(bundle.Manifest.GameVersion)
	if err != nil {
		return RegisteredAI{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	expectedMajor, _ := catalog.MajorVersion(game.Game.GameVersion)
	if major != expectedMajor {
		return RegisteredAI{}, fmt.Errorf("%w: service: AI bundle game version is incompatible", ErrBadRequest)
	}
	if err := s.bundles.Put(ctx, bundle); err != nil {
		return RegisteredAI{}, err
	}
	id := strings.TrimSpace(req.AISubmissionID)
	if id == "" {
		id = s.newAISubmissionIDFn()
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" {
		name = bundle.Manifest.AIID
	}
	record := RegisteredAI{AISubmissionID: id, GameRegistrationID: game.RegistrationID, Game: game.Game, ArtifactID: bundle.Digest, DisplayName: name, RuntimeKind: runtime.KindWASMWASI, AIID: bundle.Manifest.AIID, ValidationState: ValidationReady, Source: SourceManual}
	if err := s.submissions.Save(ctx, record); err != nil {
		return RegisteredAI{}, wrapConflict(err)
	}
	return record, nil
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
	if strings.TrimSpace(req.ArtifactID) != "" {
		return s.registerArtifactBackedGame(ctx, req)
	}
	registrationID := strings.TrimSpace(req.RegistrationID)
	if registrationID == "" {
		registrationID = defaultGameRegistrationID(req.Game)
	}
	record, err := s.buildRegisteredGame(ctx, registrationID, req.Game, SourceManual, "")
	if err != nil {
		return RegisteredGame{}, err
	}
	if err := s.games.Save(ctx, record); err != nil {
		return RegisteredGame{}, wrapConflict(err)
	}
	return record, nil
}

func (s *GeneralSubmissionService) registerArtifactBackedGame(ctx context.Context, req GameRegistrationRequest) (RegisteredGame, error) {
	if s.bundles == nil {
		return RegisteredGame{}, fmt.Errorf("%w: service: game bundle upload is not configured", ErrBadRequest)
	}
	artifactID := strings.TrimSpace(req.ArtifactID)
	data, err := s.bundles.Read(ctx, artifactID)
	if err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	bundle, err := artifactbundle.Read(data)
	if err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %v", ErrBadRequest, err)
	}
	if bundle.Digest != artifactID || bundle.Manifest.ArtifactKind != "game" {
		return RegisteredGame{}, fmt.Errorf("%w: service: selected artifact is not a game bundle", ErrBadRequest)
	}
	descriptor, err := s.registry.LookupArtifact(ctx, artifactID)
	if err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	if descriptor.ArtifactID != artifactID || descriptor.GameID != bundle.Manifest.GameID || descriptor.GameVersion != bundle.Manifest.GameVersion || descriptor.BuildMode != registry.BuildModeWASMWASI || descriptor.BuilderID != "artifact/"+artifactID {
		return RegisteredGame{}, fmt.Errorf("%w: service: admitted game descriptor does not match artifact manifest", ErrBadRequest)
	}
	rulesetVersion := strings.TrimSpace(req.RulesetVersion)
	if rulesetVersion == "" {
		rulesetVersion = strings.TrimSpace(req.Game.RulesetVersion)
	}
	requestedGameID := strings.TrimSpace(req.Game.GameID)
	requestedGameVersion := strings.TrimSpace(req.Game.GameVersion)
	requestedGameRuleset := strings.TrimSpace(req.Game.RulesetVersion)
	requestedRuleset := strings.TrimSpace(req.RulesetVersion)
	if (requestedGameID != "" && requestedGameID != bundle.Manifest.GameID) || (requestedGameVersion != "" && requestedGameVersion != bundle.Manifest.GameVersion) || (requestedGameRuleset != "" && requestedRuleset != "" && requestedGameRuleset != requestedRuleset) {
		return RegisteredGame{}, fmt.Errorf("%w: service: requested game metadata does not match selected artifact", ErrBadRequest)
	}
	game := contract.GameMetadata{GameID: bundle.Manifest.GameID, GameVersion: bundle.Manifest.GameVersion, RulesetVersion: rulesetVersion}
	if err := catalog.ValidateMetadata(catalog.GameMetadata(game)); err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	var playerCount, maxActiveBotsPerOwner int
	foundRuleset := false
	for _, ruleset := range bundle.Manifest.Rulesets {
		if ruleset.RulesetVersion == rulesetVersion {
			playerCount = ruleset.PlayerCount
			maxActiveBotsPerOwner = ruleset.MaxActiveBotsPerOwner
			foundRuleset = true
			break
		}
	}
	if !foundRuleset {
		return RegisteredGame{}, fmt.Errorf("%w: service: selected ruleset %q is not declared by artifact manifest", ErrBadRequest, rulesetVersion)
	}
	if !slicesContain(descriptor.BuildConstraints.SupportedRulesets, rulesetVersion) {
		return RegisteredGame{}, fmt.Errorf("%w: service: admitted game descriptor does not support ruleset %q", ErrBadRequest, rulesetVersion)
	}
	if playerCount < 1 || maxActiveBotsPerOwner < 1 {
		return RegisteredGame{}, fmt.Errorf("%w: service: selected ruleset is missing player or bot limits", ErrBadRequest)
	}
	registrationID := strings.TrimSpace(req.RegistrationID)
	if registrationID == "" {
		registrationID = defaultGameRegistrationID(game)
	}
	record := RegisteredGame{RegistrationID: registrationID, Game: game, ArtifactID: artifactID, PlayerCount: playerCount, MaxActiveBotsPerOwner: maxActiveBotsPerOwner, BuildMode: descriptor.BuildMode, BuilderID: descriptor.BuilderID, SupportedRulesets: append([]string(nil), descriptor.BuildConstraints.SupportedRulesets...), Source: SourceManual}
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

	aiSubmissionID := strings.TrimSpace(req.AISubmissionID)
	if aiSubmissionID == "" {
		aiSubmissionID = s.newAISubmissionIDFn()
	}
	record, err := s.buildRegisteredAI(aiSubmissionID, game, req.ArtifactRef, req.DisplayName, SourceManual, "")
	if err != nil {
		return RegisteredAI{}, err
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

// GetGame returns one registered game by id.
func (s *GeneralSubmissionService) GetGame(ctx context.Context, registrationID string) (RegisteredGame, error) {
	return s.games.Get(ctx, registrationID)
}

// GetAI returns one admitted AI submission by id.
func (s *GeneralSubmissionService) GetAI(ctx context.Context, aiSubmissionID string) (RegisteredAI, error) {
	return s.submissions.Get(ctx, aiSubmissionID)
}

func defaultGameRegistrationID(game contract.GameMetadata) string {
	major, err := catalog.MajorVersion(game.GameVersion)
	if err != nil {
		return strings.TrimSpace(game.GameID)
	}
	return fmt.Sprintf("%s-v%d-%s", strings.TrimSpace(game.GameID), major, strings.TrimSpace(game.RulesetVersion))
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

func (s *GeneralSubmissionService) buildRegisteredGame(ctx context.Context, registrationID string, game contract.GameMetadata, source RegistrationSource, sourceID string) (RegisteredGame, error) {
	if err := catalog.ValidateMetadata(catalog.GameMetadata(game)); err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	descriptor, err := s.registry.LookupVersion(ctx, game.GameID, game.GameVersion)
	if err != nil {
		return RegisteredGame{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	if !slicesContain(descriptor.BuildConstraints.SupportedRulesets, game.RulesetVersion) {
		return RegisteredGame{}, fmt.Errorf("%w: service: ruleset %q is not supported for game %q version %q", ErrBadRequest, game.RulesetVersion, game.GameID, game.GameVersion)
	}

	record := RegisteredGame{
		RegistrationID:    registrationID,
		Game:              game,
		ArtifactID:        descriptor.ArtifactID,
		BuildMode:         descriptor.BuildMode,
		BuilderID:         descriptor.BuilderID,
		SupportedRulesets: append([]string(nil), descriptor.BuildConstraints.SupportedRulesets...),
		Source:            source,
		SourceID:          sourceID,
	}
	if record.ArtifactID != "" && s.bundles != nil {
		data, err := s.bundles.Read(ctx, record.ArtifactID)
		if err != nil {
			return RegisteredGame{}, err
		}
		bundle, err := artifactbundle.Read(data)
		if err != nil {
			return RegisteredGame{}, err
		}
		for _, ruleset := range bundle.Manifest.Rulesets {
			if ruleset.RulesetVersion == game.RulesetVersion {
				record.PlayerCount = ruleset.PlayerCount
				record.MaxActiveBotsPerOwner = ruleset.MaxActiveBotsPerOwner
				break
			}
		}
	}
	return record, nil
}

func (s *GeneralSubmissionService) buildRegisteredAI(aiSubmissionID string, game RegisteredGame, artifactRef string, displayName string, source RegistrationSource, sourceID string) (RegisteredAI, error) {
	loaded, err := validateRegisteredArtifact(s.baseDir, game.Game, artifactRef)
	if err != nil {
		return RegisteredAI{}, fmt.Errorf("%w: %w", ErrBadRequest, err)
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = loaded.AIID
	}
	return RegisteredAI{
		AISubmissionID:     aiSubmissionID,
		GameRegistrationID: game.RegistrationID,
		Game:               game.Game,
		ArtifactRef:        strings.TrimSpace(artifactRef),
		DisplayName:        name,
		RuntimeKind:        loaded.Runtime.Kind,
		AIID:               loaded.AIID,
		ValidationState:    ValidationReady,
		Source:             source,
		SourceID:           sourceID,
	}, nil
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
