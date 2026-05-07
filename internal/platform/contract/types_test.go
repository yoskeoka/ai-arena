package contract

import (
	"encoding/json"
	"testing"
)

func TestValidateActionStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ActionStatus
		wantOK bool
	}{
		{
			name: "accepted without failure reason",
			status: ActionStatus{
				PlayerID:     "p1",
				ActionStatus: ActionAccepted,
				Action:       json.RawMessage(`{"action":"rock"}`),
			},
			wantOK: true,
		},
		{
			name: "accepted rejects failure reason",
			status: ActionStatus{
				PlayerID:      "p1",
				ActionStatus:  ActionAccepted,
				FailureReason: ReasonTimeout,
				Action:        json.RawMessage(`{"action":"rock"}`),
			},
		},
		{
			name: "accepted requires action payload",
			status: ActionStatus{
				PlayerID:     "p1",
				ActionStatus: ActionAccepted,
			},
		},
		{
			name: "no_action accepts failure reason",
			status: ActionStatus{
				PlayerID:      "p2",
				ActionStatus:  ActionNoAction,
				FailureReason: ReasonTimeout,
			},
			wantOK: true,
		},
		{
			name: "no_action rejects action payload",
			status: ActionStatus{
				PlayerID:     "p2",
				ActionStatus: ActionNoAction,
				Action:       json.RawMessage(`{"action":"rock"}`),
			},
		},
		{
			name: "player_id is required",
			status: ActionStatus{
				ActionStatus: ActionNoAction,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateActionStatus(tc.status)
			if tc.wantOK && err != nil {
				t.Fatalf("ValidateActionStatus() error = %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("ValidateActionStatus() returned nil error")
			}
		})
	}
}
