// Command invalid-action-ai returns semantically invalid echo actions.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "invalid-action-ai", InvalidAction: true}); err != nil {
		log.Fatal(err)
	}
}
