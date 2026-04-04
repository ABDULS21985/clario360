package visus

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/aggregator"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type KPI struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Value        float64 `json:"value"`
	DisplayValue *string `json:"display_value,omitempty"`
	Unit         string  `json:"unit"`
	Category     string  `json:"category"`
	SubCategory  string  `json:"sub_category"`
	Trend        string  `json:"trend"`
	TrendPct     float64 `json:"trend_pct"`
	UpdatedAt    string  `json:"updated_at"`
}

type CTIKPIProvider struct {
	ctiClient     *CTIClient
	cache         *CTICache
	tokenProvider *aggregator.ServiceTokenProvider
	kpis          *repository.KPIRepository
	serviceUserID uuid.UUID
	logger        zerolog.Logger
}

func NewCTIKPIProvider(
	ctiClient *CTIClient,
	cache *CTICache,
	tokenProvider *aggregator.ServiceTokenProvider,
	kpis *repository.KPIRepository,
	serviceUserID uuid.UUID,
	logger zerolog.Logger,
) *CTIKPIProvider {
	return &CTIKPIProvider{
		ctiClient:     ctiClient,
		cache:         cache,
		tokenProvider: tokenProvider,
		kpis:          kpis,
		serviceUserID: serviceUserID,
		logger:        logger.With().Str("component", "visus_cti_kpi_provider").Logger(),
	}
}

func (p *CTIKPIProvider) GetKPIs(ctx context.Context, tenantID string, authToken string) ([]KPI, error) {
	cacheKey := fmt.Sprintf("visus:cti:%s:kpis", tenantID)
	var dashboard CTIExecutiveDashboardResponse
	if err := p.cache.GetOrFetch(ctx, cacheKey, &dashboard, func() (interface{}, error) {
		return p.ctiClient.GetExecutiveDashboard(ctx, tenantID, authToken)
	}); err != nil {
		return nil, fmt.Errorf("cti kpi provider: %w", err)
	}

	snap := dashboard.Snapshot
	topOrigin := "—"
	if snap.TopThreatOriginCountry != nil && strings.TrimSpace(*snap.TopThreatOriginCountry) != "" {
		topOrigin = *snap.TopThreatOriginCountry
	}

	kpis := []KPI{
		{
			ID:          "cti.risk_score",
			Name:        "CTI Risk Score",
			Value:       snap.RiskScoreOverall,
			Unit:        "score",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.events_24h",
			Name:        "Threat Events (24h)",
			Value:       float64(snap.TotalEvents24h),
			Unit:        "count",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.events_trend",
			Name:        "Threat Trend",
			Value:       snap.TrendPercentage,
			Unit:        "percentage",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.active_campaigns",
			Name:        "Active Campaigns",
			Value:       float64(snap.ActiveCampaignsCount),
			Unit:        "count",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.critical_campaigns",
			Name:        "Critical Campaigns",
			Value:       float64(snap.CriticalCampaignsCount),
			Unit:        "count",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.total_iocs",
			Name:        "Indicators of Compromise",
			Value:       float64(snap.TotalIOCs),
			Unit:        "count",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.brand_abuse_critical",
			Name:        "Brand Abuse (Critical)",
			Value:       float64(snap.BrandAbuseCriticalCount),
			Unit:        "count",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.mttd",
			Name:        "Mean Time to Detect",
			Value:       ctiFloat64(snap.MeanTimeToDetectHours),
			Unit:        "hours",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:          "cti.mttr",
			Name:        "Mean Time to Respond",
			Value:       ctiFloat64(snap.MeanTimeToRespondHours),
			Unit:        "hours",
			Category:    "cybersecurity",
			SubCategory: "threat_intelligence",
			Trend:       snap.TrendDirection,
			TrendPct:    snap.TrendPercentage,
			UpdatedAt:   snap.ComputedAt,
		},
		{
			ID:           "cti.top_origin",
			Name:         "Top Threat Origin",
			Value:        0,
			DisplayValue: &topOrigin,
			Unit:         "label",
			Category:     "cybersecurity",
			SubCategory:  "threat_intelligence",
			Trend:        snap.TrendDirection,
			TrendPct:     snap.TrendPercentage,
			UpdatedAt:    snap.ComputedAt,
		},
	}

	return kpis, nil
}

