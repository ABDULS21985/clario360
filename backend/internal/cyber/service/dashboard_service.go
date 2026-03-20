package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	cyberdash "github.com/clario360/platform/internal/cyber/dashboard"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	cyberrisk "github.com/clario360/platform/internal/cyber/risk"
)

type DashboardService struct {
	cache        *cyberdash.Cache
	repo         *repository.DashboardRepository
	kpis         *cyberdash.KPICalculator
	timeline     *cyberdash.TimelineCalculator
	trends       *cyberdash.TrendCalculator
	mttr         *cyberdash.MTTRCalculator
	workload     *cyberdash.WorkloadCalculator
	mitreHeatmap *cyberdash.MITREHeatmapCalculator
	riskScorer   *cyberrisk.RiskScorer
	logger       zerolog.Logger
	metrics      *metrics.Metrics
}

func NewDashboardService(
	cache *cyberdash.Cache,
	repo *repository.DashboardRepository,
	kpis *cyberdash.KPICalculator,
	timeline *cyberdash.TimelineCalculator,
	trends *cyberdash.TrendCalculator,
	mttr *cyberdash.MTTRCalculator,
	workload *cyberdash.WorkloadCalculator,
	mitreHeatmap *cyberdash.MITREHeatmapCalculator,
	riskScorer *cyberrisk.RiskScorer,
	m *metrics.Metrics,
	logger zerolog.Logger,
) *DashboardService {
	return &DashboardService{
		cache:        cache,
		repo:         repo,
		kpis:         kpis,
		timeline:     timeline,
		trends:       trends,
		mttr:         mttr,
		workload:     workload,
		mitreHeatmap: mitreHeatmap,
		riskScorer:   riskScorer,
		logger:       logger.With().Str("service", "dashboard").Logger(),
		metrics:      m,
	}
}

func (s *DashboardService) GetSOCDashboard(ctx context.Context, tenantID uuid.UUID) (*model.SOCDashboard, error) {
	start := time.Now()
	if cached, ok, err := s.cache.Get(ctx, tenantID); err == nil && ok {
		if s.metrics != nil && s.metrics.DashboardCacheHitTotal != nil {
			s.metrics.DashboardCacheHitTotal.Inc()
		}
		s.observeRequest("dashboard", start)
		return cached, nil
	}
	if s.metrics != nil && s.metrics.DashboardCacheMissTotal != nil {
		s.metrics.DashboardCacheMissTotal.Inc()
	}

	var (
		dashboard model.SOCDashboard
		mttr      *model.MTTRReport
		riskScore *model.OrganizationRiskScore
		failures  []string
		mu        sync.Mutex
	)

	run := func(group *errgroup.Group, name string, fn func() error) {
		group.Go(func() error {
			queryStart := time.Now()
			err := fn()
			s.observeQuery(name, queryStart)
			if err != nil {
				mu.Lock()
				failures = append(failures, name+": "+err.Error())
				mu.Unlock()
				s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Str("section", name).Msg("dashboard section failed")
			}
			return nil
		})
	}

	var group errgroup.Group
	run(&group, "kpis", func() error {
		kpis, err := s.kpis.Calculate(ctx, tenantID)
		if err == nil {
			dashboard.KPIs = kpis
		}
		return err
	})
	run(&group, "alert_timeline", func() error {
		timeline, err := s.timeline.AlertTimeline(ctx, tenantID, 24*time.Hour)
		if err == nil {
			dashboard.AlertTimeline = timeline
		}
		return err
	})
	run(&group, "severity_distribution", func() error {
		distribution, err := s.timeline.SeverityDistribution(ctx, tenantID)
		if err == nil {
			dashboard.SeverityDistribution = distribution
		}
		return err
	})
	run(&group, "alert_trend", func() error {
		trend, err := s.trends.AlertTrend(ctx, tenantID, 30)
		if err == nil {
			dashboard.AlertTrend = trend
		}
		return err
	})
	run(&group, "vulnerability_trend", func() error {
		trend, err := s.trends.VulnTrend(ctx, tenantID, 30)
		if err == nil {
			dashboard.VulnerabilityTrend = trend
		}
		return err
	})
	run(&group, "recent_alerts", func() error {
		items, err := s.repo.RecentAlerts(ctx, tenantID, 10)
		if err == nil {
			dashboard.RecentAlerts = items
		}
		return err
	})
	run(&group, "top_attacked_assets", func() error {
		items, err := s.repo.TopAttackedAssets(ctx, tenantID, 10)
		if err == nil {
			dashboard.TopAttackedAssets = items
		}
		return err
	})
	run(&group, "analyst_workload", func() error {
		items, err := s.workload.Calculate(ctx, tenantID)
		if err == nil {
			dashboard.AnalystWorkload = items
		}
		return err
	})
	run(&group, "mitre_heatmap", func() error {
		data, err := s.mitreHeatmap.Heatmap(ctx, tenantID, 90)
		if err == nil {
			dashboard.MITREHeatmap = data
		}
		return err
	})
	run(&group, "mttr", func() error {
		report, err := s.mttr.Calculate(ctx, tenantID)
		if err == nil {
			mttr = report
		}
		return err
	})
	run(&group, "risk_score", func() error {
		score, err := s.riskScorer.CalculateOrganizationRisk(ctx, tenantID)
		if err == nil {
			riskScore = score
		}
		return err
	})
	_ = group.Wait()

	if mttr != nil {
		dashboard.KPIs.MeanTimeToRespond = mttr.Overall.AvgResponseHours
		if mttr.Overall.AvgResolveHours != nil {
			dashboard.KPIs.MeanTimeToResolve = *mttr.Overall.AvgResolveHours
		}
	}
	if riskScore != nil {
		dashboard.RiskScore = riskScore
		dashboard.KPIs.RiskScore = riskScore.OverallScore
		dashboard.KPIs.RiskGrade = riskScore.Grade
	}

	dashboard.CalculatedAt = time.Now().UTC()
	if len(failures) > 0 {
		dashboard.PartialFailures = failures
		if s.metrics != nil && s.metrics.DashboardPartialFailureTotal != nil {
			s.metrics.DashboardPartialFailureTotal.Inc()
		}
	}
	_ = s.cache.Set(ctx, tenantID, &dashboard)
	s.observeRequest("dashboard", start)
	return &dashboard, nil
}

