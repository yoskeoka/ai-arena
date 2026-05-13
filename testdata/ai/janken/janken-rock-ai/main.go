// Command janken-rock-ai always returns rock.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/janken/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID:    "janken-rock-ai",
		Actions: []string{"rock", "rock", "rock", "rock", "rock"},
	}); err != nil {
		log.Fatal(err)
	}
}
