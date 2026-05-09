package fixturebot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon"
	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

type Behavior struct {
	AIID    string
	Actions []dungeon.Action
}

func Run(behavior Behavior) error {
	dec := protocol.NewDecoder(os.Stdin)
	enc := protocol.NewEncoder(os.Stdout)
	turnIndex := 0

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
				VisibleState dungeon.VisibleState `json:"visible_state"`
			}
			if err := json.Unmarshal(req.Params, &payload); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%s turn %d\n", behavior.AIID, payload.VisibleState.Turn)
			action := pickAction(behavior.Actions, turnIndex)
			turnIndex++
			resp, err := protocol.NewResponse(req.ID, action)
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

func pickAction(actions []dungeon.Action, turnIndex int) dungeon.Action {
	if len(actions) == 0 {
		return dungeon.Action{Action: "wait"}
	}
	if turnIndex < 0 {
		turnIndex = 0
	}
	if turnIndex >= len(actions) {
		return actions[len(actions)-1]
	}
	return actions[turnIndex]
}
