package dungeon

import "testing"

func TestTurnEngineNormalizesActionsBeforeMovement(t *testing.T) {
	match := restoreFixedMapScenario(t, []string{"p1", "p2"}, fixedMapState(
		mustNewFixedMapMatch(t, []string{"p1", "p2"}),
		FullState{
			Players: []PlayerState{
				{PlayerID: "p1", X: 1, Y: 1},
				{PlayerID: "p2", X: 7, Y: 1},
			},
		},
	))

	engine := newTurnEngine(match, map[string]Action{
		"p1": {Action: "move", Direction: "left"},
	})
	engine.normalizeActions()

	if got := engine.frame.actions["p1"]; got != (Action{Action: "wait"}) {
		t.Fatalf("normalized p1 action = %+v, want wait", got)
	}
	if got := engine.frame.actions["p2"]; got != (Action{Action: "wait"}) {
		t.Fatalf("missing p2 action = %+v, want wait", got)
	}
}

func TestTurnEngineTracksMovementAndChestClaimantsByPhase(t *testing.T) {
	match := restoreFixedMapScenario(t, []string{"p1", "p2", "p3"}, fixedMapState(
		mustNewFixedMapMatch(t, []string{"p1", "p2", "p3"}),
		FullState{
			Turn: 5,
			Players: []PlayerState{
				{PlayerID: "p1", X: 1, Y: 6},
				{PlayerID: "p2", X: 3, Y: 6},
				{PlayerID: "p3", X: 5, Y: 6},
			},
			UncollectedChests: []ChestState{
				{X: 2, Y: 6, Points: 12},
			},
		},
	))

	engine := newTurnEngine(match, map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "left"},
		"p3": {Action: "wait"},
	})

	engine.normalizeActions()
	engine.resolveMovement()

	if got := engine.frame.nextPositions["p1"]; got != (Position{X: 2, Y: 6}) {
		t.Fatalf("p1 next position = %+v, want (2,6)", got)
	}
	if got := engine.frame.nextPositions["p2"]; got != (Position{X: 2, Y: 6}) {
		t.Fatalf("p2 next position = %+v, want (2,6)", got)
	}

	engine.resolveInteractions()

	claimants := engine.frame.chestClaimants["2,6"]
	if len(claimants) != 2 || claimants[0] != "p1" || claimants[1] != "p2" {
		t.Fatalf("claimants = %+v, want [p1 p2]", claimants)
	}
	assertPlayerScore(t, match, "p1", 6, 0, 6)
	assertPlayerScore(t, match, "p2", 6, 0, 6)
	assertPlayerScore(t, match, "p3", 0, 0, 0)
}
