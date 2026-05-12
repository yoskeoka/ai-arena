// Command echo-ai runs the happy-path echo fixture bot.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "echo-ai"}); err != nil {
		log.Fatal(err)
	}
}