func (s *DashboardService) GetKPIs(ctx context.Context, tenantID uuid.UUID) (model.KPICards, error) {
	start := time.Now()
	kpis, err := s.kpis.Calculate(ctx, tenantID)
	if err != nil {
		return kpis, err
	}
	if mttr, err := s.mttr.Calculate(ctx, tenantID); err == nil {
		kpis.MeanTimeToRespond = mttr.Overall.AvgResponseHours
		if mttr.Overall.AvgResolveHours != nil {
			kpis.MeanTimeToResolve = *mttr.Overall.AvgResolveHours
		}
	}
	if riskScore, err := s.riskScorer.CalculateOrganizationRisk(ctx, tenantID); err == nil {
		kpis.RiskScore = riskScore.OverallScore
		kpis.RiskGrade = riskScore.Grade
	}
	s.observeRequest("dashboard_kpis", start)
	return kpis, nil
}

func (s *DashboardService) GetAlertTimeline(ctx context.Context, tenantID uuid.UUID) (model.AlertTimelineData, error) {
	start := time.Now()
	out, err := s.timeline.AlertTimeline(ctx, tenantID, 24*time.Hour)
	s.observeRequest("dashboard_alerts_timeline", start)
	return out, err
}

func (s *DashboardService) GetSeverityDistribution(ctx context.Context, tenantID uuid.UUID) (model.SeverityDistribution, error) {
	start := time.Now()
	out, err := s.timeline.SeverityDistribution(ctx, tenantID)
	s.observeRequest("dashboard_severity_distribution", start)
	return out, err
}

func (s *DashboardService) GetMTTR(ctx context.Context, tenantID uuid.UUID) (*model.MTTRReport, error) {
	start := time.Now()
	out, err := s.mttr.Calculate(ctx, tenantID)
	s.observeRequest("dashboard_mttr", start)
	return out, err
}

func (s *DashboardService) GetAnalystWorkload(ctx context.Context, tenantID uuid.UUID) ([]model.AnalystWorkloadEntry, error) {
	start := time.Now()
	out, err := s.workload.Calculate(ctx, tenantID)
	s.observeRequest("dashboard_analyst_workload", start)
	return out, err
}

