package main

import (
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon/botlogic"
)

func main() {
	fmt.Fprintln(os.Stderr, "dungeon-goal-rush-ai-wasm policy=goal-rush")
	if err := botlogic.RunWithPolicy(os.Stdin, os.Stdout, botlogic.GoalRushPolicy()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
