package contract

import (
	"encoding/json"
	"fmt"
)

type GameMetadata struct {
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}

type DecisionMode string

const (
	Sequential   DecisionMode = "sequential"
	Simultaneous DecisionMode = "simultaneous"
)

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

type ActionDecision string

const (
	ActionAccepted ActionDecision = "accepted"
	ActionNoAction ActionDecision = "no_action"
)

type FailureReason string

const (
	ReasonTimeout       FailureReason = "invalid-timeout"
	ReasonMalformed     FailureReason = "invalid-protocol-malformed"
	ReasonMismatchedID  FailureReason = "invalid-protocol-mismatched-id"
	ReasonLateResponse  FailureReason = "invalid-protocol-late-response"
	ReasonIllegalAction FailureReason = "invalid-illegal-action"
	ReasonRuntimeStop   FailureReason = "runtime-stopped"
)

type ActionStatus struct {
	PlayerID      string          `json:"player_id"`
	ActionStatus  ActionDecision  `json:"action_status"`
	FailureReason FailureReason   `json:"failure_reason,omitempty"`
	Action        json.RawMessage `json:"action,omitempty"`
}

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
