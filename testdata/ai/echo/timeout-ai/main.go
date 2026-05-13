// Command timeout-ai times out on the first turn request.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "timeout-ai", TimeoutOnFirstTurn: true}); err != nil {
		log.Fatal(err)
	}
}
