package protocol

import (
	"io"

	publicgm "github.com/yoskeoka/ai-arena/gamemaster"
)

var (
	// ErrMalformedJSON reports invalid JSON input.
	ErrMalformedJSON = publicgm.ErrMalformedJSON
	// ErrInvalidEnvelope reports syntactically valid but invalid protocol envelopes.
	ErrInvalidEnvelope = publicgm.ErrInvalidEnvelope
	// ErrMismatchedID reports a response whose id does not match the request.
	ErrMismatchedID = publicgm.ErrMismatchedID
)

// Request is a JSON-RPC request envelope.
type Request = publicgm.Request

// Response is a JSON-RPC response envelope.
type Response = publicgm.Response

// ErrorObject is the JSON-RPC error payload shape.
type ErrorObject = publicgm.ErrorObject

// Encoder writes NDJSON-framed JSON-RPC messages.
type Encoder = publicgm.Encoder

// Decoder reads NDJSON-framed JSON-RPC messages.
type Decoder = publicgm.Decoder

// NewEncoder builds an encoder that writes one JSON value per line.
func NewEncoder(w io.Writer) *Encoder {
	return publicgm.NewEncoder(w)
}

// NewDecoder builds a decoder with a larger NDJSON line buffer.
func NewDecoder(r io.Reader) *Decoder {
	return publicgm.NewDecoder(r)
}

// NewRequest marshals a JSON-RPC request envelope.
func NewRequest(id, method string, params any) (Request, error) {
	return publicgm.NewRequest(id, method, params)
}

// NewNotification marshals a JSON-RPC notification envelope.
func NewNotification(method string, params any) (Request, error) {
	return publicgm.NewNotification(method, params)
}

// NewResponse marshals a JSON-RPC success response envelope.
func NewResponse(id string, result any) (Response, error) {
	return publicgm.NewResponse(id, result)
}

// DecodeRequestLine validates one raw request line.
func DecodeRequestLine(line []byte) (Request, error) {
	return publicgm.DecodeRequestLine(line)
}

// DecodeResponseLine validates one raw response line.
func DecodeResponseLine(line []byte) (Response, error) {
	return publicgm.DecodeResponseLine(line)
}

// MatchResponseID checks that the response id matches the expected request id.
func MatchResponseID(expected string, resp Response) error {
	return publicgm.MatchResponseID(expected, resp)
}
