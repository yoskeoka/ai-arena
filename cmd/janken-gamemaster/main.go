// Command janken-gamemaster runs the WASI-compatible janken game-master bridge.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/internal/games/janken"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	meta := gameMetadata{GameID: janken.GameID, GameVersion: janken.GameVersion, RulesetVersion: janken.RulesetRegular}
	decoder, encoder := protocol.NewDecoder(os.Stdin), protocol.NewEncoder(os.Stdout)
	state := &serverState{}
	for {
		req, err := decoder.DecodeRequest()
		if err != nil {
			return err
		}
		response, shutdown, err := handle(context.Background(), meta, state, req)
		if err != nil {
			response = protocol.Response{JSONRPC: "2.0", ID: req.ID, Error: &protocol.ErrorObject{Code: -32000, Message: err.Error()}}
		}
		if err := encoder.Encode(response); err != nil {
			return err
		}
		if shutdown {
			return nil
		}
	}
}

type gameMetadata struct {
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}
type serverState struct {
	master  game.Master
	players []game.Player
}

func handle(ctx context.Context, meta gameMetadata, state *serverState, req protocol.Request) (protocol.Response, bool, error) {
	respond := func(value any) (protocol.Response, bool, error) {
		response, err := protocol.NewResponse(req.ID, value)
		return response, false, err
	}
	if req.Method == "metadata" {
		return respond(meta)
	}
	if req.Method == "shutdown" {
		response, err := protocol.NewResponse(req.ID, map[string]bool{"ok": true})
		return response, true, err
	}
	if req.Method == "initialize_match" {
		var params gamemaster.InitializeMatchParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		var err error
		if params.ResumeSnapshot == nil {
			state.master, err = janken.New(janken.Config{GameVersion: meta.GameVersion, Ruleset: meta.RulesetVersion, Players: append([]game.Player(nil), params.Players...)})
		} else {
			state.master, err = janken.NewFromSnapshot(janken.Config{GameVersion: meta.GameVersion, Ruleset: meta.RulesetVersion, Players: append([]game.Player(nil), params.Players...)}, *params.ResumeSnapshot)
		}
		if err != nil {
			return protocol.Response{}, false, err
		}
		state.players = append([]game.Player(nil), params.Players...)
		initState, err := state.master.Init(ctx)
		if err != nil {
			return protocol.Response{}, false, err
		}
		return respond(gamemaster.InitializeMatchResult{InitState: initState})
	}
	if state.master == nil {
		return protocol.Response{}, false, fmt.Errorf("match is not initialized")
	}
	switch req.Method {
	case "next_decision_step":
		value, err := state.master.NextStep(ctx)
		if err != nil {
			return protocol.Response{}, false, err
		}
		return respond(value)
	case "normalize_action":
		var params struct {
			Request      game.DecisionRequest `json:"request"`
			ActionStatus game.ActionStatus    `json:"action_status"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		return respond(state.master.NormalizeAction(params.Request, params.ActionStatus))
	case "apply_decision_results":
		var params gamemaster.ApplyDecisionResultsParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		if err := state.master.ApplyStep(ctx, params.Step, params.ActionStatuses); err != nil {
			return protocol.Response{}, false, err
		}
		return respond(map[string]bool{"ok": true})
	case "current_snapshot":
		snapshot := state.master.Snapshot()
		if snapshot.PerPlayer == nil {
			snapshot.PerPlayer = make(map[string]game.PlayerSnapshot)
		}
		for _, player := range state.players {
			item := snapshot.PerPlayer[player.PlayerID]
			if len(item.VisibleState) == 0 {
				item.VisibleState = state.master.VisibleState(player.PlayerID)
				snapshot.PerPlayer[player.PlayerID] = item
			}
		}
		return respond(snapshot)
	case "current_exported_snapshot":
		return respond(state.master.ExportedSnapshot())
	case "current_result":
		return respond(state.master.Result())
	default:
		return protocol.Response{}, false, fmt.Errorf("unsupported method %q", req.Method)
	}
}
