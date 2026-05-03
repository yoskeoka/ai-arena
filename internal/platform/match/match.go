package match

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/catalog"
	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
	"github.com/yoskeoka/ai-arena/internal/platform/session"
)

type PlayerSession interface {
	Init(context.Context, session.Request) session.Result
	Turn(context.Context, session.Request) session.Result
	GameOver(context.Context, any) error
	Close(context.Context) error
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
	matchID     string
	players     []game.Player
	master      game.Master
	sessions    map[string]PlayerSession
	events      []Event
	nextSeq     int
	lastSeen    map[string]json.RawMessage
	lastResult  map[string]game.ActionOutcome
	phase       game.MatchStatus
	status      game.MatchStatus
	terminalErr error
}

const (
	initDeadline     = time.Second
	shutdownDeadline = time.Second
)

type turnExecution struct {
	request game.DecisionRequest
	result  session.Result
	outcome game.ActionOutcome
}

func NewRunner(matchID string, players []game.Player, master game.Master, sessions map[string]PlayerSession) *Runner {
	return &Runner{
		matchID:    matchID,
		players:    players,
		master:     master,
		sessions:   sessions,
		lastSeen:   make(map[string]json.RawMessage),
		lastResult: make(map[string]game.ActionOutcome),
		phase:      game.StatusStarting,
		status:     game.StatusStarting,
	}
}

func (r *Runner) Run(ctx context.Context) (record Record, runErr error) {
	meta := r.master.Metadata()
	r.appendEvent("match_started", 0, "", map[string]any{"match_id": r.matchID})

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownDeadline)
		defer cancel()

		r.shutdownSessions(shutdownCtx)
		r.emitTerminalEvent()
		record = r.buildRecord(meta)
	}()

	if err := r.initializeSessions(ctx, meta); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			runErr = r.cancel(err)
			return record, runErr
		}
		runErr = r.fail(err)
		return record, runErr
	}

	if err := r.runDecisionLoop(ctx); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			runErr = r.cancel(err)
			return record, runErr
		}
		runErr = r.fail(err)
		return record, runErr
	}

	r.phase = game.StatusFinishing
	r.status = game.StatusCompleted
	return record, nil
}

func (r *Runner) initializeSessions(ctx context.Context, meta catalog.GameMetadata) error {
	r.phase = game.StatusInitializing

	initState, err := r.master.Init(ctx)
	if err != nil {
		return err
	}

	for _, player := range r.players {
		state := initState.PerPlayer[player.PlayerID]
		params := map[string]any{
			"match_id":        r.matchID,
			"player_id":       player.PlayerID,
			"game_id":         meta.GameID,
			"game_version":    meta.GameVersion,
			"ruleset_version": meta.RulesetVersion,
			"deadline_ms":     initDeadline.Milliseconds(),
			"state":           json.RawMessage(state),
		}
		result := r.sessions[player.PlayerID].Init(ctx, session.Request{
			ID:       "init",
			Method:   "init",
			Params:   params,
			Deadline: initDeadline,
		})
		r.appendEvent("session_initialized", 0, player.PlayerID, result)
		if result.FailureReason == session.ReasonRuntimeStop {
			r.appendRuntimeExited(0, player.PlayerID, map[string]any{"stage": "init"})
		}
		if result.Outcome != session.OutcomeAccepted {
			return fmt.Errorf("init failed for %s: %s", player.PlayerID, result.FailureReason)
		}
	}

	return nil
}

func (r *Runner) runDecisionLoop(ctx context.Context) error {
	r.phase = game.StatusRunning

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		step, err := r.master.NextStep(ctx)
		if err != nil {
			return err
		}
		if step == nil {
			return nil
		}

		switch step.Mode {
		case game.Simultaneous:
			outcomes := r.runSimultaneousStep(ctx, *step)
			if err := r.master.ApplyStep(ctx, *step, outcomes); err != nil {
				return err
			}
		case game.Sequential:
			if len(step.Requests) != 1 {
				return fmt.Errorf("sequential step must contain exactly one request, got %d", len(step.Requests))
			}
			req := step.Requests[0]
			r.prepareTurn(step.Turn, req)
			exec := r.executeTurn(ctx, step.Turn, req)
			outcome := r.recordTurn(step.Turn, exec)
			if err := r.master.ApplyStep(ctx, *step, []game.ActionOutcome{outcome}); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported decision mode %q", step.Mode)
		}
	}
}

func (r *Runner) runSimultaneousStep(ctx context.Context, step game.DecisionStep) []game.ActionOutcome {
	executions := make([]turnExecution, len(step.Requests))
	for i, req := range step.Requests {
		r.prepareTurn(step.Turn, req)
		executions[i].request = req
	}

	var wg sync.WaitGroup
	for i, req := range step.Requests {
		i := i
		req := req
		wg.Add(1)
		go func() {
			defer wg.Done()
			executions[i] = r.executeTurn(ctx, step.Turn, req)
		}()
	}
	wg.Wait()

	outcomes := make([]game.ActionOutcome, len(executions))
	for i, exec := range executions {
		outcomes[i] = r.recordTurn(step.Turn, exec)
	}
	return outcomes
}

func (r *Runner) shutdownSessions(ctx context.Context) {
	for _, player := range r.players {
		playerID := player.PlayerID

		if r.status == game.StatusCompleted {
			if err := r.sessions[playerID].GameOver(ctx, map[string]any{"match_id": r.matchID}); err != nil {
				r.appendEvent("session_shutdown_failed", 0, playerID, map[string]any{
					"stage": "game_over",
					"error": err.Error(),
				})
			} else {
				r.appendEvent("game_over_sent", 0, playerID, map[string]any{"match_id": r.matchID})
			}
		}

		r.appendEvent("session_shutdown_started", 0, playerID, map[string]any{"phase": r.phase})
		if err := r.sessions[playerID].Close(ctx); err != nil {
			r.appendRuntimeExited(0, playerID, map[string]any{"stage": "shutdown", "error": err.Error()})
			r.appendEvent("session_shutdown_failed", 0, playerID, map[string]any{
				"stage": "close",
				"error": err.Error(),
			})
			continue
		}
		r.appendEvent("session_shutdown_completed", 0, playerID, map[string]any{"phase": r.phase})
	}
}

