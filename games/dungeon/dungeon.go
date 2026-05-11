package dungeon

import (
	"encoding/json"
	"fmt"
	"time"
)

// Match owns one in-memory dungeon match lifecycle.
type Match struct {
	meta        Metadata
	ruleset     Ruleset
	layout      GeneratedLayout
	playerOrder []string
	state       matchState
	rngSeed     string
}

// SupportedRulesets returns the ruleset identifiers accepted by this package.
func SupportedRulesets() []string {
	return []string{RulesetFixedMapV1, RulesetSeededMazeV1}
}

// MetadataForSelection resolves metadata plus static rules for one game selection.
func MetadataForSelection(gameVersion, ruleset string) (Metadata, Ruleset, error) {
	if gameVersion != GameVersion {
		return Metadata{}, Ruleset{}, fmt.Errorf("dungeon: unsupported game version %q", gameVersion)
	}
	switch ruleset {
	case RulesetFixedMapV1:
		return Metadata{GameID: GameID, GameVersion: GameVersion, RulesetVersion: ruleset}, fixedMapRuleset(), nil
	case RulesetSeededMazeV1:
		return Metadata{GameID: GameID, GameVersion: GameVersion, RulesetVersion: ruleset}, seededMazeBaseRuleset(), nil
	default:
		return Metadata{}, Ruleset{}, fmt.Errorf("dungeon: unsupported ruleset %q", ruleset)
	}
}

// New creates a fresh dungeon match from configuration.
func New(cfg Config) (*Match, error) {
	meta, ruleset, err := MetadataForSelection(cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}
	rngSeed, err := normalizeSeedOrGenerate(cfg.RNGSeed)
	if err != nil {
		return nil, err
	}
	layout, err := buildLayout(cfg.Ruleset, rngSeed)
	if err != nil {
		return nil, err
	}
	state, playerOrder, err := buildInitialState(layout, append([]string(nil), cfg.PlayerIDs...))
	if err != nil {
		return nil, err
	}
	match := newMatch(meta, ruleset, layout, playerOrder, state, rngSeed)
	match.state.refreshDiscoveries(match.ruleset, match.layout, match.playerOrder)
	return match, nil
}

// NewFromFullState restores a dungeon match from a previously captured full state.
func NewFromFullState(cfg Config, state FullState) (*Match, error) {
	meta, ruleset, err := MetadataForSelection(cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}
	rngSeed := cfg.RNGSeed
	if rngSeed == "" {
		rngSeed = state.RNGSeed
	}
	rngSeed, err = normalizeSeed(rngSeed)
	if err != nil {
		return nil, err
	}
	layout, err := buildLayout(cfg.Ruleset, rngSeed)
	if err != nil {
		return nil, err
	}
	resumedState, playerOrder, err := restoreMatchState(cfg, ruleset, layout, state, rngSeed)
	if err != nil {
		return nil, err
	}
	match := newMatch(meta, ruleset, layout, playerOrder, resumedState, rngSeed)
	match.state.refreshDiscoveries(match.ruleset, match.layout, match.playerOrder)
	return match, nil
}

func newMatch(meta Metadata, ruleset Ruleset, layout GeneratedLayout, playerOrder []string, state matchState, rngSeed string) *Match {
	return &Match{
		meta:        meta,
		ruleset:     ruleset,
		layout:      layout,
		playerOrder: playerOrder,
		state:       state,
		rngSeed:     rngSeed,
	}
}

func fixedMapRuleset() Ruleset {
	return Ruleset{
		MapID:        RulesetFixedMapV1,
		MaxTurns:     16,
		ViewRadius:   2,
		GoalBonuses:  []int{100, 50, 25, 10},
		TurnDeadline: 100 * time.Millisecond,
	}
}

func seededMazeBaseRuleset() Ruleset {
	return Ruleset{
		MapID:        RulesetSeededMazeV1,
		MaxTurns:     seededMazeMaxTurns,
		ViewRadius:   2,
		GoalBonuses:  append([]int(nil), seededGoalBonuses...),
		TurnDeadline: 100 * time.Millisecond,
	}
}

// Metadata returns the resolved game metadata for this match.
func (m *Match) Metadata() Metadata {
	return m.meta
}

// Ruleset returns a defensive copy of the static rules for this match.
func (m *Match) Ruleset() Ruleset {
	return cloneRuleset(m.ruleset)
}

// Layout returns a defensive copy of the generated layout for this match.
func (m *Match) Layout() GeneratedLayout {
	return cloneLayout(m.layout)
}

// Terminal reports whether the match has already ended.
func (m *Match) Terminal() bool {
	return m.state.terminal(m.ruleset, m.playerOrder)
}

// Turn returns the number of turns already applied.
func (m *Match) Turn() int {
	return m.state.turn
}

// PendingPlayerIDs returns players that still need an action in the current turn.
func (m *Match) PendingPlayerIDs() []string {
	if m.Terminal() {
		return nil
	}
	playerIDs := make([]string, 0, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		if m.state.playerStates[playerID].FinishedTurn == nil {
			playerIDs = append(playerIDs, playerID)
		}
	}
	return playerIDs
}

// CurrentVisibleState builds the current player-specific observation payload.
func (m *Match) CurrentVisibleState(playerID string) (VisibleState, error) {
	return m.state.visibleState(m.ruleset, m.layout, m.playerOrder, playerID)
}

// FullState returns the full resumable snapshot for this match.
func (m *Match) FullState() FullState {
	return m.state.fullState(m.ruleset, m.layout, m.playerOrder, m.rngSeed)
}

// PublicState returns the spectator-facing snapshot for this match.
func (m *Match) PublicState() PublicState {
	return m.state.publicState(m.ruleset, m.layout, m.playerOrder, m.rngSeed)
}

