package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/janken/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID:    "janken-cycle-ai",
		Actions: []string{"rock", "paper", "scissors", "rock", "paper"},
	}); err != nil {
		log.Fatal(err)
	}
}
