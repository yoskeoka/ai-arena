package dungeon

import "testing"

type targetedScenario struct {
	name       string
	playerIDs  []string
	start      FullState
	before     func(t *testing.T, match *Match)
	turns      []scenarioTurn
	finalCheck func(t *testing.T, match *Match)
}

type scenarioTurn struct {
	actions map[string]Action
	check   func(t *testing.T, match *Match)
}

func TestTargetedScenarioCatalog(t *testing.T) {
	for _, scenario := range targetedScenarioCatalog(t) {
		t.Run(scenario.name, func(t *testing.T) {
			match := restoreFixedMapScenario(t, scenario.playerIDs, scenario.start)
			if scenario.before != nil {
				scenario.before(t, match)
			}
			for i, turn := range scenario.turns {
				if err := match.Apply(turn.actions); err != nil {
					t.Fatalf("turn %d Apply: %v", i+1, err)
				}
				if turn.check != nil {
					turn.check(t, match)
				}
			}
			if scenario.finalCheck != nil {
				scenario.finalCheck(t, match)
			}
		})
	}
}

func targetedScenarioCatalog(t *testing.T) []targetedScenario {
	base := mustNewFixedMapMatch(t, []string{"p1", "p2", "p3"})
	return []targetedScenario{
		{
			name:      "contested-chest-goal-race",
			playerIDs: []string{"p1", "p2", "p3"},
			start: fixedMapState(base, FullState{
				Turn: 5,
				Players: []PlayerState{
					{PlayerID: "p1", X: 1, Y: 6},
					{PlayerID: "p2", X: 3, Y: 6},
					{PlayerID: "p3", X: 5, Y: 6},
				},
				UncollectedChests: []ChestState{
					{X: 2, Y: 6, Points: 12},
				},
			}),
			turns: []scenarioTurn{
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "right"},
						"p2": {Action: "move", Direction: "left"},
						"p3": {Action: "wait"},
					},
					check: func(t *testing.T, match *Match) {
						assertPlayerScore(t, match, "p1", 6, 0, 6)
						assertPlayerScore(t, match, "p2", 6, 0, 6)
						assertPlayerScore(t, match, "p3", 0, 0, 0)
						if got := len(match.UncollectedChests()); got != 0 {
							t.Fatalf("remaining chests = %d, want 0", got)
						}
					},
				},
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "right"},
						"p2": {Action: "move", Direction: "right"},
						"p3": {Action: "move", Direction: "right"},
					},
					check: func(t *testing.T, match *Match) {
						assertFinishedTurn(t, match, "p3", 7)
						assertPlayerScore(t, match, "p3", 0, 100, 100)
					},
				},
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "right"},
						"p2": {Action: "move", Direction: "right"},
					},
				},
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "right"},
						"p2": {Action: "move", Direction: "right"},
					},
				},
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "right"},
						"p2": {Action: "move", Direction: "right"},
					},
				},
			},
			finalCheck: func(t *testing.T, match *Match) {
				assertPlayerScore(t, match, "p1", 6, 50, 56)
				assertPlayerScore(t, match, "p2", 6, 50, 56)
				assertPlayerScore(t, match, "p3", 0, 100, 100)
				placements := match.Placements()
				if len(placements) != 3 {
					t.Fatalf("placements = %d, want 3", len(placements))
				}
				if placements[0].Place != 1 || placements[1].Place != 2 || placements[2].Place != 2 {
					t.Fatalf("placements = %+v, want competition ranking 1,2,2", placements)
				}
			},
		},
		{
			name:      "discovery-persists-and-cleans-up",
			playerIDs: []string{"p1", "p2"},
			start: fixedMapState(base, FullState{
				Players: []PlayerState{
					{PlayerID: "p1", X: 4, Y: 6},
					{PlayerID: "p2", X: 1, Y: 6},
				},
				UncollectedChests: []ChestState{
					{X: 2, Y: 6, Points: 12},
				},
			}),
			before: func(t *testing.T, match *Match) {
				visible, err := match.CurrentVisibleState("p1")
				if err != nil {
					t.Fatalf("CurrentVisibleState before turn: %v", err)
				}
				if visible.KnownGoal == nil || *visible.KnownGoal != (Position{X: 6, Y: 6}) {
					t.Fatalf("known goal = %+v, want (6,6)", visible.KnownGoal)
				}
				if len(visible.KnownChests) != 1 || visible.KnownChests[0] != (ChestState{X: 2, Y: 6, Points: 12}) {
					t.Fatalf("known chests = %+v, want only (2,6,12)", visible.KnownChests)
				}
			},
			turns: []scenarioTurn{
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "up"},
						"p2": {Action: "move", Direction: "right"},
					},
					check: func(t *testing.T, match *Match) {
						visible, err := match.CurrentVisibleState("p1")
						if err != nil {
							t.Fatalf("CurrentVisibleState after turn: %v", err)
						}
						if visible.KnownGoal == nil || *visible.KnownGoal != (Position{X: 6, Y: 6}) {
							t.Fatalf("known goal after leaving vision = %+v, want persisted goal", visible.KnownGoal)
						}
						if len(visible.KnownChests) != 0 {
							t.Fatalf("known chests after collection = %+v, want empty", visible.KnownChests)
						}
					},
				},
			},
		},
		{
			name:      "last-turn-finish-clamps-visible-state",
			playerIDs: []string{"p1", "p2"},
			start: fixedMapState(base, FullState{
				Turn: 15,
				Players: []PlayerState{
					{PlayerID: "p1", X: 5, Y: 6},
					{PlayerID: "p2", X: 1, Y: 1},
				},
				UncollectedChests: []ChestState{},
			}),
			before: func(t *testing.T, match *Match) {
				visible, err := match.CurrentVisibleState("p1")
				if err != nil {
					t.Fatalf("CurrentVisibleState before final turn: %v", err)
				}
				if visible.Turn != 16 || visible.RemainingTurns != 1 {
					t.Fatalf("visible before final turn = turn:%d remaining:%d, want turn:16 remaining:1", visible.Turn, visible.RemainingTurns)
				}
			},
			turns: []scenarioTurn{
				{
					actions: map[string]Action{
						"p1": {Action: "move", Direction: "right"},
						"p2": {Action: "wait"},
					},
				},
			},
			finalCheck: func(t *testing.T, match *Match) {
				if !match.Terminal() {
					t.Fatal("match should be terminal after max turn")
				}
				visible, err := match.CurrentVisibleState("p1")
				if err != nil {
					t.Fatalf("CurrentVisibleState after final turn: %v", err)
				}
				if visible.Turn != 16 || visible.RemainingTurns != 0 {
					t.Fatalf("visible after final turn = turn:%d remaining:%d, want turn:16 remaining:0", visible.Turn, visible.RemainingTurns)
				}
				assertFinishedTurn(t, match, "p1", 16)
				assertPlayerScore(t, match, "p1", 0, 100, 100)
			},
		},
	}
}

