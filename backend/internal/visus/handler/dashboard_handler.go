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

type DashboardHandler struct {
	baseHandler
	service *service.DashboardService
}

func NewDashboardHandler(service *service.DashboardService, logger zerolog.Logger) *DashboardHandler {
	return &DashboardHandler{baseHandler: baseHandler{logger: logger}, service: service}
}

func (h *DashboardHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok || userID == nil {
		return
	}
	var req dto.CreateDashboardRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Create(r.Context(), &model.Dashboard{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		GridColumns: req.GridColumns,
		Visibility:  model.DashboardVisibility(req.Visibility),
		SharedWith:  req.SharedWith,
		IsDefault:   req.IsDefault,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
		CreatedBy:   *userID,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

var dashboardSortColumns = map[string]string{
	"name":       "name",
	"visibility": "visibility",
	"created_at": "created_at",
	"updated_at": "updated_at",
}

func (h *DashboardHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	sortCol, sortDir := suiteapi.ParseSort(r, dashboardSortColumns, "updated_at", "desc")
	search := r.URL.Query().Get("search")
	visibility := r.URL.Query().Get("visibility")
	items, total, err := h.service.List(r.Context(), tenantID, userID, page, perPage, sortCol, sortDir, search, visibility)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Get(r.Context(), tenantID, userID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *DashboardHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok || userID == nil {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateDashboardRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Update(r.Context(), &model.Dashboard{
		ID:          id,
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		GridColumns: req.GridColumns,
		Visibility:  model.DashboardVisibility(req.Visibility),
		SharedWith:  req.SharedWith,
		IsDefault:   req.IsDefault,
		Tags:        req.Tags,
		Metadata:    req.Metadata,
		CreatedBy:   *userID,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *DashboardHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if err := h.service.Delete(r.Context(), tenantID, id); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DashboardHandler) Duplicate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok || userID == nil {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Duplicate(r.Context(), tenantID, *userID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *DashboardHandler) Share(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.ShareDashboardRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Share(r.Context(), tenantID, id, model.DashboardVisibility(req.Visibility), req.SharedWith)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *DashboardHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, repository.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, repository.ErrConflict):
		suiteapi.WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
	case errors.Is(err, service.ErrForbidden):
		suiteapi.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	default:
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
