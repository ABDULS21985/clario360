package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/suiteapi"
)

type DriftHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewDriftHandler(services Services, logger zerolog.Logger) *DriftHandler {
	return &DriftHandler{services: services, logger: logger.With().Str("handler", "ai_drift").Logger()}
}

func (h *DriftHandler) Latest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.services.Drift.Latest(r.Context(), tenantID, modelID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *DriftHandler) History(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.services.Drift.History(r.Context(), tenantID, modelID, limit)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *DriftHandler) Performance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var query aigovdto.PerformanceQuery
	query.Period = r.URL.Query().Get("period")
	items, err := h.services.Drift.Performance(r.Context(), tenantID, modelID, query.Period)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *DriftHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{id}/drift", h.Latest)
	r.Get("/{id}/drift/history", h.History)
	r.Get("/{id}/performance", h.Performance)
	return r
}
