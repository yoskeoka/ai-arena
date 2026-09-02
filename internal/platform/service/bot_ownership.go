package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrBotNotFound reports that a requested bot identity does not exist.
	ErrBotNotFound = errors.New("service: bot not found")
	// ErrBotQuotaExceeded reports that creating another active bot would exceed the scope quota.
	ErrBotQuotaExceeded = errors.New("service: active bot quota exceeded")
)

// BotLifecycleState controls whether a bot can be selected for a new match.
type BotLifecycleState string

const (
	// BotActive marks a bot as eligible for new revisions and match selection.
	BotActive BotLifecycleState = "active"
	// BotRetired marks a bot as excluded from new match selection while retaining its history.
	BotRetired BotLifecycleState = "retired"
)

// CompetitionScope is the stable major-version and ruleset selection boundary.
type CompetitionScope struct {
	ScopeID               string `json:"scope_id"`
	GameID                string `json:"game_id"`
	GameVersionMajor      int    `json:"game_version_major"`
	RulesetVersion        string `json:"ruleset_version"`
	MaxActiveBotsPerOwner int    `json:"max_active_bots_per_owner"`
}

// OwnedBot is a stable, owner-scoped ranking identity.
type OwnedBot struct {
	BotID             string            `json:"bot_id"`
	OwnerAccountID    string            `json:"owner_account_id"`
	ScopeID           string            `json:"scope_id"`
	BotName           string            `json:"bot_name"`
	NormalizedBotName string            `json:"-"`
	LifecycleState    BotLifecycleState `json:"lifecycle_state"`
	ActiveRevisionID  string            `json:"active_submission_id,omitempty"`
}

// EligibleBot is the minimum operator composition projection for one ready bot.
type EligibleBot struct {
	BotID            string `json:"bot_id"`
	ScopeID          string `json:"scope_id"`
	BotName          string `json:"bot_name"`
	ActiveRevisionID string `json:"active_submission_id"`
}

// ResolvedEligibleBot contains the immutable revision selected at admission.
type ResolvedEligibleBot struct {
	EligibleBot
	ArtifactID string `json:"artifact_id"`
}

// AISubmissionRevision is an immutable admitted artifact revision for a bot.
type AISubmissionRevision struct {
	AISubmissionID  string          `json:"ai_submission_id"`
	BotID           string          `json:"bot_id"`
	ArtifactID      string          `json:"artifact_id"`
	ValidationState ValidationState `json:"validation_state"`
	CreatedAt       time.Time       `json:"created_at"`
}

// BotRevisionRequest chooses either a new bot or a new revision of an existing bot.
type BotRevisionRequest struct {
	Scope           CompetitionScope `json:"scope,omitempty"`
	ScopeID         string           `json:"scope_id,omitempty"`
	OwnerAccountID  string           `json:"-"`
	BotID           string           `json:"bot_id,omitempty"`
	BotName         string           `json:"bot_name,omitempty"`
	ArtifactID      string           `json:"artifact_id"`
	ValidationState ValidationState  `json:"validation_state,omitempty"`
	RuntimeKind     string           `json:"-"`
	AIID            string           `json:"-"`
}

// BotOwnershipStore is the durable transaction boundary for bot lifecycle changes.
type BotOwnershipStore interface {
	CreateOrRevise(context.Context, BotRevisionRequest) (OwnedBot, AISubmissionRevision, error)
	Retire(context.Context, string, string) (OwnedBot, error)
	ListByOwner(context.Context, string, string, bool) ([]OwnedBot, error)
	ListEligible(context.Context, string) ([]EligibleBot, error)
	ResolveEligible(context.Context, string, []string) ([]ResolvedEligibleBot, error)
}

func (s *InMemoryBotOwnershipStore) ResolveEligible(ctx context.Context, scopeID string, ids []string) ([]ResolvedEligibleBot, error) {
	eligible, err := s.ListEligible(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]EligibleBot, len(eligible))
	for _, item := range eligible {
		byID[item.BotID] = item
	}
	items := make([]ResolvedEligibleBot, 0, len(ids))
	for _, id := range ids {
		item, ok := byID[strings.TrimSpace(id)]
		if !ok {
			return nil, fmt.Errorf("%w: bot is not eligible in scope", ErrBadRequest)
		}
		items = append(items, ResolvedEligibleBot{EligibleBot: item, ArtifactID: s.revisions[item.ActiveRevisionID].ArtifactID})
	}
	return items, nil
}

// ListEligible returns every active bot with a ready active revision in a scope.
func (s *InMemoryBotOwnershipStore) ListEligible(_ context.Context, scopeID string) ([]EligibleBot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]EligibleBot, 0)
	for _, bot := range s.bots {
		if bot.ScopeID != strings.TrimSpace(scopeID) || bot.LifecycleState != BotActive || bot.ActiveRevisionID == "" {
			continue
		}
		revision, ok := s.revisions[bot.ActiveRevisionID]
		if !ok || revision.ValidationState != ValidationReady {
			continue
		}
		items = append(items, EligibleBot{BotID: bot.BotID, ScopeID: bot.ScopeID, BotName: bot.BotName, ActiveRevisionID: bot.ActiveRevisionID})
	}
	return items, nil
}

