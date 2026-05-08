package dungeon

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// GameID is the stable identifier for the dungeon game.
	GameID = "dungeon"
	// GameVersion is the current dungeon game contract version.
	GameVersion = "1.0.0"
	// RulesetFixedMapV1 is the fixed-map ruleset used by this MVP.
	RulesetFixedMapV1 = "fixed-map-v1"
	// BuilderIDSubprocess identifies the local subprocess bot builder.
	BuilderIDSubprocess = "dungeon/local-subprocess"
	// DefaultRNGSeed is the default deterministic seed used by local helpers.
	DefaultRNGSeed int64 = 0
)

const (
	// TileWall marks an impassable wall tile.
	TileWall = "wall"
	// TileFloor marks a traversable floor tile.
	TileFloor = "floor"
	// TileChest marks a chest tile.
	TileChest = "chest"
	// TileGoal marks the goal tile.
	TileGoal = "goal"
)

// Metadata identifies a concrete dungeon game/ruleset selection.
type Metadata struct {
	GameID         string
	GameVersion    string
	RulesetVersion string
}

// Config configures a new or resumed dungeon match.
type Config struct {
	GameVersion string
	Ruleset     string
	PlayerIDs   []string
	RNGSeed     int64
}

// Ruleset describes the fixed map, scoring, and turn limits for a match.
type Ruleset struct {
	MapID          string
	MaxTurns       int
	ViewRadius     int
	ChestPoints    int
	GoalBonuses    []int
	TurnDeadline   time.Duration
	Tiles          []string
	SpawnPoints    []Position
	Goal           Position
	ChestPositions []Position
}

// Position is a tile coordinate in the dungeon grid.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Action is a player command for the current turn.
type Action struct {
	Action    string `json:"action"`
	Direction string `json:"direction,omitempty"`
}

