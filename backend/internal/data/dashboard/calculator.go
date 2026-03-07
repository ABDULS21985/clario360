package dashboard

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/quality"
	"github.com/clario360/platform/internal/data/repository"
)

type Calculator struct {
	sourceRepo        *repository.SourceRepository
	pipelineRepo      *repository.PipelineRepository
	dashboardRepo     *repository.DashboardRepository
	contradictionRepo *repository.ContradictionRepository
	darkDataRepo      *repository.DarkDataRepository
	lineageRepo       *repository.LineageRepository
	qualityScorer     *quality.Scorer
	cache             *Cache
	logger            zerolog.Logger
}

func NewCalculator(sourceRepo *repository.SourceRepository, pipelineRepo *repository.PipelineRepository, dashboardRepo *repository.DashboardRepository, contradictionRepo *repository.ContradictionRepository, darkDataRepo *repository.DarkDataRepository, lineageRepo *repository.LineageRepository, qualityScorer *quality.Scorer, cache *Cache, logger zerolog.Logger) *Calculator {
	return &Calculator{
		sourceRepo:        sourceRepo,
		pipelineRepo:      pipelineRepo,
		dashboardRepo:     dashboardRepo,
		contradictionRepo: contradictionRepo,
		darkDataRepo:      darkDataRepo,
		lineageRepo:       lineageRepo,
		qualityScorer:     qualityScorer,
		cache:             cache,
		logger:            logger,
	}
}

func (c *Calculator) Calculate(ctx context.Context, tenantID uuid.UUID) (*dto.DataSuiteDashboard, error) {
	if cached, err := c.cache.Get(ctx, tenantID); err == nil {
		cachedAt := time.Now().UTC()
		cached.CachedAt = &cachedAt
		return cached, nil
	}

	dashboard := &dto.DataSuiteDashboard{
		SourcesByType:            map[string]int{},
		SourcesByStatus:          map[string]int{},
		PipelinesByStatus:        map[string]int{},
		ContradictionsByType:     map[string]int{},
		ContradictionsBySeverity: map[string]int{},
		LineageStats:             map[string]any{},
		DarkDataStats:            map[string]any{},
		CalculatedAt:             time.Now().UTC(),
		PartialFailures:          make([]string, 0),
	}

	var (
		sourceStats             *dto.AggregateSourceStatsResponse
		pipelineStats           *model.PipelineStats
		qualityScore            *model.QualityScore
		qualityTrend            []dto.DailyMetric
		qualityByModel          []dto.ModelQualitySummary
		topFailures             []dto.QualityFailureSummary
		recentRuns              []dto.PipelineRunSummary
		pipelineTrend           []dto.DailyMetric
		pipelineSuccessRate     float64
		failedPipelines24h      int
		sourceDelta             int
		contradictionsDelta     int
		totalModels             int
		byContradictionType     map[string]int
		byContradictionSeverity map[string]int
		openContradictions      int
		darkDataStats           *model.DarkDataStatsSummary
		lineageEdges            []*model.LineageEdgeRecord
	)

	g, gctx := errgroup.WithContext(ctx)
	var failuresMu sync.Mutex
	run := func(name string, fn func(context.Context) error) {
		g.Go(func() error {
			if err := fn(gctx); err != nil {
				c.logger.Error().Err(err).Str("section", name).Msg("data dashboard section failed")
				failuresMu.Lock()
				dashboard.PartialFailures = append(dashboard.PartialFailures, name)
				failuresMu.Unlock()
			}
			return nil
		})
	}

	run("sources", func(ctx context.Context) error {
		var err error
		sourceStats, err = c.sourceRepo.AggregateStats(ctx, tenantID)
		if err == nil {
			dashboard.SourcesByType = sourceStats.ByType
			dashboard.SourcesByStatus = sourceStats.ByStatus
		}
		return err
	})
	run("pipelines", func(ctx context.Context) error {
		var err error
		pipelineStats, err = c.pipelineRepo.Stats(ctx, tenantID)
		if err == nil {
			dashboard.PipelinesByStatus = pipelineStats.ByStatus
		}
		return err
	})
	run("recent_runs", func(ctx context.Context) error {
		var err error
		recentRuns, err = c.dashboardRepo.RecentRuns(ctx, tenantID, 10)
		dashboard.RecentRuns = recentRuns
		return err
	})
	run("pipeline_trend", func(ctx context.Context) error {
		var err error
		pipelineTrend, err = c.dashboardRepo.PipelineTrend(ctx, tenantID, 30)
		dashboard.PipelineTrend = pipelineTrend
		return err
	})
	run("pipeline_success_rate", func(ctx context.Context) error {
		var err error
		pipelineSuccessRate, err = c.dashboardRepo.PipelineSuccessRate(ctx, tenantID, 30)
		dashboard.PipelineSuccessRate = pipelineSuccessRate
		return err
	})
	run("quality_score", func(ctx context.Context) error {
		var err error
		qualityScore, err = c.qualityScorer.CalculateScore(ctx, tenantID)
		if err == nil {
			dashboard.QualityScore = dto.QualityScoreSummary{
				OverallScore: qualityScore.OverallScore,
				Grade:        qualityScore.Grade,
				PassedRules:  qualityScore.PassedRules,
				FailedRules:  qualityScore.FailedRules,
				WarningRules: qualityScore.WarningRules,
				PassRate:     qualityScore.PassRate,
			}
		}
		return err
	})
	run("quality_trend", func(ctx context.Context) error {
		var err error
		qualityTrend, err = c.dashboardRepo.QualityTrend(ctx, tenantID, 30)
		dashboard.QualityTrend = qualityTrend
		return err
	})
	run("quality_by_model", func(ctx context.Context) error {
		var err error
		qualityByModel, err = c.dashboardRepo.DataDashboardQualityByModel(ctx, tenantID, 10)
		dashboard.QualityByModel = qualityByModel
		return err
	})
	run("quality_failures", func(ctx context.Context) error {
		var err error
		topFailures, err = c.dashboardRepo.DataDashboardTopFailures(ctx, tenantID, 10)
		dashboard.TopFailures = topFailures
		return err
	})
	run("contradictions", func(ctx context.Context) error {
		var err error
		byContradictionType, byContradictionSeverity, openContradictions, err = c.dashboardRepo.ContradictionBreakdown(ctx, tenantID)
		if err == nil {
			dashboard.ContradictionsByType = byContradictionType
			dashboard.ContradictionsBySeverity = byContradictionSeverity
			dashboard.OpenContradictions = openContradictions
		}
		return err
	})
	run("dark_data", func(ctx context.Context) error {
		var err error
		darkDataStats, err = c.darkDataRepo.Stats(ctx, tenantID)
		if err == nil {
			dashboard.DarkDataStats = map[string]any{
				"total_assets":         darkDataStats.TotalAssets,
				"by_reason":            darkDataStats.ByReason,
				"by_type":              darkDataStats.ByType,
				"by_governance_status": darkDataStats.ByGovernanceStatus,
				"pii_assets":           darkDataStats.PIIAssets,
				"high_risk_assets":     darkDataStats.HighRiskAssets,
				"total_size_bytes":     darkDataStats.TotalSizeBytes,
				"average_risk_score":   darkDataStats.AverageRiskScore,
				"governed_assets":      darkDataStats.GovernedAssets,
				"scheduled_deletions":  darkDataStats.ScheduledDeletionCount,
			}
		}
		return err
	})
	run("lineage", func(ctx context.Context) error {
		var err error
		lineageEdges, err = c.lineageRepo.ListActive(ctx, tenantID)
		if err == nil {
			dashboard.LineageStats = summarizeLineage(lineageEdges)
		}
		return err
	})
	run("kpi_support", func(ctx context.Context) error {
		var err error
		failedPipelines24h, err = c.dashboardRepo.FailedPipelines24h(ctx, tenantID)
		if err != nil {
			return err
		}
		sourceDelta, err = c.dashboardRepo.SourceCountDelta(ctx, tenantID)
		if err != nil {
			return err
		}
		contradictionsDelta, err = c.dashboardRepo.ContradictionsDelta(ctx, tenantID)
		if err != nil {
			return err
		}
		totalModels, err = c.dashboardRepo.TotalModels(ctx, tenantID)
		return err
	})

	_ = g.Wait()
	dashboard.KPIs = dto.DataKPIs{
		TotalSources:        safeTotalSources(sourceStats),
		ActivePipelines:     safeActivePipelines(pipelineStats),
		QualityScore:        safeQualityScore(qualityScore),
		QualityGrade:        safeQualityGrade(qualityScore),
		OpenContradictions:  openContradictions,
		DarkDataAssets:      safeDarkDataAssets(darkDataStats),
		TotalModels:         totalModels,
		FailedPipelines24h:  failedPipelines24h,
		SourcesDelta:        sourceDelta,
		QualityDelta:        calculateQualityDelta(qualityTrend),
		ContradictionsDelta: contradictionsDelta,
	}

	_ = c.cache.Set(ctx, tenantID, dashboard)
	return dashboard, nil
}

