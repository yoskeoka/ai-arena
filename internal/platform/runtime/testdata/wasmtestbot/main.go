// Command wasmtestbot acts as a small WASM runtime fixture bot.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	switch os.Getenv("BOT_MODE") {
	case "boot-response":
		fmt.Fprintln(os.Stderr, "runtime stderr")
		fmt.Println(`{"jsonrpc":"2.0","id":"boot","result":{"ready":true}}`)
	case "bad-json":
		fmt.Println("not-json")
	case "exit-after-init":
		runExitAfterInit()
	case "session-bot":
		runSessionBot()
	default:
		fmt.Fprintln(os.Stderr, "unknown BOT_MODE")
		os.Exit(2)
	}
}

func runSessionBot() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.Contains(line, `"id":"init"`):
			fmt.Println(`{"jsonrpc":"2.0","id":"init","result":{"ready":true}}`)
		case strings.Contains(line, `"id":"turn-timeout"`):
			time.Sleep(200 * time.Millisecond)
			fmt.Println(`{"jsonrpc":"2.0","id":"turn-timeout","result":{"action":"late"}}`)
		case strings.Contains(line, `"id":"turn-next"`):
			fmt.Println(`{"jsonrpc":"2.0","id":"turn-next","result":{"action":"paper"}}`)
		case strings.Contains(line, `"method":"game_over"`):
			fmt.Fprintln(os.Stderr, "game over received")
			fmt.Println(`{"jsonrpc":"2.0","id":"game-over","result":{"ack":true}}`)
			return
		}
	}
}

func runExitAfterInit() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, `"id":"init"`) {
			fmt.Println(`{"jsonrpc":"2.0","id":"init","result":{"ready":true}}`)
			return
		}
	}
}
