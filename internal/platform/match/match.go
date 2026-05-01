package match

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

type PlayerSession interface {
	Init(context.Context, session.Request) session.Result
	Turn(context.Context, session.Request) session.Result
	GameOver(context.Context, any) error
	StderrSnapshot() runtime.StderrSnapshot
}

type Record struct {
	MatchID          string                `json:"match_id"`
	Game             catalog.GameMetadata  `json:"game"`
	Players          []game.Player         `json:"players"`
	Status           string                `json:"status"`
	Result           game.MatchResult      `json:"result"`
	EventLog         []Event               `json:"event_log"`
	Snapshot         game.Snapshot         `json:"snapshot"`
	ExportedSnapshot game.ExportedSnapshot `json:"exported_snapshot"`
}

type Event struct {
	Seq      int             `json:"seq"`
	Kind     string          `json:"kind"`
	Turn     int             `json:"turn"`
	PlayerID string          `json:"player_id,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type Runner struct {
	matchID    string
	players    []game.Player
	master     game.Master
	sessions   map[string]PlayerSession
	events     []Event
	nextSeq    int
	lastSeen   map[string]json.RawMessage
	lastResult map[string]game.ActionOutcome
}

func NewRunner(matchID string, players []game.Player, master game.Master, sessions map[string]PlayerSession) *Runner {
	return &Runner{
		matchID:    matchID,
		players:    players,
		master:     master,
		sessions:   sessions,
		lastSeen:   make(map[string]json.RawMessage),
		lastResult: make(map[string]game.ActionOutcome),
	}
}

func (r *Runner) Run(ctx context.Context) (Record, error) {
	r.appendEvent("match_started", 0, "", map[string]any{"match_id": r.matchID})

	initState, err := r.master.Init(ctx)
	if err != nil {
		return Record{}, err
	}

	meta := r.master.Metadata()
	for _, player := range r.players {
		params := map[string]any{
			"match_id":        r.matchID,
			"player_id":       player.PlayerID,
			"game_id":         meta.GameID,
			"game_version":    meta.GameVersion,
			"ruleset_version": meta.RulesetVersion,
			"deadline_ms":     1000,
			"state":           json.RawMessage(initState.PerPlayer[player.PlayerID]),
		}
		result := r.sessions[player.PlayerID].Init(ctx, session.Request{
			ID:       "init",
			Method:   "init",
			Params:   params,
			Deadline: 1_000_000_000,
		})
		r.appendEvent("session_initialized", 0, player.PlayerID, result)
		if result.Outcome != session.OutcomeAccepted {
			return Record{}, fmt.Errorf("init failed for %s: %s", player.PlayerID, result.FailureReason)
		}
	}

	for {
		window, err := r.master.NextDecision(ctx)
		if err != nil {
			return Record{}, err
		}
		if window == nil {
			break
		}

		if window.Mode == game.Simultaneous {
			outcomes := r.runSimultaneous(ctx, *window)
			if err := r.master.ApplyDecision(ctx, *window, outcomes); err != nil {
				return Record{}, err
			}
			continue
		}

		for _, req := range window.Requests {
			outcome := r.runTurn(ctx, window.Turn, req)
			partial := game.DecisionWindow{
				Turn:     window.Turn,
				Mode:     game.Sequential,
				Requests: []game.DecisionRequest{req},
			}
			if err := r.master.ApplyDecision(ctx, partial, []game.ActionOutcome{outcome}); err != nil {
				return Record{}, err
			}
		}
	}

	for _, player := range r.players {
		if err := r.sessions[player.PlayerID].GameOver(ctx, map[string]any{"match_id": r.matchID}); err == nil {
			r.appendEvent("game_over_sent", 0, player.PlayerID, map[string]any{"match_id": r.matchID})
		}
	}

	snapshot := r.master.Snapshot()
	snapshot.MatchID = r.matchID
	snapshot.Status = "completed"
	if snapshot.PerPlayer == nil {
		snapshot.PerPlayer = make(map[string]game.PlayerSnapshot)
	}

	for _, player := range r.players {
		stderr := r.sessions[player.PlayerID].StderrSnapshot()
		snapshot.PerPlayer[player.PlayerID] = game.PlayerSnapshot{
			VisibleState: r.lastSeen[player.PlayerID],
			LastOutcome:  r.lastResult[player.PlayerID],
			StderrBytes:  stderr.BytesRead,
		}
	}

	exported := r.master.ExportedSnapshot()
	exported.MatchID = r.matchID
	exported.Status = "completed"
	if len(exported.Players) == 0 {
		for _, player := range r.players {
			exported.Players = append(exported.Players, game.ExportedPlayerSnapshot{
				PlayerID:    player.PlayerID,
				LastOutcome: r.lastResult[player.PlayerID],
			})
		}
		sort.Slice(exported.Players, func(i, j int) bool {
			return exported.Players[i].PlayerID < exported.Players[j].PlayerID
		})
	}

	record := Record{
		MatchID:          r.matchID,
		Game:             meta,
		Players:          r.players,
		Status:           "completed",
		Result:           r.master.Result(),
		EventLog:         r.events,
		Snapshot:         snapshot,
		ExportedSnapshot: exported,
	}
	r.appendEvent("match_completed", snapshot.Turn, "", record.Result)
	record.EventLog = r.events
	return record, nil
}

func (r *Runner) runSimultaneous(ctx context.Context, window game.DecisionWindow) []game.ActionOutcome {
	outcomes := make([]game.ActionOutcome, len(window.Requests))
	var wg sync.WaitGroup
	for i, req := range window.Requests {
		i := i
		req := req
		wg.Add(1)
		go func() {
			defer wg.Done()
			outcomes[i] = r.runTurn(ctx, window.Turn, req)
		}()
	}
	wg.Wait()
	return outcomes
}

func (r *Runner) runTurn(ctx context.Context, turn int, req game.DecisionRequest) game.ActionOutcome {
	r.lastSeen[req.PlayerID] = req.VisibleState
	r.appendEvent("turn_requested", turn, req.PlayerID, req)

	result := r.sessions[req.PlayerID].Turn(ctx, session.Request{
		ID:       fmt.Sprintf("turn-%d-%s", turn, req.PlayerID),
		Method:   "turn",
		Params:   map[string]any{"turn": turn, "visible_state": json.RawMessage(req.VisibleState), "legal_action_hint": json.RawMessage(req.LegalActionHint), "deadline_ms": req.Deadline.Milliseconds()},
		Deadline: req.Deadline,
	})

	outcome := game.ActionOutcome{
		PlayerID:      req.PlayerID,
		Outcome:       result.Outcome,
		FailureReason: result.FailureReason,
		Action:        result.Payload,
	}
	r.lastResult[req.PlayerID] = outcome

	switch result.FailureReason {
	case "":
		r.appendEvent("turn_result", turn, req.PlayerID, outcome)
	case session.ReasonTimeout:
		r.appendEvent("turn_timeout", turn, req.PlayerID, outcome)
	case session.ReasonLateResponse:
		r.appendEvent("late_response_ignored", turn, req.PlayerID, outcome)
	default:
		r.appendEvent("protocol_error", turn, req.PlayerID, outcome)
	}
	return outcome
}

func (r *Runner) appendEvent(kind string, turn int, playerID string, payload any) {
	r.nextSeq++
	raw, _ := json.Marshal(payload)
	r.events = append(r.events, Event{
		Seq:      r.nextSeq,
		Kind:     kind,
		Turn:     turn,
		PlayerID: playerID,
		Payload:  raw,
	})
}
