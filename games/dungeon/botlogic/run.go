package botlogic

import (
	"encoding/json"
	"io"

	"github.com/yoskeoka/ai-arena/games/dungeon"
)

// Run serves the balanced reference bot over NDJSON JSON-RPC.
func Run(r io.Reader, w io.Writer) error {
	return RunWithPolicy(r, w, BalancedPolicy())
}

// RunWithPolicy serves a named policy variant over the shared dungeon AI protocol.
func RunWithPolicy(r io.Reader, w io.Writer, policy Policy) error {
	dec := newDecoder(r)
	enc := newEncoder(w)
	bot := NewWithPolicy(policy)

	for {
		req, err := dec.decodeRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch req.Method {
		case "init":
			resp, err := newResponse(req.ID, map[string]any{"ready": true, "policy": policy.Name()})
			if err != nil {
				return err
			}
			if err := enc.encode(resp); err != nil {
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
			resp, err := newResponse(req.ID, action)
			if err != nil {
				return err
			}
			if err := enc.encode(resp); err != nil {
				return err
			}
		case "game_over":
			resp, err := newResponse(req.ID, map[string]any{"ack": true})
			if err != nil {
				return err
			}
			if err := enc.encode(resp); err != nil {
				return err
			}
			return nil
		}
	}
}
