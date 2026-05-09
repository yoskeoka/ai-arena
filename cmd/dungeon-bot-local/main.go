// Command dungeon-bot-local runs a local dungeon bot over NDJSON JSON-RPC.
//
// This command lives in the monorepo for convenience, but it is expected to
// remain movable to a separate dungeon repository. Keep it free of ai-arena
// internal dependencies so local verification can reflect the portable game
// boundary directly.
package main

import (
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon/botlogic"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	return botlogic.Run(os.Stdin, os.Stdout)
}
