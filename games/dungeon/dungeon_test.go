package dungeon

import "testing"

func TestSeededMazeGenerationIsDeterministic(t *testing.T) {
	cfg := Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     "alpha",
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
		RNGSeed:     "alpha",
	})
	if err != nil {
		t.Fatalf("New alpha: %v", err)
	}
	second, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     "beta",
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
		RNGSeed:     "alpha",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	total := 0
	got := append([]ChestState(nil), match.Ruleset().InitialChests...)
	for _, chest := range got {
		total += chest.Points
	}
	if total != 48 {
		t.Fatalf("total chest points = %d, want 48", total)
	}
	expected := map[int]int{24: 1, 12: 2}
	for _, chest := range got {
		expected[chest.Points]--
	}
	for points, count := range expected {
		if count != 0 {
			t.Fatalf("score set mismatch for %d: remaining %d", points, count)
		}
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
		RNGSeed:     "alpha",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state := match.FullState()
	state.RNGSeed = "beta"
	if _, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     "alpha",
	}, state); err == nil {
		t.Fatal("expected rng seed mismatch")
	}
}

func TestCurrentVisibleStateClampsTerminalTurn(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     "alpha",
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
