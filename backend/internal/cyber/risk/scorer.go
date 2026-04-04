package risk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/cyber/metrics"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
)

const riskCacheTTL = 5 * time.Minute

type RiskComponent interface {
	Name() string
	Weight() float64
	Calculate(ctx context.Context, tenantID uuid.UUID) (*model.RiskComponentResult, error)
}

type RiskScorer struct {
	components       []RiskComponent
	contribs         *ContributorAnalyzer
	recommends       *RecommendationEngine
	historyRepo      *repository.RiskHistoryRepository
	cache            *redis.Client
	db               *pgxpool.Pool
	logger           zerolog.Logger
	metrics          *metrics.Metrics
	predictionLogger *aigovmiddleware.PredictionLogger
}

func NewRiskScorer(
	db *pgxpool.Pool,
	cache *redis.Client,
	historyRepo *repository.RiskHistoryRepository,
	contribs *ContributorAnalyzer,
	recommends *RecommendationEngine,
	m *metrics.Metrics,
	logger zerolog.Logger,
	components ...RiskComponent,
) *RiskScorer {
	return &RiskScorer{
		components:  components,
		contribs:    contribs,
		recommends:  recommends,
		historyRepo: historyRepo,
		cache:       cache,
		db:          db,
		logger:      logger.With().Str("component", "risk-scorer").Logger(),
		metrics:     m,
	}
}

func (rs *RiskScorer) CalculateOrganizationRisk(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error) {
	return rs.calculate(ctx, tenantID, false)
}

func (rs *RiskScorer) CalculateOrganizationRiskFresh(ctx context.Context, tenantID uuid.UUID) (*model.OrganizationRiskScore, error) {
	return rs.calculate(ctx, tenantID, true)
}

func (rs *RiskScorer) InvalidateCache(ctx context.Context, tenantID uuid.UUID) error {
	if rs.cache == nil {
		return nil
	}
	return rs.cache.Del(ctx, riskCacheKey(tenantID)).Err()
}

