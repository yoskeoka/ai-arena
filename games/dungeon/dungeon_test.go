package dungeon

import "testing"

func TestChestSplitAndGoalBonuses(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Turn 1: move both players toward their nearest visible chest path.
	if err := match.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "down"},
		"p2": {Action: "move", Direction: "down"},
	}); err != nil {
		t.Fatalf("turn1: %v", err)
	}
	if err := match.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "down"},
		"p2": {Action: "move", Direction: "down"},
	}); err != nil {
		t.Fatalf("turn2: %v", err)
	}
	if err := match.Apply(map[string]Action{
		"p1": {Action: "move", Direction: "right"},
		"p2": {Action: "move", Direction: "left"},
	}); err != nil {
		t.Fatalf("turn3: %v", err)
	}

	full := match.FullState()
	if got := full.Players[0].ChestPoints + full.Players[1].ChestPoints; got != 24 {
		t.Fatalf("combined chest points = %d, want 24", got)
	}

	for !match.Terminal() {
		if err := match.Apply(map[string]Action{
			"p1": {Action: "wait"},
			"p2": {Action: "wait"},
		}); err != nil {
			t.Fatalf("advance to terminal: %v", err)
		}
	}
	if !match.Terminal() {
		t.Fatal("expected terminal after max turns")
	}
	if match.Turn() != match.Ruleset().MaxTurns {
		t.Fatalf("turn = %d, want %d", match.Turn(), match.Ruleset().MaxTurns)
	}
}

func TestCompetitionRankingGoalBonus(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2", "p3"},
		RNGSeed:     DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Manually restore a near-finish state to verify ranking math.
	restored, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2", "p3"},
		RNGSeed:     DefaultRNGSeed,
	}, FullState{
		MapID:    RulesetFixedMapV1,
		RNGSeed:  DefaultRNGSeed,
		Turn:     5,
		MaxTurns: 16,
		Goal:     match.Ruleset().Goal,
		Players: []PlayerState{
			{PlayerID: "p1", X: 5, Y: 6},
			{PlayerID: "p2", X: 5, Y: 6},
			{PlayerID: "p3", X: 6, Y: 5},
		},
		UncollectedChests: match.UncollectedChests(),
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
		"p2": {Action: "move", Direction: "right"},
		"p3": {Action: "wait"},
	}); err != nil {
		t.Fatalf("Apply first finish turn: %v", err)
	}
	if err := restored.Apply(map[string]Action{
		"p3": {Action: "move", Direction: "down"},
	}); err != nil {
		t.Fatalf("Apply second finish turn: %v", err)
	}
	scores := restored.scoreboardWithPositions()
	for _, player := range scores {
		switch player.PlayerID {
		case "p1", "p2":
			if player.GoalBonus != 100 {
				t.Fatalf("%s goal bonus = %d, want 100", player.PlayerID, player.GoalBonus)
			}
		case "p3":
			if player.GoalBonus != 25 {
				t.Fatalf("p3 goal bonus = %d, want 25", player.GoalBonus)
			}
		}
	}
}

func TestCurrentVisibleStateClampsTerminalTurn(t *testing.T) {
	match, err := New(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     DefaultRNGSeed,
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

func TestNewFromFullStateValidatesResumeMetadata(t *testing.T) {
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
	state.RNGSeed = 99
	if _, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     DefaultRNGSeed,
	}, state); err == nil {
		t.Fatal("expected rng seed mismatch")
	}

	state = match.FullState()
	state.MaxTurns++
	if _, err := NewFromFullState(Config{
		GameVersion: GameVersion,
		Ruleset:     RulesetFixedMapV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     DefaultRNGSeed,
	}, state); err == nil {
		t.Fatal("expected max_turns mismatch")
	}
}
