package botlogic

import "github.com/yoskeoka/ai-arena/games/dungeon"

// Memory stores only information the bot has actually observed through visible_state.
type Memory struct {
	knownTiles    map[string]string
	knownGoal     *dungeon.Position
	knownChests   map[string]dungeon.ChestState
	exploreTarget *dungeon.Position
}

// NewMemory returns empty persistent bot memory.
func NewMemory() *Memory {
	return &Memory{
		knownTiles:  make(map[string]string),
		knownChests: make(map[string]dungeon.ChestState),
	}
}

// Observe updates memory from the latest visible state.
func (m *Memory) Observe(state dungeon.VisibleState) {
	for _, tile := range state.VisibleTiles {
		pos := dungeon.Position{X: tile.X, Y: tile.Y}
		m.knownTiles[posKey(pos)] = tile.Tile
		if tile.Tile == dungeon.TileGoal {
			goal := pos
			m.knownGoal = &goal
		}
		if tile.Tile == dungeon.TileChest {
			m.knownChests[posKey(pos)] = dungeon.ChestState{X: pos.X, Y: pos.Y}
		}
	}
	if state.KnownGoal != nil {
		goal := *state.KnownGoal
		m.knownGoal = &goal
	}
	currentKnown := make(map[string]struct{}, len(state.KnownChests))
	for _, chest := range state.KnownChests {
		key := posKey(dungeon.Position{X: chest.X, Y: chest.Y})
		currentKnown[key] = struct{}{}
		m.knownChests[key] = chest
	}
	for key := range m.knownChests {
		if _, ok := currentKnown[key]; !ok {
			delete(m.knownChests, key)
		}
	}
}

func (m *Memory) KnownGoal() *dungeon.Position {
	if m.knownGoal == nil {
		return nil
	}
	goal := *m.knownGoal
	return &goal
}

func (m *Memory) KnownChests() []dungeon.ChestState {
	return sortedChestMap(m.knownChests)
}

func (m *Memory) ExploreTarget() *dungeon.Position {
	if m.exploreTarget == nil {
		return nil
	}
	target := *m.exploreTarget
	return &target
}

func (m *Memory) SetExploreTarget(target *dungeon.Position) {
	if target == nil {
		m.exploreTarget = nil
		return
	}
	next := *target
	m.exploreTarget = &next
}
