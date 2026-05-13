// Package fixturebot provides lightweight janken fixture bots for tests.
package fixturebot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

// Behavior configures how the janken fixture bot responds.
type Behavior struct {
	AIID          string
	Actions       []string
	TimeoutRounds map[int]bool
}

// Run serves the JSON-RPC loop for one configured fixture bot.
func Run(behavior Behavior) error {
	dec := protocol.NewDecoder(os.Stdin)
	enc := protocol.NewEncoder(os.Stdout)

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
			fmt.Fprintf(os.Stderr, "%s init\n", behavior.AIID)
			resp, err := protocol.NewResponse(req.ID, map[string]any{"ready": true})
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
		case "turn":
			var payload struct {
				VisibleState struct {
					Round int `json:"round"`
				} `json:"visible_state"`
			}
			if err := json.Unmarshal(req.Params, &payload); err != nil {
				return err
			}
			round := payload.VisibleState.Round
			fmt.Fprintf(os.Stderr, "%s turn %d\n", behavior.AIID, round)
			if behavior.TimeoutRounds[round] {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			action := pickAction(behavior.Actions, round)
			resp, err := protocol.NewResponse(req.ID, map[string]any{"action": action})
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
		case "game_over":
			fmt.Fprintf(os.Stderr, "%s game_over\n", behavior.AIID)
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

func pickAction(actions []string, round int) string {
	if len(actions) == 0 {
		return "rock"
	}
	index := round - 1
	if index < 0 {
		index = 0
	}
	return actions[index%len(actions)]
}
