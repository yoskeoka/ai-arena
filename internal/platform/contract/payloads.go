package contract

import "encoding/json"

type InitParams struct {
	MatchID        string          `json:"match_id"`
	PlayerID       string          `json:"player_id"`
	GameID         string          `json:"game_id"`
	GameVersion    string          `json:"game_version"`
	RulesetVersion string          `json:"ruleset_version"`
	DeadlineMS     int64           `json:"deadline_ms"`
	State          json.RawMessage `json:"state"`
}

type TurnParams struct {
	Turn            int             `json:"turn"`
	VisibleState    json.RawMessage `json:"visible_state"`
	LegalActionHint json.RawMessage `json:"legal_action_hint"`
	DeadlineMS      int64           `json:"deadline_ms"`
}

type GameOverParams struct {
	MatchID           string          `json:"match_id"`
	FinalVisibleState json.RawMessage `json:"final_visible_state"`
	Summary           any             `json:"summary"`
	ShutdownAfterMS   int64           `json:"shutdown_after_ms"`
}
