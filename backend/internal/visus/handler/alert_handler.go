package handler

import (
	"errors"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
	"github.com/clario360/platform/internal/visus/dto"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
	"github.com/clario360/platform/internal/visus/service"
)

type AlertHandler struct {
	baseHandler
	service *service.AlertService
}

func NewAlertHandler(service *service.AlertService, logger zerolog.Logger) *AlertHandler {
	return &AlertHandler{baseHandler: baseHandler{logger: logger}, service: service}
}

func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.List(r.Context(), tenantID, repository.AlertListFilters{
		Status:       suiteapi.ParseCSVParam(r, "status"),
		Severity:     suiteapi.ParseCSVParam(r, "severity"),
		Category:     suiteapi.ParseCSVParam(r, "category"),
		SourceSuites: suiteapi.ParseCSVParam(r, "source_suite"),
		Search:       r.URL.Query().Get("search"),
	}, page, perPage)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AlertHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateAlertStatusRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.UpdateStatus(r.Context(), tenantID, id, model.AlertStatus(req.Status), userID, req.ActionNotes, req.DismissReason)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *AlertHandler) Count(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	count, err := h.service.Count(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]int{"count": count})
}

func (h *AlertHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	stats, err := h.service.Stats(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, stats)
}

func (h *AlertHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	default:
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
