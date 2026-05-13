// Command janken-invalid-ai returns invalid janken actions.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/janken/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID:    "janken-invalid-ai",
		Actions: []string{"lizard", "lizard", "lizard", "lizard", "lizard"},
	}); err != nil {
		log.Fatal(err)
	}
}
