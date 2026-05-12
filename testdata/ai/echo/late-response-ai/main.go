// Command late-response-ai emits a response after the deadline.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "late-response-ai", LateResponse: true}); err != nil {
		log.Fatal(err)
	}
}
