package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/suiteapi"
)

type baseHandler struct {
	logger zerolog.Logger
}

func (h *baseHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", cleanError(err), nil)
	case errors.Is(err, service.ErrConflict):
		suiteapi.WriteError(w, r, http.StatusConflict, "CONFLICT", cleanError(err), nil)
	case errors.Is(err, service.ErrTooManyRequests):
		suiteapi.WriteError(w, r, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", cleanError(err), nil)
	case errors.Is(err, service.ErrForbiddenOperation):
		suiteapi.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", cleanError(err), nil)
	case errors.Is(err, service.ErrConnectionTestFailed):
		suiteapi.WriteError(w, r, http.StatusUnprocessableEntity, "CONNECTION_TEST_FAILED", cleanError(err), nil)
	case errors.Is(err, service.ErrTimeout):
		suiteapi.WriteError(w, r, http.StatusGatewayTimeout, "TIMEOUT", cleanError(err), nil)
	case errors.Is(err, service.ErrUnsupportedType):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "UNSUPPORTED_TYPE", cleanError(err), nil)
	case errors.Is(err, pgx.ErrNoRows):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "resource not found", nil)
	default:
		h.logger.Error().Err(err).Msg("data suite request failed")
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func cleanError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	message = strings.TrimPrefix(message, service.ErrValidation.Error()+": ")
	message = strings.TrimPrefix(message, service.ErrConflict.Error()+": ")
	message = strings.TrimPrefix(message, service.ErrTooManyRequests.Error()+": ")
	message = strings.TrimPrefix(message, service.ErrForbiddenOperation.Error()+": ")
	message = strings.TrimPrefix(message, service.ErrConnectionTestFailed.Error()+": ")
	message = strings.TrimPrefix(message, service.ErrTimeout.Error()+": ")
	message = strings.TrimPrefix(message, service.ErrUnsupportedType.Error()+": ")
	return message
}