func (rs *RiskScorer) calculate(ctx context.Context, tenantID uuid.UUID, force bool) (*model.OrganizationRiskScore, error) {
	start := time.Now()
	if !force && rs.cache != nil {
		cached, err := rs.cache.Get(ctx, riskCacheKey(tenantID)).Bytes()
		if err == nil {
			var score model.OrganizationRiskScore
			if unmarshalErr := json.Unmarshal(cached, &score); unmarshalErr == nil {
				if rs.metrics != nil && rs.metrics.RiskCacheHitTotal != nil {
					rs.metrics.RiskCacheHitTotal.Inc()
				}
				return &score, nil
			}
		}
	}
	if rs.metrics != nil && rs.metrics.RiskCacheMissTotal != nil {
		rs.metrics.RiskCacheMissTotal.Inc()
	}

	latestAny, _ := rs.historyRepo.Latest(ctx, tenantID)
	latestDaily, _ := rs.historyRepo.LatestDaily(ctx, tenantID)
	latestAnyComponents := decodeHistoricalComponents(latestAny)
	latestDailyComponents := decodeHistoricalComponents(latestDaily)

	type namedResult struct {
		name   string
		weight float64
		result *model.RiskComponentResult
	}
	results := make(chan namedResult, len(rs.components))
	group, groupCtx := errgroup.WithContext(ctx)
	for _, component := range rs.components {
		component := component
		group.Go(func() error {
			result, err := component.Calculate(groupCtx, tenantID)
			if err != nil {
				rs.logger.Error().
					Err(err).
					Str("tenant_id", tenantID.String()).
					Str("component", component.Name()).
					Msg("risk component calculation failed; using historical fallback")
				result = fallbackComponentResult(component.Name(), latestAnyComponents)
			}
			if result == nil {
				result = &model.RiskComponentResult{
					Description: "component unavailable",
					Details:     map[string]interface{}{},
				}
			}
			results <- namedResult{name: component.Name(), weight: component.Weight(), result: result}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	close(results)

	componentScores := model.RiskComponents{}
	var overall float64
	for entry := range results {
		previous := previousComponentScore(entry.name, latestDailyComponents)
		score := componentScoreFromResult(entry.result, entry.weight, previous)
		assignComponentScore(&componentScores, entry.name, score)
		overall += score.Weighted
	}
	overall = clampScore(roundTo2(overall))

	contextData, err := rs.loadContext(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load risk context: %w", err)
	}

	score := &model.OrganizationRiskScore{
		TenantID:     tenantID,
		OverallScore: overall,
		Grade:        gradeForScore(overall),
		Components:   componentScores,
		Context:      contextData,
		CalculatedAt: time.Now().UTC(),
	}
	if latestDaily != nil {
		score.Trend, score.TrendDelta = trendForScore(overall, latestDaily.OverallScore)
	} else {
		score.Trend = "stable"
		score.TrendDelta = 0
	}

	if rs.contribs != nil {
		contributors, err := rs.contribs.Analyze(ctx, tenantID)
		if err != nil {
			rs.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("risk contributor analysis failed")
		} else {
			score.TopContributors = contributors
		}
	}

	if rs.recommends != nil {
		recommendations, err := rs.recommends.Generate(ctx, tenantID, score)
		if err != nil {
			rs.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("risk recommendation generation failed")
		} else {
			score.Recommendations = recommendations
		}
	}

	if rs.cache != nil {
		if encoded, err := json.Marshal(score); err == nil {
			_ = rs.cache.Set(ctx, riskCacheKey(tenantID), encoded, riskCacheTTL).Err()
		}
	}

	if rs.metrics != nil {
		if rs.metrics.RiskCalculationDuration != nil {
			rs.metrics.RiskCalculationDuration.Observe(time.Since(start).Seconds())
		}
		if rs.metrics.RiskScoreCurrent != nil {
			rs.metrics.RiskScoreCurrent.WithLabelValues(tenantID.String(), score.Grade).Set(score.OverallScore)
		}
		if rs.metrics.RiskComponentScore != nil {
			rs.metrics.RiskComponentScore.WithLabelValues(tenantID.String(), "vulnerability").Set(score.Components.VulnerabilityRisk.Score)
			rs.metrics.RiskComponentScore.WithLabelValues(tenantID.String(), "threat").Set(score.Components.ThreatExposure.Score)
			rs.metrics.RiskComponentScore.WithLabelValues(tenantID.String(), "configuration").Set(score.Components.ConfigurationRisk.Score)
			rs.metrics.RiskComponentScore.WithLabelValues(tenantID.String(), "surface").Set(score.Components.AttackSurfaceRisk.Score)
			rs.metrics.RiskComponentScore.WithLabelValues(tenantID.String(), "compliance").Set(score.Components.ComplianceGapRisk.Score)
		}
	}
	rs.recordPrediction(ctx, tenantID, score)

	return score, nil
}

func (rs *RiskScorer) SetPredictionLogger(predictionLogger *aigovmiddleware.PredictionLogger) {
	rs.predictionLogger = predictionLogger
}

func (rs *RiskScorer) loadContext(ctx context.Context, tenantID uuid.UUID) (model.RiskContext, error) {
	var out model.RiskContext
	err := rs.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM vulnerabilities WHERE tenant_id = $1 AND status IN ('open','in_progress') AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM alerts WHERE tenant_id = $1 AND status IN ('new','acknowledged','investigating') AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM threats WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL),
			(SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL
				AND ('internet-facing' = ANY(tags) OR 'dmz' = ANY(tags) OR 'public' = ANY(tags))),
			(SELECT COUNT(*)::int FROM assets WHERE tenant_id = $1 AND status = 'active' AND criticality = 'critical' AND deleted_at IS NULL)`,
		tenantID,
	).Scan(
		&out.TotalAssets,
		&out.TotalOpenVulns,
		&out.TotalOpenAlerts,
		&out.TotalActiveThreats,
		&out.InternetFacingAssets,
		&out.CriticalAssets,
	)
	return out, err
}

func componentScoreFromResult(result *model.RiskComponentResult, weight, previous float64) model.ComponentScore {
	score := clampScore(roundTo2(result.Score))
	trend, delta := trendForScore(score, previous)
	details := result.Details
	if details == nil {
		details = map[string]interface{}{}
	}
	return model.ComponentScore{
		Score:       score,
		Weight:      weight,
		Weighted:    roundTo2(score * weight),
		Trend:       trend,
		TrendDelta:  roundTo2(delta),
		Description: result.Description,
		Details:     details,
	}
}

func riskCacheKey(tenantID uuid.UUID) string {
	return "cyber:risk:" + tenantID.String()
}

func decodeHistoricalComponents(history *model.RiskScoreHistory) *model.RiskComponents {
	if history == nil || len(history.Components) == 0 {
		return nil
	}
	var components model.RiskComponents
	if err := json.Unmarshal(history.Components, &components); err != nil {
		return nil
	}
	return &components
}

func fallbackComponentResult(name string, previous *model.RiskComponents) *model.RiskComponentResult {
	if previous == nil {
		return &model.RiskComponentResult{
			Score:       0,
			Description: "no historical value available",
			Details:     map[string]interface{}{},
		}
	}
	component := previousComponent(name, previous)
	return &model.RiskComponentResult{
		Score:       component.Score,
		Trend:       component.Trend,
		TrendDelta:  component.TrendDelta,
		Description: "historical fallback applied",
		Details:     component.Details,
	}
}

func previousComponentScore(name string, previous *model.RiskComponents) float64 {
	if previous == nil {
		return 0
	}
	return previousComponent(name, previous).Score
}

func previousComponent(name string, previous *model.RiskComponents) model.ComponentScore {
	switch name {
	case "vulnerability_risk":
		return previous.VulnerabilityRisk
	case "threat_exposure":
		return previous.ThreatExposure
	case "configuration_risk":
		return previous.ConfigurationRisk
	case "attack_surface_risk":
		return previous.AttackSurfaceRisk
	case "compliance_gap_risk":
		return previous.ComplianceGapRisk
	default:
		return model.ComponentScore{}
	}
}

func assignComponentScore(target *model.RiskComponents, name string, score model.ComponentScore) {
	switch name {
	case "vulnerability_risk":
		target.VulnerabilityRisk = score
	case "threat_exposure":
		target.ThreatExposure = score
	case "configuration_risk":
		target.ConfigurationRisk = score
	case "attack_surface_risk":
		target.AttackSurfaceRisk = score
	case "compliance_gap_risk":
		target.ComplianceGapRisk = score
	}
}

func gradeForScore(score float64) string {
	switch {
	case score <= 20:
		return "A"
	case score <= 40:
		return "B"
	case score <= 60:
		return "C"
	case score <= 80:
		return "D"
	default:
		return "F"
	}
}

func trendForScore(current, previous float64) (string, float64) {
	if previous == 0 && current == 0 {
		return "stable", 0
	}
	delta := roundTo2(current - previous)
	switch {
	case delta > 2:
		return "worsening", delta
	case delta < -2:
		return "improving", delta
	default:
		return "stable", delta
	}
}

func clampScore(score float64) float64 {
	return math.Min(100, math.Max(0, score))
}

func roundTo2(value float64) float64 {
	return math.Round(value*100) / 100
}

func ignoreNotFound(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	return err
}
