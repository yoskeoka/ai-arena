// Command dungeon-gamemaster runs the dungeon game master sidecar.
//
// The dungeon-specific parts of this command are intended to stay movable to a
// separate repository. Avoid introducing new ai-arena internal dependencies on
// the dungeon side of the boundary; the remaining platform coupling tracked by
// docs/issues/dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md
// should be reduced rather than expanded.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/yoskeoka/ai-arena/games/dungeon"
	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var gameVersion string
	var ruleset string

	fs := flag.NewFlagSet("dungeon-gamemaster", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&gameVersion, "game-version", "", "game version")
	fs.StringVar(&ruleset, "ruleset", "", "ruleset")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	meta, _, err := dungeon.MetadataForSelection(gameVersion, ruleset)
	if err != nil {
		return err
	}
	server := &serverState{
		meta: catalog.GameMetadata{
			GameID:         meta.GameID,
			GameVersion:    meta.GameVersion,
			RulesetVersion: meta.RulesetVersion,
		},
		lastAction: make(map[string]game.ActionStatus),
	}

	dec := protocol.NewDecoder(os.Stdin)
	enc := protocol.NewEncoder(os.Stdout)
	for {
		req, err := dec.DecodeRequest()
		if err != nil {
			return err
		}
		resp, exit, err := server.handleRequest(context.Background(), req)
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
	meta       catalog.GameMetadata
	world      *dungeon.Match
	players    []game.Player
	lastAction map[string]game.ActionStatus
}

