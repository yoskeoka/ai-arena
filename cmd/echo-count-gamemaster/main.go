// Command echo-count-gamemaster runs the subprocess game-master bridge for echo-count.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/internal/games/echo"
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
	var gameID string
	var gameVersion string
	var ruleset string

	fs := flag.NewFlagSet("echo-count-gamemaster", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&gameID, "game-id", "", "game id")
	fs.StringVar(&gameVersion, "game-version", "", "game version")
	fs.StringVar(&ruleset, "ruleset", "", "ruleset")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	selectedMeta, _, _, _, err := echo.MetadataForSelectionWithGameID(gameID, gameVersion, ruleset)
	if err != nil {
		return err
	}
	meta := gameMetadata{
		GameID:         selectedMeta.GameID,
		GameVersion:    selectedMeta.GameVersion,
		RulesetVersion: selectedMeta.RulesetVersion,
	}

	dec := protocol.NewDecoder(os.Stdin)
	enc := protocol.NewEncoder(os.Stdout)

	state := &serverState{}
	for {
		req, err := dec.DecodeRequest()
		if err != nil {
			return err
		}
		resp, exit, err := handleRequest(context.Background(), meta, state, req)
		if err != nil {
			resp = protocol.Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &protocol.ErrorObject{
					Code:    -32000,
					Message: err.Error(),
				},
			}
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
		if exit {
			return nil
		}
	}
}

type serverState struct {
	master  game.Master
	players []game.Player
}

func handleRequest(ctx context.Context, meta gameMetadata, state *serverState, req protocol.Request) (protocol.Response, bool, error) {
	switch req.Method {
	case "metadata":
		resp, err := protocol.NewResponse(req.ID, meta)
		return resp, false, err
	case "initialize_match":
		var params gamemaster.InitializeMatchParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		cfg := echo.Config{
			GameID:      meta.GameID,
			GameVersion: meta.GameVersion,
			Ruleset:     meta.RulesetVersion,
			Players:     append([]game.Player(nil), params.Players...),
		}
		var impl game.Master
		var err error
		if params.ResumeSnapshot == nil {
			impl, err = echo.New(cfg)
		} else {
			impl, err = echo.NewFromSnapshot(cfg, *params.ResumeSnapshot)
		}
		if err != nil {
			return protocol.Response{}, false, err
		}
		state.master = impl
		state.players = append([]game.Player(nil), params.Players...)
		initState, err := impl.Init(ctx)
		if err != nil {
			return protocol.Response{}, false, err
		}
		resp, err := protocol.NewResponse(req.ID, gamemaster.InitializeMatchResult{InitState: initState})
		return resp, false, err
	case "next_decision_step":
		if state.master == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		step, err := state.master.NextStep(ctx)
		if err != nil {
			return protocol.Response{}, false, err
		}
		resp, err := protocol.NewResponse(req.ID, step)
		return resp, false, err
	case "normalize_action":
		if state.master == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		var params struct {
			Request      game.DecisionRequest `json:"request"`
			ActionStatus game.ActionStatus    `json:"action_status"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		normalized := state.master.NormalizeAction(params.Request, params.ActionStatus)
		resp, err := protocol.NewResponse(req.ID, normalized)
		return resp, false, err
	case "apply_decision_results":
		if state.master == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		var params gamemaster.ApplyDecisionResultsParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		if err := state.master.ApplyStep(ctx, params.Step, params.ActionStatuses); err != nil {
			return protocol.Response{}, false, err
		}
		resp, err := protocol.NewResponse(req.ID, map[string]bool{"ok": true})
		return resp, false, err
	case "current_snapshot":
		if state.master == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		snapshot := state.master.Snapshot()
		if snapshot.PerPlayer == nil {
			snapshot.PerPlayer = make(map[string]game.PlayerSnapshot)
		}
		for _, player := range state.players {
			playerState := snapshot.PerPlayer[player.PlayerID]
			if len(playerState.VisibleState) == 0 {
				playerState.VisibleState = state.master.VisibleState(player.PlayerID)
				snapshot.PerPlayer[player.PlayerID] = playerState
			}
		}
		resp, err := protocol.NewResponse(req.ID, snapshot)
		return resp, false, err
	case "current_exported_snapshot":
		if state.master == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		resp, err := protocol.NewResponse(req.ID, state.master.ExportedSnapshot())
		return resp, false, err
	case "current_result":
		if state.master == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		resp, err := protocol.NewResponse(req.ID, state.master.Result())
		return resp, false, err
	case "shutdown":
		resp, err := protocol.NewResponse(req.ID, map[string]bool{"ok": true})
		return resp, true, err
	default:
		return protocol.Response{}, false, fmt.Errorf("unsupported method %q", req.Method)
	}
}

type gameMetadata struct {
	GameID         string `json:"game_id"`
	GameVersion    string `json:"game_version"`
	RulesetVersion string `json:"ruleset_version"`
}
