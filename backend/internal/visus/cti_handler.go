package visus

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/suiteapi"
)

type CTIWidgetHandler struct {
	ctiClient   *CTIClient
	cache       *CTICache
	logger      zerolog.Logger
	kpiProvider *CTIKPIProvider
}

func NewCTIWidgetHandler(ctiClient *CTIClient, cache *CTICache, logger zerolog.Logger) *CTIWidgetHandler {
	return &CTIWidgetHandler{
		ctiClient: ctiClient,
		cache:     cache,
		logger:    logger.With().Str("component", "visus_cti_widget_handler").Logger(),
	}
}

func (h *CTIWidgetHandler) WithKPIProvider(provider *CTIKPIProvider) *CTIWidgetHandler {
	h.kpiProvider = provider
	return h
}

func (h *CTIWidgetHandler) GetCTIOverview(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	h.ensureKPIs(r, tenantID)

	cacheKey := fmt.Sprintf("visus:cti:%s:overview", tenantID)
	var result CTIExecutiveDashboardResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		return h.ctiClient.GetExecutiveDashboard(r.Context(), tenantID, authToken)
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("failed to get CTI overview")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI overview", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) GetCTIThreatMap(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	period := normalizeCTIPeriod(r.URL.Query().Get("period"))
	cacheKey := fmt.Sprintf("visus:cti:%s:threat_map:%s", tenantID, period)

	var result CTIGlobalThreatMapResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		return h.ctiClient.GetGlobalThreatMap(r.Context(), tenantID, authToken, period)
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Str("period", period).Msg("failed to get CTI threat map")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI threat map", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) GetCTISectorOverview(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	period := normalizeCTIPeriod(r.URL.Query().Get("period"))
	cacheKey := fmt.Sprintf("visus:cti:%s:sectors:%s", tenantID, period)

	var result CTISectorThreatResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		return h.ctiClient.GetSectorThreatOverview(r.Context(), tenantID, authToken, period)
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Str("period", period).Msg("failed to get CTI sectors")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI sector overview", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) GetCTICampaigns(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	limit := parsePositiveLimit(r.URL.Query().Get("limit"), 5, 50)
	cacheKey := fmt.Sprintf("visus:cti:%s:campaigns:%d", tenantID, limit)

	var result CTICampaignListResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		return h.ctiClient.GetActiveCampaigns(r.Context(), tenantID, authToken, limit)
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Int("limit", limit).Msg("failed to get CTI campaigns")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI campaigns", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) GetCTIBrandAbuse(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	limit := parsePositiveLimit(r.URL.Query().Get("limit"), 5, 50)
	cacheKey := fmt.Sprintf("visus:cti:%s:brand_abuse:%d", tenantID, limit)

	var result CTIBrandAbuseListResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		return h.ctiClient.GetCriticalBrandAbuse(r.Context(), tenantID, authToken, limit)
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Int("limit", limit).Msg("failed to get CTI brand abuse")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI brand abuse", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) GetCTIActors(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	limit := parsePositiveLimit(r.URL.Query().Get("limit"), 5, 50)
	cacheKey := fmt.Sprintf("visus:cti:%s:actors:%d", tenantID, limit)

	var result CTIActorListResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		return h.ctiClient.GetThreatActors(r.Context(), tenantID, authToken, limit)
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Int("limit", limit).Msg("failed to get CTI actors")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI actors", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) GetCTIRiskScore(w http.ResponseWriter, r *http.Request) {
	tenantID, authToken, ok := h.tenantAndToken(w, r)
	if !ok {
		return
	}
	cacheKey := fmt.Sprintf("visus:cti:%s:risk_score", tenantID)

	var result CTIRiskScoreResponse
	if err := h.cache.GetOrFetch(r.Context(), cacheKey, &result, func() (interface{}, error) {
		dashboard, err := h.ctiClient.GetExecutiveDashboard(r.Context(), tenantID, authToken)
		if err != nil {
			return nil, err
		}
		snapshot := dashboard.Snapshot
		return &CTIRiskScoreResponse{
			RiskScore:       snapshot.RiskScoreOverall,
			TrendDirection:  snapshot.TrendDirection,
			TrendPercentage: snapshot.TrendPercentage,
			TotalEvents24h:  snapshot.TotalEvents24h,
			MTTDHours:       ctiFloat64(snapshot.MeanTimeToDetectHours),
			MTTRHours:       ctiFloat64(snapshot.MeanTimeToRespondHours),
			ComputedAt:      snapshot.ComputedAt,
		}, nil
	}); err != nil {
		h.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("failed to get CTI risk score")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "CTI_FETCH_FAILED", "Failed to fetch CTI risk score", nil)
		return
	}

	suiteapi.WriteData(w, http.StatusOK, result)
}

func (h *CTIWidgetHandler) tenantAndToken(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return "", "", false
	}
	authToken, err := extractBearerToken(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error(), nil)
		return "", "", false
	}
	return tenantID.String(), authToken, true
}

func (h *CTIWidgetHandler) ensureKPIs(r *http.Request, tenantID string) {
	if h.kpiProvider == nil {
		return
	}
	if err := h.kpiProvider.EnsureDefinitions(r.Context(), tenantID); err != nil {
		h.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("failed to ensure CTI KPI definitions")
	}
}

func extractBearerToken(r *http.Request) (string, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", fmt.Errorf("authorization header must use Bearer token")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", fmt.Errorf("authorization token is empty")
	}
	return token, nil
}

func normalizeCTIPeriod(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "24h", "7d", "30d", "90d":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "24h"
	}
}

func parsePositiveLimit(raw string, fallback int, max int) int {
	if fallback <= 0 {
		fallback = 10
	}
	if max < fallback {
		max = fallback
	}
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}
