package botlogic

import (
	"testing"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

func TestMemoryObservePersistsGoalAndCleansCollectedChests(t *testing.T) {
	memory := NewMemory()

	memory.Observe(dungeon.VisibleState{
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
			tile(2, 1, dungeon.TileGoal),
			tile(2, 2, dungeon.TileChest),
		),
		KnownGoal:   &dungeon.Position{X: 2, Y: 1},
		KnownChests: []dungeon.ChestState{{X: 2, Y: 2, Points: 18}},
	})
	memory.Observe(dungeon.VisibleState{
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
		),
		KnownGoal:   &dungeon.Position{X: 2, Y: 1},
		KnownChests: nil,
	})

	if goal := memory.KnownGoal(); goal == nil || *goal != (dungeon.Position{X: 2, Y: 1}) {
		t.Fatalf("known goal = %+v, want (2,1)", goal)
	}
	if got := memory.KnownChests(); len(got) != 0 {
		t.Fatalf("known chests = %+v, want empty after cleanup", got)
	}
}
