package main

import (
	"encoding/json"
	"testing"

	"github.com/yoskeoka/ai-arena/games/dungeon"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
)

func TestCurrentExportedSnapshotHidesSeedUntilCompleted(t *testing.T) {
	world, err := dungeon.New(dungeon.Config{
		GameVersion: dungeon.GameVersion,
		Ruleset:     dungeon.RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     dungeon.DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server := serverState{
		meta: catalog.GameMetadata{
			GameID:         dungeon.GameID,
			GameVersion:    dungeon.GameVersion,
			RulesetVersion: dungeon.RulesetSeededMazeV1,
		},
		world:   world,
		players: []game.Player{{PlayerID: "p1"}, {PlayerID: "p2"}},
	}

	exported := server.currentExportedSnapshot()
	var public map[string]any
	if err := json.Unmarshal(exported.PublicState, &public); err != nil {
		t.Fatalf("unmarshal public_state: %v", err)
	}
	if _, ok := public["rng_seed"]; ok {
		t.Fatalf("rng_seed unexpectedly present before completion: %+v", public)
	}
}

func TestCurrentExportedSnapshotIncludesSeedAfterCompletion(t *testing.T) {
	world, err := dungeon.New(dungeon.Config{
		GameVersion: dungeon.GameVersion,
		Ruleset:     dungeon.RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     dungeon.DefaultRNGSeed,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	state := world.FullState()
	state.Turn = world.Ruleset().MaxTurns
	completed, err := dungeon.NewFromFullState(dungeon.Config{
		GameVersion: dungeon.GameVersion,
		Ruleset:     dungeon.RulesetSeededMazeV1,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     state.RNGSeed,
	}, state)
	if err != nil {
		t.Fatalf("NewFromFullState: %v", err)
	}
	server := serverState{
		meta: catalog.GameMetadata{
			GameID:         dungeon.GameID,
			GameVersion:    dungeon.GameVersion,
			RulesetVersion: dungeon.RulesetSeededMazeV1,
		},
		world:   completed,
		players: []game.Player{{PlayerID: "p1"}, {PlayerID: "p2"}},
	}

	exported := server.currentExportedSnapshot()
	var public struct {
		RNGSeed string `json:"rng_seed"`
	}
	if err := json.Unmarshal(exported.PublicState, &public); err != nil {
		t.Fatalf("unmarshal public_state: %v", err)
	}
	if public.RNGSeed != dungeon.DefaultRNGSeed {
		t.Fatalf("rng_seed = %q, want %q", public.RNGSeed, dungeon.DefaultRNGSeed)
	}
}
