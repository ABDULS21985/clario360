package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/suiteapi"
)

type ShadowHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewShadowHandler(services Services, logger zerolog.Logger) *ShadowHandler {
	return &ShadowHandler{services: services, logger: logger.With().Str("handler", "ai_shadow").Logger()}
}

func (h *ShadowHandler) Start(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.StartShadowRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Shadow.Start(r.Context(), tenantID, modelID, req.VersionID, userID(r))
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ShadowHandler) Stop(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.StopShadowRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Shadow.Stop(r.Context(), tenantID, modelID, req.VersionID, req.Reason)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ShadowHandler) LatestComparison(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.services.Shadow.LatestComparison(r.Context(), tenantID, modelID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ShadowHandler) History(w http.ResponseWriter, r *http.Request) {
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
	items, err := h.services.Shadow.ComparisonHistory(r.Context(), tenantID, modelID, limit)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *ShadowHandler) Divergences(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.services.Shadow.Divergences(r.Context(), tenantID, modelID, page, perPage)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *ShadowHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/start", h.Start)
	r.Post("/stop", h.Stop)
	r.Get("/comparison", h.LatestComparison)
	r.Get("/comparison/history", h.History)
	r.Get("/divergences", h.Divergences)
	return r
}
