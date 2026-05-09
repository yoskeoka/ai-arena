package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/games/dungeon"
	"github.com/yoskeoka/ai-arena/testdata/ai/dungeon/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID: "dungeon-goal-rush-ai-wasm",
		Actions: []dungeon.Action{
			{Action: "move", Direction: "left"},
			{Action: "move", Direction: "left"},
			{Action: "move", Direction: "left"},
			{Action: "move", Direction: "left"},
			{Action: "move", Direction: "down"},
			{Action: "move", Direction: "down"},
			{Action: "move", Direction: "left"},
			{Action: "move", Direction: "left"},
			{Action: "move", Direction: "down"},
			{Action: "move", Direction: "down"},
			{Action: "move", Direction: "right"},
			{Action: "move", Direction: "right"},
			{Action: "move", Direction: "right"},
			{Action: "move", Direction: "right"},
			{Action: "move", Direction: "down"},
			{Action: "move", Direction: "down"},
			{Action: "wait"},
		},
	}); err != nil {
		log.Fatal(err)
	}
}
