// Command dungeon-map-helper prints fixed-map layout and shortest-path helpers.
//
// This debug and verification CLI is allowed to live in the monorepo, but it
// should remain movable with the dungeon game to a separate repository. Keep
// the helper on public dungeon APIs and avoid ai-arena internal dependencies.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var (
		ruleset = dungeon.RulesetSeededMazeV1
		rngSeed = dungeon.DefaultRNGSeed
	)
	fs := flag.NewFlagSet("dungeon-map-helper", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&ruleset, "ruleset", ruleset, "dungeon ruleset")
	fs.StringVar(&rngSeed, "rng-seed", rngSeed, "deterministic seed")
	if err := fs.Parse(args); err != nil {
		return err
	}

	world, err := dungeon.New(dungeon.Config{
		GameVersion: dungeon.GameVersion,
		Ruleset:     ruleset,
		PlayerIDs:   []string{"p1", "p2"},
		RNGSeed:     rngSeed,
	})
	if err != nil {
		return err
	}

	fmt.Printf("map_id=%s rng_seed=%s max_turns=%d view_radius=%d goal_bonuses=%v\n",
		world.Ruleset().MapID,
		world.PublicState().RNGSeed,
		world.Ruleset().MaxTurns,
		world.Ruleset().ViewRadius,
		world.Ruleset().GoalBonuses,
	)
	for _, row := range world.PublicState().Tiles {
		fmt.Println(row)
	}
	for i, spawn := range world.SpawnPoints()[:2] {
		path, ok := world.ShortestPath(spawn, world.Ruleset().Goal)
		if !ok {
			return fmt.Errorf("no path from spawn %d to goal", i+1)
		}
		fmt.Printf("spawn_%d_to_goal steps=%d route=%v\n", i+1, len(path)-1, path)
	}
	for i, chest := range world.Ruleset().InitialChests {
		path, ok := world.ShortestPath(world.SpawnPoints()[0], dungeon.Position{X: chest.X, Y: chest.Y})
		if !ok {
			return fmt.Errorf("no path from spawn 1 to chest %d", i+1)
		}
		fmt.Printf("spawn_1_to_chest_%d points=%d steps=%d route=%v\n", i+1, chest.Points, len(path)-1, path)
	}
	return nil
}
