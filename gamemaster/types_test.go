package gamemaster

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDecisionRequestMarshalJSONUsesMilliseconds(t *testing.T) {
	req := DecisionRequest{
		PlayerID:        "p1",
		VisibleState:    json.RawMessage(`{"turn":1}`),
		LegalActionHint: json.RawMessage(`["move"]`),
		Deadline:        1500 * time.Millisecond,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got struct {
		DeadlineMS int64 `json:"deadline_ms"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal roundtrip: %v", err)
	}
	if got.DeadlineMS != 1500 {
		t.Fatalf("deadline_ms = %d, want 1500", got.DeadlineMS)
	}
}

func TestDecisionRequestUnmarshalJSONUsesMilliseconds(t *testing.T) {
	var req DecisionRequest
	if err := json.Unmarshal([]byte(`{"player_id":"p1","deadline_ms":2200}`), &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if req.Deadline != 2200*time.Millisecond {
		t.Fatalf("Deadline = %s, want 2.2s", req.Deadline)
	}
}
