package game

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

// Player identifies one participating AI player.
type Player struct {
	PlayerID string `json:"player_id"`
	AIID     string `json:"ai_id"`
}

// DecisionMode aliases the shared contract decision mode.
type DecisionMode = contract.DecisionMode

const (
	// Sequential aliases the sequential decision mode.
	Sequential = contract.Sequential
	// Simultaneous aliases the simultaneous decision mode.
	Simultaneous = contract.Simultaneous
)

// InitState contains per-player initialization payloads.
type InitState struct {
	PerPlayer map[string]json.RawMessage
}

// DecisionRequest contains the visible state and deadline for one player turn.
type DecisionRequest struct {
	PlayerID        string
	VisibleState    json.RawMessage
	LegalActionHint json.RawMessage
	Deadline        time.Duration
}

// DecisionStep describes one game-master step that players must answer.
type DecisionStep struct {
	Turn     int
	Mode     DecisionMode
	Requests []DecisionRequest
}

// MatchStatus aliases the shared contract match lifecycle status.
type MatchStatus = contract.MatchStatus

const (
	// StatusStarting aliases the starting lifecycle status.
	StatusStarting = contract.StatusStarting
	// StatusInitializing aliases the initializing lifecycle status.
	StatusInitializing = contract.StatusInitializing
	// StatusRunning aliases the running lifecycle status.
	StatusRunning = contract.StatusRunning
	// StatusFinishing aliases the finishing lifecycle status.
	StatusFinishing = contract.StatusFinishing
	// StatusCompleted aliases the completed lifecycle status.
	StatusCompleted = contract.StatusCompleted
	// StatusFailed aliases the failed lifecycle status.
	StatusFailed = contract.StatusFailed
	// StatusCanceled aliases the canceled lifecycle status.
	StatusCanceled = contract.StatusCanceled
)

// ActionStatus aliases the shared per-player turn outcome.
type ActionStatus = contract.ActionStatus

// Placement aliases the shared match placement record.
type Placement = contract.Placement

// MatchResult aliases the shared final match result.
type MatchResult = contract.MatchResult

// PlayerSnapshot aliases the shared internal player snapshot.
type PlayerSnapshot = contract.PlayerSnapshot

// Snapshot aliases the shared internal match snapshot.
type Snapshot = contract.Snapshot

// ExportedPlayerSnapshot aliases the shared public player snapshot.
type ExportedPlayerSnapshot = contract.ExportedPlayerSnapshot

// ExportedSnapshot aliases the shared public match snapshot.
type ExportedSnapshot = contract.ExportedSnapshot

// Master defines the game-master surface used by the platform core.
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
