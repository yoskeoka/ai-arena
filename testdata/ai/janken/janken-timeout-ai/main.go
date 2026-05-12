// Command janken-timeout-ai times out on configured rounds.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/janken/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID:          "janken-timeout-ai",
		Actions:       []string{"rock", "rock", "rock", "rock", "rock"},
		TimeoutRounds: map[int]bool{1: true},
	}); err != nil {
		log.Fatal(err)
	}
}
