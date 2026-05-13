// Command bad-json-ai emits malformed JSON for protocol verification.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/echo/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{AIID: "bad-json-ai", BadJSON: true}); err != nil {
		log.Fatal(err)
	}
}
