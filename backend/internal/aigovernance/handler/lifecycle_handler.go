package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/suiteapi"
)

type LifecycleHandler struct {
	services Services
	logger   zerolog.Logger
}

func NewLifecycleHandler(services Services, logger zerolog.Logger) *LifecycleHandler {
	return &LifecycleHandler{services: services, logger: logger.With().Str("handler", "ai_lifecycle").Logger()}
}

func (h *LifecycleHandler) Promote(w http.ResponseWriter, r *http.Request) {
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
	var req aigovdto.PromoteRequest
	if r.ContentLength > 0 {
		if !decodeBody(w, r, &req) {
			return
		}
	}
	item, err := h.services.Lifecycle.Promote(r.Context(), tenantID, modelID, versionID, req.ApprovedBy, req.Override)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *LifecycleHandler) Retire(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	userID := userID(r)
	if userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
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
	var req aigovdto.RetireVersionRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Lifecycle.Retire(r.Context(), tenantID, modelID, versionID, *userID, req.Reason)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *LifecycleHandler) Fail(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	userID := userID(r)
	if userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
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
	var req aigovdto.FailVersionRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Lifecycle.Fail(r.Context(), tenantID, modelID, versionID, *userID, req.Reason)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *LifecycleHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	userID := userID(r)
	if userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req aigovdto.RollbackRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.services.Lifecycle.Rollback(r.Context(), tenantID, modelID, *userID, req.Reason)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *LifecycleHandler) History(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantID(w, r)
	if !ok {
		return
	}
	modelID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	items, err := h.services.Lifecycle.History(r.Context(), tenantID, modelID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *LifecycleHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{id}/versions/{vid}/promote", h.Promote)
	r.Post("/{id}/versions/{vid}/retire", h.Retire)
	r.Post("/{id}/versions/{vid}/fail", h.Fail)
	r.Post("/{id}/rollback", h.Rollback)
	r.Get("/{id}/lifecycle-history", h.History)
	return r
}
