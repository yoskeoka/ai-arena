package contract

import (
	"encoding/json"
	"fmt"
)

// GameMetadata identifies one game build and ruleset selection.
type GameMetadata struct {
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}

// DecisionMode describes whether a turn is sequential or simultaneous.
type DecisionMode string

const (
	// Sequential runs exactly one player request at a time.
	Sequential DecisionMode = "sequential"
	// Simultaneous runs all player requests for a turn together.
	Simultaneous DecisionMode = "simultaneous"
)

// MatchStatus describes the current lifecycle phase of a match.
type MatchStatus string

const (
	// StatusStarting marks a match before initialization begins.
	StatusStarting MatchStatus = "starting"
	// StatusInitializing marks the initialization handshake phase.
	StatusInitializing MatchStatus = "initializing"
	// StatusRunning marks an active match with pending turns.
	StatusRunning MatchStatus = "running"
	// StatusFinishing marks a match after turn processing has ended.
	StatusFinishing MatchStatus = "finishing"
	// StatusCompleted marks a successfully finished match.
	StatusCompleted MatchStatus = "completed"
	// StatusFailed marks a match that ended with an error.
	StatusFailed MatchStatus = "failed"
	// StatusCanceled marks a match canceled by context shutdown.
	StatusCanceled MatchStatus = "canceled"
)

// ActionDecision describes whether a submitted action was accepted.
type ActionDecision string

const (
	// ActionAccepted means the player's action payload was accepted.
	ActionAccepted ActionDecision = "accepted"
	// ActionNoAction means the player produced no usable action.
	ActionNoAction ActionDecision = "no_action"
)

// FailureReason classifies why a player action was not accepted.
type FailureReason string

const (
	// ReasonTimeout indicates that the player missed the deadline.
	ReasonTimeout FailureReason = "invalid-timeout"
	// ReasonMalformed indicates malformed protocol data.
	ReasonMalformed FailureReason = "invalid-protocol-malformed"
	// ReasonMismatchedID indicates a response id mismatch.
	ReasonMismatchedID FailureReason = "invalid-protocol-mismatched-id"
	// ReasonLateResponse indicates that a response arrived after timeout.
	ReasonLateResponse FailureReason = "invalid-protocol-late-response"
	// ReasonIllegalAction indicates that the action payload was semantically invalid.
	ReasonIllegalAction FailureReason = "invalid-illegal-action"
	// ReasonRuntimeStop indicates that the player runtime stopped unexpectedly.
	ReasonRuntimeStop FailureReason = "runtime-stopped"
)

// ActionStatus records the normalized action result for one player turn.
type ActionStatus struct {
	PlayerID      string          `json:"player_id"`
	ActionStatus  ActionDecision  `json:"action_status"`
	FailureReason FailureReason   `json:"failure_reason,omitempty"`
	Action        json.RawMessage `json:"action,omitempty"`
}

// ValidateActionStatus checks that an action status carries a consistent payload.
func ValidateActionStatus(status ActionStatus) error {
	if status.PlayerID == "" {
		return fmt.Errorf("player_id is required")
	}

	switch status.ActionStatus {
	case ActionAccepted:
		if status.FailureReason != "" {
			return fmt.Errorf("accepted action must not carry failure_reason")
		}
		if len(status.Action) == 0 {
			return fmt.Errorf("accepted action must carry action payload")
		}
	case ActionNoAction:
		if len(status.Action) != 0 {
			return fmt.Errorf("no_action must not carry action payload")
		}
	default:
		return fmt.Errorf("unknown action_status %q", status.ActionStatus)
	}
	return nil
}
