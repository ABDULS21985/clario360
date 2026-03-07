package handler

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
	"github.com/clario360/platform/internal/visus/dto"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
	"github.com/clario360/platform/internal/visus/service"
)

type WidgetHandler struct {
	baseHandler
	service *service.WidgetService
}

func NewWidgetHandler(service *service.WidgetService, logger zerolog.Logger) *WidgetHandler {
	return &WidgetHandler{baseHandler: baseHandler{logger: logger}, service: service}
}

func (h *WidgetHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dashboardID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.CreateWidgetRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Create(r.Context(), &model.Widget{
		TenantID:               tenantID,
		DashboardID:            dashboardID,
		Title:                  req.Title,
		Subtitle:               req.Subtitle,
		Type:                   model.WidgetType(req.Type),
		Config:                 req.Config,
		Position:               model.WidgetPosition(req.Position),
		RefreshIntervalSeconds: req.RefreshIntervalSeconds,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *WidgetHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dashboardID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	items, err := h.service.List(r.Context(), tenantID, dashboardID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *WidgetHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dashboardID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	widgetID, err := suiteapi.UUIDParam(r, "wid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateWidgetRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Update(r.Context(), &model.Widget{
		ID:                     widgetID,
		TenantID:               tenantID,
		DashboardID:            dashboardID,
		Title:                  req.Title,
		Subtitle:               req.Subtitle,
		Config:                 req.Config,
		Position:               model.WidgetPosition(req.Position),
		RefreshIntervalSeconds: req.RefreshIntervalSeconds,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *WidgetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dashboardID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	widgetID, err := suiteapi.UUIDParam(r, "wid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	if err := h.service.Delete(r.Context(), tenantID, dashboardID, widgetID); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *WidgetHandler) Data(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dashboardID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	widgetID, err := suiteapi.UUIDParam(r, "wid")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	items, err := h.service.List(r.Context(), tenantID, dashboardID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	for _, item := range items {
		if item.ID != widgetID {
			continue
		}
		data, err := h.service.GetWidgetData(r.Context(), tenantID, &item)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		suiteapi.WriteData(w, http.StatusOK, data)
		return
	}
	suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "widget not found", nil)
}

func (h *WidgetHandler) UpdateLayout(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	dashboardID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateWidgetLayoutRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	positions := make(map[uuid.UUID]model.WidgetPosition, len(req.Positions))
	for _, position := range req.Positions {
		positions[position.WidgetID] = model.WidgetPosition{X: position.X, Y: position.Y, W: position.W, H: position.H}
	}
	if err := h.service.UpdateLayout(r.Context(), tenantID, dashboardID, positions); err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]any{"updated": len(positions)})
}

func (h *WidgetHandler) Types(w http.ResponseWriter, r *http.Request) {
	suiteapi.WriteData(w, http.StatusOK, []map[string]any{
		{"type": "kpi_card", "schema": map[string]any{"kpi_id": "uuid", "show_trend": "bool", "show_target": "bool"}},
		{"type": "line_chart", "schema": map[string]any{"suite": "string", "data_source": "string", "x_axis": "string", "y_axis": []string{}}},
		{"type": "bar_chart", "schema": map[string]any{"suite": "string", "data_source": "string", "x_axis": "string", "y_axis": []string{}}},
		{"type": "area_chart", "schema": map[string]any{"suite": "string", "data_source": "string", "x_axis": "string", "y_axis": []string{}}},
		{"type": "pie_chart", "schema": map[string]any{"suite": "string", "data_source": "string", "label_path": "string", "value_path": "string"}},
		{"type": "gauge", "schema": map[string]any{"kpi_id": "uuid"}},
		{"type": "table", "schema": map[string]any{"suite": "string", "data_source": "string", "columns": []map[string]any{}}},
		{"type": "alert_feed", "schema": map[string]any{"alert_sources": []string{}, "severity_filter": []string{}}},
		{"type": "text", "schema": map[string]any{"content": "string"}},
		{"type": "sparkline", "schema": map[string]any{"kpi_id": "uuid", "points": "number"}},
		{"type": "heatmap", "schema": map[string]any{"suite": "string", "data_source": "string", "x_axis": "string", "y_axis": "string", "value_key": "string"}},
		{"type": "status_grid", "schema": map[string]any{"items": []map[string]any{}}},
		{"type": "trend_indicator", "schema": map[string]any{"kpi_id": "uuid"}},
	})
}

func (h *WidgetHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	case errors.Is(err, repository.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	default:
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
}
