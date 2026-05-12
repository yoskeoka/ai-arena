package contract

import "encoding/json"

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
