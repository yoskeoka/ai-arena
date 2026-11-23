// Command echo-ai runs the happy-path echo fixture bot.
package main

import (
	"log"
	"os"
	"runtime"

	"github.com/yoskeoka/ai-arena/gamemaster"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if _, exists := os.LookupEnv(gamemaster.WASIRulesetVersionEnv); runtime.GOOS == "wasip1" && exists {
		log.Fatal("game master ruleset environment reached a player AI")
	}
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "echo-ai"}); err != nil {
		log.Fatal(err)
	}
}