func mustNewFixedMapMatch(t *testing.T, playerIDs []string) *Match {
	t.Helper()

	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   append([]string(nil), playerIDs...),
		RNGSeed:     DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New fixed map: %v", err)
	}
	return match
}

func fixedMapState(base *Match, state FullState) FullState {
	full := base.FullState()
	full.Turn = state.Turn
	if state.Players != nil {
		full.Players = append([]PlayerState(nil), state.Players...)
	}
	if state.UncollectedChests != nil {
		full.UncollectedChests = append([]ChestState(nil), state.UncollectedChests...)
	}
	if state.Discovery != nil {
		full.Discovery = state.Discovery
	} else {
		full.Discovery = blankDiscovery(full.Players)
	}
	return full
}

func restoreFixedMapScenario(t *testing.T, playerIDs []string, state FullState) *Match {
	t.Helper()

	match, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   append([]string(nil), playerIDs...),
		RNGSeed:     DefaultRNGSeed,
	}, state)
	if err != nil {
		t.Fatalf("NewFromFullState: %v", err)
	}
	return match
}

func blankDiscovery(players []PlayerState) map[string]DiscoveryState {
	discovery := make(map[string]DiscoveryState, len(players))
	for _, player := range players {
		discovery[player.PlayerID] = DiscoveryState{}
	}
	return discovery
}

func assertPlayerScore(t *testing.T, match *Match, playerID string, chest, goal, score int) {
	t.Helper()

	player := mustFindPlayerState(t, match.scoreboardWithPositions(), playerID)
	if player.ChestPoints != chest || player.GoalBonus != goal || player.Score != score {
		t.Fatalf("%s = chest:%d goal:%d score:%d, want chest:%d goal:%d score:%d",
			player.PlayerID, player.ChestPoints, player.GoalBonus, player.Score,
			chest, goal, score)
	}
}

func assertFinishedTurn(t *testing.T, match *Match, playerID string, want int) {
	t.Helper()

	player := mustFindPlayerState(t, match.scoreboardWithPositions(), playerID)
	if player.FinishedTurn == nil || *player.FinishedTurn != want {
		t.Fatalf("%s finished turn = %v, want %d", player.PlayerID, player.FinishedTurn, want)
	}
}

func mustFindPlayerState(t *testing.T, players []PlayerState, playerID string) PlayerState {
	t.Helper()

	for _, player := range players {
		if player.PlayerID == playerID {
			return player
		}
	}
	t.Fatalf("player %q missing", playerID)
	return PlayerState{}
}
