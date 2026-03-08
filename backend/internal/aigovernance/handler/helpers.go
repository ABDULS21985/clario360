package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/suiteapi"
)

func tenantID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "MISSING_TENANT", err.Error(), nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

func userID(r *http.Request) *uuid.UUID {
	value, _ := suiteapi.UserID(r)
	return value
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := suiteapi.DecodeJSON(r, dst); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_BODY", "invalid request body", err.Error())
		return false
	}
	return true
}

func writeError(logger zerolog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case isValidationError(err):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		logger.Error().Err(err).Msg("ai governance request failed")
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "request failed", nil)
	}
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, token := range []string{
		"required",
		"invalid",
		"must",
		"reject",
		"approval",
		"rollback",
		"shadow mode",
		"transition",
	} {
		if strings.Contains(text, token) {
			return true
		}
	}
	return false
}

type Services struct {
	Registry     *aigovservice.RegistryService
	Predictions  *aigovservice.PredictionService
	Explanations *aigovservice.ExplanationService
	Shadow       *aigovservice.ShadowService
	Lifecycle    *aigovservice.LifecycleService
	Drift        *aigovservice.DriftService
	Dashboard    *aigovservice.DashboardService
}
