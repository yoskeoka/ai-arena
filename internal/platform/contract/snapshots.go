package contract

import "encoding/json"

type Placement struct {
	PlayerID string `json:"player_id"`
	Place    int    `json:"place"`
}

type MatchResult struct {
	Placements []Placement `json:"placements"`
}

type PlayerSnapshot struct {
	VisibleState     json.RawMessage `json:"visible_state,omitempty"`
	LastActionStatus ActionStatus    `json:"last_action_status"`
	StderrBytes      int             `json:"stderr_bytes"`
}

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

type ExportedPlayerSnapshot struct {
	PlayerID         string       `json:"player_id"`
	LastActionStatus ActionStatus `json:"last_action_status"`
}

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
