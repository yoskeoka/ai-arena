package service

import (
	"encoding/json"
	"testing"

	"github.com/yoskeoka/ai-arena/internal/platform/contract"
)

func TestValidateSubmission(t *testing.T) {
	valid := MatchSubmission{
		SubmissionID: "sub-1",
		MatchID:      "match-1",
		Game: contract.GameMetadata{
			GameID:         "janken",
			GameVersion:    "2.1.0",
			RulesetVersion: "regular",
		},
		Players: []SubmittedPlayer{
			{
				PlayerID:    "p1",
				ArtifactRef: "file:///tmp/p1.wasm",
			},
			{
				PlayerID:    "p2",
				ArtifactRef: "s3://bucket/p2.wasm",
			},
		},
		OutputDir:    "arena-service-output",
		AttemptCount: 1,
	}
	clone := func() MatchSubmission {
		sub := valid
		sub.Players = append([]SubmittedPlayer(nil), valid.Players...)
		return sub
	}

	tests := []struct {
		name       string
		submission MatchSubmission
		wantOK     bool
	}{
		{name: "valid", submission: valid, wantOK: true},
		{
			name: "requires attempt count one",
			submission: func() MatchSubmission {
				sub := clone()
				sub.AttemptCount = 2
				return sub
			}(),
		},
		{
			name: "requires output dir",
			submission: func() MatchSubmission {
				sub := clone()
				sub.OutputDir = " "
				return sub
			}(),
		},
		{
			name: "rejects duplicate player ids",
			submission: func() MatchSubmission {
				sub := clone()
				sub.Players[1].PlayerID = "p1"
				return sub
			}(),
		},
		{
			name: "requires artifact ref",
			submission: func() MatchSubmission {
				sub := clone()
				sub.Players[1].ArtifactRef = ""
				return sub
			}(),
		},
		{
			name: "requires game metadata",
			submission: func() MatchSubmission {
				sub := clone()
				sub.Game.GameVersion = ""
				return sub
			}(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSubmission(tc.submission)
			if tc.wantOK && err != nil {
				t.Fatalf("ValidateSubmission() error = %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("ValidateSubmission() returned nil error")
			}
		})
	}
}

func TestQueueRecordJSONOmitEmpty(t *testing.T) {
	record := QueueRecord{
		Submission: MatchSubmission{
			SubmissionID: "sub-1",
			MatchID:      "match-1",
			Game: contract.GameMetadata{
				GameID:         "janken",
				GameVersion:    "2.1.0",
				RulesetVersion: "regular",
			},
			Players: []SubmittedPlayer{
				{
					PlayerID:    "p1",
					ArtifactRef: "file:///tmp/p1.wasm",
				},
			},
			OutputDir:    "arena-service-output",
			AttemptCount: 1,
		},
		State: StateQueued,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(data) == "" {
		t.Fatal("json.Marshal() returned empty payload")
	}
	if contains := string(data); contains != "" {
		if jsonContainsField(t, data, "lease") {
			t.Fatalf("json = %s, unexpected lease field", data)
		}
		if jsonContainsField(t, data, "terminal") {
			t.Fatalf("json = %s, unexpected terminal field", data)
		}
	}
}

func TestSubmittedPlayerJSONShape(t *testing.T) {
	player := SubmittedPlayer{
		PlayerID:    "p1",
		ArtifactRef: "file:///tmp/p1.wasm",
	}

	data, err := json.Marshal(player)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(data) != `{"player_id":"p1","artifact_ref":"file:///tmp/p1.wasm"}` {
		t.Fatalf("json = %s", data)
	}
}

func jsonContainsField(t *testing.T, data []byte, field string) bool {
	t.Helper()

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	_, ok := decoded[field]
	return ok
}

func TestValidateTransition(t *testing.T) {
	tests := []struct {
		name   string
		from   LifecycleState
		to     LifecycleState
		wantOK bool
	}{
		{name: "queued to leased", from: StateQueued, to: StateLeased, wantOK: true},
		{name: "queued to canceled", from: StateQueued, to: StateCanceled, wantOK: true},
		{name: "leased to running", from: StateLeased, to: StateRunning, wantOK: true},
		{name: "running to persisting", from: StateRunning, to: StatePersisting, wantOK: true},
		{name: "persisting to completed", from: StatePersisting, to: StateCompleted, wantOK: true},
		{name: "persisting to failed", from: StatePersisting, to: StateFailed, wantOK: true},
		{name: "same state is idempotent", from: StateQueued, to: StateQueued, wantOK: true},
		{name: "queued cannot complete", from: StateQueued, to: StateCompleted},
		{name: "leased cannot cancel", from: StateLeased, to: StateCanceled},
		{name: "completed is terminal", from: StateCompleted, to: StateFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTransition(tc.from, tc.to)
			if tc.wantOK && err != nil {
				t.Fatalf("ValidateTransition() error = %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("ValidateTransition() returned nil error")
			}
		})
	}
}
