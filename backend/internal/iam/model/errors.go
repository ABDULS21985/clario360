package model

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrUnauthorized  = errors.New("invalid credentials")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("already exists")
	ErrValidation    = errors.New("validation failed")
	ErrAccountLocked = errors.New("account temporarily locked")
	ErrMFARequired   = errors.New("mfa verification required")
	ErrInvalidMFA    = errors.New("invalid mfa code")
	ErrInvalidToken  = errors.New("invalid or expired token")
	ErrSystemRole    = errors.New("cannot modify system role")
)
