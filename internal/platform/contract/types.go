package contract

import (
	"github.com/yoskeoka/ai-arena/gamemaster"
)

// GameMetadata identifies one game build and ruleset selection.
type GameMetadata = gamemaster.GameMetadata

// DecisionMode describes whether a turn is sequential or simultaneous.
type DecisionMode = gamemaster.DecisionMode

const (
	// Sequential runs exactly one player request at a time.
	Sequential DecisionMode = gamemaster.Sequential
	// Simultaneous runs all player requests for a turn together.
	Simultaneous DecisionMode = gamemaster.Simultaneous
)

// MatchStatus describes the current lifecycle phase of a match.
type MatchStatus = gamemaster.MatchStatus

const (
	// StatusStarting marks a match before initialization begins.
	StatusStarting MatchStatus = gamemaster.StatusStarting
	// StatusInitializing marks the initialization handshake phase.
	StatusInitializing MatchStatus = gamemaster.StatusInitializing
	// StatusRunning marks an active match with pending turns.
	StatusRunning MatchStatus = gamemaster.StatusRunning
	// StatusFinishing marks a match after turn processing has ended.
	StatusFinishing MatchStatus = gamemaster.StatusFinishing
	// StatusCompleted marks a successfully finished match.
	StatusCompleted MatchStatus = gamemaster.StatusCompleted
	// StatusFailed marks a match that ended with an error.
	StatusFailed MatchStatus = gamemaster.StatusFailed
	// StatusCanceled marks a match canceled by context shutdown.
	StatusCanceled MatchStatus = gamemaster.StatusCanceled
)

// ActionDecision describes whether a submitted action was accepted.
type ActionDecision = gamemaster.ActionDecision

const (
	// ActionAccepted means the player's action payload was accepted.
	ActionAccepted ActionDecision = gamemaster.ActionAccepted
	// ActionNoAction means the player produced no usable action.
	ActionNoAction ActionDecision = gamemaster.ActionNoAction
)

// FailureReason classifies why a player action was not accepted.
type FailureReason = gamemaster.FailureReason

const (
	// ReasonTimeout indicates that the player missed the deadline.
	ReasonTimeout FailureReason = gamemaster.ReasonTimeout
	// ReasonMalformed indicates malformed protocol data.
	ReasonMalformed FailureReason = gamemaster.ReasonMalformed
	// ReasonMismatchedID indicates a response id mismatch.
	ReasonMismatchedID FailureReason = gamemaster.ReasonMismatchedID
	// ReasonLateResponse indicates that a response arrived after timeout.
	ReasonLateResponse FailureReason = gamemaster.ReasonLateResponse
	// ReasonIllegalAction indicates that the action payload was semantically invalid.
	ReasonIllegalAction FailureReason = gamemaster.ReasonIllegalAction
	// ReasonRuntimeStop indicates that the player runtime stopped unexpectedly.
	ReasonRuntimeStop FailureReason = gamemaster.ReasonRuntimeStop
)

// ActionStatus records the normalized action result for one player turn.
type ActionStatus = gamemaster.ActionStatus

// ValidateActionStatus checks that an action status carries a consistent payload.
func ValidateActionStatus(status ActionStatus) error {
	return gamemaster.ValidateActionStatus(status)
}
