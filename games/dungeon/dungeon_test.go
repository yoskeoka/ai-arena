package dungeon

import "testing"

const (
	testSeedAlpha = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
	testSeedBeta  = "ffeeddccbbaa99887766554433221100ffeeddccbbaa99887766554433221100"
)

func TestSeededMazeGenerationIsDeterministic(t *testing.T) {
	cfg := Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	}
	first, err := New(cfg)
	if err != nil {
		t.Fatalf("first New: %v", err)
	}
	second, err := New(cfg)
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	if !equalStringSlices(first.Ruleset().Tiles, second.Ruleset().Tiles) {
		t.Fatal("tiles differ for same seed")
	}
	if !equalPositions(first.Ruleset().SpawnPoints, second.Ruleset().SpawnPoints) {
		t.Fatal("spawn points differ for same seed")
	}
	if first.Ruleset().Goal != second.Ruleset().Goal {
		t.Fatal("goal differs for same seed")
	}
	if !equalChests(first.Ruleset().InitialChests, second.Ruleset().InitialChests) {
		t.Fatal("initial chests differ for same seed")
	}
}

func TestSeededMazeGenerationVariesAcrossSeeds(t *testing.T) {
	first, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New alpha: %v", err)
	}
	second, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedBeta,
	})
	if err != nil {
		t.Fatalf("New beta: %v", err)
	}
	sameTiles := equalStringSlices(first.Ruleset().Tiles, second.Ruleset().Tiles)
	sameGoal := first.Ruleset().Goal == second.Ruleset().Goal
	sameChests := equalChests(first.Ruleset().InitialChests, second.Ruleset().InitialChests)
	if sameTiles && sameGoal && sameChests {
		t.Fatal("expected different generated state for different seeds")
	}
}

func TestSeededMazeUsesFixedChestScoreSet(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	total := 0
	got := append([]ChestState(nil), match.Ruleset().InitialChests...)
	for _, chest := range got {
		total += chest.Points
	}
	if total != 54 {
		t.Fatalf("total chest points = %d, want 54", total)
	}
	expected := map[int]int{24: 1, 18: 1, 12: 1}
	for _, chest := range got {
		expected[chest.Points]--
	}
	for points, count := range expected {
		if count != 0 {
			t.Fatalf("score set mismatch for %d: remaining %d", points, count)
		}
	}
}

func TestSeededMazeUsesExpandedTurnBudget(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if match.Ruleset().MaxTurns != 50 {
		t.Fatalf("max turns = %d, want 50", match.Ruleset().MaxTurns)
	}
	visible, err := match.CurrentVisibleState("p1")
	if err != nil {
		t.Fatalf("CurrentVisibleState: %v", err)
	}
	if visible.RemainingTurns != 50 {
		t.Fatalf("remaining turns = %d, want 50", visible.RemainingTurns)
	}
}

func TestNewGeneratesSeedWhenOmitted(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(match.FullState().RNGSeed) != 64 {
		t.Fatalf("generated rng seed length = %d, want 64", len(match.FullState().RNGSeed))
	}
}

func TestFixedMapRulesetRemainsResumable(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state := match.FullState()
	restored, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     DefaultRNGSeed,
	}, state)
	if err != nil {
		t.Fatalf("NewFromFullState: %v", err)
	}
	if !equalStringSlices(restored.Ruleset().Tiles, match.Ruleset().Tiles) {
		t.Fatal("restored tiles differ")
	}
	if !equalChests(restored.Ruleset().InitialChests, match.Ruleset().InitialChests) {
		t.Fatal("restored chests differ")
	}
}

func TestNewFromFullStateValidatesGeneratedSeed(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state := match.FullState()
	state.RNGSeed = testSeedBeta
	if _, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	}, state); err == nil {
		t.Fatal("expected rng seed mismatch")
	}
}

