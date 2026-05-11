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
	"slices"

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
	layout := world.Layout()
	rulesetDef := world.Ruleset()

	fmt.Printf("map_id=%s rng_seed=%s max_turns=%d view_radius=%d goal_bonuses=%v\n",
		rulesetDef.MapID,
		world.PublicState().RNGSeed,
		rulesetDef.MaxTurns,
		rulesetDef.ViewRadius,
		rulesetDef.GoalBonuses,
	)
	chestTotal := 0
	for _, chest := range layout.InitialChests {
		chestTotal += chest.Points
	}
	majorityFloor := chestTotal/2 + 1
	thirdPlaceBonus := 0
	if len(rulesetDef.GoalBonuses) >= 3 {
		thirdPlaceBonus = rulesetDef.GoalBonuses[2]
	}
	chestValues := make([]int, 0, len(layout.InitialChests))
	for _, chest := range layout.InitialChests {
		chestValues = append(chestValues, chest.Points)
	}
	slices.Sort(chestValues)
	majorityChestMin := chestTotal
	for mask := 1; mask < 1<<len(chestValues); mask++ {
		sum := 0
		for i, points := range chestValues {
			if mask&(1<<i) != 0 {
				sum += points
			}
		}
		if sum >= majorityFloor && sum < majorityChestMin {
			majorityChestMin = sum
		}
	}
	fmt.Printf("balance chest_total=%d majority_threshold=%d first_no_chest=%d third_with_min_majority=%d\n",
		chestTotal,
		majorityFloor,
		rulesetDef.GoalBonuses[0],
		thirdPlaceBonus+majorityChestMin,
	)
	for _, row := range world.PublicState().Tiles {
		fmt.Println(row)
	}
	for i, spawn := range world.SpawnPoints()[:2] {
		path, ok := world.ShortestPath(spawn, layout.Goal)
		if !ok {
			return fmt.Errorf("no path from spawn %d to goal", i+1)
		}
		fmt.Printf("spawn_%d_to_goal steps=%d route=%v\n", i+1, len(path)-1, path)
	}
	for i, chest := range layout.InitialChests {
		path, ok := world.ShortestPath(world.SpawnPoints()[0], dungeon.Position{X: chest.X, Y: chest.Y})
		if !ok {
			return fmt.Errorf("no path from spawn 1 to chest %d", i+1)
		}
		fmt.Printf("spawn_1_to_chest_%d points=%d steps=%d route=%v\n", i+1, chest.Points, len(path)-1, path)
	}
	return nil
}
