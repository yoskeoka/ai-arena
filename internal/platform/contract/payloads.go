package contract

import "encoding/json"

// InitParams is the payload sent to an AI player's init request.
type InitParams struct {
	MatchID        string          `json:"match_id"`
	PlayerID       string          `json:"player_id"`
	GameID         string          `json:"game_id"`
	GameVersion    string          `json:"game_version"`
	RulesetVersion string          `json:"ruleset_version"`
	DeadlineMS     int64           `json:"deadline_ms"`
	State          json.RawMessage `json:"state"`
}

// TurnParams is the payload sent to an AI player's turn request.
type TurnParams struct {
	Turn            int             `json:"turn"`
	VisibleState    json.RawMessage `json:"visible_state"`
	LegalActionHint json.RawMessage `json:"legal_action_hint"`
	DeadlineMS      int64           `json:"deadline_ms"`
}

// GameOverParams is the payload sent during final shutdown notification.
type GameOverParams struct {
	MatchID           string          `json:"match_id"`
	FinalVisibleState json.RawMessage `json:"final_visible_state"`
	Summary           any             `json:"summary"`
	ShutdownAfterMS   int64           `json:"shutdown_after_ms"`
}
