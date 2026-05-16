package game

import (
	"context"
	"encoding/json"

	publicgm "github.com/yoskeoka/ai-arena/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
)

// Player identifies one participating AI player.
type Player = publicgm.Player

// DecisionMode aliases the shared contract decision mode.
type DecisionMode = publicgm.DecisionMode

const (
	// Sequential aliases the sequential decision mode.
	Sequential = publicgm.Sequential
	// Simultaneous aliases the simultaneous decision mode.
	Simultaneous = publicgm.Simultaneous
)

// InitState contains per-player initialization payloads.
type InitState = publicgm.InitState

// DecisionRequest contains the visible state and deadline for one player turn.
type DecisionRequest = publicgm.DecisionRequest

// DecisionStep describes one game-master step that players must answer.
type DecisionStep = publicgm.DecisionStep

// MatchStatus aliases the shared contract match lifecycle status.
type MatchStatus = publicgm.MatchStatus

const (
	// StatusStarting aliases the starting lifecycle status.
	StatusStarting = publicgm.StatusStarting
	// StatusInitializing aliases the initializing lifecycle status.
	StatusInitializing = publicgm.StatusInitializing
	// StatusRunning aliases the running lifecycle status.
	StatusRunning = publicgm.StatusRunning
	// StatusFinishing aliases the finishing lifecycle status.
	StatusFinishing = publicgm.StatusFinishing
	// StatusCompleted aliases the completed lifecycle status.
	StatusCompleted = publicgm.StatusCompleted
	// StatusFailed aliases the failed lifecycle status.
	StatusFailed = publicgm.StatusFailed
	// StatusCanceled aliases the canceled lifecycle status.
	StatusCanceled = publicgm.StatusCanceled
)

// ActionStatus aliases the shared per-player turn outcome.
type ActionStatus = publicgm.ActionStatus

// Placement aliases the shared match placement record.
type Placement = publicgm.Placement

// MatchResult aliases the shared final match result.
type MatchResult = publicgm.MatchResult

// PlayerSnapshot aliases the shared internal player snapshot.
type PlayerSnapshot = publicgm.PlayerSnapshot

// Snapshot aliases the shared internal match snapshot.
type Snapshot = publicgm.Snapshot

// ExportedPlayerSnapshot aliases the shared public player snapshot.
type ExportedPlayerSnapshot = publicgm.ExportedPlayerSnapshot

// ExportedSnapshot aliases the shared public match snapshot.
type ExportedSnapshot = publicgm.ExportedSnapshot

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
