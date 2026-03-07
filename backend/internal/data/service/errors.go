package service

import "errors"

var (
	ErrValidation           = errors.New("validation error")
	ErrConflict             = errors.New("conflict")
	ErrConnectionTestFailed = errors.New("connection test failed")
	ErrUnsupportedType      = errors.New("unsupported source type")
	ErrTooManyRequests      = errors.New("too many requests")
	ErrForbiddenOperation   = errors.New("forbidden operation")
)
