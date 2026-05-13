// Command dungeon-go-wasm-ai runs the Go WASM dungeon fixture bot.
package main

import (
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon/botlogic"
)

func main() {
	if err := botlogic.Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
