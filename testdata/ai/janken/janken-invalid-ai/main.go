package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/janken/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID:          "janken-invalid-ai",
		Actions:       []string{"rock", "rock", "rock", "rock", "rock"},
		InvalidRounds: map[int]bool{1: true},
	}); err != nil {
		log.Fatal(err)
	}
}