func (s *serverState) handleRequest(ctx context.Context, req protocol.Request) (protocol.Response, bool, error) {
	switch req.Method {
	case "metadata":
		resp, err := protocol.NewResponse(req.ID, s.meta)
		return resp, false, err
	case "initialize_match":
		var params gamemaster.InitializeMatchParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		playerIDs := make([]string, 0, len(params.Players))
		for _, player := range params.Players {
			playerIDs = append(playerIDs, player.PlayerID)
		}
		cfg := dungeon.Config{
			GameVersion: s.meta.GameVersion,
			Ruleset:     s.meta.RulesetVersion,
			PlayerIDs:   playerIDs,
			RNGSeed:     params.RNGSeed,
		}
		var world *dungeon.Match
		var err error
		if params.ResumeSnapshot == nil {
			world, err = dungeon.New(cfg)
		} else {
			fullState, decodeErr := snapshotToFullState(*params.ResumeSnapshot)
			if decodeErr != nil {
				return protocol.Response{}, false, decodeErr
			}
			world, err = dungeon.NewFromFullState(cfg, fullState)
		}
		if err != nil {
			return protocol.Response{}, false, err
		}
		s.world = world
		s.players = append([]game.Player(nil), params.Players...)
		s.lastAction = make(map[string]game.ActionStatus, len(params.Players))
		initState := make(map[string]json.RawMessage, len(params.Players))
		for _, player := range params.Players {
			visible, err := s.world.CurrentVisibleState(player.PlayerID)
			if err != nil {
				return protocol.Response{}, false, err
			}
			initState[player.PlayerID] = mustJSON(visible)
			s.lastAction[player.PlayerID] = game.ActionStatus{
				PlayerID:     player.PlayerID,
				ActionStatus: session.StatusNoAction,
			}
		}
		resp, err := protocol.NewResponse(req.ID, gamemaster.InitializeMatchResult{
			InitState: game.InitState{PerPlayer: initState},
		})
		return resp, false, err
	case "next_decision_step":
		if s.world == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		if s.world.Terminal() {
			resp, err := protocol.NewResponse(req.ID, (*game.DecisionStep)(nil))
			return resp, false, err
		}
		ruleset := s.world.Ruleset()
		requests := make([]game.DecisionRequest, 0)
		for _, playerID := range s.world.PendingPlayerIDs() {
			visible, err := s.world.CurrentVisibleState(playerID)
			if err != nil {
				return protocol.Response{}, false, err
			}
			requests = append(requests, game.DecisionRequest{
				PlayerID:        playerID,
				VisibleState:    mustJSON(visible),
				LegalActionHint: s.world.LegalActionHint(),
				Deadline:        ruleset.TurnDeadline,
			})
		}
		resp, err := protocol.NewResponse(req.ID, &game.DecisionStep{
			Turn:     s.world.Turn() + 1,
			Mode:     game.Simultaneous,
			Requests: requests,
		})
		return resp, false, err
	case "normalize_action":
		if s.world == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		var params struct {
			Request      game.DecisionRequest `json:"request"`
			ActionStatus game.ActionStatus    `json:"action_status"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		normalized := normalizeAction(s.world, params.Request.PlayerID, params.ActionStatus)
		resp, err := protocol.NewResponse(req.ID, normalized)
		return resp, false, err
	case "apply_decision_results":
		if s.world == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		var params gamemaster.ApplyDecisionResultsParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return protocol.Response{}, false, err
		}
		actions := make(map[string]dungeon.Action, len(params.ActionStatuses))
		for _, status := range params.ActionStatuses {
			s.lastAction[status.PlayerID] = status
			if status.ActionStatus == session.StatusAccepted {
				action, err := dungeon.ParseAction(status.Action)
				if err != nil {
					return protocol.Response{}, false, err
				}
				actions[status.PlayerID] = action
				continue
			}
			actions[status.PlayerID] = dungeon.Action{Action: "wait"}
		}
		if err := s.world.Apply(actions); err != nil {
			return protocol.Response{}, false, err
		}
		resp, err := protocol.NewResponse(req.ID, map[string]bool{"ok": true})
		return resp, false, err
	case "current_snapshot":
		if s.world == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		snapshot, err := s.currentSnapshot()
		if err != nil {
			return protocol.Response{}, false, err
		}
		resp, err := protocol.NewResponse(req.ID, snapshot)
		return resp, false, err
	case "current_exported_snapshot":
		if s.world == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		resp, err := protocol.NewResponse(req.ID, s.currentExportedSnapshot())
		return resp, false, err
	case "current_result":
		if s.world == nil {
			return protocol.Response{}, false, fmt.Errorf("match is not initialized")
		}
		resp, err := protocol.NewResponse(req.ID, s.currentResult())
		return resp, false, err
	case "shutdown":
		resp, err := protocol.NewResponse(req.ID, map[string]bool{"ok": true})
		return resp, true, err
	default:
		return protocol.Response{}, false, fmt.Errorf("unsupported method %q", req.Method)
	}
}

func (s *serverState) currentSnapshot() (game.Snapshot, error) {
	full := s.world.FullState()
	perPlayer := make(map[string]game.PlayerSnapshot, len(s.players))
	for _, player := range s.players {
		visible, err := s.world.CurrentVisibleState(player.PlayerID)
		if err != nil {
			return game.Snapshot{}, err
		}
		status := s.lastAction[player.PlayerID]
		if status.PlayerID == "" {
			status = game.ActionStatus{PlayerID: player.PlayerID, ActionStatus: session.StatusNoAction}
		}
		perPlayer[player.PlayerID] = game.PlayerSnapshot{
			VisibleState:     mustJSON(visible),
			LastActionStatus: status,
		}
	}
	return game.Snapshot{
		GameID:         s.meta.GameID,
		GameVersion:    s.meta.GameVersion,
		RulesetVersion: s.meta.RulesetVersion,
		Turn:           s.world.Turn(),
		Status:         statusForWorld(s.world),
		GameState:      mustJSON(full),
		PerPlayer:      perPlayer,
	}, nil
}

func (s *serverState) currentExportedSnapshot() game.ExportedSnapshot {
	public := s.world.PublicState()
	if statusForWorld(s.world) != game.StatusCompleted {
		public.RNGSeed = ""
	}
	players := make([]game.ExportedPlayerSnapshot, 0, len(s.players))
	for _, player := range s.players {
		status := s.lastAction[player.PlayerID]
		if status.PlayerID == "" {
			status = game.ActionStatus{PlayerID: player.PlayerID, ActionStatus: session.StatusNoAction}
		}
		players = append(players, game.ExportedPlayerSnapshot{
			PlayerID:         player.PlayerID,
			LastActionStatus: status,
		})
	}
	return game.ExportedSnapshot{
		GameID:         s.meta.GameID,
		GameVersion:    s.meta.GameVersion,
		RulesetVersion: s.meta.RulesetVersion,
		Turn:           s.world.Turn(),
		Status:         statusForWorld(s.world),
		PublicState:    mustJSON(public),
		Players:        players,
	}
}

func (s *serverState) currentResult() game.MatchResult {
	placements := s.world.Placements()
	result := game.MatchResult{Placements: make([]game.Placement, 0, len(placements))}
	for _, placement := range placements {
		result.Placements = append(result.Placements, game.Placement{
			PlayerID: placement.PlayerID,
			Place:    placement.Place,
		})
	}
	return result
}

func normalizeAction(world *dungeon.Match, playerID string, status game.ActionStatus) game.ActionStatus {
	if status.PlayerID == "" {
		status.PlayerID = playerID
	}
	if status.ActionStatus != session.StatusAccepted {
		status.Action = nil
		return status
	}
	action, err := dungeon.ParseAction(status.Action)
	if err != nil || !world.CanApply(playerID, action) {
		return game.ActionStatus{
			PlayerID:      playerID,
			ActionStatus:  session.StatusNoAction,
			FailureReason: contract.ReasonIllegalAction,
		}
	}
	return status
}

func statusForWorld(world *dungeon.Match) game.MatchStatus {
	if world.Terminal() {
		return game.StatusCompleted
	}
	return game.StatusRunning
}

func snapshotToFullState(snapshot game.Snapshot) (dungeon.FullState, error) {
	var state dungeon.FullState
	if err := json.Unmarshal(snapshot.GameState, &state); err != nil {
		return dungeon.FullState{}, fmt.Errorf("decode dungeon snapshot game_state: %w", err)
	}
	return state, nil
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
