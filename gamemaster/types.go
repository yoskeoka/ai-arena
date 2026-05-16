package gamemaster

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// MethodMetadata requests the game master's metadata tuple.
	MethodMetadata = "metadata"
	// MethodInitializeMatch starts or resumes a match.
	MethodInitializeMatch = "initialize_match"
	// MethodNextDecisionStep requests the next decision step.
	MethodNextDecisionStep = "next_decision_step"
	// MethodNormalizeAction asks the game master to normalize one action.
	MethodNormalizeAction = "normalize_action"
	// MethodApplyDecisionResults applies one resolved decision step.
	MethodApplyDecisionResults = "apply_decision_results"
	// MethodCurrentSnapshot requests the authoritative resume snapshot.
	MethodCurrentSnapshot = "current_snapshot"
	// MethodCurrentExportedSnapshot requests the public snapshot.
	MethodCurrentExportedSnapshot = "current_exported_snapshot"
	// MethodCurrentResult requests the current match result.
	MethodCurrentResult = "current_result"
	// MethodShutdown asks the game master to shut down cleanly.
	MethodShutdown = "shutdown"
)

// GameMetadata identifies one game build and ruleset selection.
type GameMetadata struct {
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}

// Player identifies one participating AI player.
type Player struct {
	PlayerID string `json:"player_id"`
	AIID     string `json:"ai_id"`
}

// DecisionMode describes whether a turn is sequential or simultaneous.
type DecisionMode string

const (
	// Sequential runs exactly one player request at a time.
	Sequential DecisionMode = "sequential"
	// Simultaneous runs all player requests for a turn together.
	Simultaneous DecisionMode = "simultaneous"
)

// InitState contains per-player initialization payloads.
type InitState struct {
	PerPlayer map[string]json.RawMessage `json:"per_player"`
}

// DecisionRequest contains the visible state and deadline for one player turn.
type DecisionRequest struct {
	PlayerID        string          `json:"player_id"`
	VisibleState    json.RawMessage `json:"visible_state,omitempty"`
	LegalActionHint json.RawMessage `json:"legal_action_hint,omitempty"`
	Deadline        time.Duration   `json:"deadline_ms"`
}

// DecisionStep describes one game-master step that players must answer.
type DecisionStep struct {
	Turn     int               `json:"turn"`
	Mode     DecisionMode      `json:"mode"`
	Requests []DecisionRequest `json:"requests"`
}

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

// Placement records one player's final placement.
type Placement struct {
	PlayerID string `json:"player_id"`
	Place    int    `json:"place"`
}

// MatchResult is the persisted placement summary for a match.
type MatchResult struct {
	Placements []Placement `json:"placements"`
}

// PlayerSnapshot stores the player-scoped portion of a match snapshot.
type PlayerSnapshot struct {
	VisibleState     json.RawMessage `json:"visible_state,omitempty"`
	LastActionStatus ActionStatus    `json:"last_action_status"`
	StderrBytes      int             `json:"stderr_bytes"`
}

// Snapshot stores the full internal match state used for replay and resume.
type Snapshot struct {
	MatchID        string                    `json:"match_id"`
	GameID         string                    `json:"game_id,omitempty"`
	GameVersion    string                    `json:"game_version,omitempty"`
	RulesetVersion string                    `json:"ruleset_version,omitempty"`
	Turn           int                       `json:"turn"`
	Status         MatchStatus               `json:"status"`
	GameState      json.RawMessage           `json:"game_state,omitempty"`
	PerPlayer      map[string]PlayerSnapshot `json:"per_player"`
}

// ExportedPlayerSnapshot stores the public per-player snapshot fields.
type ExportedPlayerSnapshot struct {
	PlayerID         string       `json:"player_id"`
	LastActionStatus ActionStatus `json:"last_action_status"`
}

// ExportedSnapshot stores the public snapshot fields safe to expose externally.
type ExportedSnapshot struct {
	MatchID        string                   `json:"match_id"`
	GameID         string                   `json:"game_id,omitempty"`
	GameVersion    string                   `json:"game_version,omitempty"`
	RulesetVersion string                   `json:"ruleset_version,omitempty"`
	Turn           int                      `json:"turn"`
	Status         MatchStatus              `json:"status"`
	PublicState    json.RawMessage          `json:"public_state,omitempty"`
	Players        []ExportedPlayerSnapshot `json:"players"`
}

// InitializeMatchParams contains the inputs for starting or resuming a match.
type InitializeMatchParams struct {
	Players        []Player  `json:"players"`
	RNGSeed        string    `json:"rng_seed,omitempty"`
	ResumeSnapshot *Snapshot `json:"resume_snapshot,omitempty"`
}

// InitializeMatchResult returns the initial per-player state from a game master.
type InitializeMatchResult struct {
	InitState InitState `json:"init_state"`
}

// NormalizeActionParams carries one action-normalization request.
type NormalizeActionParams struct {
	Request      DecisionRequest `json:"request"`
	ActionStatus ActionStatus    `json:"action_status"`
}

// ApplyDecisionResultsParams carries one resolved step back to the game master.
type ApplyDecisionResultsParams struct {
	Step           DecisionStep   `json:"step"`
	ActionStatuses []ActionStatus `json:"action_statuses"`
}
