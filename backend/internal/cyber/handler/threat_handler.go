package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/service"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

// ThreatHandler handles threat and indicator endpoints.
type ThreatHandler struct {
	svc *service.ThreatService
}

// NewThreatHandler creates a new ThreatHandler.
func NewThreatHandler(svc *service.ThreatService) *ThreatHandler {
	return &ThreatHandler{svc: svc}
}

func (h *ThreatHandler) CreateThreat(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateThreatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.CreateThreat(r.Context(), tenantID, userID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *ThreatHandler) ListThreats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.ListThreats(r.Context(), tenantID, parseThreatListParams(r), actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *ThreatHandler) GetThreat(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	item, err := h.svc.GetThreat(r.Context(), tenantID, threatID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "GET_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatHandler) UpdateThreat(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.UpdateThreatRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.UpdateThreat(r.Context(), tenantID, threatID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatHandler) DeleteThreat(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.svc.DeleteThreat(r.Context(), tenantID, threatID, actorFromRequest(r)); err != nil {
		writeError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *ThreatHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ThreatStatusUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateThreatStatus(r.Context(), tenantID, threatID, actorFromRequest(r), req.Status)
	if err != nil {
		writeError(w, http.StatusBadRequest, "STATUS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatHandler) ListIndicatorsForThreat(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.ListThreatIndicators(r.Context(), tenantID, threatID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INDICATORS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *ThreatHandler) AddIndicatorToThreat(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.ThreatIndicatorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	existed, item, err := h.svc.AddThreatIndicator(r.Context(), tenantID, threatID, userID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INDICATOR_ADD_FAILED", err.Error(), nil)
		return
	}
	if existed {
		writeJSON(w, http.StatusOK, envelope{"data": item, "existed": true})
	} else {
		writeJSON(w, http.StatusCreated, envelope{"data": item})
	}
}

func (h *ThreatHandler) UpdateIndicatorStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	indicatorID, ok := parseUUID(w, chi.URLParam(r, "indicatorId"))
	if !ok {
		return
	}
	var req dto.ThreatIndicatorStatusUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateIndicatorStatus(r.Context(), tenantID, indicatorID, actorFromRequest(r), req.Active)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INDICATOR_STATUS_FAILED", err.Error(), nil)
		return
	}
	item.Normalize()
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.ThreatStats(r.Context(), tenantID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), nil)
		return
	}
	stats.Normalize()
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

func (h *ThreatHandler) Trend(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := parseDashboardTrendParams(r)
	items, err := h.svc.ThreatTrend(r.Context(), tenantID, actorFromRequest(r), params.Days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TREND_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *ThreatHandler) RelatedAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.ListThreatAlerts(r.Context(), tenantID, threatID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "ALERTS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *ThreatHandler) Timeline(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	threatID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	items, err := h.svc.ListThreatTimeline(r.Context(), tenantID, threatID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TIMELINE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *ThreatHandler) CheckIndicators(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.IndicatorCheckRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := h.svc.CheckIndicators(r.Context(), tenantID, actorFromRequest(r), req.Values)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CHECK_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *ThreatHandler) BulkImportIndicators(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.IndicatorBulkImportRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	count, err := h.svc.BulkImport(r.Context(), tenantID, userID, actorFromRequest(r), req.Payload, req.Source, req.ConflictMode)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BULK_IMPORT_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": map[string]any{"count": count}})
}

func (h *ThreatHandler) BatchCreateIndicators(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.IndicatorBatchRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	result, err := h.svc.BatchCreateIndicators(r.Context(), tenantID, userID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BATCH_CREATE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": result})
}

func (h *ThreatHandler) ListIndicators(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params, err := parseIndicatorListParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	result, err := h.svc.ListIndicators(r.Context(), tenantID, params, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error(), nil)
		return
	}
	result.Normalize()
	writeJSON(w, http.StatusOK, result)
}

func (h *ThreatHandler) IndicatorStats(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.svc.IndicatorStats(r.Context(), tenantID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "STATS_FAILED", err.Error(), nil)
		return
	}
	stats.Normalize()
	writeJSON(w, http.StatusOK, envelope{"data": stats})
}

func (h *ThreatHandler) CreateIndicator(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.StandaloneIndicatorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.CreateIndicator(r.Context(), tenantID, userID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CREATE_FAILED", err.Error(), nil)
		return
	}
	item.Normalize()
	writeJSON(w, http.StatusCreated, envelope{"data": item})
}

func (h *ThreatHandler) GetIndicator(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	indicatorID, ok := parseUUID(w, chi.URLParam(r, "indicatorId"))
	if !ok {
		return
	}
	item, err := h.svc.GetIndicator(r.Context(), tenantID, indicatorID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "GET_FAILED", err.Error(), nil)
		return
	}
	item.Normalize()
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatHandler) UpdateIndicator(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	indicatorID, ok := parseUUID(w, chi.URLParam(r, "indicatorId"))
	if !ok {
		return
	}
	var req dto.StandaloneIndicatorRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if fieldErrs := pkgvalidator.Validate(req); fieldErrs != nil {
		writeValidationError(w, fieldErrs)
		return
	}
	item, err := h.svc.UpdateIndicator(r.Context(), tenantID, indicatorID, actorFromRequest(r), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error(), nil)
		return
	}
	item.Normalize()
	writeJSON(w, http.StatusOK, envelope{"data": item})
}

func (h *ThreatHandler) DeleteIndicator(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	indicatorID, ok := parseUUID(w, chi.URLParam(r, "indicatorId"))
	if !ok {
		return
	}
	if err := h.svc.DeleteIndicator(r.Context(), tenantID, indicatorID, actorFromRequest(r)); err != nil {
		writeError(w, http.StatusBadRequest, "DELETE_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": map[string]any{"deleted": true}})
}

func (h *ThreatHandler) IndicatorEnrichment(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	indicatorID, ok := parseUUID(w, chi.URLParam(r, "indicatorId"))
	if !ok {
		return
	}
	data, err := h.svc.IndicatorEnrichment(r.Context(), tenantID, indicatorID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "ENRICHMENT_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": data})
}

func (h *ThreatHandler) IndicatorMatches(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	indicatorID, ok := parseUUID(w, chi.URLParam(r, "indicatorId"))
	if !ok {
		return
	}
	data, err := h.svc.IndicatorMatches(r.Context(), tenantID, indicatorID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "MATCHES_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": data})
}
