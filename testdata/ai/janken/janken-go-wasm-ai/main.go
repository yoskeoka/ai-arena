// Command janken-go-wasm-ai runs the Go WASM janken fixture bot.
package main

import (
	"log"

	"github.com/yoskeoka/ai-arena/testdata/ai/janken/fixturebot"
)

func main() {
	if err := fixturebot.Run(fixturebot.Behavior{
		AIID:    "janken-go-wasm-ai",
		Actions: []string{"paper", "paper", "paper", "paper", "paper"},
	}); err != nil {
		log.Fatal(err)
	}
}
