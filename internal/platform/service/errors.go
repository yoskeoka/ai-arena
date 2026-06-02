package service

import "errors"

var (
	// ErrBadRequest marks one operator-visible validation or admission failure.
	ErrBadRequest = errors.New("service: bad request")
	// ErrConflict marks one operator-visible state conflict.
	ErrConflict = errors.New("service: conflict")
)