// InMemoryBotOwnershipStore provides the same serialized invariant for local mode.
type InMemoryBotOwnershipStore struct {
	mu        sync.Mutex
	bots      map[string]OwnedBot
	revisions map[string]AISubmissionRevision
}

// NewInMemoryBotOwnershipStore creates a process-local store with serialized bot lifecycle state.
func NewInMemoryBotOwnershipStore() *InMemoryBotOwnershipStore {
	return &InMemoryBotOwnershipStore{bots: map[string]OwnedBot{}, revisions: map[string]AISubmissionRevision{}}
}

// CreateOrRevise creates a bot or appends an immutable revision while enforcing scope quota and name uniqueness.
func (s *InMemoryBotOwnershipStore) CreateOrRevise(_ context.Context, req BotRevisionRequest) (OwnedBot, AISubmissionRevision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(req.OwnerAccountID) == "" || strings.TrimSpace(req.Scope.ScopeID) == "" || strings.TrimSpace(req.ArtifactID) == "" {
		return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: owner, scope, and artifact are required", ErrBadRequest)
	}
	if req.Scope.MaxActiveBotsPerOwner < 1 {
		return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: max active bots per owner must be positive", ErrBadRequest)
	}
	var bot OwnedBot
	if id := strings.TrimSpace(req.BotID); id != "" {
		var ok bool
		bot, ok = s.bots[id]
		if !ok {
			return OwnedBot{}, AISubmissionRevision{}, ErrBotNotFound
		}
		if bot.OwnerAccountID != req.OwnerAccountID || bot.ScopeID != req.Scope.ScopeID || bot.LifecycleState != BotActive {
			return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: bot is not an active bot owned in this scope", ErrBadRequest)
		}
	} else {
		name := normalizeBotName(req.BotName)
		if name == "" {
			return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: bot name is required", ErrBadRequest)
		}
		active := 0
		for _, candidate := range s.bots {
			if candidate.OwnerAccountID == req.OwnerAccountID && candidate.ScopeID == req.Scope.ScopeID && candidate.LifecycleState == BotActive {
				active++
				if candidate.NormalizedBotName == name {
					return OwnedBot{}, AISubmissionRevision{}, fmt.Errorf("%w: bot name already exists in scope", ErrConflict)
				}
			}
		}
		if active >= req.Scope.MaxActiveBotsPerOwner {
			return OwnedBot{}, AISubmissionRevision{}, ErrBotQuotaExceeded
		}
		bot = OwnedBot{BotID: "bot-" + uuid.NewString(), OwnerAccountID: req.OwnerAccountID, ScopeID: req.Scope.ScopeID, BotName: strings.TrimSpace(req.BotName), NormalizedBotName: name, LifecycleState: BotActive}
	}
	revision := AISubmissionRevision{AISubmissionID: "ai-" + uuid.NewString(), BotID: bot.BotID, ArtifactID: req.ArtifactID, ValidationState: req.ValidationState, CreatedAt: time.Now().UTC()}
	if revision.ValidationState == "" {
		revision.ValidationState = ValidationReady
	}
	bot.ActiveRevisionID = revision.AISubmissionID
	s.bots[bot.BotID] = bot
	s.revisions[revision.AISubmissionID] = revision
	return bot, revision, nil
}

// Retire marks an owner-owned bot as retired without deleting its identity or revisions.
func (s *InMemoryBotOwnershipStore) Retire(_ context.Context, ownerAccountID, botID string) (OwnedBot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	bot, ok := s.bots[strings.TrimSpace(botID)]
	if !ok {
		return OwnedBot{}, ErrBotNotFound
	}
	if bot.OwnerAccountID != strings.TrimSpace(ownerAccountID) {
		return OwnedBot{}, fmt.Errorf("%w: bot is not owned by principal", ErrBadRequest)
	}
	bot.LifecycleState = BotRetired
	s.bots[bot.BotID] = bot
	return bot, nil
}

// ListByOwner returns bots owned by an account in a scope, optionally including retired bots.
func (s *InMemoryBotOwnershipStore) ListByOwner(_ context.Context, ownerAccountID, scopeID string, includeRetired bool) ([]OwnedBot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]OwnedBot, 0)
	for _, bot := range s.bots {
		if bot.OwnerAccountID == ownerAccountID && bot.ScopeID == scopeID && (includeRetired || bot.LifecycleState == BotActive) {
			items = append(items, bot)
		}
	}
	return items, nil
}

func normalizeBotName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}
