package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/aggregator"
	"github.com/clario360/platform/internal/visus/dto"
	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/report"
)

type ExecutiveService struct {
	aggregator *aggregator.CrossSuiteAggregator
	publisher  Publisher
	metrics    *visusmetrics.Metrics
	logger     zerolog.Logger
}

func NewExecutiveService(aggregator *aggregator.CrossSuiteAggregator, publisher Publisher, metrics *visusmetrics.Metrics, logger zerolog.Logger) *ExecutiveService {
	return &ExecutiveService{
		aggregator: aggregator,
		publisher:  publisher,
		metrics:    metrics,
		logger:     logger.With().Str("component", "visus_executive_service").Logger(),
	}
}

func (s *ExecutiveService) GetView(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID) (*model.ExecutiveView, error) {
	view, err := s.aggregator.GetExecutiveView(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if userID != nil {
		_ = publishEvent(ctx, s.publisher, tenantID, "visus.executive.viewed", map[string]any{
			"user_id":   *userID,
			"tenant_id": tenantID,
			"viewed_at": time.Now().UTC(),
		})
	}
	return view, nil
}

func (s *ExecutiveService) GetSummary(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID) (*dto.ExecutiveSummaryResponse, error) {
	view, err := s.GetView(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	sections := map[string]interface{}{
		"security_posture": map[string]any{
			"available":       view.CyberSecurity != nil,
			"risk_score":      valueFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) float64 { return item.RiskScore }),
			"grade":           stringFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) string { return item.RiskGrade }),
			"prev_risk_score": valueFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) float64 { return item.RiskScore }),
			"trend_word":      stringFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) string { return item.Trend }),
			"open_alerts":     valueFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) float64 { return float64(item.OpenAlerts) }),
			"critical_alerts": valueFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) float64 { return float64(item.CriticalAlerts) }),
			"mttr_hours":      valueFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) float64 { return item.MTTRHours }),
			"coverage":        valueFromCyber(view.CyberSecurity, func(item *model.CyberSecuritySummary) float64 { return item.MITRECoverage }),
		},
		"data_intelligence": map[string]any{
			"available":           view.DataIntelligence != nil,
			"quality_score":       valueFromData(view.DataIntelligence, func(item *model.DataIntelligenceSummary) float64 { return item.QualityScore }),
			"quality_grade":       stringFromData(view.DataIntelligence, func(item *model.DataIntelligenceSummary) string { return item.QualityGrade }),
			"success_rate":        valueFromData(view.DataIntelligence, func(item *model.DataIntelligenceSummary) float64 { return item.PipelineSuccessRate }),
			"failed_count":        valueFromData(view.DataIntelligence, func(item *model.DataIntelligenceSummary) float64 { return float64(item.FailedPipelines24h) }),
			"contradiction_count": valueFromData(view.DataIntelligence, func(item *model.DataIntelligenceSummary) float64 { return float64(item.OpenContradictions) }),
		},
		"governance": map[string]any{
			"available":        view.Governance != nil,
			"compliance_score": valueFromGov(view.Governance, func(item *model.GovernanceSummary) float64 { return item.ComplianceScore }),
			"meeting_count":    valueFromGov(view.Governance, func(item *model.GovernanceSummary) float64 { return float64(item.UpcomingMeetings) }),
			"overdue_count":    valueFromGov(view.Governance, func(item *model.GovernanceSummary) float64 { return float64(item.OverdueActionItems) }),
			"minutes_pending":  valueFromGov(view.Governance, func(item *model.GovernanceSummary) float64 { return float64(item.MinutesPending) }),
		},
		"legal": map[string]any{
			"available":        view.Legal != nil,
			"active_contracts": valueFromLegal(view.Legal, func(item *model.LegalSummary) float64 { return float64(item.ActiveContracts) }),
			"value":            valueFromLegal(view.Legal, func(item *model.LegalSummary) float64 { return item.TotalContractValue }),
			"expiring_count":   valueFromLegal(view.Legal, func(item *model.LegalSummary) float64 { return float64(item.ExpiringIn30Days) }),
			"high_risk_count":  valueFromLegal(view.Legal, func(item *model.LegalSummary) float64 { return float64(item.HighRiskContracts) }),
		},
		"recommendations": map[string]any{
			"items": []string{
				"Review open executive alerts and critical KPI breaches across all suites.",
			},
		},
	}
	narrative := report.GenerateNarrative(sections, [2]time.Time{time.Now().UTC().AddDate(0, 0, -30), time.Now().UTC()})
	return &dto.ExecutiveSummaryResponse{Narrative: narrative}, nil
}

func (s *ExecutiveService) Health(ctx context.Context, tenantID uuid.UUID) (map[string]model.SuiteStatus, error) {
	view, err := s.aggregator.GetExecutiveView(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return view.SuiteHealth, nil
}

func valueFromCyber(item *model.CyberSecuritySummary, fn func(*model.CyberSecuritySummary) float64) float64 {
	if item == nil {
		return 0
	}
	return fn(item)
}

func stringFromCyber(item *model.CyberSecuritySummary, fn func(*model.CyberSecuritySummary) string) string {
	if item == nil {
		return ""
	}
	return fn(item)
}

func valueFromData(item *model.DataIntelligenceSummary, fn func(*model.DataIntelligenceSummary) float64) float64 {
	if item == nil {
		return 0
	}
	return fn(item)
}

func stringFromData(item *model.DataIntelligenceSummary, fn func(*model.DataIntelligenceSummary) string) string {
	if item == nil {
		return ""
	}
	return fn(item)
}

func valueFromGov(item *model.GovernanceSummary, fn func(*model.GovernanceSummary) float64) float64 {
	if item == nil {
		return 0
	}
	return fn(item)
}

func valueFromLegal(item *model.LegalSummary, fn func(*model.LegalSummary) float64) float64 {
	if item == nil {
		return 0
	}
	return fn(item)
}
