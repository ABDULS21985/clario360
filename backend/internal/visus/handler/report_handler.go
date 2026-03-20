package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
	"github.com/clario360/platform/internal/visus/dto"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
	"github.com/clario360/platform/internal/visus/service"
)

type ReportHandler struct {
	baseHandler
	service *service.ReportService
}

func NewReportHandler(service *service.ReportService, logger zerolog.Logger) *ReportHandler {
	return &ReportHandler{baseHandler: baseHandler{logger: logger}, service: service}
}

func (h *ReportHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok || userID == nil {
		return
	}
	var req dto.CreateReportRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Create(r.Context(), &model.ReportDefinition{
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		ReportType:        model.ReportType(req.ReportType),
		Sections:          req.Sections,
		Period:            req.Period,
		CustomPeriodStart: parseOptionalDate(valueOrEmpty(req.CustomPeriodStart)),
		CustomPeriodEnd:   parseOptionalDate(valueOrEmpty(req.CustomPeriodEnd)),
		Schedule:          req.Schedule,
		Recipients:        req.Recipients,
		AutoSend:          req.AutoSend,
		CreatedBy:         *userID,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

var reportSortColumns = map[string]string{
	"name":              "name",
	"report_type":       "report_type",
	"schedule":          "schedule",
	"last_generated_at": "last_generated_at",
	"updated_at":        "updated_at",
	"created_at":        "created_at",
}

func (h *ReportHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	sortCol, sortDir := suiteapi.ParseSort(r, reportSortColumns, "updated_at", "desc")
	search := r.URL.Query().Get("search")
	reportType := r.URL.Query().Get("report_type")
	var autoSend *bool
	if raw := r.URL.Query().Get("auto_send"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "auto_send must be a boolean", nil)
			return
		}
		autoSend = &value
	}
	items, total, err := h.service.List(r.Context(), tenantID, page, perPage, sortCol, sortDir, search, reportType, autoSend)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *ReportHandler) Get(w http.ResponseWriter, r *http.Request) {
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

func (h *ReportHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok || userID == nil {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateReportRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Update(r.Context(), &model.ReportDefinition{
		ID:                id,
		TenantID:          tenantID,
		Name:              req.Name,
		Description:       req.Description,
		ReportType:        model.ReportType(req.ReportType),
		Sections:          req.Sections,
		Period:            req.Period,
		CustomPeriodStart: parseOptionalDate(valueOrEmpty(req.CustomPeriodStart)),
		CustomPeriodEnd:   parseOptionalDate(valueOrEmpty(req.CustomPeriodEnd)),
		Schedule:          req.Schedule,
		Recipients:        req.Recipients,
		AutoSend:          req.AutoSend,
		CreatedBy:         *userID,
	})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ReportHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

func (h *ReportHandler) Generate(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.Generate(r.Context(), id, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusAccepted, item)
}

func (h *ReportHandler) Snapshots(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	items, err := h.service.ListSnapshots(r.Context(), tenantID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *ReportHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	reportID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	snapshotID, err := suiteapi.UUIDParam(r, "snapId")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GetSnapshot(r.Context(), tenantID, reportID, snapshotID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ReportHandler) LatestSnapshot(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	reportID, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.LatestSnapshot(r.Context(), tenantID, reportID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (h *ReportHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
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
