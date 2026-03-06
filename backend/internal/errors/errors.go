package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for type checking.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation error")
	ErrInternal     = errors.New("internal error")
)

// AppError is a structured application error that carries an HTTP status code,
// a machine-readable code, and a human-readable message.
type AppError struct {
	// HTTP status code
	Status int `json:"status"`
	// Machine-readable error code (e.g., "TENANT_NOT_FOUND")
	Code string `json:"code"`
	// Human-readable message
	Message string `json:"message"`
	// Optional field-level validation errors
	Fields map[string]string `json:"fields,omitempty"`
	// Wrapped internal error (not serialized)
	Err error `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewNotFound creates a 404 error.
func NewNotFound(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusNotFound,
		Code:    code,
		Message: message,
		Err:     ErrNotFound,
	}
}

// NewUnauthorized creates a 401 error.
func NewUnauthorized(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusUnauthorized,
		Code:    code,
		Message: message,
		Err:     ErrUnauthorized,
	}
}

// NewForbidden creates a 403 error.
func NewForbidden(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusForbidden,
		Code:    code,
		Message: message,
		Err:     ErrForbidden,
	}
}

// NewConflict creates a 409 error.
func NewConflict(code, message string) *AppError {
	return &AppError{
		Status:  http.StatusConflict,
		Code:    code,
		Message: message,
		Err:     ErrConflict,
	}
}

// NewValidation creates a 422 error with field-level details.
func NewValidation(code, message string, fields map[string]string) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Code:    code,
		Message: message,
		Fields:  fields,
		Err:     ErrValidation,
	}
}

// NewInternal creates a 500 error.
func NewInternal(code, message string, err error) *AppError {
	return &AppError{
		Status:  http.StatusInternalServerError,
		Code:    code,
		Message: message,
		Err:     fmt.Errorf("%w: %v", ErrInternal, err),
	}
}

// IsNotFound checks if the error is a not-found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsConflict checks if the error is a conflict error.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

// HTTPStatus extracts the HTTP status code from an error.
// Falls back to 500 for untyped errors.
func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrUnauthorized) {
		return http.StatusUnauthorized
	}
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden
	}
	if errors.Is(err, ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrValidation) {
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}
