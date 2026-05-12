// Command dungeon-wait-ai always returns the wait action.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/dungeon/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID: "dungeon-wait-ai",
	}); err != nil {
		log.Fatal(err)
	}
}
