package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/suiteapi"
)

type PredictionHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewPredictionHandler(services Services, logger zerolog.Logger) *PredictionHandler {
	return &PredictionHandler{services: services, logger: logger.With().Str("handler", "ai_predictions").Logger()}
}

func (h *PredictionHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	var modelID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("model_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid model_id", nil)
			return
		}
		modelID = &parsed
	}
	var isShadow *bool
	if raw := strings.TrimSpace(r.URL.Query().Get("is_shadow")); raw == "true" || raw == "false" {
		value := raw == "true"
		isShadow = &value
	}
	items, total, err := h.services.Predictions.List(r.Context(), tenantID, aigovdto.PredictionQuery{
		ModelID:    modelID,
		Suite:      r.URL.Query().Get("suite"),
		UseCase:    r.URL.Query().Get("use_case"),
		EntityType: r.URL.Query().Get("entity_type"),
		IsShadow:   isShadow,
		Search:     r.URL.Query().Get("search"),
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *PredictionHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	predictionID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.services.Predictions.Get(r.Context(), tenantID, predictionID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *PredictionHandler) Feedback(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	userID := userID(r)
	if userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	predictionID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.PredictionFeedbackRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if err := h.services.Predictions.SubmitFeedback(r.Context(), tenantID, *userID, predictionID, req); err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]string{"message": "feedback recorded"})
}

func (h *PredictionHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	items, err := h.services.Predictions.Stats(r.Context(), tenantID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *PredictionHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/stats", h.Stats)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/feedback", h.Feedback)
	return r
}