func TestNewFromFullStateUsesSnapshotSeedWhenConfigOmitted(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state := match.FullState()
	restored, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
	}, state)
	if err != nil {
		t.Fatalf("NewFromFullState: %v", err)
	}
	if restored.FullState().RNGSeed != state.RNGSeed {
		t.Fatalf("restored rng seed = %q, want %q", restored.FullState().RNGSeed, state.RNGSeed)
	}
}

func TestNewRejectsInvalidSeedFormat(t *testing.T) {
	if _, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     "alpha",
	}); err == nil {
		t.Fatal("expected invalid seed format error")
	}
}

func TestChestSplitAndGoalBonusesStillApply(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2", "p3"},
		RNGSeed:     DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	restored, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2", "p3"},
		RNGSeed:     DefaultRNGSeed,
	}, FullState{
		MapID:         RulesetFixedMapV1,
		RNGSeed:       DefaultRNGSeed,
		Turn:          5,
		MaxTurns:      match.Ruleset().MaxTurns,
		Tiles:         append([]string(nil), match.Ruleset().Tiles...),
		SpawnPoints:   append([]Position(nil), match.Ruleset().SpawnPoints...),
		Goal:          match.Ruleset().Goal,
		InitialChests: append([]ChestState(nil), match.Ruleset().InitialChests...),
		Players: []PlayerState{
			{PlayerID: "p1", X: 1, Y: 6},
			{PlayerID: "p2", X: 3, Y: 6},
			{PlayerID: "p3", X: 5, Y: 6},
		},
		UncollectedChests: []ChestState{
			{X: 2, Y: 6, Points: 12},
		},
		Discovery: map[string]DiscoveryState{
			"p1": {},
			"p2": {},
			"p3": {},
		},
	})
	if err != nil {
		t.Fatalf("NewFromFullState: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "left"},
		"p3": {Action: "wait"},
	}); err != nil {
		t.Fatalf("Apply contested chest turn: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "right"},
		"p3": {Action: "move", Direction: "right"},
	}); err != nil {
		t.Fatalf("Apply first finish turn: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "right"},
	}); err != nil {
		t.Fatalf("Apply advance turn: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "right"},
	}); err != nil {
		t.Fatalf("Apply advance turn 2: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "right"},
	}); err != nil {
		t.Fatalf("Apply second finish turn: %v", err)
	}
	players := restored.scoreboardWithPositions()
	want := map[string]struct {
		chest int
		goal  int
		score int
	}{
		"p1": {chest: 6, goal: 50, score: 56},
		"p2": {chest: 6, goal: 50, score: 56},
		"p3": {chest: 0, goal: 100, score: 100},
	}
	for _, player := range players {
		expected := want[player.PlayerID]
		if player.ChestPoints != expected.chest || player.GoalBonus != expected.goal || player.Score != expected.score {
			t.Fatalf("%s = chest:%d goal:%d score:%d, want chest:%d goal:%d score:%d",
				player.PlayerID, player.ChestPoints, player.GoalBonus, player.Score,
				expected.chest, expected.goal, expected.score)
		}
	}
	placements := restored.Placements()
	if placements[0].Place != 1 || placements[1].Place != 2 || placements[2].Place != 2 {
		t.Fatalf("placements = %+v, want competition ranking 1,2,2", placements)
	}
}