// LegalActionHint returns a JSON Schema-like hint for supported action payloads.
func (m *Match) LegalActionHint() json.RawMessage {
	return mustJSON(map[string]any{
		"type": "object",
		"oneOf": []map[string]any{
			{
				"type":     "object",
				"required": []string{"action", "direction"},
				"properties": map[string]any{
					"action": map[string]any{"type": "string", "const": "move"},
					"direction": map[string]any{
						"type": "string",
						"enum": []string{"up", "down", "left", "right"},
					},
				},
			},
			{
				"type":     "object",
				"required": []string{"action"},
				"properties": map[string]any{
					"action": map[string]any{"type": "string", "const": "wait"},
				},
			},
		},
	})
}

// Apply resolves one turn of actions and advances the match state.
func (m *Match) Apply(actions map[string]Action) error {
	if m.Terminal() {
		return fmt.Errorf("dungeon: match is already terminal")
	}

	activePlayers := m.PendingPlayerIDs()
	nextPositions := make(map[string]Position, len(activePlayers))
	for _, playerID := range activePlayers {
		current := m.state.playerStates[playerID].position()
		action := actions[playerID]
		if action.Action == "" {
			action = Action{Action: "wait"}
		}
		if !m.CanApply(playerID, action) {
			action = Action{Action: "wait"}
		}
		nextPositions[playerID] = m.resolvePosition(current, action)
	}

	for _, playerID := range activePlayers {
		player := m.state.playerStates[playerID]
		target := nextPositions[playerID]
		player.X = target.X
		player.Y = target.Y
		m.state.playerStates[playerID] = player
	}

	for chestID, chest := range chestsCopy(m.state.uncollectedChests) {
		claimants := make([]string, 0, len(activePlayers))
		for _, playerID := range activePlayers {
			player := m.state.playerStates[playerID]
			if player.X == chest.X && player.Y == chest.Y {
				claimants = append(claimants, playerID)
			}
		}
		if len(claimants) == 0 {
			continue
		}
		share := chest.Points / len(claimants)
		for _, playerID := range claimants {
			player := m.state.playerStates[playerID]
			player.ChestPoints += share
			player.Score += share
			m.state.playerStates[playerID] = player
		}
		delete(m.state.uncollectedChests, chestID)
		for _, known := range m.state.discoveredChests {
			delete(known, chestID)
		}
	}

	for _, playerID := range activePlayers {
		player := m.state.playerStates[playerID]
		if player.FinishedTurn == nil && player.X == m.layout.Goal.X && player.Y == m.layout.Goal.Y {
			finishedTurn := m.state.turn + 1
			player.FinishedTurn = &finishedTurn
			m.state.playerStates[playerID] = player
		}
	}

	m.state.turn++
	m.state.applyGoalBonuses(m.ruleset, m.playerOrder)
	m.state.refreshDiscoveries(m.ruleset, m.layout, m.playerOrder)
	return nil
}

// CanApply reports whether an action is legal for the player's current state.
func (m *Match) CanApply(playerID string, action Action) bool {
	player, ok := m.state.playerStates[playerID]
	if !ok {
		return false
	}
	if player.FinishedTurn != nil {
		return action.Action == "wait"
	}
	switch action.Action {
	case "wait":
		return action.Direction == ""
	case "move":
		next, ok := step(player.position(), action.Direction)
		return ok && m.state.isWalkable(m.layout, next)
	default:
		return false
	}
}

// ParseAction decodes and validates one dungeon action payload.
func ParseAction(raw json.RawMessage) (Action, error) {
	var action Action
	if err := json.Unmarshal(raw, &action); err != nil {
		return Action{}, fmt.Errorf("decode dungeon action: %w", err)
	}
	switch action.Action {
	case "wait":
		if action.Direction != "" {
			return Action{}, fmt.Errorf("wait action must not include direction")
		}
	case "move":
		switch action.Direction {
		case "up", "down", "left", "right":
		default:
			return Action{}, fmt.Errorf("move action requires valid direction")
		}
	default:
		return Action{}, fmt.Errorf("unsupported action %q", action.Action)
	}
	return action, nil
}

// Placements computes final rank ordering from the current scoreboard.
func (m *Match) Placements() []Placement {
	scores := m.state.scoreboard(m.playerOrder)
	placements := make([]Placement, 0, len(scores))
	lastScore := 0
	lastPlace := 0
	for i, player := range scores {
		if i == 0 || player.Score != lastScore {
			lastPlace = i + 1
			lastScore = player.Score
		}
		placements = append(placements, Placement{
			PlayerID: player.PlayerID,
			Place:    lastPlace,
		})
	}
	return placements
}

// ShortestPath returns the layout path between two positions when one exists.
func (m *Match) ShortestPath(from, to Position) ([]Position, bool) {
	return shortestPath(m.layout.Tiles, from, to)
}

// SpawnPoints returns the initial spawn positions for the current layout.
func (m *Match) SpawnPoints() []Position {
	return append([]Position(nil), m.layout.SpawnPoints...)
}

// UncollectedChests returns the remaining chest states in stable order.
func (m *Match) UncollectedChests() []ChestState {
	return chestsFromMap(m.state.uncollectedChests)
}

func (m *Match) scoreboardWithPositions() []PlayerState {
	return m.state.scoreboardWithPositions(m.playerOrder)
}

func (m *Match) resolvePosition(current Position, action Action) Position {
	if action.Action != "move" {
		return current
	}
	next, ok := step(current, action.Direction)
	if !ok || !m.state.isWalkable(m.layout, next) {
		return current
	}
	return next
}
