package service

import "errors"

var (
	// ErrBadRequest marks one operator-visible validation or admission failure.
	ErrBadRequest = errors.New("service: bad request")
	// ErrConflict marks one operator-visible state conflict.
	ErrConflict = errors.New("service: conflict")
	// ErrWorkerQueueOwned reports that another process holds the queue worker ownership lock.
	ErrWorkerQueueOwned = errors.New("service: worker queue already owned")
	// ErrWorkerOwnershipTimeout reports that a worker could not acquire queue ownership before its handoff deadline.
	ErrWorkerOwnershipTimeout = errors.New("service: worker ownership wait timed out")
)
