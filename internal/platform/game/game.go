package game

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

type Player struct {
	PlayerID string `json:"player_id"`
	AIID     string `json:"ai_id"`
}

type DecisionMode = contract.DecisionMode

const (
	Sequential   = contract.Sequential
	Simultaneous = contract.Simultaneous
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

type MatchStatus = contract.MatchStatus

const (
	StatusStarting     = contract.StatusStarting
	StatusInitializing = contract.StatusInitializing
	StatusRunning      = contract.StatusRunning
	StatusFinishing    = contract.StatusFinishing
	StatusCompleted    = contract.StatusCompleted
	StatusFailed       = contract.StatusFailed
	StatusCanceled     = contract.StatusCanceled
)

type ActionStatus = contract.ActionStatus
type Placement = contract.Placement
type MatchResult = contract.MatchResult
type PlayerSnapshot = contract.PlayerSnapshot
type Snapshot = contract.Snapshot
type ExportedPlayerSnapshot = contract.ExportedPlayerSnapshot
type ExportedSnapshot = contract.ExportedSnapshot

type Master interface {
	Metadata() catalog.GameMetadata
	Init(context.Context) (InitState, error)
	NextStep(context.Context) (*DecisionStep, error)
	NormalizeAction(DecisionRequest, ActionStatus) ActionStatus
	ApplyStep(context.Context, DecisionStep, []ActionStatus) error
	VisibleState(playerID string) json.RawMessage
	Snapshot() Snapshot
	ExportedSnapshot() ExportedSnapshot
	Result() MatchResult
}
