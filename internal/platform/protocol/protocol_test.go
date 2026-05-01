package protocol

import (
	"bytes"
	"errors"
	"testing"
)

func TestEncodeDecodeRequestAndResponse(t *testing.T) {
	req, err := NewRequest("turn-1", "turn", map[string]any{"turn": 1})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	resp, err := NewResponse("turn-1", map[string]any{"action": "rock"})
	if err != nil {
		t.Fatalf("NewResponse: %v", err)
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(req); err != nil {
		t.Fatalf("Encode request: %v", err)
	}
	if err := enc.Encode(resp); err != nil {
		t.Fatalf("Encode response: %v", err)
	}

	dec := NewDecoder(bytes.NewReader(buf.Bytes()))
	decodedReq, err := dec.DecodeRequest()
	if err != nil {
		t.Fatalf("DecodeRequest: %v", err)
	}
	if decodedReq.ID != "turn-1" || decodedReq.Method != "turn" {
		t.Fatalf("decoded request mismatch: %+v", decodedReq)
	}

	decodedResp, err := dec.DecodeResponse()
	if err != nil {
		t.Fatalf("DecodeResponse: %v", err)
	}
	if decodedResp.ID != "turn-1" {
		t.Fatalf("decoded response id = %q, want turn-1", decodedResp.ID)
	}
}

func TestNDJSONFramingAcceptsCRLF(t *testing.T) {
	line := []byte("{\"jsonrpc\":\"2.0\",\"id\":\"init\",\"result\":{\"ready\":true}}\r\n")
	resp, err := DecodeResponseLine(line)
	if err != nil {
		t.Fatalf("DecodeResponseLine: %v", err)
	}
	if resp.ID != "init" {
		t.Fatalf("resp.ID = %q, want init", resp.ID)
	}
}

func TestMalformedAndInvalidEnvelope(t *testing.T) {
	if _, err := DecodeResponseLine([]byte("{")); !errors.Is(err, ErrMalformedJSON) {
		t.Fatalf("expected ErrMalformedJSON, got %v", err)
	}
	if _, err := DecodeResponseLine([]byte("{\"jsonrpc\":\"1.0\",\"id\":\"x\",\"result\":{}}")); !errors.Is(err, ErrInvalidEnvelope) {
		t.Fatalf("expected ErrInvalidEnvelope, got %v", err)
	}
}

func TestMismatchedID(t *testing.T) {
	resp, err := NewResponse("turn-2", map[string]any{"action": "paper"})
	if err != nil {
		t.Fatalf("NewResponse: %v", err)
	}
	if err := MatchResponseID("turn-1", resp); !errors.Is(err, ErrMismatchedID) {
		t.Fatalf("expected ErrMismatchedID, got %v", err)
	}
}
