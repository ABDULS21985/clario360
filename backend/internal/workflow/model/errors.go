package model

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrConflict         = errors.New("already exists")
	ErrConcurrencyConfl = errors.New("concurrency conflict: row was modified by another process")
	ErrTaskNotClaimable = errors.New("task is not in a claimable state")
	ErrTaskNotOwned     = errors.New("task is not owned by this user")
)
