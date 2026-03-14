package handler

import (
	"net/http"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	predictengine "github.com/clario360/platform/internal/cyber/vciso/predict/engine"
)

// AnalyticsHandler serves the /api/v1/cyber/analytics/* endpoints, delegating
// to the existing predictive engine and threat repository.
type AnalyticsHandler struct {
	forecastEngine *predictengine.ForecastEngine
	threatRepo     *repository.ThreatRepository
	logger         zerolog.Logger
}

// NewAnalyticsHandler creates a new AnalyticsHandler.
func NewAnalyticsHandler(
	forecastEngine *predictengine.ForecastEngine,
	threatRepo *repository.ThreatRepository,
	logger zerolog.Logger,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		forecastEngine: forecastEngine,
		threatRepo:     threatRepo,
		logger:         logger,
	}
}

// ThreatForecast handles GET /api/v1/cyber/analytics/threat-forecast.
func (h *AnalyticsHandler) ThreatForecast(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	horizon := 30
	if v := r.URL.Query().Get("horizon_days"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 && h <= 90 {
			horizon = h
		}
	}
	resp, err := h.forecastEngine.PredictTechniqueTrends(r.Context(), tenantID, horizon)
	if err != nil {
		h.logger.Error().Err(err).Msg("threat forecast failed")
		writeError(w, http.StatusInternalServerError, "FORECAST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": resp})
}

// AlertForecast handles GET /api/v1/cyber/analytics/alert-forecast.
func (h *AnalyticsHandler) AlertForecast(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	horizon := 30
	if v := r.URL.Query().Get("horizon_days"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 && h <= 90 {
			horizon = h
		}
	}
	resp, err := h.forecastEngine.ForecastAlertVolume(r.Context(), tenantID, horizon)
	if err != nil {
		h.logger.Error().Err(err).Msg("alert forecast failed")
		writeError(w, http.StatusInternalServerError, "FORECAST_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": resp})
}

// TechniqueTrends handles GET /api/v1/cyber/analytics/technique-trends.
func (h *AnalyticsHandler) TechniqueTrends(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	horizon := 30
	if v := r.URL.Query().Get("horizon_days"); v != "" {
		if h, err := strconv.Atoi(v); err == nil && h > 0 && h <= 90 {
			horizon = h
		}
	}
	resp, err := h.forecastEngine.PredictTechniqueTrends(r.Context(), tenantID, horizon)
	if err != nil {
		h.logger.Error().Err(err).Msg("technique trends failed")
		writeError(w, http.StatusInternalServerError, "TRENDS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": resp})
}

// Campaigns handles GET /api/v1/cyber/analytics/campaigns.
func (h *AnalyticsHandler) Campaigns(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	lookback := 30
	if v := r.URL.Query().Get("lookback_days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 && d <= 180 {
			lookback = d
		}
	}
	resp, err := h.forecastEngine.DetectCampaigns(r.Context(), tenantID, lookback)
	if err != nil {
		h.logger.Error().Err(err).Msg("campaign detection failed")
		writeError(w, http.StatusInternalServerError, "CAMPAIGNS_FAILED", err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": resp})
}

// Landscape handles GET /api/v1/cyber/analytics/landscape — aggregated threat landscape.
func (h *AnalyticsHandler) Landscape(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	stats, err := h.threatRepo.Stats(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("landscape failed")
		writeError(w, http.StatusInternalServerError, "LANDSCAPE_FAILED", err.Error(), nil)
		return
	}

	// Build landscape response from existing ThreatStats
	landscape := model.ThreatLandscape{
		ActiveThreatCount: stats.Active,
		TotalThreats:      stats.Total,
		IndicatorsTotal:   stats.IndicatorsTotal,
		ByType:            stats.ByType,
		BySeverity:        stats.BySeverity,
	}
	// Extract top tactic/technique names from stats
	if len(stats.ByType) > 0 {
		landscape.TopThreatType = stats.ByType[0].Name
	}

	writeJSON(w, http.StatusOK, envelope{"data": landscape})
}
