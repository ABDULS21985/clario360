package repository

import "errors"

// ErrNotFound is returned when a requested record does not exist or belongs
// to a different tenant.
var ErrNotFound = errors.New("record not found")
