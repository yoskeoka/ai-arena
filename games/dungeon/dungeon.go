package dungeon

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	randv2 "math/rand/v2"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	// GameID is the stable identifier for the dungeon game.
	GameID = "dungeon"
	// GameVersion is the current dungeon game contract version.
	GameVersion = "1.0.0"
	// RulesetFixedMapV1 is the fixed-map ruleset used by the first MVP slice.
	RulesetFixedMapV1 = "fixed-map-v1"
	// RulesetSeededMazeV1 is the seeded maze ruleset used by the generated-map slice.
	RulesetSeededMazeV1 = "seeded-maze-v1"
	// BuilderIDSubprocess identifies the local subprocess bot builder.
	BuilderIDSubprocess = "dungeon/local-subprocess"
	// DefaultRNGSeed is the default deterministic seed used by local helpers.
	DefaultRNGSeed = "0000000000000000000000000000000000000000000000000000000000000000"
)

const (
	TileWall  = "wall"
	TileFloor = "floor"
	TileChest = "chest"
	TileGoal  = "goal"
)

var (
	seededGoalBonuses  = []int{42, 28, 14, 7}
	seededChestPoints  = []int{24, 18, 12}
	seededMazeMaxTurns = 50
)

type Metadata struct {
	GameID         string
	GameVersion    string
	RulesetVersion string
}

type Config struct {
	GameVersion string
	Ruleset     string
	PlayerIDs   []string
	RNGSeed     string
}

type Ruleset struct {
	MapID         string
	MaxTurns      int
	ViewRadius    int
	GoalBonuses   []int
	TurnDeadline  time.Duration
	Tiles         []string
	SpawnPoints   []Position
	Goal          Position
	InitialChests []ChestState
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ChestState struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Points int `json:"points"`
}

type Action struct {
	Action    string `json:"action"`
	Direction string `json:"direction,omitempty"`
}

