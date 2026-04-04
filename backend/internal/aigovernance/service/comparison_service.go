package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	aishadow "github.com/clario360/platform/internal/aigovernance/shadow"
	"github.com/clario360/platform/internal/events"
)

type ComparisonService struct {
	registryRepo   *repository.ModelRegistryRepository
	predictionRepo *repository.PredictionLogRepository
	repo           *repository.ShadowComparisonRepository
	metrics        *Metrics
	producer       *events.Producer
	logger         zerolog.Logger
}

func NewComparisonService(registryRepo *repository.ModelRegistryRepository, predictionRepo *repository.PredictionLogRepository, repo *repository.ShadowComparisonRepository, producer *events.Producer, metrics *Metrics, logger zerolog.Logger) *ComparisonService {
	return &ComparisonService{
		registryRepo:   registryRepo,
		predictionRepo: predictionRepo,
		repo:           repo,
		metrics:        metrics,
		producer:       producer,
		logger:         logger.With().Str("component", "ai_comparison_service").Logger(),
	}
}

func (s *ComparisonService) AggregateAllShadowModels(ctx context.Context) error {
	shadowVersions, err := s.registryRepo.ListShadowVersions(ctx)
	if err != nil {
		return err
	}
	for idx := range shadowVersions {
		shadowVersion := shadowVersions[idx]
		productionVersion, err := s.registryRepo.GetCurrentProductionVersion(ctx, shadowVersion.TenantID, shadowVersion.ModelID)
		if err != nil {
			s.logger.Warn().Err(err).Str("shadow_version_id", shadowVersion.ID.String()).Msg("skipping shadow aggregation without production version")
			continue
		}
		if _, err := s.Build(ctx, shadowVersion.TenantID, productionVersion, &shadowVersion, time.Hour); err != nil {
			s.logger.Error().Err(err).Str("shadow_version_id", shadowVersion.ID.String()).Msg("failed to build shadow comparison")
		}
	}
	return nil
}

func (s *ComparisonService) Build(ctx context.Context, tenantID uuid.UUID, productionVersion, shadowVersion *aigovmodel.ModelVersion, period time.Duration) (*aigovmodel.ShadowComparison, error) {
	if productionVersion == nil || shadowVersion == nil {
		return nil, fmt.Errorf("production and shadow versions are required")
	}
	if period <= 0 {
		period = time.Hour
	}
	end := time.Now().UTC()
	start := end.Add(-period)
	productionLogs, err := s.predictionRepo.ListByVersionAndWindow(ctx, tenantID, productionVersion.ID, start, end, boolPtr(false))
	if err != nil {
		return nil, err
	}
	shadowLogs, err := s.predictionRepo.ListByVersionAndWindow(ctx, tenantID, shadowVersion.ID, start, end, boolPtr(true))
	if err != nil {
		return nil, err
	}

	pairs := pairShadowLogs(productionLogs, shadowLogs)
	divergences := make([]aigovmodel.ShadowDivergence, 0)
	disagreementByUseCase := make(map[string]map[string]any)
	agreementCount := 0
	for _, pair := range pairs {
		agree, divergence := aishadow.ComparePredictionLogs(&pair.production, &pair.shadow)
		if agree {
			agreementCount++
		} else if divergence != nil {
			divergences = append(divergences, *divergence)
		}
		entry := disagreementByUseCase[pair.production.UseCase]
		if entry == nil {
			entry = map[string]any{"count": 0, "disagreements": 0}
			disagreementByUseCase[pair.production.UseCase] = entry
		}
		entry["count"] = entry["count"].(int) + 1
		if !agree {
			entry["disagreements"] = entry["disagreements"].(int) + 1
		}
	}

	total := len(pairs)
	agreementRate := 0.0
	if total > 0 {
		agreementRate = float64(agreementCount) / float64(total)
	}
	prodMetrics := summarizeShadowMetrics(productionLogs)
	shadowMetrics := summarizeShadowMetrics(shadowLogs)
	recommendation, reason, factors := aishadow.Recommend(agreementRate, prodMetrics, shadowMetrics)

	divergenceSamples := make([]aigovmodel.ShadowDivergence, 0, minInt(len(divergences), 10))
	for idx, item := range divergences {
		if idx >= 10 {
			break
		}
		divergenceSamples = append(divergenceSamples, item)
	}
	for useCase, item := range disagreementByUseCase {
		count := item["count"].(int)
		disagreements := item["disagreements"].(int)
		agreement := 1.0
		if count > 0 {
			agreement = 1 - float64(disagreements)/float64(count)
		}
		item["agreement_rate"] = agreement
		item["use_case"] = useCase
	}

	comparison := &aigovmodel.ShadowComparison{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		ModelID:              shadowVersion.ModelID,
		ProductionVersionID:  productionVersion.ID,
		ShadowVersionID:      shadowVersion.ID,
		PeriodStart:          start,
		PeriodEnd:            end,
		TotalPredictions:     total,
		AgreementCount:       agreementCount,
		DisagreementCount:    len(divergences),
		AgreementRate:        agreementRate,
		ProductionMetrics:    mustJSON(prodMetrics),
		ShadowMetrics:        mustJSON(shadowMetrics),
		MetricsDelta:         mustJSON(aishadow.DeltaMetrics(prodMetrics, shadowMetrics)),
		DivergenceSamples:    mustJSON(divergenceSamples),
		DivergenceByUseCase:  mustJSON(disagreementByUseCase),
		Recommendation:       recommendation,
		RecommendationReason: reason,
		RecommendationFactors: mustJSON(factors),
		CreatedAt:            time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, comparison); err != nil {
		return nil, err
	}
	if s.metrics != nil {
		s.metrics.ShadowAgreementRate.WithLabelValues(shadowVersion.ModelSlug).Set(agreementRate)
		if len(divergences) > 0 {
			s.metrics.ShadowDivergencesTotal.WithLabelValues(shadowVersion.ModelSlug).Add(float64(len(divergences)))
		}
	}
	s.publish(ctx, "com.clario360.ai.shadow.comparison_ready", tenantID, map[string]any{
		"model_id":       shadowVersion.ModelID,
		"comparison_id":  comparison.ID,
		"recommendation": comparison.Recommendation,
	})
	if len(divergences) > 0 {
		s.publish(ctx, "com.clario360.ai.shadow.divergence_detected", tenantID, map[string]any{
			"model_id":        shadowVersion.ModelID,
			"divergence_count": len(divergences),
			"agreement_rate":  agreementRate,
		})
	}
	return comparison, nil
}

