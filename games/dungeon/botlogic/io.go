package botlogic

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

const jsonRPCVersion = "2.0"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
}

type encoder struct {
	enc *json.Encoder
}

type decoder struct {
	scanner *bufio.Scanner
}

func newEncoder(w io.Writer) *encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &encoder{enc: enc}
}

func (e *encoder) encode(v any) error {
	return e.enc.Encode(v)
}

func newDecoder(r io.Reader) *decoder {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &decoder{scanner: scanner}
}

func (d *decoder) decodeRequest() (request, error) {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return request{}, err
		}
		return request{}, io.EOF
	}

	var req request
	if err := json.Unmarshal(d.scanner.Bytes(), &req); err != nil {
		return request{}, err
	}
	if req.JSONRPC != jsonRPCVersion {
		return request{}, fmt.Errorf("unsupported jsonrpc version %q", req.JSONRPC)
	}
	if req.Method == "" {
		return request{}, fmt.Errorf("request method is required")
	}
	return req, nil
}

func newResponse(id string, result any) (response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return response{}, err
	}
	return response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Result:  raw,
	}, nil
}
