package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	apperrors "github.com/clario360/platform/internal/errors"
	"github.com/clario360/platform/internal/suiteapi"
)

type baseHandler struct {
	logger zerolog.Logger
}

func (h *baseHandler) tenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

func (h *baseHandler) userID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, err := suiteapi.UserID(r)
	if err != nil || userID == nil {
		message := "authentication required"
		if err != nil {
			message = err.Error()
		}
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", message, nil)
		return uuid.Nil, false
	}
	return *userID, true
}

func (h *baseHandler) tenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	userID, ok := h.userID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, userID, true
}

func (h *baseHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		details := any(nil)
		if len(appErr.Fields) > 0 {
			details = appErr.Fields
		}
		suiteapi.WriteError(w, r, appErr.Status, appErr.Code, appErr.Message, details)
		return
	}
	h.logger.Error().Err(err).Msg("acta request failed")
	suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
}

func parseOptionalUUID(raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalDateTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			parsed = parsed.UTC()
			return &parsed, nil
		}
	}
	return nil, errors.New("invalid datetime format")
}

func parseOptionalMonth(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	layouts := []string{
		"2006-01",
		"2006-01-02",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			month := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.UTC)
			return &month, nil
		}
	}
	return nil, errors.New("invalid month format")
}

func parseOptionalInt(raw string) (*int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func parseOptionalBool(raw string) (*bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
