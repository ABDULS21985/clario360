package aggregator

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type CrossSuiteAggregator struct {
	client        *SuiteClient
	kpiSnapshots  *repository.KPISnapshotRepository
	alerts        *repository.AlertRepository
	metrics       *visusmetrics.Metrics
}

func NewCrossSuiteAggregator(client *SuiteClient, kpiSnapshots *repository.KPISnapshotRepository, alerts *repository.AlertRepository, metrics *visusmetrics.Metrics) *CrossSuiteAggregator {
	return &CrossSuiteAggregator{
		client:       client,
		kpiSnapshots: kpiSnapshots,
		alerts:       alerts,
		metrics:      metrics,
	}
}

func (a *CrossSuiteAggregator) GetExecutiveView(ctx context.Context, tenantID uuid.UUID) (*model.ExecutiveView, error) {
	start := time.Now()
	view := &model.ExecutiveView{
		SuiteHealth: make(map[string]model.SuiteStatus, 4),
		CacheStatus: make(map[string]string, 4),
		GeneratedAt: time.Now().UTC(),
	}
	group, gctx := errgroup.WithContext(ctx)

	type suiteResult struct {
		name string
		meta FetchMetadata
	}
	results := make(chan suiteResult, 4)

	group.Go(func() error {
		var payload map[string]any
		meta := a.client.Fetch(gctx, "cyber", "/dashboard", tenantID, &payload)
		results <- suiteResult{name: "cyber", meta: meta}
		if meta.Status == "unavailable" {
			return nil
		}
		view.CyberSecurity = buildCyberSummary(payload)
		return nil
	})
	group.Go(func() error {
		var payload map[string]any
		meta := a.client.Fetch(gctx, "data", "/dashboard", tenantID, &payload)
		results <- suiteResult{name: "data", meta: meta}
		if meta.Status == "unavailable" {
			return nil
		}
		view.DataIntelligence = buildDataSummary(payload)
		return nil
	})
	group.Go(func() error {
		var payload map[string]any
		meta := a.client.Fetch(gctx, "acta", "/dashboard", tenantID, &payload)
		results <- suiteResult{name: "acta", meta: meta}
		if meta.Status == "unavailable" {
			return nil
		}
		view.Governance = buildGovernanceSummary(payload)
		return nil
	})
	group.Go(func() error {
		var payload map[string]any
		meta := a.client.Fetch(gctx, "lex", "/dashboard", tenantID, &payload)
		results <- suiteResult{name: "lex", meta: meta}
		if meta.Status == "unavailable" {
			return nil
		}
		view.Legal = buildLegalSummary(payload)
		return nil
	})
	group.Go(func() error {
		snapshots, err := a.kpiSnapshots.ListLatestByTenant(gctx, tenantID)
		if err == nil {
			view.KPIs = snapshots
		}
		return nil
	})
	group.Go(func() error {
		alerts, _, err := a.alerts.List(gctx, tenantID, repository.AlertListFilters{
			Status: []string{"new", "viewed", "acknowledged", "escalated"},
		}, 1, 10, "created_at", "desc")
		if err == nil {
			view.Alerts = alerts
		}
		return nil
	})

	_ = group.Wait()
	close(results)
	for result := range results {
		if result.name == "" {
			continue
		}
		view.CacheStatus[result.name] = result.meta.Status
		view.SuiteHealth[result.name] = model.SuiteStatus{
			Available:   result.meta.Status != "unavailable",
			LastSuccess: result.meta.LastSuccess,
			LatencyMS:   int(result.meta.Latency / time.Millisecond),
			Error:       errorString(result.meta.Error),
		}
	}
	if a.metrics != nil && a.metrics.ExecutiveViewDurationSeconds != nil {
		a.metrics.ExecutiveViewDurationSeconds.Observe(time.Since(start).Seconds())
	}
	return view, nil
}

func buildCyberSummary(payload map[string]any) *model.CyberSecuritySummary {
	summary := &model.CyberSecuritySummary{
		RiskScore:       mustValue(payload, "$.data.kpis.risk_score"),
		RiskGrade:       mustString(payload, "$.data.kpis.risk_grade"),
		OpenAlerts:      int(mustValue(payload, "$.data.kpis.open_alerts")),
		CriticalAlerts:  int(mustValue(payload, "$.data.kpis.critical_alerts")),
		MTTRHours:       mustValue(payload, "$.data.kpis.mttr_hours"),
		AssetsMonitored: int(mustValue(payload, "$.data.risk_score.context.total_assets")),
		Trend:           normalizeTrend(mustString(payload, "$.data.risk_score.trend")),
	}
	if cells, err := Extract(payload, "$.data.mitre_heatmap.cells"); err == nil {
		if list, ok := cells.([]any); ok {
			total := len(list)
			covered := 0
			for _, item := range list {
				if mapped, ok := item.(map[string]any); ok && truthy(mapped["has_detection"]) {
					covered++
				}
			}
			if total > 0 {
				summary.MITRECoverage = (float64(covered) / float64(total)) * 100
			}
		}
	}
	return summary
}

