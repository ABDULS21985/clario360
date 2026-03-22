package handler

import (
	"errors"
	"net/http"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/repository"
)

type RiskHandler struct {
	svc riskService
}

func NewRiskHandler(svc riskService) *RiskHandler {
	return &RiskHandler{svc: svc}
}

func (h *RiskHandler) GetScore(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	score, err := h.svc.GetCurrentScore(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": score})
}

func (h *RiskHandler) GetTrend(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := parseRiskTrendParams(r)
	trend, err := h.svc.Trend(r.Context(), tenantID, params.Days)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": trend})
}

func (h *RiskHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	user := auth.UserFromContext(r.Context())
	if user == nil || !auth.HasAnyPermission(user.Roles, auth.PermCyberWrite, auth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		return
	}
	score, err := h.svc.Recalculate(r.Context(), tenantID, actorFromRequest(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": score})
}

func (h *RiskHandler) GetHeatmap(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	heatmap, err := h.svc.Heatmap(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": dto.HeatmapToResponse(heatmap)})
}

func (h *RiskHandler) GetTopRisks(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.TopRisks(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}

func (h *RiskHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.Recommendations(r.Context(), tenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": items})
}