type PlayerState struct {
	PlayerID     string `json:"player_id"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Score        int    `json:"score"`
	GoalBonus    int    `json:"goal_bonus"`
	ChestPoints  int    `json:"chest_points"`
	FinishedTurn *int   `json:"finished_turn"`
}

type VisibleTile struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Tile string `json:"tile"`
}

type VisibleState struct {
	Turn           int           `json:"turn"`
	RemainingTurns int           `json:"remaining_turns"`
	ViewRadius     int           `json:"view_radius"`
	Self           PlayerState   `json:"self"`
	VisibleTiles   []VisibleTile `json:"visible_tiles"`
	KnownGoal      *Position     `json:"known_goal"`
	KnownChests    []ChestState  `json:"known_chests"`
	Scores         []PlayerState `json:"scores"`
}

type DiscoveryState struct {
	KnownGoal   *Position    `json:"known_goal"`
	KnownChests []ChestState `json:"known_chests"`
}

type FullState struct {
	MapID             string                    `json:"map_id"`
	RNGSeed           string                    `json:"rng_seed"`
	Turn              int                       `json:"turn"`
	MaxTurns          int                       `json:"max_turns"`
	Tiles             []string                  `json:"tiles"`
	SpawnPoints       []Position                `json:"spawn_points"`
	Goal              Position                  `json:"goal"`
	InitialChests     []ChestState              `json:"initial_chests"`
	Players           []PlayerState             `json:"players"`
	UncollectedChests []ChestState              `json:"uncollected_chests"`
	Discovery         map[string]DiscoveryState `json:"discovery"`
}

type PublicState struct {
	MapID             string        `json:"map_id"`
	RNGSeed           string        `json:"rng_seed,omitempty"`
	Turn              int           `json:"turn"`
	MaxTurns          int           `json:"max_turns"`
	Tiles             []string      `json:"tiles"`
	SpawnPoints       []Position    `json:"spawn_points"`
	Goal              Position      `json:"goal"`
	InitialChests     []ChestState  `json:"initial_chests"`
	Players           []PlayerState `json:"players"`
	UncollectedChests []ChestState  `json:"uncollected_chests"`
}

type Placement struct {
	PlayerID string
	Place    int
}

type Match struct {
	meta              Metadata
	ruleset           Ruleset
	playerOrder       []string
	playerStates      map[string]PlayerState
	uncollectedChests map[string]ChestState
	discoveredGoal    map[string]*Position
	discoveredChests  map[string]map[string]ChestState
	turn              int
	rngSeed           string
}

func SupportedRulesets() []string {
	return []string{RulesetFixedMapV1, RulesetSeededMazeV1}
}

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

func New(cfg Config) (*Match, error) {
	meta, _, err := MetadataForSelection(cfg.GameVersion, cfg.Ruleset)
	if err != nil {
		return nil, err
	}
	rngSeed, err := normalizeSeedOrGenerate(cfg.RNGSeed)
	if err != nil {
		return nil, err
	}
	ruleset, err := buildRuleset(cfg.Ruleset, rngSeed)
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
		uncollectedChests: make(map[string]ChestState, len(ruleset.InitialChests)),
		discoveredGoal:    make(map[string]*Position, len(cfg.PlayerIDs)),
		discoveredChests:  make(map[string]map[string]ChestState, len(cfg.PlayerIDs)),
		rngSeed:           rngSeed,
	}
	for _, chest := range ruleset.InitialChests {
		match.uncollectedChests[chestKey(chest)] = chest
	}
	for _, playerID := range playerOrder {
		match.discoveredChests[playerID] = make(map[string]ChestState)
	}
	match.refreshDiscoveries()
	return match, nil
}

func NewFromFullState(cfg Config, state FullState) (*Match, error) {
	resumeSeed := cfg.RNGSeed
	if strings.TrimSpace(resumeSeed) == "" {
		resumeSeed = state.RNGSeed
	}
	match, err := New(Config{
		GameVersion: cfg.GameVersion,
		Ruleset:     cfg.Ruleset,
		PlayerIDs:   append([]string(nil), cfg.PlayerIDs...),
		RNGSeed:     resumeSeed,
	})
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
	stateSeed, err := normalizeSeed(state.RNGSeed)
	if err != nil {
		return nil, fmt.Errorf("dungeon: invalid snapshot rng_seed: %w", err)
	}
	if stateSeed != match.rngSeed {
		return nil, fmt.Errorf("dungeon: snapshot rng_seed %q does not match config %q", state.RNGSeed, match.rngSeed)
	}
	if !equalStringSlices(state.Tiles, match.ruleset.Tiles) {
		return nil, fmt.Errorf("dungeon: snapshot tiles do not match generated layout")
	}
	if !equalPositions(state.SpawnPoints, match.ruleset.SpawnPoints) {
		return nil, fmt.Errorf("dungeon: snapshot spawn_points do not match generated layout")
	}
	if state.Goal != match.ruleset.Goal {
		return nil, fmt.Errorf("dungeon: snapshot goal does not match generated layout")
	}
	if !equalChests(state.InitialChests, match.ruleset.InitialChests) {
		return nil, fmt.Errorf("dungeon: snapshot initial_chests do not match generated layout")
	}

	match.turn = state.Turn
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

	match.uncollectedChests = make(map[string]ChestState, len(state.UncollectedChests))
	for _, chest := range state.UncollectedChests {
		if !match.isOriginalChest(chest) {
			return nil, fmt.Errorf("dungeon: snapshot chest at (%d,%d) is not in generated layout", chest.X, chest.Y)
		}
		match.uncollectedChests[chestKey(chest)] = chest
	}

	match.discoveredGoal = make(map[string]*Position, len(match.playerOrder))
	match.discoveredChests = make(map[string]map[string]ChestState, len(match.playerOrder))
	for _, playerID := range match.playerOrder {
		match.discoveredChests[playerID] = make(map[string]ChestState)
		discovery := state.Discovery[playerID]
		if discovery.KnownGoal != nil {
			pos := *discovery.KnownGoal
			match.discoveredGoal[playerID] = &pos
		}
		for _, chest := range discovery.KnownChests {
			if original, ok := match.uncollectedChests[chestKey(chest)]; ok {
				match.discoveredChests[playerID][chestKey(chest)] = original
			}
		}
	}
	match.refreshDiscoveries()
	return match, nil
}

func (m *Match) Metadata() Metadata {
	return m.meta
}

func (m *Match) Ruleset() Ruleset {
	return cloneRuleset(m.ruleset)
}

func (m *Match) Terminal() bool {
	return m.turn >= m.ruleset.MaxTurns || m.allPlayersFinished()
}

func (m *Match) Turn() int {
	return m.turn
}

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
		KnownChests:    chestsFromMap(m.discoveredChests[playerID]),
		Scores:         m.scoreboard(),
	}, nil
}

func (m *Match) FullState() FullState {
	discovery := make(map[string]DiscoveryState, len(m.playerOrder))
	for _, playerID := range m.playerOrder {
		discovery[playerID] = DiscoveryState{
			KnownGoal:   clonePositionPtr(m.discoveredGoal[playerID]),
			KnownChests: chestsFromMap(m.discoveredChests[playerID]),
		}
	}
	return FullState{
		MapID:             m.ruleset.MapID,
		RNGSeed:           m.rngSeed,
		Turn:              m.turn,
		MaxTurns:          m.ruleset.MaxTurns,
		Tiles:             append([]string(nil), m.ruleset.Tiles...),
		SpawnPoints:       append([]Position(nil), m.ruleset.SpawnPoints...),
		Goal:              m.ruleset.Goal,
		InitialChests:     append([]ChestState(nil), m.ruleset.InitialChests...),
		Players:           m.scoreboardWithPositions(),
		UncollectedChests: chestsFromMap(m.uncollectedChests),
		Discovery:         discovery,
	}
}

func (m *Match) PublicState() PublicState {
	return PublicState{
		MapID:             m.ruleset.MapID,
		RNGSeed:           m.rngSeed,
		Turn:              m.turn,
		MaxTurns:          m.ruleset.MaxTurns,
		Tiles:             append([]string(nil), m.ruleset.Tiles...),
		SpawnPoints:       append([]Position(nil), m.ruleset.SpawnPoints...),
		Goal:              m.ruleset.Goal,
		InitialChests:     append([]ChestState(nil), m.ruleset.InitialChests...),
		Players:           m.scoreboardWithPositions(),
		UncollectedChests: chestsFromMap(m.uncollectedChests),
	}
}

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

	for chestID, chest := range chestsCopy(m.uncollectedChests) {
		claimants := make([]string, 0, len(activePlayers))
		for _, playerID := range activePlayers {
			player := m.playerStates[playerID]
			if player.X == chest.X && player.Y == chest.Y {
				claimants = append(claimants, playerID)
			}
		}
		if len(claimants) == 0 {
			continue
		}
		share := chest.Points / len(claimants)
		for _, playerID := range claimants {
			player := m.playerStates[playerID]
			player.ChestPoints += share
			player.Score += share
			m.playerStates[playerID] = player
		}
		delete(m.uncollectedChests, chestID)
		for _, known := range m.discoveredChests {
			delete(known, chestID)
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

func (m *Match) ShortestPath(from, to Position) ([]Position, bool) {
	return shortestPath(m.ruleset.Tiles, from, to)
}

func (m *Match) SpawnPoints() []Position {
	return append([]Position(nil), m.ruleset.SpawnPoints...)
}

func (m *Match) UncollectedChests() []ChestState {
	return chestsFromMap(m.uncollectedChests)
}

func buildRuleset(ruleset, seed string) (Ruleset, error) {
	switch ruleset {
	case RulesetFixedMapV1:
		return fixedMapRuleset(), nil
	case RulesetSeededMazeV1:
		return seededMazeRuleset(seed)
	default:
		return Ruleset{}, fmt.Errorf("dungeon: unsupported ruleset %q", ruleset)
	}
}

func fixedMapRuleset() Ruleset {
	return Ruleset{
		MapID:        RulesetFixedMapV1,
		MaxTurns:     16,
		ViewRadius:   2,
		GoalBonuses:  []int{100, 50, 25, 10},
		TurnDeadline: 100 * time.Millisecond,
		Tiles: []string{
			"#########",
			"#...#...#",
			"#...#...#",
			"#.......#",
			"#.#.#.#.#",
			"#...#...#",
			"#.......#",
			"#.......#",
			"#########",
		},
		SpawnPoints: []Position{{X: 1, Y: 1}, {X: 7, Y: 1}, {X: 1, Y: 7}, {X: 7, Y: 7}},
		Goal:        Position{X: 6, Y: 6},
		InitialChests: []ChestState{
			{X: 2, Y: 3, Points: 12},
			{X: 6, Y: 3, Points: 12},
			{X: 2, Y: 6, Points: 12},
		},
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

func seededMazeRuleset(seed string) (Ruleset, error) {
	ruleset := seededMazeBaseRuleset()
	tiles := generatePerfectMaze9x9(seed)
	walkable := walkablePositions(tiles)
	if len(walkable) < 8 {
		return Ruleset{}, fmt.Errorf("dungeon: generated maze has insufficient walkable tiles")
	}
	rng, err := newSeededRand(seed)
	if err != nil {
		return Ruleset{}, err
	}
	goal := walkable[rng.IntN(len(walkable))]
	start := farthestPosition(tiles, goal)
	spawns, err := nearestUniquePositions(tiles, start, 4, map[string]struct{}{posKey(goal): {}})
	if err != nil {
		return Ruleset{}, err
	}
	chestPositions, err := selectChestPositions(tiles, walkable, start, goal, spawns, 3, rng)
	if err != nil {
		return Ruleset{}, err
	}
	chestScores := append([]int(nil), seededChestPoints...)
	rng.Shuffle(len(chestScores), func(i, j int) {
		chestScores[i], chestScores[j] = chestScores[j], chestScores[i]
	})
	initialChests := make([]ChestState, 0, len(chestPositions))
	for i, pos := range chestPositions {
		initialChests = append(initialChests, ChestState{X: pos.X, Y: pos.Y, Points: chestScores[i]})
	}
	ruleset.Tiles = tiles
	ruleset.SpawnPoints = spawns
	ruleset.Goal = goal
	ruleset.InitialChests = initialChests
	return ruleset, nil
}

func generatePerfectMaze9x9(seed string) []string {
	const size = 9
	grid := make([][]byte, size)
	for y := range grid {
		grid[y] = make([]byte, size)
		for x := range grid[y] {
			grid[y][x] = '#'
		}
	}
	type cell struct{ x, y int }
	cells := []cell{}
	for y := 1; y < size; y += 2 {
		for x := 1; x < size; x += 2 {
			cells = append(cells, cell{x: x, y: y})
		}
	}
	rng, err := newSeededRand(seed)
	if err != nil {
		panic(err)
	}
	start := cells[rng.IntN(len(cells))]
	stack := []cell{start}
	visited := map[cell]struct{}{start: {}}
	grid[start.y][start.x] = '.'
	dirs := []cell{{x: 0, y: -2}, {x: 2, y: 0}, {x: 0, y: 2}, {x: -2, y: 0}}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		order := append([]cell(nil), dirs...)
		rng.Shuffle(len(order), func(i, j int) {
			order[i], order[j] = order[j], order[i]
		})
		advanced := false
		for _, dir := range order {
			next := cell{x: current.x + dir.x, y: current.y + dir.y}
			if next.x <= 0 || next.x >= size-1 || next.y <= 0 || next.y >= size-1 {
				continue
			}
			if _, ok := visited[next]; ok {
				continue
			}
			grid[current.y+dir.y/2][current.x+dir.x/2] = '.'
			grid[next.y][next.x] = '.'
			visited[next] = struct{}{}
			stack = append(stack, next)
			advanced = true
			break
		}
		if !advanced {
			stack = stack[:len(stack)-1]
		}
	}
	rows := make([]string, size)
	for i := range grid {
		rows[i] = string(grid[i])
	}
	return rows
}

func nearestUniquePositions(layout []string, from Position, count int, exclude map[string]struct{}) ([]Position, error) {
	candidates := make([]positionDistance, 0)
	for _, pos := range walkablePositions(layout) {
		if _, skip := exclude[posKey(pos)]; skip {
			continue
		}
		path, ok := shortestPath(layout, from, pos)
		if !ok {
			continue
		}
		candidates = append(candidates, positionDistance{Position: pos, Distance: len(path) - 1})
	}
	sortPositionDistances(candidates, false)
	if len(candidates) < count {
		return nil, fmt.Errorf("dungeon: insufficient spawn positions")
	}
	out := make([]Position, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, candidates[i].Position)
	}
	return out, nil
}

func selectChestPositions(layout []string, walkable []Position, start, goal Position, spawns []Position, count int, rng *deterministicRand) ([]Position, error) {
	excluded := make(map[string]struct{}, len(spawns)+1)
	excluded[posKey(goal)] = struct{}{}
	for _, spawn := range spawns {
		excluded[posKey(spawn)] = struct{}{}
	}
	candidates := make([]Position, 0, len(walkable))
	for _, pos := range walkable {
		if _, skip := excluded[posKey(pos)]; skip {
			continue
		}
		candidates = append(candidates, pos)
	}
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	selected := make([]Position, 0, count)
	seen := map[string]struct{}{}
	for _, pos := range candidates {
		if distanceBetween(layout, start, pos) <= 1 || distanceBetween(layout, goal, pos) <= 1 {
			continue
		}
		key := posKey(pos)
		if _, exists := seen[key]; exists {
			continue
		}
		selected = append(selected, pos)
		seen[key] = struct{}{}
		if len(selected) == count {
			return selected, nil
		}
	}
	for _, pos := range candidates {
		key := posKey(pos)
		if _, exists := seen[key]; exists {
			continue
		}
		selected = append(selected, pos)
		seen[key] = struct{}{}
		if len(selected) == count {
			return selected, nil
		}
	}
	return nil, fmt.Errorf("dungeon: insufficient chest positions")
}

func farthestPosition(layout []string, from Position) Position {
	best := from
	bestDistance := -1
	for _, pos := range walkablePositions(layout) {
		path, ok := shortestPath(layout, from, pos)
		if !ok {
			continue
		}
		distance := len(path) - 1
		if distance > bestDistance || (distance == bestDistance && comparePosition(pos, best) < 0) {
			best = pos
			bestDistance = distance
		}
	}
	return best
}

type positionDistance struct {
	Position
	Distance int
}

func sortPositionDistances(values []positionDistance, reverseDistance bool) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Distance != values[j].Distance {
			if reverseDistance {
				return values[i].Distance > values[j].Distance
			}
			return values[i].Distance < values[j].Distance
		}
		return comparePosition(values[i].Position, values[j].Position) < 0
	})
}

func walkablePositions(layout []string) []Position {
	positions := make([]Position, 0)
	for y, row := range layout {
		for x := 0; x < len(row); x++ {
			if row[x] != '#' {
				positions = append(positions, Position{X: x, Y: y})
			}
		}
	}
	return positions
}

type deterministicRand struct {
	mu  sync.Mutex
	rng *randv2.Rand
}

func newSeededRand(seed string) (*deterministicRand, error) {
	material, err := decodeSeedMaterial(seed)
	if err != nil {
		return nil, err
	}
	var seed32 [32]byte
	copy(seed32[:], material)
	// #nosec G404 -- deterministic gameplay generation requires a reproducible PRNG, and ChaCha8 gives a stable stream
	// while being less trivially guessable than simpler generators. This source is wrapped so multi-site callers can
	// serialize access without introducing additional independent RNG state.
	return &deterministicRand{rng: randv2.New(randv2.NewChaCha8(seed32))}, nil
}

func (r *deterministicRand) IntN(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.IntN(n)
}

func (r *deterministicRand) Shuffle(n int, swap func(i, j int)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rng.Shuffle(n, swap)
}

func normalizeSeed(seed string) (string, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return "", fmt.Errorf("rng_seed is required")
	}
	decoded, err := decodeSeedMaterial(seed)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(decoded), nil
}

func normalizeSeedOrGenerate(seed string) (string, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return GenerateSeedHex()
	}
	return normalizeSeed(seed)
}

func decodeSeedMaterial(seed string) ([]byte, error) {
	seed = strings.TrimSpace(seed)
	if seed == "" {
		return nil, fmt.Errorf("rng_seed is required")
	}
	if len(seed) != 64 {
		return nil, fmt.Errorf("rng_seed must be 64 hex characters")
	}
	decoded, err := hex.DecodeString(seed)
	if err != nil {
		return nil, fmt.Errorf("rng_seed must be lowercase/uppercase hex: %w", err)
	}
	if len(decoded) != 32 {
		return nil, fmt.Errorf("rng_seed must decode to 32 bytes")
	}
	return decoded, nil
}

func GenerateSeedHex() (string, error) {
	buf := make([]byte, 32)
	if _, err := cryptorand.Read(buf); err != nil {
		return "", fmt.Errorf("generate rng_seed: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func cloneRuleset(r Ruleset) Ruleset {
	r.GoalBonuses = append([]int(nil), r.GoalBonuses...)
	r.Tiles = append([]string(nil), r.Tiles...)
	r.SpawnPoints = append([]Position(nil), r.SpawnPoints...)
	r.InitialChests = append([]ChestState(nil), r.InitialChests...)
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
				if chest, ok := m.uncollectedChests[posKey(pos)]; ok {
					m.discoveredChests[playerID][posKey(pos)] = chest
				}
			}
		}
		for key := range m.discoveredChests[playerID] {
			if _, ok := m.uncollectedChests[key]; !ok {
				delete(m.discoveredChests[playerID], key)
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

func (m *Match) isOriginalChest(chest ChestState) bool {
	for _, original := range m.ruleset.InitialChests {
		if original == chest {
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

func chestsFromMap(values map[string]ChestState) []ChestState {
	chests := make([]ChestState, 0, len(values))
	for _, chest := range values {
		chests = append(chests, chest)
	}
	sort.Slice(chests, func(i, j int) bool {
		if chests[i].Y != chests[j].Y {
			return chests[i].Y < chests[j].Y
		}
		if chests[i].X != chests[j].X {
			return chests[i].X < chests[j].X
		}
		return chests[i].Points < chests[j].Points
	})
	return chests
}

func chestsCopy(values map[string]ChestState) map[string]ChestState {
	cloned := make(map[string]ChestState, len(values))
	for key, chest := range values {
		cloned[key] = chest
	}
	return cloned
}

func posKey(pos Position) string {
	return fmt.Sprintf("%d,%d", pos.X, pos.Y)
}

func chestKey(chest ChestState) string {
	return posKey(Position{X: chest.X, Y: chest.Y})
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

func comparePosition(a, b Position) int {
	if a.Y != b.Y {
		return a.Y - b.Y
	}
	return a.X - b.X
}

func distanceBetween(layout []string, from, to Position) int {
	path, ok := shortestPath(layout, from, to)
	if !ok {
		return 1 << 30
	}
	return len(path) - 1
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalPositions(a, b []Position) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalChests(a, b []ChestState) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
