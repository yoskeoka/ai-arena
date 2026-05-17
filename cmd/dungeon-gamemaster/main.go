// Command dungeon-gamemaster runs the dungeon game master sidecar.
//
// The dungeon-specific parts of this command are intended to stay movable to a
// separate repository. Avoid introducing new ai-arena internal dependencies on
// the dungeon side of the boundary; the remaining platform coupling tracked by
// docs/issues/0013-dungeon-sidecars-should-not-depend-on-internal-platform-protocol.md
// should be reduced rather than expanded.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	publicgm "github.com/yoskeoka/ai-arena/gamemaster"
	"github.com/yoskeoka/ai-arena/games/dungeon"
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
		meta: publicgm.GameMetadata{
			GameID:         meta.GameID,
			GameVersion:    meta.GameVersion,
			RulesetVersion: meta.RulesetVersion,
		},
		lastAction: make(map[string]publicgm.ActionStatus),
	}

	dec := publicgm.NewDecoder(os.Stdin)
	enc := publicgm.NewEncoder(os.Stdout)
	for {
		req, err := dec.DecodeRequest()
		if err != nil {
			return err
		}
		resp, exit, err := server.handleRequest(context.Background(), req)
		if err != nil {
			resp = publicgm.Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &publicgm.ErrorObject{
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
	meta       publicgm.GameMetadata
	world      *dungeon.Match
	players    []publicgm.Player
	lastAction map[string]publicgm.ActionStatus
}

func (s *serverState) handleRequest(ctx context.Context, req publicgm.Request) (publicgm.Response, bool, error) {
	switch req.Method {
	case publicgm.MethodMetadata:
		resp, err := publicgm.NewResponse(req.ID, s.meta)
		return resp, false, err
	case publicgm.MethodInitializeMatch:
		var params publicgm.InitializeMatchParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return publicgm.Response{}, false, err
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
				return publicgm.Response{}, false, decodeErr
			}
			world, err = dungeon.NewFromFullState(cfg, fullState)
		}
		if err != nil {
			return publicgm.Response{}, false, err
		}
		s.world = world
		s.players = append([]publicgm.Player(nil), params.Players...)
		s.lastAction = make(map[string]publicgm.ActionStatus, len(params.Players))
		initState := make(map[string]json.RawMessage, len(params.Players))
		for _, player := range params.Players {
			visible, err := s.world.CurrentVisibleState(player.PlayerID)
			if err != nil {
				return publicgm.Response{}, false, err
			}
			initState[player.PlayerID] = mustJSON(visible)
			s.lastAction[player.PlayerID] = publicgm.ActionStatus{
				PlayerID:     player.PlayerID,
				ActionStatus: publicgm.ActionNoAction,
			}
		}
		resp, err := publicgm.NewResponse(req.ID, publicgm.InitializeMatchResult{
			InitState: publicgm.InitState{PerPlayer: initState},
		})
		return resp, false, err
	case publicgm.MethodNextDecisionStep:
		if s.world == nil {
			return publicgm.Response{}, false, fmt.Errorf("match is not initialized")
		}
		if s.world.Terminal() {
			resp, err := publicgm.NewResponse(req.ID, (*publicgm.DecisionStep)(nil))
			return resp, false, err
		}
		ruleset := s.world.Ruleset()
		requests := make([]publicgm.DecisionRequest, 0)
		for _, playerID := range s.world.PendingPlayerIDs() {
			visible, err := s.world.CurrentVisibleState(playerID)
			if err != nil {
				return publicgm.Response{}, false, err
			}
			requests = append(requests, publicgm.DecisionRequest{
				PlayerID:        playerID,
				VisibleState:    mustJSON(visible),
				LegalActionHint: s.world.LegalActionHint(),
				Deadline:        ruleset.TurnDeadline,
			})
		}
		resp, err := publicgm.NewResponse(req.ID, &publicgm.DecisionStep{
			Turn:     s.world.Turn() + 1,
			Mode:     publicgm.Simultaneous,
			Requests: requests,
		})
		return resp, false, err
	case publicgm.MethodNormalizeAction:
		if s.world == nil {
			return publicgm.Response{}, false, fmt.Errorf("match is not initialized")
		}
		var params publicgm.NormalizeActionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return publicgm.Response{}, false, err
		}
		normalized := normalizeAction(s.world, params.Request.PlayerID, params.ActionStatus)
		resp, err := publicgm.NewResponse(req.ID, normalized)
		return resp, false, err
	case publicgm.MethodApplyDecisionResults:
		if s.world == nil {
			return publicgm.Response{}, false, fmt.Errorf("match is not initialized")
		}
		var params publicgm.ApplyDecisionResultsParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return publicgm.Response{}, false, err
		}
		actions := make(map[string]dungeon.Action, len(params.ActionStatuses))
		for _, status := range params.ActionStatuses {
			s.lastAction[status.PlayerID] = status
			if status.ActionStatus == publicgm.ActionAccepted {
				action, err := dungeon.ParseAction(status.Action)
				if err != nil {
					return publicgm.Response{}, false, err
				}
				actions[status.PlayerID] = action
				continue
			}
			actions[status.PlayerID] = dungeon.Action{Action: "wait"}
		}
		if err := s.world.Apply(actions); err != nil {
			return publicgm.Response{}, false, err
		}
		resp, err := publicgm.NewResponse(req.ID, map[string]bool{"ok": true})
		return resp, false, err
	case publicgm.MethodCurrentSnapshot:
		if s.world == nil {
			return publicgm.Response{}, false, fmt.Errorf("match is not initialized")
		}
		snapshot, err := s.currentSnapshot()
		if err != nil {
			return publicgm.Response{}, false, err
		}
		resp, err := publicgm.NewResponse(req.ID, snapshot)
		return resp, false, err
	case publicgm.MethodCurrentExportedSnapshot:
		if s.world == nil {
			return publicgm.Response{}, false, fmt.Errorf("match is not initialized")
		}
		resp, err := publicgm.NewResponse(req.ID, s.currentExportedSnapshot())
		return resp, false, err
	case publicgm.MethodCurrentResult:
		if s.world == nil {
			return publicgm.Response{}, false, fmt.Errorf("match is not initialized")
		}
		resp, err := publicgm.NewResponse(req.ID, s.currentResult())
		return resp, false, err
	case publicgm.MethodShutdown:
		resp, err := publicgm.NewResponse(req.ID, map[string]bool{"ok": true})
		return resp, true, err
	default:
		return publicgm.Response{}, false, fmt.Errorf("unsupported method %q", req.Method)
	}
}

