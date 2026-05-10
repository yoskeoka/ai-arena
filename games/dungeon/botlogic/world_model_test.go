package botlogic

import (
	"testing"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

func TestWorldModelGoalPathAndFrontierQuery(t *testing.T) {
	memory := NewMemory()
	memory.Observe(dungeon.VisibleState{
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
			tile(2, 1, dungeon.TileFloor),
			tile(3, 1, dungeon.TileGoal),
			tile(1, 2, dungeon.TileWall),
			tile(2, 2, dungeon.TileFloor),
		),
		KnownGoal: &dungeon.Position{X: 3, Y: 1},
	})

	world := NewWorldModel(memory)
	path, ok := world.GoalPath(dungeon.Position{X: 1, Y: 1})
	if !ok {
		t.Fatal("goal path missing")
	}
	if len(path) != 3 {
		t.Fatalf("goal path len = %d, want 3", len(path))
	}
	if !world.IsFrontier(dungeon.Position{X: 2, Y: 1}) {
		t.Fatal("expected known floor next to unknown tile to remain frontier")
	}
}
