package visus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/aggregator"
	visusalert "github.com/clario360/platform/internal/visus/alert"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type CTIAlertEvaluator struct {
	ctiClient     *CTIClient
	cache         *CTICache
	tokenProvider *aggregator.ServiceTokenProvider
	alerts        *visusalert.Generator
	logger        zerolog.Logger
	now           func() time.Time
}

func NewCTIAlertEvaluator(
	ctiClient *CTIClient,
	cache *CTICache,
	tokenProvider *aggregator.ServiceTokenProvider,
	alerts *visusalert.Generator,
	logger zerolog.Logger,
) *CTIAlertEvaluator {
	return &CTIAlertEvaluator{
		ctiClient:     ctiClient,
		cache:         cache,
		tokenProvider: tokenProvider,
		alerts:        alerts,
		logger:        logger.With().Str("component", "visus_cti_alert_evaluator").Logger(),
		now:           func() time.Time { return time.Now().UTC() },
	}
}

func (e *CTIAlertEvaluator) EvaluateAlerts(ctx context.Context, tenantID string, authToken string) ([]model.ExecutiveAlert, error) {
	snapshot, err := e.getSnapshot(ctx, tenantID, authToken)
	if err != nil {
		return nil, err
	}

	alerts := make([]model.ExecutiveAlert, 0, 6)

	if snapshot.RiskScoreOverall > 80 {
		alerts = append(alerts, e.newAlert(
			tenantID,
			"CTI Risk Score Critical",
			fmt.Sprintf("Cyber threat risk score is %.1f (threshold: 80). %d critical campaigns active.", snapshot.RiskScoreOverall, snapshot.CriticalCampaignsCount),
			model.AlertCategoryRisk,
			model.AlertSeverityCritical,
			"/cyber/cti",
			"risk_score_critical",
			map[string]any{
				"risk_score":         snapshot.RiskScoreOverall,
				"critical_campaigns": snapshot.CriticalCampaignsCount,
				"threshold":          80,
			},
		))
	}

	if snapshot.TrendDirection == "increasing" && snapshot.TrendPercentage > 50 {
		alerts = append(alerts, e.newAlert(
			tenantID,
			"Threat Activity Surge Detected",
			fmt.Sprintf("Threat events increased %.1f%% in the last 24 hours (%d events). Investigate active campaigns.", snapshot.TrendPercentage, snapshot.TotalEvents24h),
			model.AlertCategoryRisk,
			model.AlertSeverityHigh,
			"/cyber/cti/events",
			"event_volume_spike",
			map[string]any{
				"trend_direction":  snapshot.TrendDirection,
				"trend_percentage": snapshot.TrendPercentage,
				"total_events_24h": snapshot.TotalEvents24h,
				"threshold":        50,
			},
		))
	}

	if snapshot.CriticalCampaignsCount > 3 {
		alerts = append(alerts, e.newAlert(
			tenantID,
			"Multiple Critical Campaigns Active",
			fmt.Sprintf("%d critical-severity campaigns are currently active. Executive review recommended.", snapshot.CriticalCampaignsCount),
			model.AlertCategoryRisk,
			model.AlertSeverityHigh,
			"/cyber/cti/campaigns?severity=critical",
			"critical_campaigns",
			map[string]any{
				"critical_campaigns": snapshot.CriticalCampaignsCount,
				"threshold":          3,
			},
		))
	}

	if snapshot.BrandAbuseCriticalCount > 5 {
		alerts = append(alerts, e.newAlert(
			tenantID,
			"Elevated Brand Abuse Risk",
			fmt.Sprintf("%d critical brand abuse incidents detected. Takedown actions may be required.", snapshot.BrandAbuseCriticalCount),
			model.AlertCategoryRisk,
			model.AlertSeverityHigh,
			"/cyber/cti/brand-abuse?risk_level=critical",
			"brand_abuse_critical",
			map[string]any{
				"critical_brand_abuse": snapshot.BrandAbuseCriticalCount,
				"threshold":            5,
			},
		))
	}

	if ctiFloat64(snapshot.MeanTimeToDetectHours) > 24 {
		alerts = append(alerts, e.newAlert(
			tenantID,
			"Slow Threat Detection",
			fmt.Sprintf("Mean time to detect threats is %.1f hours (target: <24h). Detection capabilities may need improvement.", ctiFloat64(snapshot.MeanTimeToDetectHours)),
			model.AlertCategoryOperational,
			model.AlertSeverityMedium,
			"/cyber/cti",
			"slow_detection",
			map[string]any{
				"mttd_hours": ctiFloat64(snapshot.MeanTimeToDetectHours),
				"threshold":  24,
			},
		))
	}

	if ctiFloat64(snapshot.MeanTimeToRespondHours) > 48 {
		alerts = append(alerts, e.newAlert(
			tenantID,
			"Slow Threat Response",
			fmt.Sprintf("Mean time to respond to threats is %.1f hours (target: <48h). Response procedures may need review.", ctiFloat64(snapshot.MeanTimeToRespondHours)),
			model.AlertCategoryOperational,
			model.AlertSeverityMedium,
			"/cyber/cti",
			"slow_response",
			map[string]any{
				"mttr_hours": ctiFloat64(snapshot.MeanTimeToRespondHours),
				"threshold":  48,
			},
		))
	}

	return alerts, nil
}

