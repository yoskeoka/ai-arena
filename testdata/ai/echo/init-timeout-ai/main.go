package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "init-timeout-ai", InitTimeout: true}); err != nil {
		log.Fatal(err)
	}
}
