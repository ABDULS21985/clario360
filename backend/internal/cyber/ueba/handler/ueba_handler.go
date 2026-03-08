package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	cyberrepo "github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/ueba/dto"
	uebasvc "github.com/clario360/platform/internal/cyber/ueba/service"
)

type UEBAHandler struct {
	svc *uebasvc.UEBAService
}

func NewUEBAHandler(svc *uebasvc.UEBAService) *UEBAHandler {
	return &UEBAHandler{svc: svc}
}

func (h *UEBAHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := &dto.ProfileListParams{}
	params.Page, params.PerPage = parsePageParams(r, 25)
	params.Status = dto.NormalizeStatus(r.URL.Query().Get("status"))
	result, err := h.svc.ListProfiles(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *UEBAHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetProfile(r.Context(), tenantID, chi.URLParam(r, "entityId"))
	if err != nil {
		if errors.Is(err, cyberrepo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "GET_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *UEBAHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	page, perPage := parsePageParams(r, 50)
	result, err := h.svc.GetTimeline(r.Context(), tenantID, chi.URLParam(r, "entityId"), page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "TIMELINE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *UEBAHandler) GetHeatmap(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			days = value
		}
	}
	result, err := h.svc.GetHeatmap(r.Context(), tenantID, chi.URLParam(r, "entityId"), days)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "HEATMAP_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *UEBAHandler) UpdateProfileStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.ProfileStatusUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	profile, err := h.svc.UpdateProfileStatus(r.Context(), tenantID, chi.URLParam(r, "entityId"), &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "STATUS_UPDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": profile})
}

func (h *UEBAHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := &dto.AlertListParams{
		EntityID: r.URL.Query().Get("entity_id"),
		Status:   r.URL.Query().Get("status"),
	}
	params.Page, params.PerPage = parsePageParams(r, 25)
	result, err := h.svc.ListAlerts(r.Context(), tenantID, params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "LIST_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *UEBAHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	alert, err := h.svc.GetAlert(r.Context(), tenantID, alertID)
	if err != nil {
		if errors.Is(err, cyberrepo.ErrNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "alert not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "GET_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *UEBAHandler) UpdateAlertStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.AlertStatusUpdateRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	alert, err := h.svc.UpdateAlertStatus(r.Context(), tenantID, alertID, &userID, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "STATUS_UPDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *UEBAHandler) MarkFalsePositive(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	alertID, ok := parseUUID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req dto.FalsePositiveRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	alert, err := h.svc.MarkFalsePositive(r.Context(), tenantID, alertID, &userID, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "FALSE_POSITIVE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": alert})
}

func (h *UEBAHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetDashboard(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DASHBOARD_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *UEBAHandler) GetRiskRanking(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	items, err := h.svc.GetRiskRanking(r.Context(), tenantID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "RANKING_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *UEBAHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := requireTenantAndUser(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": h.svc.GetConfig()})
}

func (h *UEBAHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := requireTenantAndUser(w, r); !ok {
		return
	}
	var req dto.UEBAConfigDTO
	if !decodeJSON(w, r, &req) {
		return
	}
	cfg, err := h.svc.UpdateConfig(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "CONFIG_UPDATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": cfg})
}
