package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon"
	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	dec := protocol.NewDecoder(os.Stdin)
	enc := protocol.NewEncoder(os.Stdout)
	bot := dungeon.NewBot()

	for {
		req, err := dec.DecodeRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch req.Method {
		case "init":
			resp, err := protocol.NewResponse(req.ID, map[string]any{"ready": true})
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
		case "turn":
			var payload struct {
				VisibleState dungeon.VisibleState `json:"visible_state"`
			}
			if err := json.Unmarshal(req.Params, &payload); err != nil {
				return err
			}
			action := bot.Decide(payload.VisibleState)
			resp, err := protocol.NewResponse(req.ID, action)
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
		case "game_over":
			resp, err := protocol.NewResponse(req.ID, map[string]any{"ack": true})
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
			return nil
		}
	}
}
