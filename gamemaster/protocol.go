package gamemaster

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const version = "2.0"

var (
	// ErrMalformedJSON reports invalid JSON input.
	ErrMalformedJSON = errors.New("protocol: malformed json")
	// ErrInvalidEnvelope reports syntactically valid but invalid protocol envelopes.
	ErrInvalidEnvelope = errors.New("protocol: invalid envelope")
	// ErrMismatchedID reports a response whose id does not match the request.
	ErrMismatchedID = errors.New("protocol: mismatched id")
)

// Request is a JSON-RPC request envelope.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC response envelope.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

// ErrorObject is the JSON-RPC error payload shape.
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Encoder writes NDJSON-framed JSON-RPC messages.
type Encoder struct {
	enc *json.Encoder
}

// Decoder reads NDJSON-framed JSON-RPC messages.
type Decoder struct {
	scanner *bufio.Scanner
}

// NewEncoder builds an encoder that writes one JSON value per line.
func NewEncoder(w io.Writer) *Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Encoder{enc: enc}
}

// NewDecoder builds a decoder with a larger NDJSON line buffer.
func NewDecoder(r io.Reader) *Decoder {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &Decoder{scanner: scanner}
}

// NewRequest marshals a JSON-RPC request envelope.
func NewRequest(id, method string, params any) (Request, error) {
	raw, err := marshalParams(params)
	if err != nil {
		return Request{}, err
	}
	return Request{
		JSONRPC: version,
		ID:      id,
		Method:  method,
		Params:  raw,
	}, nil
}

// NewNotification marshals a JSON-RPC notification envelope.
func NewNotification(method string, params any) (Request, error) {
	raw, err := marshalParams(params)
	if err != nil {
		return Request{}, err
	}
	return Request{
		JSONRPC: version,
		Method:  method,
		Params:  raw,
	}, nil
}

// NewResponse marshals a JSON-RPC success response envelope.
func NewResponse(id string, result any) (Response, error) {
	raw, err := marshalParams(result)
	if err != nil {
		return Response{}, err
	}
	return Response{
		JSONRPC: version,
		ID:      id,
		Result:  raw,
	}, nil
}

// Encode writes one JSON value followed by a newline.
func (e *Encoder) Encode(v any) error {
	return e.enc.Encode(v)
}

// DecodeRequest reads and validates the next request line.
func (d *Decoder) DecodeRequest() (Request, error) {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return Request{}, err
		}
		return Request{}, io.EOF
	}
	return DecodeRequestLine(d.scanner.Bytes())
}

// DecodeResponse reads and validates the next response line.
func (d *Decoder) DecodeResponse() (Response, error) {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return Response{}, err
		}
		return Response{}, io.EOF
	}
	return DecodeResponseLine(d.scanner.Bytes())
}

// DecodeRequestLine validates one raw request line.
func DecodeRequestLine(line []byte) (Request, error) {
	line = bytes.TrimRight(line, "\r\n")
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return Request{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if err := validateRequest(req); err != nil {
		return Request{}, err
	}
	return req, nil
}

// DecodeResponseLine validates one raw response line.
func DecodeResponseLine(line []byte) (Response, error) {
	line = bytes.TrimRight(line, "\r\n")
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if err := validateResponse(resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}

// MatchResponseID checks that the response id matches the expected request id.
func MatchResponseID(expected string, resp Response) error {
	if expected != resp.ID {
		return fmt.Errorf("%w: expected %q, got %q", ErrMismatchedID, expected, resp.ID)
	}
	return nil
}

func marshalParams(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func validateRequest(req Request) error {
	if req.JSONRPC != version {
		return fmt.Errorf("%w: jsonrpc must be %q", ErrInvalidEnvelope, version)
	}
	if req.Method == "" {
		return fmt.Errorf("%w: method is required", ErrInvalidEnvelope)
	}
	return nil
}

func validateResponse(resp Response) error {
	if resp.JSONRPC != version {
		return fmt.Errorf("%w: jsonrpc must be %q", ErrInvalidEnvelope, version)
	}
	if resp.ID == "" {
		return fmt.Errorf("%w: id is required", ErrInvalidEnvelope)
	}
	hasResult := len(resp.Result) > 0
	hasError := resp.Error != nil
	if hasResult == hasError {
		return fmt.Errorf("%w: exactly one of result or error is required", ErrInvalidEnvelope)
	}
	if hasError && resp.Error.Message == "" {
		return fmt.Errorf("%w: error.message is required", ErrInvalidEnvelope)
	}
	return nil
}
