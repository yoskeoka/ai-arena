package fixturebot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

type Behavior struct {
	AIID               string
	InitTimeout        bool
	ExitAfterInit      bool
	TimeoutOnFirstTurn bool
	InvalidAction      bool
	BadJSON            bool
	MismatchedID       bool
	LateResponse       bool
	HungAfterGameOver  bool
}

type turnState struct {
	Turn     int `json:"turn"`
	Expected int `json:"expected"`
}

func Run(behavior Behavior) error {
	dec := protocol.NewDecoder(os.Stdin)
	enc := protocol.NewEncoder(os.Stdout)
	turnCount := 0

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
			if behavior.InitTimeout {
				time.Sleep(2 * time.Second)
				continue
			}
			resp, err := protocol.NewResponse(req.ID, map[string]any{"ready": true})
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
			if behavior.ExitAfterInit {
				return nil
			}
		case "turn":
			turnCount++
			fmt.Fprintf(os.Stderr, "%s turn %d\n", behavior.AIID, turnCount)

			if behavior.TimeoutOnFirstTurn && turnCount == 1 {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if behavior.BadJSON {
				_, _ = fmt.Fprintln(os.Stdout, "{bad json")
				continue
			}

			var state turnState
			if params := req.Params; len(params) > 0 {
				var payload struct {
					VisibleState turnState `json:"visible_state"`
				}
				_ = json.Unmarshal(params, &payload)
				state = payload.VisibleState
			}

			if behavior.LateResponse && turnCount == 1 {
				time.Sleep(120 * time.Millisecond)
			}

			respID := req.ID
			if behavior.MismatchedID {
				respID = req.ID + "-wrong"
			}

			result := map[string]any{"echo": state.Expected}
			if behavior.InvalidAction {
				result = map[string]any{"echo": state.Expected + 1000}
			}

			resp, err := protocol.NewResponse(respID, result)
			if err != nil {
				return err
			}
			if err := enc.Encode(resp); err != nil {
				return err
			}
		case "game_over":
			fmt.Fprintf(os.Stderr, "%s game_over\n", behavior.AIID)
			if behavior.HungAfterGameOver {
				ch := make(chan os.Signal, 1)
				signal.Notify(ch, os.Interrupt)
				defer signal.Stop(ch)
				for {
					select {
					case <-ch:
					case <-time.After(50 * time.Millisecond):
					}
				}
			}
			return nil
		}
	}
}
