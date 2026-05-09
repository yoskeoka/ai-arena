package botlogic

import (
	"testing"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

func TestBotPrefersHighValueChestWhenDetourIsAffordable(t *testing.T) {
	bot := New()
	action := bot.Decide(dungeon.VisibleState{
		RemainingTurns: 10,
		Self:           dungeon.PlayerState{PlayerID: "p1", X: 1, Y: 1},
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
			tile(2, 1, dungeon.TileFloor),
			tile(3, 1, dungeon.TileFloor),
			tile(4, 1, dungeon.TileFloor),
			tile(5, 1, dungeon.TileGoal),
			tile(3, 2, dungeon.TileChest),
			tile(4, 2, dungeon.TileFloor),
			tile(5, 2, dungeon.TileFloor),
		),
		KnownGoal:   &dungeon.Position{X: 5, Y: 1},
		KnownChests: []dungeon.ChestState{{X: 3, Y: 2, Points: 24}},
		Scores: []dungeon.PlayerState{
			{PlayerID: "p1", Score: 0},
			{PlayerID: "p2", Score: 18},
		},
	})
	if action.Direction != "right" {
		t.Fatalf("direction = %q, want right toward chest route", action.Direction)
	}
}

func TestBotPivotsToGoalWhenTurnsAreTight(t *testing.T) {
	bot := New()
	action := bot.Decide(dungeon.VisibleState{
		RemainingTurns: 4,
		Self:           dungeon.PlayerState{PlayerID: "p1", X: 1, Y: 1, ChestPoints: 18, Score: 18},
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
			tile(2, 1, dungeon.TileFloor),
			tile(3, 1, dungeon.TileFloor),
			tile(4, 1, dungeon.TileFloor),
			tile(5, 1, dungeon.TileGoal),
			tile(3, 2, dungeon.TileChest),
			tile(4, 2, dungeon.TileFloor),
			tile(5, 2, dungeon.TileFloor),
		),
		KnownGoal:   &dungeon.Position{X: 5, Y: 1},
		KnownChests: []dungeon.ChestState{{X: 3, Y: 2, Points: 24}},
	})
	if action.Direction != "right" {
		t.Fatalf("direction = %q, want right toward goal", action.Direction)
	}
}

func TestBotSkipsLowValueChestWhenGoalIsClose(t *testing.T) {
	bot := New()
	action := bot.Decide(dungeon.VisibleState{
		RemainingTurns: 8,
		Self:           dungeon.PlayerState{PlayerID: "p1", X: 1, Y: 1},
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
			tile(2, 1, dungeon.TileFloor),
			tile(3, 1, dungeon.TileGoal),
			tile(2, 2, dungeon.TileChest),
			tile(3, 2, dungeon.TileFloor),
		),
		KnownGoal:   &dungeon.Position{X: 3, Y: 1},
		KnownChests: []dungeon.ChestState{{X: 2, Y: 2, Points: 12}},
	})
	if action.Direction != "right" {
		t.Fatalf("direction = %q, want right toward nearby goal", action.Direction)
	}
}

func TestBotExploresWhenNoGoalOrChestKnown(t *testing.T) {
	bot := New()
	action := bot.Decide(dungeon.VisibleState{
		RemainingTurns: 20,
		Self:           dungeon.PlayerState{PlayerID: "p1", X: 1, Y: 1},
		VisibleTiles: visibleTiles(
			tile(1, 1, dungeon.TileFloor),
			tile(2, 1, dungeon.TileFloor),
		),
	})
	if action.Direction != "right" {
		t.Fatalf("direction = %q, want right toward frontier", action.Direction)
	}
}

func tile(x, y int, kind string) dungeon.VisibleTile {
	return dungeon.VisibleTile{X: x, Y: y, Tile: kind}
}

func visibleTiles(tiles ...dungeon.VisibleTile) []dungeon.VisibleTile {
	return append([]dungeon.VisibleTile(nil), tiles...)
}
