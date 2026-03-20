package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/suiteapi"
)

type ValidationHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewValidationHandler(services Services, logger zerolog.Logger) *ValidationHandler {
	return &ValidationHandler{services: services, logger: logger.With().Str("handler", "ai_validation").Logger()}
}

func (h *ValidationHandler) Preview(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	versionID, err := suiteapi.UUIDParam(r, "vid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.ValidateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Validation.Preview(r.Context(), tenantID, modelID, versionID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ValidationHandler) Run(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	versionID, err := suiteapi.UUIDParam(r, "vid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.ValidateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Validation.Validate(r.Context(), tenantID, modelID, versionID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ValidationHandler) Latest(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	versionID, err := suiteapi.UUIDParam(r, "vid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.services.Validation.Latest(r.Context(), tenantID, versionID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ValidationHandler) History(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	versionID, err := suiteapi.UUIDParam(r, "vid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.services.Validation.History(r.Context(), tenantID, versionID, limit)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *ValidationHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/preview", h.Preview)
	r.Post("/", h.Run)
	r.Get("/", h.Latest)
	r.Get("/history", h.History)
	return r
}
