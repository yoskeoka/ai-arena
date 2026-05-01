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

type DecisionWindow struct {
	Turn     int
	Mode     DecisionMode
	Requests []DecisionRequest
}

type ActionOutcome struct {
	PlayerID      string          `json:"player_id"`
	Outcome       string          `json:"outcome"`
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
	VisibleState json.RawMessage `json:"visible_state,omitempty"`
	LastOutcome  ActionOutcome   `json:"last_outcome"`
	StderrBytes  int             `json:"stderr_bytes"`
}

type Snapshot struct {
	MatchID   string                    `json:"match_id"`
	Turn      int                       `json:"turn"`
	Status    string                    `json:"status"`
	GameState json.RawMessage           `json:"game_state,omitempty"`
	PerPlayer map[string]PlayerSnapshot `json:"per_player"`
}

type ExportedPlayerSnapshot struct {
	PlayerID    string        `json:"player_id"`
	LastOutcome ActionOutcome `json:"last_outcome"`
}

type ExportedSnapshot struct {
	MatchID     string                   `json:"match_id"`
	Turn        int                      `json:"turn"`
	Status      string                   `json:"status"`
	PublicState json.RawMessage          `json:"public_state,omitempty"`
	Players     []ExportedPlayerSnapshot `json:"players"`
}

type Master interface {
	Metadata() catalog.GameMetadata
	Init(context.Context) (InitState, error)
	NextDecision(context.Context) (*DecisionWindow, error)
	ApplyDecision(context.Context, DecisionWindow, []ActionOutcome) error
	Snapshot() Snapshot
	ExportedSnapshot() ExportedSnapshot
	Result() MatchResult
}