// PlayerState is the score and position state tracked for one player.
type PlayerState struct {
	PlayerID     string `json:"player_id"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Score        int    `json:"score"`
	GoalBonus    int    `json:"goal_bonus"`
	ChestPoints  int    `json:"chest_points"`
	FinishedTurn *int   `json:"finished_turn"`
}

// VisibleTile is one tile in a player's visible viewport.
type VisibleTile struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Tile string `json:"tile"`
}

// VisibleState is the per-player state sent to a bot on its turn.
type VisibleState struct {
	Turn           int           `json:"turn"`
	RemainingTurns int           `json:"remaining_turns"`
	ViewRadius     int           `json:"view_radius"`
	Self           PlayerState   `json:"self"`
	VisibleTiles   []VisibleTile `json:"visible_tiles"`
	KnownGoal      *Position     `json:"known_goal"`
	KnownChests    []Position    `json:"known_chests"`
	Scores         []PlayerState `json:"scores"`
}

// DiscoveryState stores the persistent discoveries known to one player.
type DiscoveryState struct {
	KnownGoal   *Position  `json:"known_goal"`
	KnownChests []Position `json:"known_chests"`
}

// FullState is the resumable full dungeon game state snapshot.
type FullState struct {
	MapID             string                    `json:"map_id"`
	RNGSeed           int64                     `json:"rng_seed"`
	Turn              int                       `json:"turn"`
	MaxTurns          int                       `json:"max_turns"`
	Goal              Position                  `json:"goal"`
	Players           []PlayerState             `json:"players"`
	UncollectedChests []Position                `json:"uncollected_chests"`
	Discovery         map[string]DiscoveryState `json:"discovery"`
}

// PublicState is the spectator-safe state shared outside per-player fog.
type PublicState struct {
	MapID             string        `json:"map_id"`
	RNGSeed           int64         `json:"rng_seed"`
	Turn              int           `json:"turn"`
	MaxTurns          int           `json:"max_turns"`
	Tiles             []string      `json:"tiles"`
	Goal              Position      `json:"goal"`
	Players           []PlayerState `json:"players"`
	UncollectedChests []Position    `json:"uncollected_chests"`
}

// Placement records the final ranking for one player.
type Placement struct {
	PlayerID string
	Place    int
}

// Match owns one running dungeon match instance.
type Match struct {
	meta              Metadata
	ruleset           Ruleset
	playerOrder       []string
	playerStates      map[string]PlayerState
	uncollectedChests map[string]Position
	discoveredGoal    map[string]*Position
	discoveredChests  map[string]map[string]Position
	turn              int
	rngSeed           int64
}

// SupportedRulesets returns the rulesets exposed by this package.
func SupportedRulesets() []string {
	return []string{RulesetFixedMapV1}
}

// MetadataForSelection validates a game/ruleset selection and returns its metadata.
func MetadataForSelection(gameVersion, ruleset string) (Metadata, Ruleset, error) {
	if gameVersion != GameVersion {
		return Metadata{}, Ruleset{}, fmt.Errorf("dungeon: unsupported game version %q", gameVersion)
	}
	if ruleset != RulesetFixedMapV1 {
		return Metadata{}, Ruleset{}, fmt.Errorf("dungeon: unsupported ruleset %q", ruleset)
	}
	rs := fixedMapRuleset()
	return Metadata{
		GameID:         GameID,
		GameVersion:    GameVersion,
		RulesetVersion: RulesetFixedMapV1,
	}, rs, nil
}

// New creates a new dungeon match from the provided config.
func New(cfg Config) (*Match, error) {
	meta, ruleset, err := MetadataForSelection(cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}
	if len(cfg.PlayerIDs) < 2 || len(cfg.PlayerIDs) > len(ruleset.SpawnPoints) {
		return nil, fmt.Errorf("dungeon: player count must be between 2 and %d", len(ruleset.SpawnPoints))
	}

	playerStates := make(map[string]PlayerState, len(cfg.PlayerIDs))
	playerOrder := make([]string, 0, len(cfg.PlayerIDs))
	for i, playerID := range cfg.PlayerIDs {
		if strings.TrimSpace(playerID) == "" {
			return nil, fmt.Errorf("dungeon: player_id is required")
		}
		if _, exists := playerStates[playerID]; exists {
			return nil, fmt.Errorf("dungeon: duplicate player_id %q", playerID)
		}
		spawn := ruleset.SpawnPoints[i]
		playerStates[playerID] = PlayerState{
			PlayerID: playerID,
			X:        spawn.X,
			Y:        spawn.Y,
		}
		playerOrder = append(playerOrder, playerID)
	}

	match := &Match{
		meta:              meta,
		ruleset:           ruleset,
		playerOrder:       playerOrder,
		playerStates:      playerStates,
		uncollectedChests: make(map[string]Position, len(ruleset.ChestPositions)),
		discoveredGoal:    make(map[string]*Position, len(cfg.PlayerIDs)),
		discoveredChests:  make(map[string]map[string]Position, len(cfg.PlayerIDs)),
		rngSeed:           cfg.RNGSeed,
	}
	for _, chest := range ruleset.ChestPositions {
		match.uncollectedChests[posKey(chest)] = chest
	}
	for _, playerID := range playerOrder {
		match.discoveredChests[playerID] = make(map[string]Position)
	}
	match.refreshDiscoveries()
	return match, nil
}

// NewFromFullState restores a dungeon match from a previously saved full state.
func NewFromFullState(cfg Config, state FullState) (*Match, error) {
	match, err := New(cfg)
	if err != nil {
		return nil, err
	}
	if state.MapID != match.ruleset.MapID {
		return nil, fmt.Errorf("dungeon: snapshot map_id %q does not match %q", state.MapID, match.ruleset.MapID)
	}
	if state.MaxTurns != match.ruleset.MaxTurns {
		return nil, fmt.Errorf("dungeon: snapshot max_turns %d does not match ruleset %d", state.MaxTurns, match.ruleset.MaxTurns)
	}
	if state.Turn < 0 || state.Turn > match.ruleset.MaxTurns {
		return nil, fmt.Errorf("dungeon: snapshot turn %d out of range", state.Turn)
	}
	if state.RNGSeed != cfg.RNGSeed {
		return nil, fmt.Errorf("dungeon: snapshot rng_seed %d does not match config %d", state.RNGSeed, cfg.RNGSeed)
	}
	match.turn = state.Turn
	match.rngSeed = state.RNGSeed

	seenPlayers := make(map[string]struct{}, len(state.Players))
	for _, player := range state.Players {
		if _, ok := match.playerStates[player.PlayerID]; !ok {
			return nil, fmt.Errorf("dungeon: snapshot has unknown player %q", player.PlayerID)
		}
		if !match.isWalkable(Position{X: player.X, Y: player.Y}) {
			return nil, fmt.Errorf("dungeon: snapshot player %q has invalid position (%d,%d)", player.PlayerID, player.X, player.Y)
		}
		match.playerStates[player.PlayerID] = clonePlayerState(player)
		seenPlayers[player.PlayerID] = struct{}{}
	}
	if len(seenPlayers) != len(match.playerOrder) {
		return nil, fmt.Errorf("dungeon: snapshot player count does not match config")
	}

	match.uncollectedChests = make(map[string]Position, len(state.UncollectedChests))
	for _, chest := range state.UncollectedChests {
		if !match.isOriginalChest(chest) {
			return nil, fmt.Errorf("dungeon: snapshot chest at (%d,%d) is not in fixed map", chest.X, chest.Y)
		}
		match.uncollectedChests[posKey(chest)] = chest
	}

	match.discoveredGoal = make(map[string]*Position, len(match.playerOrder))
	match.discoveredChests = make(map[string]map[string]Position, len(match.playerOrder))
	for _, playerID := range match.playerOrder {
		match.discoveredChests[playerID] = make(map[string]Position)
		discovery := state.Discovery[playerID]
		if discovery.KnownGoal != nil {
			pos := *discovery.KnownGoal
			match.discoveredGoal[playerID] = &pos
		}
		for _, chest := range discovery.KnownChests {
			if _, ok := match.uncollectedChests[posKey(chest)]; ok {
				match.discoveredChests[playerID][posKey(chest)] = chest
			}
		}
	}
	match.refreshDiscoveries()
	return match, nil
}

// Metadata returns the match metadata.
func (m *Match) Metadata() Metadata {
	return m.meta
}

// Ruleset returns a defensive copy of the match ruleset.
func (m *Match) Ruleset() Ruleset {
	return cloneRuleset(m.ruleset)
}

// Terminal reports whether the match has ended.
func (m *Match) Terminal() bool {
	return m.turn >= m.ruleset.MaxTurns || m.allPlayersFinished()
}

// Turn returns the zero-based number of completed turns.
func (m *Match) Turn() int {
	return m.turn
}

// PendingPlayerIDs returns the players expected to submit actions this turn.
func (m *Match) PendingPlayerIDs() []string {
	if m.Terminal() {
		return nil
	}
	playerIDs := make([]string, 0, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		if m.playerStates[playerID].FinishedTurn == nil {
			playerIDs = append(playerIDs, playerID)
		}
	}
	return playerIDs
}

// CurrentVisibleState returns the visible state for the requested player.
func (m *Match) CurrentVisibleState(playerID string) (VisibleState, error) {
	player, ok := m.playerStates[playerID]
	if !ok {
		return VisibleState{}, fmt.Errorf("dungeon: unknown player %q", playerID)
	}
	turn := m.turn + 1
	remainingTurns := m.ruleset.MaxTurns - m.turn
	if m.Terminal() {
		turn = m.turn
		remainingTurns = 0
	}
	return VisibleState{
		Turn:           turn,
		RemainingTurns: remainingTurns,
		ViewRadius:     m.ruleset.ViewRadius,
		Self:           clonePlayerState(player),
		VisibleTiles:   m.visibleTiles(player.position(), m.ruleset.ViewRadius),
		KnownGoal:      clonePositionPtr(m.discoveredGoal[playerID]),
		KnownChests:    positionsFromMap(m.discoveredChests[playerID]),
		Scores:         m.scoreboard(),
	}, nil
}

// FullState returns a resumable snapshot of the full match state.
func (m *Match) FullState() FullState {
	discovery := make(map[string]DiscoveryState, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		discovery[playerID] = DiscoveryState{
			KnownGoal:   clonePositionPtr(m.discoveredGoal[playerID]),
			KnownChests: positionsFromMap(m.discoveredChests[playerID]),
		}
	}
	return FullState{
		MapID:             m.ruleset.MapID,
		RNGSeed:           m.rngSeed,
		Turn:              m.turn,
		MaxTurns:          m.ruleset.MaxTurns,
		Goal:              m.ruleset.Goal,
		Players:           m.scoreboardWithPositions(),
		UncollectedChests: positionsFromMap(m.uncollectedChests),
		Discovery:         discovery,
	}
}

// PublicState returns the spectator-safe public match state.
func (m *Match) PublicState() PublicState {
	return PublicState{
		MapID:             m.ruleset.MapID,
		RNGSeed:           m.rngSeed,
		Turn:              m.turn,
		MaxTurns:          m.ruleset.MaxTurns,
		Tiles:             append([]string(nil), m.ruleset.Tiles...),
		Goal:              m.ruleset.Goal,
		Players:           m.scoreboardWithPositions(),
		UncollectedChests: positionsFromMap(m.uncollectedChests),
	}
}

// LegalActionHint returns a machine-readable hint describing valid actions.
func (m *Match) LegalActionHint() json.RawMessage {
	return mustJSON(map[string]any{
		"type": "object",
		"oneOf": []map[string]any{
			{
				"type":     "object",
				"required": []string{"action", "direction"},
				"properties": map[string]any{
					"action": map[string]any{
						"type":  "string",
						"const": "move",
					},
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
					"action": map[string]any{
						"type":  "string",
						"const": "wait",
					},
				},
			},
		},
	})
}

// Apply applies one turn of actions to the match state.
func (m *Match) Apply(actions map[string]Action) error {
	if m.Terminal() {
		return fmt.Errorf("dungeon: match is already terminal")
	}

	activePlayers := m.PendingPlayerIDs()
	nextPositions := make(map[string]Position, len(activePlayers))
	for _, playerID := range activePlayers {
		current := m.playerStates[playerID].position()
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
		player := m.playerStates[playerID]
		target := nextPositions[playerID]
		player.X = target.X
		player.Y = target.Y
		m.playerStates[playerID] = player
	}

	for chestKey, chestPos := range positionsCopy(m.uncollectedChests) {
		claimants := make([]string, 0, len(activePlayers))
		for _, playerID := range activePlayers {
			player := m.playerStates[playerID]
			if player.X == chestPos.X && player.Y == chestPos.Y {
				claimants = append(claimants, playerID)
			}
		}
		if len(claimants) == 0 {
			continue
		}
		share := m.ruleset.ChestPoints / len(claimants)
		for _, playerID := range claimants {
			player := m.playerStates[playerID]
			player.ChestPoints += share
			player.Score += share
			m.playerStates[playerID] = player
		}
		delete(m.uncollectedChests, chestKey)
		for _, known := range m.discoveredChests {
			delete(known, chestKey)
		}
	}

	for _, playerID := range activePlayers {
		player := m.playerStates[playerID]
		if player.FinishedTurn == nil && player.X == m.ruleset.Goal.X && player.Y == m.ruleset.Goal.Y {
			finishedTurn := m.turn + 1
			player.FinishedTurn = &finishedTurn
			m.playerStates[playerID] = player
		}
	}

	m.turn++
	m.applyGoalBonuses()
	m.refreshDiscoveries()
	return nil
}

// CanApply reports whether the given action is legal for the player.
func (m *Match) CanApply(playerID string, action Action) bool {
	player, ok := m.playerStates[playerID]
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
		return ok && m.isWalkable(next)
	default:
		return false
	}
}

// ParseAction decodes and validates an action payload.
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

// Placements returns the final match ranking in ascending place order.
func (m *Match) Placements() []Placement {
	scores := m.scoreboard()
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

// ShortestPath returns the shortest traversable path on the fixed map.
func (m *Match) ShortestPath(from, to Position) ([]Position, bool) {
	return shortestPath(m.ruleset.Tiles, from, to)
}

// SpawnPoints returns the configured spawn points in player order.
func (m *Match) SpawnPoints() []Position {
	return append([]Position(nil), m.ruleset.SpawnPoints...)
}

// UncollectedChests returns the remaining chest positions.
func (m *Match) UncollectedChests() []Position {
	return positionsFromMap(m.uncollectedChests)
}

func fixedMapRuleset() Ruleset {
	return Ruleset{
		MapID:        RulesetFixedMapV1,
		MaxTurns:     16,
		ViewRadius:   2,
		ChestPoints:  12,
		GoalBonuses:  []int{100, 50, 25, 10},
		TurnDeadline: 100 * time.Millisecond,
		Tiles: []string{
			"#########",
			"#A..#..B#",
			"#...#...#",
			"#.C...C.#",
			"#.#.#.#.#",
			"#...#...#",
			"#.C...G.#",
			"#.......#",
			"#########",
		},
		SpawnPoints: []Position{{X: 1, Y: 1}, {X: 7, Y: 1}, {X: 1, Y: 7}, {X: 7, Y: 7}},
		Goal:        Position{X: 6, Y: 6},
		ChestPositions: []Position{
			{X: 2, Y: 3},
			{X: 6, Y: 3},
			{X: 2, Y: 6},
		},
	}
}

func cloneRuleset(r Ruleset) Ruleset {
	r.GoalBonuses = append([]int(nil), r.GoalBonuses...)
	r.Tiles = append([]string(nil), r.Tiles...)
	r.SpawnPoints = append([]Position(nil), r.SpawnPoints...)
	r.ChestPositions = append([]Position(nil), r.ChestPositions...)
	return r
}

func (m *Match) allPlayersFinished() bool {
	for _, playerID := range m.playerOrder {
		if m.playerStates[playerID].FinishedTurn == nil {
			return false
		}
	}
	return true
}

func (m *Match) resolvePosition(current Position, action Action) Position {
	if action.Action != "move" {
		return current
	}
	next, ok := step(current, action.Direction)
	if !ok || !m.isWalkable(next) {
		return current
	}
	return next
}

func (m *Match) applyGoalBonuses() {
	finished := make([]PlayerState, 0, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		player := m.playerStates[playerID]
		if player.FinishedTurn != nil {
			finished = append(finished, clonePlayerState(player))
		}
	}
	sort.SliceStable(finished, func(i, j int) bool {
		if *finished[i].FinishedTurn != *finished[j].FinishedTurn {
			return *finished[i].FinishedTurn < *finished[j].FinishedTurn
		}
		return finished[i].PlayerID < finished[j].PlayerID
	})

	lastTurn := -1
	lastRank := 0
	for i, player := range finished {
		if i == 0 || *player.FinishedTurn != lastTurn {
			lastRank = i + 1
			lastTurn = *player.FinishedTurn
		}
		bonus := 0
		if lastRank-1 < len(m.ruleset.GoalBonuses) {
			bonus = m.ruleset.GoalBonuses[lastRank-1]
		}
		state := m.playerStates[player.PlayerID]
		state.GoalBonus = bonus
		state.Score = state.ChestPoints + state.GoalBonus
		m.playerStates[player.PlayerID] = state
	}
	for _, playerID := range m.playerOrder {
		state := m.playerStates[playerID]
		if state.FinishedTurn == nil {
			state.GoalBonus = 0
			state.Score = state.ChestPoints
			m.playerStates[playerID] = state
		}
	}
}

func (m *Match) refreshDiscoveries() {
	for _, playerID := range m.playerOrder {
		state := m.playerStates[playerID]
		visible := m.visibleTiles(state.position(), m.ruleset.ViewRadius)
		for _, tile := range visible {
			pos := Position{X: tile.X, Y: tile.Y}
			switch tile.Tile {
			case TileGoal:
				goal := pos
				m.discoveredGoal[playerID] = &goal
			case TileChest:
				if _, ok := m.uncollectedChests[posKey(pos)]; ok {
					m.discoveredChests[playerID][posKey(pos)] = pos
				}
			}
		}
		for chestKey := range m.discoveredChests[playerID] {
			if _, ok := m.uncollectedChests[chestKey]; !ok {
				delete(m.discoveredChests[playerID], chestKey)
			}
		}
	}
}

func (m *Match) visibleTiles(center Position, radius int) []VisibleTile {
	tiles := make([]VisibleTile, 0)
	for y := center.Y - radius; y <= center.Y+radius; y++ {
		for x := center.X - radius; x <= center.X+radius; x++ {
			if !m.inBounds(Position{X: x, Y: y}) {
				continue
			}
			if manhattan(center, Position{X: x, Y: y}) > radius {
				continue
			}
			tiles = append(tiles, VisibleTile{
				X:    x,
				Y:    y,
				Tile: m.tileAt(Position{X: x, Y: y}),
			})
		}
	}
	return tiles
}

func (m *Match) tileAt(pos Position) string {
	if pos == m.ruleset.Goal {
		return TileGoal
	}
	if _, ok := m.uncollectedChests[posKey(pos)]; ok {
		return TileChest
	}
	if m.ruleset.Tiles[pos.Y][pos.X] == '#' {
		return TileWall
	}
	return TileFloor
}

func (m *Match) inBounds(pos Position) bool {
	return pos.Y >= 0 && pos.Y < len(m.ruleset.Tiles) && pos.X >= 0 && pos.X < len(m.ruleset.Tiles[pos.Y])
}

func (m *Match) isWalkable(pos Position) bool {
	if !m.inBounds(pos) {
		return false
	}
	return m.ruleset.Tiles[pos.Y][pos.X] != '#'
}

func (m *Match) isOriginalChest(pos Position) bool {
	for _, chest := range m.ruleset.ChestPositions {
		if chest == pos {
			return true
		}
	}
	return false
}

func (m *Match) scoreboard() []PlayerState {
	scores := make([]PlayerState, 0, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		state := m.playerStates[playerID]
		scores = append(scores, clonePlayerState(PlayerState{
			PlayerID:     state.PlayerID,
			Score:        state.Score,
			GoalBonus:    state.GoalBonus,
			ChestPoints:  state.ChestPoints,
			FinishedTurn: cloneIntPtr(state.FinishedTurn),
		}))
	}
	sort.SliceStable(scores, func(i, j int) bool {
		if scores[i].Score != scores[j].Score {
			return scores[i].Score > scores[j].Score
		}
		if finishOrder(scores[i].FinishedTurn) != finishOrder(scores[j].FinishedTurn) {
			return finishOrder(scores[i].FinishedTurn) < finishOrder(scores[j].FinishedTurn)
		}
		return scores[i].PlayerID < scores[j].PlayerID
	})
	return scores
}

func (m *Match) scoreboardWithPositions() []PlayerState {
	players := make([]PlayerState, 0, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		players = append(players, clonePlayerState(m.playerStates[playerID]))
	}
	sort.SliceStable(players, func(i, j int) bool {
		return players[i].PlayerID < players[j].PlayerID
	})
	return players
}

func (p PlayerState) position() Position {
	return Position{X: p.X, Y: p.Y}
}

func clonePlayerState(p PlayerState) PlayerState {
	p.FinishedTurn = cloneIntPtr(p.FinishedTurn)
	return p
}

func cloneIntPtr(v *int) *int {
	if v == nil {
		return nil
	}
	copy := *v
	return &copy
}

func clonePositionPtr(v *Position) *Position {
	if v == nil {
		return nil
	}
	copy := *v
	return &copy
}

func positionsFromMap(values map[string]Position) []Position {
	positions := make([]Position, 0, len(values))
	for _, pos := range values {
		positions = append(positions, pos)
	}
	sort.Slice(positions, func(i, j int) bool {
		if positions[i].Y != positions[j].Y {
			return positions[i].Y < positions[j].Y
		}
		return positions[i].X < positions[j].X
	})
	return positions
}

func positionsCopy(values map[string]Position) map[string]Position {
	cloned := make(map[string]Position, len(values))
	for key, pos := range values {
		cloned[key] = pos
	}
	return cloned
}

func posKey(pos Position) string {
	return fmt.Sprintf("%d,%d", pos.X, pos.Y)
}

func step(pos Position, direction string) (Position, bool) {
	switch direction {
	case "up":
		return Position{X: pos.X, Y: pos.Y - 1}, true
	case "down":
		return Position{X: pos.X, Y: pos.Y + 1}, true
	case "left":
		return Position{X: pos.X - 1, Y: pos.Y}, true
	case "right":
		return Position{X: pos.X + 1, Y: pos.Y}, true
	default:
		return Position{}, false
	}
}

func manhattan(a, b Position) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}
	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

func shortestPath(layout []string, from, to Position) ([]Position, bool) {
	if from == to {
		return []Position{from}, true
	}
	queue := []Position{from}
	prev := map[string]Position{}
	seen := map[string]struct{}{posKey(from): {}}
	directions := []string{"up", "left", "right", "down"}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, direction := range directions {
			next, ok := step(current, direction)
			if !ok || next.Y < 0 || next.Y >= len(layout) || next.X < 0 || next.X >= len(layout[next.Y]) {
				continue
			}
			if layout[next.Y][next.X] == '#' {
				continue
			}
			key := posKey(next)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			prev[key] = current
			if next == to {
				path := []Position{to}
				cursor := to
				for cursor != from {
					cursor = prev[posKey(cursor)]
					path = append(path, cursor)
				}
				reversePositions(path)
				return path, true
			}
			queue = append(queue, next)
		}
	}
	return nil, false
}

func reversePositions(path []Position) {
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
}

func finishOrder(turn *int) int {
	if turn == nil {
		return 1 << 30
	}
	return *turn
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
