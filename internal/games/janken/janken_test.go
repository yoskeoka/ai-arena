package janken

import (
	"os"
	"testing"
)

func TestGameIDMatchesSpec(t *testing.T) {
	if GameID != "janken" {
		t.Fatalf("GameID = %q, want janken", GameID)
	}
}

func TestSkeletonTracksFollowUpPlan(t *testing.T) {
	if _, err := os.Stat("../../../docs/exec-plan/todo/platform-phase2-05-janken-richer-integration.md"); err != nil {
		t.Fatalf("follow-up plan missing: %v", err)
	}
}
