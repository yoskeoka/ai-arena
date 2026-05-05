package janken

import "testing"

func TestGameIDMatchesSpec(t *testing.T) {
	if GameID != "janken" {
		t.Fatalf("GameID = %q, want janken", GameID)
	}
}

func TestSkeletonTracksFollowUpPlan(t *testing.T) {
	t.Skip("platform-phase2-05-janken-richer-integration will add the master implementation and richer verification")
}
