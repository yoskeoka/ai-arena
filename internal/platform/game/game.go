package game

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
)

type Player struct {
	PlayerID string `json:"player_id"`
	AIID     string `json:"ai_id"`
}

type DecisionMode string

const (
	Sequential   DecisionMode = "sequential"
	Simultaneous DecisionMode = "simultaneous"
)

type InitState struct {
	PerPlayer map[string]json.RawMessage
}

type DecisionRequest struct {
	PlayerID        string
	VisibleState    json.RawMessage
	LegalActionHint json.RawMessage
	Deadline        time.Duration
}

type DecisionStep struct {
	Turn     int
	Mode     DecisionMode
	Requests []DecisionRequest
}

type MatchStatus string

const (
	StatusStarting     MatchStatus = "starting"
	StatusInitializing MatchStatus = "initializing"
	StatusRunning      MatchStatus = "running"
	StatusFinishing    MatchStatus = "finishing"
	StatusCompleted    MatchStatus = "completed"
	StatusFailed       MatchStatus = "failed"
	StatusCanceled     MatchStatus = "canceled"
)

type ActionStatus struct {
	PlayerID      string          `json:"player_id"`
	ActionStatus  string          `json:"action_status"`
	FailureReason string          `json:"failure_reason,omitempty"`
	Action        json.RawMessage `json:"action,omitempty"`
}

type Placement struct {
	PlayerID string `json:"player_id"`
	Place    int    `json:"place"`
}

type MatchResult struct {
	Placements []Placement `json:"placements"`
}

type PlayerSnapshot struct {
	VisibleState     json.RawMessage `json:"visible_state,omitempty"`
	LastActionStatus ActionStatus    `json:"last_action_status"`
	StderrBytes      int             `json:"stderr_bytes"`
}

type Snapshot struct {
	MatchID        string                    `json:"match_id"`
	GameID         string                    `json:"game_id,omitempty"`
	GameVersion    string                    `json:"game_version,omitempty"`
	RulesetVersion string                    `json:"ruleset_version,omitempty"`
	Turn           int                       `json:"turn"`
	Status         string                    `json:"status"`
	GameState      json.RawMessage           `json:"game_state,omitempty"`
	PerPlayer      map[string]PlayerSnapshot `json:"per_player"`
}

type ExportedPlayerSnapshot struct {
	PlayerID         string       `json:"player_id"`
	LastActionStatus ActionStatus `json:"last_action_status"`
}

type ExportedSnapshot struct {
	MatchID        string                   `json:"match_id"`
	GameID         string                   `json:"game_id,omitempty"`
	GameVersion    string                   `json:"game_version,omitempty"`
	RulesetVersion string                   `json:"ruleset_version,omitempty"`
	Turn           int                      `json:"turn"`
	Status         string                   `json:"status"`
	PublicState    json.RawMessage          `json:"public_state,omitempty"`
	Players        []ExportedPlayerSnapshot `json:"players"`
}

type Master interface {
	Metadata() catalog.GameMetadata
	Init(context.Context) (InitState, error)
	NextStep(context.Context) (*DecisionStep, error)
	NormalizeAction(DecisionRequest, ActionStatus) ActionStatus
	ApplyStep(context.Context, DecisionStep, []ActionStatus) error
	Snapshot() Snapshot
	ExportedSnapshot() ExportedSnapshot
	Result() MatchResult
}
