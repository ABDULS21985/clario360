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

type KPIHandler struct {
	baseHandler
	service *service.KPIService
}

func NewKPIHandler(service *service.KPIService, logger zerolog.Logger) *KPIHandler {
	return &KPIHandler{baseHandler: baseHandler{logger: logger}, service: service}
}

func (h *KPIHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok || userID == nil {
		return
	}
	var req dto.CreateKPIRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item, err := h.service.Create(r.Context(), &model.KPIDefinition{
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		Category:          model.KPICategory(req.Category),
		Suite:             model.KPISuite(req.Suite),
		Icon:              req.Icon,
		QueryEndpoint:     req.QueryEndpoint,
		QueryParams:       req.QueryParams,
		ValuePath:         req.ValuePath,
		Unit:              model.KPIUnit(req.Unit),
		FormatPattern:     req.FormatPattern,
		TargetValue:       req.TargetValue,
		WarningThreshold:  req.WarningThreshold,
		CriticalThreshold: req.CriticalThreshold,
		Direction:         model.KPIDirection(req.Direction),
		CalculationType:   model.KPICalculationType(req.CalculationType),
		CalculationWindow: req.CalculationWindow,
		SnapshotFrequency: model.KPISnapshotFrequency(req.SnapshotFrequency),
		Enabled:           enabled,
		Tags:              req.Tags,
		CreatedBy:         *userID,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *KPIHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.service.List(r.Context(), tenantID, page, perPage)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *KPIHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, history, err := h.service.Get(r.Context(), tenantID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]any{"definition": item, "history": history})
}

func (h *KPIHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateKPIRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item, err := h.service.Update(r.Context(), &model.KPIDefinition{
		ID:                id,
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		Category:          model.KPICategory(req.Category),
		Suite:             model.KPISuite(req.Suite),
		Icon:              req.Icon,
		QueryEndpoint:     req.QueryEndpoint,
		QueryParams:       req.QueryParams,
		ValuePath:         req.ValuePath,
		Unit:              model.KPIUnit(req.Unit),
		FormatPattern:     req.FormatPattern,
		TargetValue:       req.TargetValue,
		WarningThreshold:  req.WarningThreshold,
		CriticalThreshold: req.CriticalThreshold,
		Direction:         model.KPIDirection(req.Direction),
		CalculationType:   model.KPICalculationType(req.CalculationType),
		CalculationWindow: req.CalculationWindow,
		SnapshotFrequency: model.KPISnapshotFrequency(req.SnapshotFrequency),
		Enabled:           enabled,
		Tags:              req.Tags,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *KPIHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *KPIHandler) History(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	limit := 100
	if _, perPage := suiteapi.ParsePagination(r); perPage > 0 {
		limit = perPage
	}
	history, err := h.service.History(r.Context(), tenantID, id, parseOptionalDate(r.URL.Query().Get("start")), parseOptionalDate(r.URL.Query().Get("end")), limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, history)
}

func (h *KPIHandler) TriggerSnapshot(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	if err := h.service.TriggerSnapshot(r.Context(), tenantID); err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusAccepted, map[string]any{"status": "started"})
}

func (h *KPIHandler) Summary(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	items, err := h.service.Summary(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *KPIHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, repository.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, repository.ErrConflict):
		suiteapi.WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
	default:
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