func (s *DashboardService) GetTopAttackedAssets(ctx context.Context, tenantID uuid.UUID) ([]model.AssetAlertSummary, error) {
	start := time.Now()
	out, err := s.repo.TopAttackedAssets(ctx, tenantID, 10)
	s.observeRequest("dashboard_top_attacked_assets", start)
	return out, err
}

func (s *DashboardService) GetMITREHeatmap(ctx context.Context, tenantID uuid.UUID) (model.MITREHeatmapData, error) {
	start := time.Now()
	out, err := s.mitreHeatmap.Heatmap(ctx, tenantID, 90)
	s.observeRequest("dashboard_mitre_heatmap", start)
	return out, err
}

func (s *DashboardService) GetTrends(ctx context.Context, tenantID uuid.UUID, days int) (*dto.DashboardTrendsResponse, error) {
	start := time.Now()
	alertTrend, err := s.trends.AlertTrend(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}
	vulnTrend, err := s.trends.VulnTrend(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}
	threatTrend, err := s.trends.ThreatTrend(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}
	s.observeRequest("dashboard_trends", start)
	return &dto.DashboardTrendsResponse{
		Days:        days,
		AlertTrend:  alertTrend,
		VulnTrend:   vulnTrend,
		ThreatTrend: threatTrend,
	}, nil
}

// GetMetrics returns the aggregated secondary metrics strip data.
// Each section runs in parallel; partial failures yield nil for that field
// rather than failing the entire request.
func (s *DashboardService) GetMetrics(ctx context.Context, tenantID uuid.UUID) (*dto.DashboardMetricsResponse, error) {
	start := time.Now()
	resp := &dto.DashboardMetricsResponse{}
	var mu sync.Mutex

	var group errgroup.Group

	// MTTR, MTTA, SLA from the existing calculator
	group.Go(func() error {
		report, err := s.mttr.Calculate(ctx, tenantID)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("metrics: mttr calculation failed")
			return nil
		}
		mu.Lock()
		defer mu.Unlock()

		mttaMin := report.Overall.AvgResponseHours * 60
		resp.MTTAMinutes = &mttaMin

		var mttrMin float64
		if report.Overall.AvgResolveHours != nil {
			mttrMin = *report.Overall.AvgResolveHours * 60
		}
		resp.MTTRMinutes = &mttrMin

		sla := report.Overall.SLACompliance
		resp.SLACompliancePct = &sla
		return nil
	})

	// Active incidents (open critical/high alerts)
	group.Go(func() error {
		count, err := s.repo.ActiveIncidents(ctx, tenantID)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("metrics: active incidents query failed")
			return nil
		}
		mu.Lock()
		resp.ActiveIncidents = &count
		mu.Unlock()
		return nil
	})

	// Active users today
	group.Go(func() error {
		count, err := s.repo.ActiveUsersToday(ctx, tenantID)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("metrics: active users query failed")
			return nil
		}
		mu.Lock()
		resp.ActiveUsersToday = &count
		mu.Unlock()
		return nil
	})

	// Pending reviews (unacknowledged critical/high alerts)
	group.Go(func() error {
		count, err := s.repo.PendingReviews(ctx, tenantID)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("metrics: pending reviews query failed")
			return nil
		}
		mu.Lock()
		resp.PendingReviews = &count
		mu.Unlock()
		return nil
	})

	_ = group.Wait()
	s.observeRequest("dashboard_metrics", start)
	return resp, nil
}

func (s *DashboardService) InvalidateCache(ctx context.Context, tenantID uuid.UUID) error {
	return s.cache.Invalidate(ctx, tenantID)
}

func (s *DashboardService) observeRequest(endpoint string, start time.Time) {
	if s.metrics != nil && s.metrics.DashboardRequestDuration != nil {
		s.metrics.DashboardRequestDuration.WithLabelValues(endpoint).Observe(time.Since(start).Seconds())
	}
}

func (s *DashboardService) observeQuery(name string, start time.Time) {
	if s.metrics != nil && s.metrics.DashboardQueryDuration != nil {
		s.metrics.DashboardQueryDuration.WithLabelValues(name).Observe(time.Since(start).Seconds())
	}
}
