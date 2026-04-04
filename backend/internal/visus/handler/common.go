package handler

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
)

type baseHandler struct {
	logger zerolog.Logger
}

func (h baseHandler) tenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

func (h baseHandler) tenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, *uuid.UUID, bool) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return uuid.Nil, nil, false
	}
	userID, err := suiteapi.UserID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return uuid.Nil, nil, false
	}
	return tenantID, userID, true
}

func parseOptionalDate(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return &parsed
	}
	return nil
}
