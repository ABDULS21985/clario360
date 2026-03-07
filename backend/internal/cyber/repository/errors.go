package repository

import "errors"

// ErrNotFound is returned when a requested record does not exist or belongs
// to a different tenant.
var ErrNotFound = errors.New("record not found")

// ErrConflict is returned when a uniqueness or state transition constraint is violated.
var ErrConflict = errors.New("record conflict")

// ErrInvalidInput is returned when a request fails domain-level validation beyond struct tags.
var ErrInvalidInput = errors.New("invalid input")