func buildDataSummary(payload map[string]any) *model.DataIntelligenceSummary {
	return &model.DataIntelligenceSummary{
		QualityScore:        mustValue(payload, "$.data.kpis.quality_score"),
		QualityGrade:        mustString(payload, "$.data.kpis.quality_grade"),
		ActivePipelines:     int(mustValue(payload, "$.data.kpis.active_pipelines")),
		FailedPipelines24h:  int(mustValue(payload, "$.data.kpis.failed_pipelines_24h")),
		PipelineSuccessRate: mustValue(payload, "$.data.pipeline_success_rate_30d"),
		OpenContradictions:  int(mustValue(payload, "$.data.kpis.open_contradictions")),
		DarkDataAssets:      int(mustValue(payload, "$.data.kpis.dark_data_assets")),
		Trend:               trendFromDelta(mustValue(payload, "$.data.kpis.quality_delta")),
	}
}

func buildGovernanceSummary(payload map[string]any) *model.GovernanceSummary {
	score := mustValue(payload, "$.data.kpis.compliance_score")
	return &model.GovernanceSummary{
		UpcomingMeetings:   int(mustValue(payload, "$.data.kpis.upcoming_meetings_30d")),
		OverdueActionItems: int(mustValue(payload, "$.data.kpis.overdue_action_items")),
		ComplianceScore:    score,
		ComplianceGrade:    gradeForPercent(score),
		OpenActionItems:    int(mustValue(payload, "$.data.kpis.open_action_items")),
		MinutesPending:     int(mustValue(payload, "$.data.kpis.minutes_pending_approval")),
		Trend:              trendFromCompliance(payload),
	}
}

func buildLegalSummary(payload map[string]any) *model.LegalSummary {
	summary := &model.LegalSummary{
		ActiveContracts:      int(mustValue(payload, "$.data.kpis.active_contracts")),
		TotalContractValue:   mustValue(payload, "$.data.kpis.total_active_value"),
		ExpiringIn30Days:     int(mustValue(payload, "$.data.kpis.expiring_in_30_days")),
		HighRiskContracts:    int(mustValue(payload, "$.data.kpis.high_risk_contracts")),
		OpenComplianceAlerts: int(mustValue(payload, "$.data.kpis.open_compliance_alerts")),
		PendingReview:        int(mustValue(payload, "$.data.kpis.pending_review")),
		Trend:                trendFromMonthlyActivity(payload),
	}
	if highRisk, err := Extract(payload, "$.data.high_risk_contracts"); err == nil {
		if items, ok := highRisk.([]any); ok && len(items) > 0 {
			total := 0.0
			count := 0
			for _, item := range items {
				mapped, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if score, err := toFloat64(mapped["risk_score"], "$.data.high_risk_contracts.risk_score"); err == nil {
					total += score
					count++
				}
			}
			if count > 0 {
				summary.AvgRiskScore = total / float64(count)
			}
		}
	}
	return summary
}

func mustValue(payload map[string]any, path string) float64 {
	value, err := ExtractValue(payload, path)
	if err != nil {
		return 0
	}
	return value
}

func mustString(payload map[string]any, path string) string {
	value, err := Extract(payload, path)
	if err != nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func gradeForPercent(value float64) string {
	switch {
	case value >= 90:
		return "A"
	case value >= 80:
		return "B"
	case value >= 70:
		return "C"
	case value >= 60:
		return "D"
	default:
		return "F"
	}
}

func normalizeTrend(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "improving", "stable", "degrading":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "stable"
	}
}

func trendFromDelta(delta float64) string {
	switch {
	case delta > 1:
		return "improving"
	case delta < -1:
		return "degrading"
	default:
		return "stable"
	}
}

func trendFromCompliance(payload map[string]any) string {
	score := mustValue(payload, "$.data.kpis.compliance_score")
	if score >= 90 {
		return "improving"
	}
	if score < 75 {
		return "degrading"
	}
	return "stable"
}

func trendFromMonthlyActivity(payload map[string]any) string {
	activity, err := Extract(payload, "$.data.monthly_activity")
	if err != nil {
		return "stable"
	}
	items, ok := activity.([]any)
	if !ok || len(items) < 2 {
		return "stable"
	}
	curr, _ := items[len(items)-1].(map[string]any)
	prev, _ := items[len(items)-2].(map[string]any)
	if must, err := toFloat64(curr["activated"], "$.data.monthly_activity.activated"); err == nil {
		prevVal, _ := toFloat64(prev["activated"], "$.data.monthly_activity.activated")
		if must > prevVal {
			return "improving"
		}
		if must < prevVal {
			return "degrading"
		}
	}
	return "stable"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