func (r *Runner) buildRecord(meta catalog.GameMetadata) Record {
	snapshot := r.master.Snapshot()
	snapshot.MatchID = r.matchID
	snapshot.Status = string(r.status)
	if snapshot.PerPlayer == nil {
		snapshot.PerPlayer = make(map[string]game.PlayerSnapshot)
	}

	for _, player := range r.players {
		playerID := player.PlayerID
		stderr := r.sessions[playerID].StderrSnapshot()
		snapshot.PerPlayer[playerID] = game.PlayerSnapshot{
			VisibleState: r.lastSeen[playerID],
			LastOutcome:  r.lastResult[playerID],
			StderrBytes:  stderr.BytesRead,
		}
	}

	exported := r.master.ExportedSnapshot()
	exported.MatchID = r.matchID
	exported.Status = string(r.status)
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

	return Record{
		MatchID:          r.matchID,
		Game:             meta,
		Players:          r.players,
		Status:           string(r.status),
		Result:           r.master.Result(),
		EventLog:         append([]Event(nil), r.events...),
		Snapshot:         snapshot,
		ExportedSnapshot: exported,
	}
}

func (r *Runner) fail(err error) error {
	r.phase = game.StatusFailed
	r.status = game.StatusFailed
	r.terminalErr = err
	return err
}

func (r *Runner) cancel(err error) error {
	r.phase = game.StatusCanceled
	r.status = game.StatusCanceled
	r.terminalErr = err
	return err
}

func (r *Runner) emitTerminalEvent() {
	turn := r.master.Snapshot().Turn

	switch r.status {
	case game.StatusCompleted:
		r.phase = game.StatusCompleted
		r.appendEvent("match_completed", turn, "", r.master.Result())
	case game.StatusCanceled:
		r.appendEvent("match_canceled", turn, "", map[string]any{"error": errString(r.terminalErr)})
	case game.StatusFailed:
		r.appendEvent("match_failed", turn, "", map[string]any{"error": errString(r.terminalErr)})
	default:
		r.phase = game.StatusFailed
		r.status = game.StatusFailed
		r.appendEvent("match_failed", turn, "", map[string]any{"error": "match terminated without explicit status"})
	}
}

func (r *Runner) prepareTurn(turn int, req game.DecisionRequest) {
	r.lastSeen[req.PlayerID] = req.VisibleState
	r.appendEvent("turn_requested", turn, req.PlayerID, req)
}

func (r *Runner) executeTurn(ctx context.Context, turn int, req game.DecisionRequest) turnExecution {
	result := r.sessions[req.PlayerID].Turn(ctx, session.Request{
		ID:     fmt.Sprintf("turn-%d-%s", turn, req.PlayerID),
		Method: "turn",
		Params: map[string]any{
			"turn":              turn,
			"visible_state":     json.RawMessage(req.VisibleState),
			"legal_action_hint": json.RawMessage(req.LegalActionHint),
			"deadline_ms":       req.Deadline.Milliseconds(),
		},
		Deadline: req.Deadline,
	})

	return turnExecution{
		request: req,
		result:  result,
		outcome: r.master.NormalizeAction(req, game.ActionOutcome{
			PlayerID:      req.PlayerID,
			Outcome:       result.Outcome,
			FailureReason: result.FailureReason,
			Action:        result.Payload,
		}),
	}
}

func (r *Runner) recordTurn(turn int, exec turnExecution) game.ActionOutcome {
	r.lastResult[exec.request.PlayerID] = exec.outcome
	for _, lateID := range exec.result.IgnoredLateResponseIDs {
		r.appendEvent("late_response_ignored", turn, exec.request.PlayerID, map[string]any{"response_id": lateID})
	}

	switch exec.result.FailureReason {
	case "":
		r.appendEvent("turn_result", turn, exec.request.PlayerID, exec.outcome)
	case session.ReasonTimeout:
		r.appendEvent("turn_timeout", turn, exec.request.PlayerID, exec.outcome)
	case session.ReasonRuntimeStop:
		r.appendRuntimeExited(turn, exec.request.PlayerID, exec.outcome)
	default:
		r.appendEvent("protocol_error", turn, exec.request.PlayerID, exec.outcome)
	}
	return exec.outcome
}

func (r *Runner) appendRuntimeExited(turn int, playerID string, payload any) {
	r.appendEvent("runtime_exited", turn, playerID, payload)
}

func (r *Runner) appendEvent(kind string, turn int, playerID string, payload any) {
	r.nextSeq++
	raw := mustMarshalPayload(payload)
	r.events = append(r.events, Event{
		Seq:      r.nextSeq,
		Kind:     kind,
		Turn:     turn,
		PlayerID: playerID,
		Payload:  raw,
	})
}

func mustMarshalPayload(payload any) json.RawMessage {
	raw, err := json.Marshal(payload)
	if err == nil {
		return raw
	}

	fallback, fallbackErr := json.Marshal(map[string]any{
		"marshal_error": err.Error(),
		"payload_type":  fmt.Sprintf("%T", payload),
	})
	if fallbackErr == nil {
		return fallback
	}

	return json.RawMessage(`{"marshal_error":"failed to encode event payload"}`)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
