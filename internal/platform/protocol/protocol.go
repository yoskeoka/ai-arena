package protocol

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
	ErrMalformedJSON   = errors.New("protocol: malformed json")
	ErrInvalidEnvelope = errors.New("protocol: invalid envelope")
	ErrMismatchedID    = errors.New("protocol: mismatched id")
)

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ErrorObject    `json:"error,omitempty"`
}

type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Encoder struct {
	enc *json.Encoder
}

type Decoder struct {
	scanner *bufio.Scanner
}

func NewEncoder(w io.Writer) *Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Encoder{enc: enc}
}

func NewDecoder(r io.Reader) *Decoder {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &Decoder{scanner: scanner}
}

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

func (e *Encoder) Encode(v any) error {
	return e.enc.Encode(v)
}

func (d *Decoder) DecodeRequest() (Request, error) {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return Request{}, err
		}
		return Request{}, io.EOF
	}
	return DecodeRequestLine(d.scanner.Bytes())
}

func (d *Decoder) DecodeResponse() (Response, error) {
	if !d.scanner.Scan() {
		if err := d.scanner.Err(); err != nil {
			return Response{}, err
		}
		return Response{}, io.EOF
	}
	return DecodeResponseLine(d.scanner.Bytes())
}

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
