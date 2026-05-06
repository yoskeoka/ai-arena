package contract

import "testing"

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
			},
			wantOK: true,
		},
		{
			name: "accepted rejects failure reason",
			status: ActionStatus{
				PlayerID:      "p1",
				ActionStatus:  ActionAccepted,
				FailureReason: ReasonTimeout,
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