func (s *serverState) currentSnapshot() (publicgm.Snapshot, error) {
	full := s.world.FullState()
	perPlayer := make(map[string]publicgm.PlayerSnapshot, len(s.players))
	for _, player := range s.players {
		visible, err := s.world.CurrentVisibleState(player.PlayerID)
		if err != nil {
			return publicgm.Snapshot{}, err
		}
		status := s.lastAction[player.PlayerID]
		if status.PlayerID == "" {
			status = publicgm.ActionStatus{PlayerID: player.PlayerID, ActionStatus: publicgm.ActionNoAction}
		}
		perPlayer[player.PlayerID] = publicgm.PlayerSnapshot{
			VisibleState:     mustJSON(visible),
			LastActionStatus: status,
		}
	}
	return publicgm.Snapshot{
		GameID:         s.meta.GameID,
		GameVersion:    s.meta.GameVersion,
		RulesetVersion: s.meta.RulesetVersion,
		Turn:           s.world.Turn(),
		Status:         statusForWorld(s.world),
		GameState:      mustJSON(full),
		PerPlayer:      perPlayer,
	}, nil
}

func (s *serverState) currentExportedSnapshot() publicgm.ExportedSnapshot {
	public := s.world.PublicState()
	if statusForWorld(s.world) != publicgm.StatusCompleted {
		public.RNGSeed = ""
	}
	players := make([]publicgm.ExportedPlayerSnapshot, 0, len(s.players))
	for _, player := range s.players {
		status := s.lastAction[player.PlayerID]
		if status.PlayerID == "" {
			status = publicgm.ActionStatus{PlayerID: player.PlayerID, ActionStatus: publicgm.ActionNoAction}
		}
		players = append(players, publicgm.ExportedPlayerSnapshot{
			PlayerID:         player.PlayerID,
			LastActionStatus: status,
		})
	}
	return publicgm.ExportedSnapshot{
		GameID:         s.meta.GameID,
		GameVersion:    s.meta.GameVersion,
		RulesetVersion: s.meta.RulesetVersion,
		Turn:           s.world.Turn(),
		Status:         statusForWorld(s.world),
		PublicState:    mustJSON(public),
		Players:        players,
	}
}

func (s *serverState) currentResult() publicgm.MatchResult {
	placements := s.world.Placements()
	result := publicgm.MatchResult{Placements: make([]publicgm.Placement, 0, len(placements))}
	for _, placement := range placements {
		result.Placements = append(result.Placements, publicgm.Placement{
			PlayerID: placement.PlayerID,
			Place:    placement.Place,
		})
	}
	return result
}

func normalizeAction(world *dungeon.Match, playerID string, status publicgm.ActionStatus) publicgm.ActionStatus {
	if status.PlayerID == "" {
		status.PlayerID = playerID
	}
	if status.ActionStatus != publicgm.ActionAccepted {
		status.Action = nil
		return status
	}
	action, err := dungeon.ParseAction(status.Action)
	if err != nil || !world.CanApply(playerID, action) {
		return publicgm.ActionStatus{
			PlayerID:      playerID,
			ActionStatus:  publicgm.ActionNoAction,
			FailureReason: publicgm.ReasonIllegalAction,
		}
	}
	return status
}

func statusForWorld(world *dungeon.Match) publicgm.MatchStatus {
	if world.Terminal() {
		return publicgm.StatusCompleted
	}
	return publicgm.StatusRunning
}

func snapshotToFullState(snapshot publicgm.Snapshot) (dungeon.FullState, error) {
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
