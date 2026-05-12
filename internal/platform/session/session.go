package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

const (
	// StatusAccepted aliases the accepted action decision.
	StatusAccepted = contract.ActionAccepted
	// StatusNoAction aliases the no-action decision.
	StatusNoAction = contract.ActionNoAction

	// ReasonTimeout aliases the timeout failure reason.
	ReasonTimeout = contract.ReasonTimeout
	// ReasonMalformed aliases the malformed protocol failure reason.
	ReasonMalformed = contract.ReasonMalformed
	// ReasonMismatchedID aliases the mismatched-id failure reason.
	ReasonMismatchedID = contract.ReasonMismatchedID
	// ReasonLateResponse aliases the late-response failure reason.
	ReasonLateResponse = contract.ReasonLateResponse
	// ReasonRuntimeStop aliases the runtime-stopped failure reason.
	ReasonRuntimeStop = contract.ReasonRuntimeStop
)

// Transport abstracts the runtime transport used by a player session.
type Transport interface {
	Send(protocol.Request) error
	Incoming() <-chan runtime.Message
	Close(context.Context) error
	StderrSnapshot() runtime.StderrSnapshot
}

// Request describes one player-facing protocol call.
type Request struct {
	ID       string
	Method   string
	Params   any
	Deadline time.Duration
}

// Result is the normalized outcome of a player-facing protocol call.
type Result struct {
	Status                 contract.ActionDecision
	FailureReason          contract.FailureReason
	Payload                json.RawMessage
	IgnoredLateResponseIDs []string
}

// Session manages one player's request/response lifecycle over a Transport.
type Session struct {
	transport   Transport
	lateIDs     map[string]struct{}
	onLate      func(string)
	onMalformed func(error)
}

// New constructs a session over the provided transport.
func New(transport Transport) *Session {
	return &Session{
		transport: transport,
		lateIDs:   make(map[string]struct{}),
	}
}

// SetLateResponseHook registers a callback for ignored late responses.
func (s *Session) SetLateResponseHook(fn func(string)) {
	s.onLate = fn
}

// SetMalformedHook registers a callback for malformed transport messages.
func (s *Session) SetMalformedHook(fn func(error)) {
	s.onMalformed = fn
}

// Init performs the initialization request flow.
func (s *Session) Init(ctx context.Context, req Request) Result {
	return s.call(ctx, req)
}

// Turn performs one turn request flow.
func (s *Session) Turn(ctx context.Context, req Request) Result {
	return s.call(ctx, req)
}

// GameOver performs the final game-over request flow.
func (s *Session) GameOver(ctx context.Context, req Request) Result {
	return s.call(ctx, req)
}

// Close closes the underlying transport.
func (s *Session) Close(ctx context.Context) error {
	return s.transport.Close(ctx)
}

// StderrSnapshot returns the latest captured stderr snapshot.
func (s *Session) StderrSnapshot() runtime.StderrSnapshot {
	return s.transport.StderrSnapshot()
}

func (s *Session) call(ctx context.Context, req Request) Result {
	var ignoredLateResponseIDs []string

	msg, err := protocol.NewRequest(req.ID, req.Method, req.Params)
	if err != nil {
		return Result{Status: StatusNoAction, FailureReason: ReasonMalformed}
	}
	if err := s.transport.Send(msg); err != nil {
		return Result{Status: StatusNoAction, FailureReason: ReasonRuntimeStop}
	}

	timer := time.NewTimer(req.Deadline)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			s.lateIDs[req.ID] = struct{}{}
			return Result{Status: StatusNoAction, FailureReason: ReasonTimeout}
		case <-timer.C:
			s.lateIDs[req.ID] = struct{}{}
			return Result{Status: StatusNoAction, FailureReason: ReasonTimeout}
		case incoming, ok := <-s.transport.Incoming():
			if !ok {
				return Result{Status: StatusNoAction, FailureReason: ReasonRuntimeStop}
			}
			if incoming.Err != nil {
				if s.onMalformed != nil {
					s.onMalformed(incoming.Err)
				}
				return Result{Status: StatusNoAction, FailureReason: ReasonMalformed}
			}
			if incoming.Response == nil {
				continue
			}
			if _, timedOut := s.lateIDs[incoming.Response.ID]; timedOut {
				delete(s.lateIDs, incoming.Response.ID)
				ignoredLateResponseIDs = append(ignoredLateResponseIDs, incoming.Response.ID)
				if s.onLate != nil {
					s.onLate(incoming.Response.ID)
				}
				continue
			}
			if err := protocol.MatchResponseID(req.ID, *incoming.Response); err != nil {
				if errors.Is(err, protocol.ErrMismatchedID) {
					return Result{Status: StatusNoAction, FailureReason: ReasonMismatchedID}
				}
				return Result{Status: StatusNoAction, FailureReason: ReasonMalformed}
			}
			if incoming.Response.Error != nil {
				return Result{
					Status:                 StatusNoAction,
					FailureReason:          ReasonMalformed,
					IgnoredLateResponseIDs: ignoredLateResponseIDs,
				}
			}
			return Result{
				Status:                 StatusAccepted,
				Payload:                incoming.Response.Result,
				IgnoredLateResponseIDs: ignoredLateResponseIDs,
			}
		}
	}
}
