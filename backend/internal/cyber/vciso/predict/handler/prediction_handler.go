package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	predictengine "github.com/clario360/platform/internal/cyber/vciso/predict/engine"
	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
	"github.com/clario360/platform/internal/suiteapi"
)

type PredictionHandler struct {
	engine *predictengine.ForecastEngine
	logger zerolog.Logger
}

func NewPredictionHandler(engine *predictengine.ForecastEngine, logger zerolog.Logger) *PredictionHandler {
	return &PredictionHandler{
		engine: engine,
		logger: logger.With().Str("component", "vciso_predict_handler").Logger(),
	}
}

func (h *PredictionHandler) Forecast(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	forecastType := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("forecast_type")))
	horizon := parseHorizon(r.URL.Query().Get("time_horizon"), 7)
	switch forecastType {
	case "", "alert_volume":
		response, err := h.engine.ForecastAlertVolume(r.Context(), tenantID, horizon)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		suiteapi.WriteJSON(w, http.StatusOK, response)
	case "technique_trend":
		response, err := h.engine.PredictTechniqueTrends(r.Context(), tenantID, horizon)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		suiteapi.WriteJSON(w, http.StatusOK, response)
	case "campaign_detection":
		response, err := h.engine.DetectCampaigns(r.Context(), tenantID, horizon)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		suiteapi.WriteJSON(w, http.StatusOK, response)
	default:
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "unsupported forecast_type", nil)
	}
}

func (h *PredictionHandler) Assets(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	limit := parseInt(r.URL.Query().Get("limit"), 10)
	assetType := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("asset_type")))
	response, err := h.engine.PredictAssetRisk(r.Context(), tenantID, limit, assetType)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, response)
}

func (h *PredictionHandler) Vulnerabilities(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	limit := parseInt(r.URL.Query().Get("limit"), 20)
	minProbability := parseFloat(r.URL.Query().Get("min_probability"), 0.5)
	response, err := h.engine.PredictVulnerabilityPriority(r.Context(), tenantID, limit, minProbability)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, response)
}

func (h *PredictionHandler) Techniques(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	response, err := h.engine.PredictTechniqueTrends(r.Context(), tenantID, parseHorizon(r.URL.Query().Get("time_horizon"), 30))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, response)
}

func (h *PredictionHandler) InsiderThreats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	response, err := h.engine.ForecastInsiderThreats(
		r.Context(),
		tenantID,
		parseHorizon(r.URL.Query().Get("time_horizon"), 30),
		parseInt(r.URL.Query().Get("threshold"), 70),
	)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, response)
}

func (h *PredictionHandler) Campaigns(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	response, err := h.engine.DetectCampaigns(r.Context(), tenantID, parseHorizon(r.URL.Query().Get("time_horizon"), 30))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, response)
}

func (h *PredictionHandler) Accuracy(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return
	}
	dashboard, err := h.engine.AccuracyDashboard(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, map[string]any{
		"generated_at": time.Now().UTC(),
		"dashboard":    dashboard,
	})
}

func (h *PredictionHandler) Retrain(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	modelType := predictmodel.PredictionType(strings.TrimSpace(chi.URLParam(r, "model_type")))
	response, err := h.engine.Retrain(r.Context(), tenantID, modelType, "manual")
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusAccepted, response)
}

func (h *PredictionHandler) requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context", nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

func (h *PredictionHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantID, ok := h.requireTenant(w, r)
	if !ok {
		return uuid.Nil, false
	}
	ctx := r.Context()
	allowed := false
	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		allowed = auth.HasAnyPermission(claims.Roles, auth.PermAdminAll, auth.PermCyberWrite)
	}
	if !allowed {
		if user := auth.UserFromContext(ctx); user != nil {
			allowed = auth.HasAnyPermission(user.Roles, auth.PermAdminAll, auth.PermCyberWrite)
		}
	}
	if !allowed {
		suiteapi.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "vciso predictive admin permission required", nil)
		return uuid.Nil, false
	}
	return tenantID, true
}

func (h *PredictionHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error().Err(err).Msg("vciso predictive request failed")
	suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", sanitizeError(err.Error()), nil)
}

func parseHorizon(value string, fallback int) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "7_days", "7":
		return 7
	case "30_days", "30":
		return 30
	case "90_days", "90":
		return 90
	default:
		return fallback
	}
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseFloat(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func sanitizeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "request failed"
	}
	if len(value) > 240 {
		return value[:240] + "..."
	}
	return value
}