func summarizeLineage(edges []*model.LineageEdgeRecord) map[string]any {
	nodes := make(map[string]struct{})
	relationships := make(map[string]int)
	for _, edge := range edges {
		nodes[string(edge.SourceType)+":"+edge.SourceID.String()] = struct{}{}
		nodes[string(edge.TargetType)+":"+edge.TargetID.String()] = struct{}{}
		relationships[string(edge.Relationship)]++
	}
	return map[string]any{
		"node_count":    len(nodes),
		"edge_count":    len(edges),
		"relationships": relationships,
	}
}

func safeTotalSources(stats *dto.AggregateSourceStatsResponse) int {
	if stats == nil {
		return 0
	}
	return stats.TotalSources
}

func safeActivePipelines(stats *model.PipelineStats) int {
	if stats == nil {
		return 0
	}
	return stats.ActivePipelines
}

func safeQualityScore(score *model.QualityScore) float64 {
	if score == nil {
		return 0
	}
	return score.OverallScore
}

func safeQualityGrade(score *model.QualityScore) string {
	if score == nil {
		return "F"
	}
	return score.Grade
}

func safeDarkDataAssets(stats *model.DarkDataStatsSummary) int {
	if stats == nil {
		return 0
	}
	return stats.TotalAssets
}

func calculateQualityDelta(trend []dto.DailyMetric) float64 {
	if len(trend) < 2 {
		return 0
	}
	return trend[len(trend)-1].Value - trend[len(trend)-2].Value
}
