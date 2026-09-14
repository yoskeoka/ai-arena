package registry

import (
	"context"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/game"
	"github.com/yoskeoka/ai-arena/internal/platform/gamemaster"
)

func TestCleanupSessionForwardsCurrentPublicReplayAndCleansUpOnce(t *testing.T) {
	session := &publicReplaySession{replay: game.PublicReplay{
		Format:  "reversi/replay",
		Version: "1",
		Payload: []byte(`{"moves":["d3","c3"]}`),
	}}
	cleanupCalls := 0
	wrapped := &cleanupSession{
		Session: session,
		cleanup: func() {
			cleanupCalls++
		},
	}

	provider, ok := any(wrapped).(gamemaster.PublicReplaySession)
	if !ok {
		t.Fatal("cleanup session does not expose public replay capability")
	}
	replay, err := provider.CurrentPublicReplay(context.Background())
	if err != nil {
		t.Fatalf("CurrentPublicReplay: %v", err)
	}
	if replay.Format != session.replay.Format || replay.Version != session.replay.Version || string(replay.Payload) != string(session.replay.Payload) {
		t.Fatalf("CurrentPublicReplay = %+v, want %+v", replay, session.replay)
	}

	if err := wrapped.Shutdown(context.Background()); err != nil {
		t.Fatalf("first Shutdown: %v", err)
	}
	if err := wrapped.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
	if session.shutdownCalls != 2 {
		t.Fatalf("Shutdown calls = %d, want 2", session.shutdownCalls)
	}
	if cleanupCalls != 1 {
		t.Fatalf("cleanup calls = %d, want 1", cleanupCalls)
	}
}

func TestCleanupSessionReportsUnavailableReplayForNonProvider(t *testing.T) {
	wrapped := &cleanupSession{Session: nonReplaySession{}}

	if _, err := wrapped.CurrentPublicReplay(context.Background()); err == nil {
		t.Fatal("CurrentPublicReplay error = nil, want unavailable replay error")
	}
}

type publicReplaySession struct {
	gamemaster.Session
	replay        game.PublicReplay
	shutdownCalls int
}

type nonReplaySession struct{ gamemaster.Session }

func (s *publicReplaySession) CurrentPublicReplay(context.Context) (game.PublicReplay, error) {
	return s.replay, nil
}

func (s *publicReplaySession) Shutdown(context.Context) error {
	s.shutdownCalls++
	return nil
}

var _ gamemaster.Session = (*publicReplaySession)(nil)
var _ gamemaster.PublicReplaySession = (*publicReplaySession)(nil)
var _ gamemaster.Session = nonReplaySession{}