func (e *CTIAlertEvaluator) SyncTenant(ctx context.Context, tenantID uuid.UUID) error {
	if e.tokenProvider == nil {
		return fmt.Errorf("cti alert evaluator: token provider is required")
	}
	if e.alerts == nil {
		return fmt.Errorf("cti alert evaluator: alert generator is required")
	}

	authToken, err := e.tokenProvider.Token(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("cti alert evaluator: service token: %w", err)
	}

	alerts, err := e.EvaluateAlerts(ctx, tenantID.String(), authToken)
	if err != nil {
		return err
	}

	for idx := range alerts {
		alert := alerts[idx]
		if _, err := e.alerts.CreateAlert(ctx, &alert); err != nil {
			return fmt.Errorf("cti alert evaluator: create alert %q: %w", alert.Title, err)
		}
	}
	return nil
}

func (e *CTIAlertEvaluator) getSnapshot(ctx context.Context, tenantID string, authToken string) (*CTIExecutiveSnapshot, error) {
	cacheKey := fmt.Sprintf("visus:cti:%s:snapshot_for_alerts", tenantID)
	var dashboard CTIExecutiveDashboardResponse
	if err := e.cache.GetOrFetch(ctx, cacheKey, &dashboard, func() (interface{}, error) {
		return e.ctiClient.GetExecutiveDashboard(ctx, tenantID, authToken)
	}); err != nil {
		return nil, fmt.Errorf("cti alert evaluator: fetch snapshot: %w", err)
	}
	return &dashboard.Snapshot, nil
}

func (e *CTIAlertEvaluator) newAlert(
	tenantID string,
	title string,
	description string,
	category model.AlertCategory,
	severity model.AlertSeverity,
	actionURL string,
	dedupSuffix string,
	metadata map[string]any,
) model.ExecutiveAlert {
	tenantUUID, _ := uuid.Parse(tenantID)
	now := e.now()
	sourceEventType := "cti.snapshot"
	dedupKey := fmt.Sprintf("cti:%s:%s", tenantID, dedupSuffix)
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["action_url"] = actionURL
	metadata["cti_category"] = "cyber_threat_intelligence"

	return model.ExecutiveAlert{
		TenantID:          tenantUUID,
		Title:             title,
		Description:       description,
		Category:          category,
		Severity:          severity,
		SourceSuite:       "cyber",
		SourceType:        "cti_evaluator",
		SourceEntityID:    nil,
		SourceEventType:   &sourceEventType,
		Status:            model.AlertStatusNew,
		ViewedAt:          nil,
		ViewedBy:          nil,
		ActionedAt:        nil,
		ActionedBy:        nil,
		ActionNotes:       nil,
		DismissedAt:       nil,
		DismissedBy:       nil,
		DismissReason:     nil,
		DedupKey:          &dedupKey,
		OccurrenceCount:   1,
		FirstSeenAt:       now,
		LastSeenAt:        now,
		LinkedKPIID:       nil,
		LinkedDashboardID: nil,
		Metadata:          metadata,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

type CTIAlertScheduler struct {
	evaluator  *CTIAlertEvaluator
	dashboards *repository.DashboardRepository
	interval   time.Duration
	logger     zerolog.Logger
}

func NewCTIAlertScheduler(
	evaluator *CTIAlertEvaluator,
	dashboards *repository.DashboardRepository,
	interval time.Duration,
	logger zerolog.Logger,
) *CTIAlertScheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &CTIAlertScheduler{
		evaluator:  evaluator,
		dashboards: dashboards,
		interval:   interval,
		logger:     logger.With().Str("component", "visus_cti_alert_scheduler").Logger(),
	}
}

func (s *CTIAlertScheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		if err := s.runOnce(ctx); err != nil && ctx.Err() == nil {
			s.logger.Error().Err(err).Msg("cti alert scheduler iteration failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *CTIAlertScheduler) RunOnce(ctx context.Context) error {
	return s.runOnce(ctx)
}

func (s *CTIAlertScheduler) runOnce(ctx context.Context) error {
	if s.evaluator == nil || s.dashboards == nil {
		return nil
	}
	tenantIDs, err := s.dashboards.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	if len(tenantIDs) == 0 {
		return nil
	}

	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, tenantID := range tenantIDs {
		tenantID := tenantID
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := s.evaluator.SyncTenant(ctx, tenantID); err != nil {
				s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("cti alert evaluation failed")
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return firstErr
}