func TestThirdPlaceWithMajorityChestPointsBeatsChestlessWinner(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2", "p3"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	path, ok := match.ShortestPath(match.Ruleset().SpawnPoints[0], match.Ruleset().Goal)
	if !ok || len(path) < 4 {
		t.Fatalf("shortest path length = %d, want >= 4", len(path))
	}
	p1Start := path[len(path)-2]
	p2Start := path[len(path)-3]
	p3Start := path[len(path)-4]
	directionTo := func(from, to Position) string {
		switch {
		case to.X == from.X && to.Y == from.Y-1:
			return "up"
		case to.X == from.X && to.Y == from.Y+1:
			return "down"
		case to.X == from.X-1 && to.Y == from.Y:
			return "left"
		case to.X == from.X+1 && to.Y == from.Y:
			return "right"
		default:
			t.Fatalf("non-adjacent move from %+v to %+v", from, to)
			return ""
		}
	}
	restored, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2", "p3"},
		RNGSeed:     testSeedAlpha,
	}, FullState{
		MapID:         RulesetSeededMazeV1,
		RNGSeed:       testSeedAlpha,
		Turn:          5,
		MaxTurns:      match.Ruleset().MaxTurns,
		Tiles:         append([]string(nil), match.Ruleset().Tiles...),
		SpawnPoints:   append([]Position(nil), match.Ruleset().SpawnPoints...),
		Goal:          match.Ruleset().Goal,
		InitialChests: append([]ChestState(nil), match.Ruleset().InitialChests...),
		Players: []PlayerState{
			{PlayerID: "p1", X: p1Start.X, Y: p1Start.Y},
			{PlayerID: "p2", X: p2Start.X, Y: p2Start.Y},
			{PlayerID: "p3", X: p3Start.X, Y: p3Start.Y, Score: 30, ChestPoints: 30},
		},
		UncollectedChests: []ChestState{},
		Discovery: map[string]DiscoveryState{
			"p1": {},
			"p2": {},
			"p3": {},
		},
	})
	if err != nil {
		t.Fatalf("NewFromFullState: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p1": {Action: "move", Direction: directionTo(p1Start, match.Ruleset().Goal)},
		"p2": {Action: "move", Direction: directionTo(p2Start, p1Start)},
		"p3": {Action: "move", Direction: directionTo(p3Start, p2Start)},
	}); err != nil {
		t.Fatalf("Apply first finish turn: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p2": {Action: "move", Direction: directionTo(p1Start, match.Ruleset().Goal)},
		"p3": {Action: "move", Direction: directionTo(p2Start, p1Start)},
	}); err != nil {
		t.Fatalf("Apply second finish turn: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p3": {Action: "move", Direction: directionTo(p1Start, match.Ruleset().Goal)},
	}); err != nil {
		t.Fatalf("Apply third finish turn: %v", err)
	}
	players := restored.scoreboardWithPositions()
	want := map[string]struct {
		chest int
		goal  int
		score int
	}{
		"p1": {chest: 0, goal: 42, score: 42},
		"p2": {chest: 0, goal: 28, score: 28},
		"p3": {chest: 30, goal: 14, score: 44},
	}
	for _, player := range players {
		expected := want[player.PlayerID]
		if player.ChestPoints != expected.chest || player.GoalBonus != expected.goal || player.Score != expected.score {
			t.Fatalf("%s = chest:%d goal:%d score:%d, want chest:%d goal:%d score:%d",
				player.PlayerID, player.ChestPoints, player.GoalBonus, player.Score,
				expected.chest, expected.goal, expected.score)
		}
	}
	placements := restored.Placements()
	if placements[0].PlayerID != "p3" || placements[0].Place != 1 {
		t.Fatalf("placements[0] = %+v, want p3 first", placements[0])
	}
	if placements[1].PlayerID != "p1" || placements[1].Place != 2 {
		t.Fatalf("placements[1] = %+v, want p1 second", placements[1])
	}
	if placements[2].PlayerID != "p2" || placements[2].Place != 3 {
		t.Fatalf("placements[2] = %+v, want p2 third", placements[2])
	}
}

func TestCurrentVisibleStateClampsTerminalTurn(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     testSeedAlpha,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	for !match.Terminal() {
		if err := match.Apply(map[string]Action{
			"p1": {Action: "wait"},
			"p2": {Action: "wait"},
		}); err != nil {
			t.Fatalf("Apply: %v", err)
		}
	}
	visible, err := match.CurrentVisibleState("p1")
	if err != nil {
		t.Fatalf("CurrentVisibleState: %v", err)
	}
	if visible.Turn != match.Turn() {
		t.Fatalf("visible turn = %d, want %d", visible.Turn, match.Turn())
	}
	if visible.RemainingTurns != 0 {
		t.Fatalf("remaining turns = %d, want 0", visible.RemainingTurns)
	}
}
