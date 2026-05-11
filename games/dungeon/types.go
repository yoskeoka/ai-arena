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