func (p *CTIKPIProvider) EnsureDefinitions(ctx context.Context, tenantID string) error {
	if p.kpis == nil {
		return nil
	}
	tenantUUID, err := uuid.Parse(strings.TrimSpace(tenantID))
	if err != nil {
		return fmt.Errorf("cti kpi provider: parse tenant id: %w", err)
	}

	existing, _, err := p.kpis.List(ctx, tenantUUID, 1, 500, "name", "asc", "", string(model.KPISuiteCyber), nil)
	if err != nil {
		return fmt.Errorf("cti kpi provider: list existing definitions: %w", err)
	}
	existingByName := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		existingByName[item.Name] = struct{}{}
	}

	for _, definition := range p.defaultDefinitions(tenantUUID) {
		if _, ok := existingByName[definition.Name]; ok {
			continue
		}
		if _, err := p.kpis.Create(ctx, &definition); err != nil {
			if err == repository.ErrConflict {
				continue
			}
			return fmt.Errorf("cti kpi provider: create %q: %w", definition.Name, err)
		}
	}
	return nil
}

func (p *CTIKPIProvider) defaultDefinitions(tenantID uuid.UUID) []model.KPIDefinition {
	return []model.KPIDefinition{
		p.newDefinition(tenantID, "CTI Risk Score", "/cti/dashboard/executive", "$.data.snapshot.risk_score_overall", model.KPIUnitScore, model.KPIDirectionLowerIsBetter, 60, 80),
		p.newDefinition(tenantID, "Threat Events (24h)", "/cti/dashboard/executive", "$.data.snapshot.total_events_24h", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 250, 500),
		p.newDefinition(tenantID, "Threat Trend (%)", "/cti/dashboard/executive", "$.data.snapshot.trend_percentage", model.KPIUnitPercentage, model.KPIDirectionLowerIsBetter, 25, 50),
		p.newDefinition(tenantID, "Active Campaigns", "/cti/dashboard/executive", "$.data.snapshot.active_campaigns_count", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 3, 5),
		p.newDefinition(tenantID, "Critical Campaigns", "/cti/dashboard/executive", "$.data.snapshot.critical_campaigns_count", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 2, 4),
		p.newDefinition(tenantID, "Indicators of Compromise", "/cti/dashboard/executive", "$.data.snapshot.total_iocs", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 500, 1000),
		p.newDefinition(tenantID, "Brand Abuse (Critical)", "/cti/dashboard/executive", "$.data.snapshot.brand_abuse_critical_count", model.KPIUnitCount, model.KPIDirectionLowerIsBetter, 2, 5),
		p.newDefinition(tenantID, "Mean Time to Detect", "/cti/dashboard/executive", "$.data.snapshot.mean_time_to_detect_hours", model.KPIUnitHours, model.KPIDirectionLowerIsBetter, 24, 48),
		p.newDefinition(tenantID, "Mean Time to Respond", "/cti/dashboard/executive", "$.data.snapshot.mean_time_to_respond_hours", model.KPIUnitHours, model.KPIDirectionLowerIsBetter, 48, 72),
	}
}

func (p *CTIKPIProvider) newDefinition(
	tenantID uuid.UUID,
	name string,
	endpoint string,
	valuePath string,
	unit model.KPIUnit,
	direction model.KPIDirection,
	warning float64,
	critical float64,
) model.KPIDefinition {
	description := name
	calculationWindow := "24h"
	return model.KPIDefinition{
		TenantID:          tenantID,
		Name:              name,
		Description:       description,
		Category:          model.KPICategorySecurity,
		Suite:             model.KPISuiteCyber,
		QueryEndpoint:     endpoint,
		QueryParams:       map[string]any{},
		ValuePath:         valuePath,
		Unit:              unit,
		WarningThreshold:  &warning,
		CriticalThreshold: &critical,
		Direction:         direction,
		CalculationType:   model.KPICalcDirect,
		CalculationWindow: &calculationWindow,
		SnapshotFrequency: model.KPIFrequencyHour,
		Enabled:           true,
		IsDefault:         true,
		Tags:              []string{"default", "cti", "threat_intelligence"},
		CreatedBy:         p.serviceUserID,
	}
}

func ctiFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
