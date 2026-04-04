package cti

import (
	apperrors "github.com/clario360/platform/internal/errors"
)

// CTI error constructors — wrapping the shared AppError model.

func errNotFound(entity string) *apperrors.AppError {
	return apperrors.NewNotFound("CTI_NOT_FOUND", entity+" not found")
}

func errConflict(msg string) *apperrors.AppError {
	return apperrors.NewConflict("CTI_CONFLICT", msg)
}

func errValidation(msg string) *apperrors.AppError {
	return apperrors.NewValidation("CTI_VALIDATION_ERROR", msg, nil)
}

func errInternal(msg string, err error) *apperrors.AppError {
	return apperrors.NewInternal("CTI_INTERNAL_ERROR", msg, err)
}
