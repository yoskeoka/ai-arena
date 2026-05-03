package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/yoskeoka/ai-arena/internal/platform/protocol"
	"github.com/yoskeoka/ai-arena/internal/platform/runtime"
)

const (
	StatusAccepted = "accepted"
	StatusNoAction = "no_action"

	ReasonTimeout      = "invalid-timeout"
	ReasonMalformed    = "invalid-protocol-malformed"
	ReasonMismatchedID = "invalid-protocol-mismatched-id"
	ReasonLateResponse = "invalid-protocol-late-response"
	ReasonRuntimeStop  = "runtime-stopped"
)

type Transport interface {
	Send(protocol.Request) error
	Incoming() <-chan runtime.Message
	Close(context.Context) error
	StderrSnapshot() runtime.StderrSnapshot
}

type Request struct {
	ID       string
	Method   string
	Params   any
	Deadline time.Duration
}

type Result struct {
	Status                 string
	FailureReason          string
	Payload                json.RawMessage
	IgnoredLateResponseIDs []string
}

type Session struct {
	transport   Transport
	lateIDs     map[string]struct{}
	onLate      func(string)
	onMalformed func(error)
}

func New(transport Transport) *Session {
	return &Session{
		transport: transport,
		lateIDs:   make(map[string]struct{}),
	}
}

func (s *Session) SetLateResponseHook(fn func(string)) {
	s.onLate = fn
}

func (s *Session) SetMalformedHook(fn func(error)) {
	s.onMalformed = fn
}

func (s *Session) Init(ctx context.Context, req Request) Result {
	return s.call(ctx, req)
}

func (s *Session) Turn(ctx context.Context, req Request) Result {
	return s.call(ctx, req)
}

func (s *Session) GameOver(ctx context.Context, req Request) Result {
	return s.call(ctx, req)
}

func (s *Session) Close(ctx context.Context) error {
	return s.transport.Close(ctx)
}

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