func (s *ComparisonService) Latest(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.ShadowComparison, error) {
	return s.repo.LatestByModel(ctx, tenantID, modelID)
}

func (s *ComparisonService) History(ctx context.Context, tenantID, modelID uuid.UUID, limit int) ([]aigovmodel.ShadowComparison, error) {
	return s.repo.History(ctx, tenantID, modelID, limit)
}

func (s *ComparisonService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai shadow event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai shadow event")
	}
}

type shadowPair struct {
	production aigovmodel.PredictionLog
	shadow     aigovmodel.PredictionLog
}

func pairShadowLogs(productionLogs, shadowLogs []aigovmodel.PredictionLog) []shadowPair {
	productionByHash := make(map[string][]aigovmodel.PredictionLog)
	for _, item := range productionLogs {
		productionByHash[item.InputHash] = append(productionByHash[item.InputHash], item)
	}
	pairs := make([]shadowPair, 0, minInt(len(productionLogs), len(shadowLogs)))
	for _, shadowLog := range shadowLogs {
		candidates := productionByHash[shadowLog.InputHash]
		if len(candidates) == 0 {
			continue
		}
		pairs = append(pairs, shadowPair{production: candidates[0], shadow: shadowLog})
		if len(candidates) == 1 {
			delete(productionByHash, shadowLog.InputHash)
		} else {
			productionByHash[shadowLog.InputHash] = candidates[1:]
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		return pairs[i].production.CreatedAt.Before(pairs[j].production.CreatedAt)
	})
	return pairs
}

func summarizeShadowMetrics(logs []aigovmodel.PredictionLog) aishadow.MetricsSummary {
	if len(logs) == 0 {
		return aishadow.MetricsSummary{}
	}
	out := aishadow.MetricsSummary{Total: len(logs)}
	correctCount := 0
	correctTotal := 0
	for _, item := range logs {
		if item.Confidence != nil {
			out.AvgConfidence += *item.Confidence
		}
		out.AvgLatencyMS += float64(item.LatencyMS)
		if item.FeedbackCorrect != nil {
			correctTotal++
			if *item.FeedbackCorrect {
				correctCount++
			}
		}
	}
	out.AvgConfidence = out.AvgConfidence / float64(len(logs))
	out.AvgLatencyMS = out.AvgLatencyMS / float64(len(logs))
	if correctTotal > 0 {
		out.Accuracy = float64(correctCount) / float64(correctTotal)
	}
	return out
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func boolPtr(value bool) *bool {
	v := value
	return &v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
