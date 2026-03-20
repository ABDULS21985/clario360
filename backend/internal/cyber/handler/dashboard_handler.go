package handler

import (
	"net/http"

	"github.com/clario360/platform/internal/cyber/dto"
)

type DashboardHandler struct {
	svc dashboardService
}

func NewDashboardHandler(svc dashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	dashboard, err := h.svc.GetSOCDashboard(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": dashboard})
}

func (h *DashboardHandler) GetKPIs(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	kpis, err := h.svc.GetKPIs(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": kpis})
}

func (h *DashboardHandler) GetAlertsTimeline(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	timeline, err := h.svc.GetAlertTimeline(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": timeline})
}

func (h *DashboardHandler) GetSeverityDistribution(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	distribution, err := h.svc.GetSeverityDistribution(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": distribution})
}

func (h *DashboardHandler) GetMTTR(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	report, err := h.svc.GetMTTR(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": report})
}

func (h *DashboardHandler) GetAnalystWorkload(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	workload, err := h.svc.GetAnalystWorkload(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": workload})
}

func (h *DashboardHandler) GetTopAttackedAssets(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.GetTopAttackedAssets(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *DashboardHandler) GetMITREHeatmap(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	heatmap, err := h.svc.GetMITREHeatmap(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": heatmap})
}

func (h *DashboardHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	metrics, err := h.svc.GetMetrics(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": metrics})
}

func (h *DashboardHandler) GetTrends(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := dto.DashboardTrendParams{}
	params.SetDefaults()
	if v := r.URL.Query().Get("days"); v != "" {
		params = *parseDashboardTrendParams(r)
	}
	trends, err := h.svc.GetTrends(r.Context(), tenantID, params.Days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": trends})
}
