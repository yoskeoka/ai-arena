package dungeon

import "time"

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

// Tile constants encode the public map surface used in snapshots and visible state.
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

// Metadata identifies one concrete dungeon game selection.
type Metadata struct {
	GameID         string
	GameVersion    string
	RulesetVersion string
}

// Config selects a dungeon ruleset, player roster, and optional deterministic seed.
type Config struct {
	GameVersion string
	Ruleset     string
	PlayerIDs   []string
	RNGSeed     string
}

// Ruleset defines the static rule surface for a dungeon match.
type Ruleset struct {
	MapID        string
	MaxTurns     int
	ViewRadius   int
	GoalBonuses  []int
	TurnDeadline time.Duration
}

// GeneratedLayout holds the deterministic seed-derived layout for one match.
type GeneratedLayout struct {
	Tiles         []string
	SpawnPoints   []Position
	Goal          Position
	InitialChests []ChestState
}

// Position identifies a tile coordinate in the dungeon grid.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ChestState describes one chest location and its point value.
type ChestState struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Points int `json:"points"`
}

// Action is one normalized player command for a dungeon turn.
type Action struct {
	Action    string `json:"action"`
	Direction string `json:"direction,omitempty"`
}

// PlayerState is the per-player state shared in dungeon snapshots.
type PlayerState struct {
	PlayerID     string `json:"player_id"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Score        int    `json:"score"`
	GoalBonus    int    `json:"goal_bonus"`
	ChestPoints  int    `json:"chest_points"`
	FinishedTurn *int   `json:"finished_turn"`
}

// VisibleTile is one tile currently visible to a player.
type VisibleTile struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Tile string `json:"tile"`
}

// VisibleState is the player-specific observation payload for one turn.
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

// DiscoveryState records which goal and chests one player has discovered.
type DiscoveryState struct {
	KnownGoal   *Position    `json:"known_goal"`
	KnownChests []ChestState `json:"known_chests"`
}

// FullState is the full resumable dungeon snapshot including hidden information.
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

// PublicState is the spectator-facing dungeon snapshot without discovery maps.
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

// Placement reports one player's final rank.
type Placement struct {
	PlayerID string
	Place    int
}
