package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovdrift "github.com/clario360/platform/internal/aigovernance/drift"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	"github.com/clario360/platform/internal/events"
)

type DriftService struct {
	registryRepo   *repository.ModelRegistryRepository
	predictionRepo *repository.PredictionLogRepository
	repo           *repository.DriftReportRepository
	producer       *events.Producer
	metrics        *Metrics
	logger         zerolog.Logger
}

func NewDriftService(registryRepo *repository.ModelRegistryRepository, predictionRepo *repository.PredictionLogRepository, repo *repository.DriftReportRepository, producer *events.Producer, metrics *Metrics, logger zerolog.Logger) *DriftService {
	return &DriftService{
		registryRepo:   registryRepo,
		predictionRepo: predictionRepo,
		repo:           repo,
		producer:       producer,
		metrics:        metrics,
		logger:         logger.With().Str("component", "ai_drift_service").Logger(),
	}
}

func (s *DriftService) Latest(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.DriftReport, error) {
	return s.repo.LatestByModel(ctx, tenantID, modelID)
}

func (s *DriftService) History(ctx context.Context, tenantID, modelID uuid.UUID, limit int) ([]aigovmodel.DriftReport, error) {
	return s.repo.History(ctx, tenantID, modelID, limit)
}

func (s *DriftService) Performance(ctx context.Context, tenantID, modelID uuid.UUID, period string) ([]aigovmodel.PerformancePoint, error) {
	return s.predictionRepo.PerformanceSeries(ctx, tenantID, modelID, sinceForPeriod(period))
}

func (s *DriftService) RunAllProductionModels(ctx context.Context) error {
	versions, err := s.registryRepo.ListProductionVersions(ctx)
	if err != nil {
		return err
	}
	for idx := range versions {
		if _, err := s.RunVersion(ctx, &versions[idx], "7d"); err != nil {
			s.logger.Error().Err(err).Str("version_id", versions[idx].ID.String()).Msg("drift calculation failed")
		}
	}
	return nil
}

func (s *DriftService) RunVersion(ctx context.Context, version *aigovmodel.ModelVersion, period string) (*aigovmodel.DriftReport, error) {
	if version == nil {
		return nil, fmt.Errorf("model version is required")
	}
	if period == "" {
		period = "7d"
	}
	now := time.Now().UTC()
	currentStart := now.AddDate(0, 0, -7)
	referenceStart := version.CreatedAt
	if version.PromotedToProductionAt != nil {
		referenceStart = version.PromotedToProductionAt.UTC()
	}
	referenceEnd := referenceStart.AddDate(0, 0, 7)
	if referenceEnd.After(now) {
		referenceEnd = now
	}
	referenceLogs, err := s.predictionRepo.ListByVersionAndWindow(ctx, version.TenantID, version.ID, referenceStart, referenceEnd, boolPtr(false))
	if err != nil {
		return nil, err
	}
	currentLogs, err := s.predictionRepo.ListByVersionAndWindow(ctx, version.TenantID, version.ID, currentStart, now, boolPtr(false))
	if err != nil {
		return nil, err
	}

	referenceOutput := extractOutputDistribution(referenceLogs)
	currentOutput := extractOutputDistribution(currentLogs)
	referenceConfidence := extractConfidenceSeries(referenceLogs)
	currentConfidence := extractConfidenceSeries(currentLogs)

	var outputPSI *float64
	outputLevel := aigovmodel.DriftLevelNone
	if len(referenceOutput) > 0 && len(currentOutput) > 0 {
		psi, err := aigovdrift.CalculatePSI(referenceOutput, currentOutput, 10)
		if err == nil {
			outputPSI = &psi
			outputLevel = aigovdrift.LevelForPSI(psi)
		}
	}
	var confidencePSI *float64
	confidenceLevel := aigovmodel.DriftLevelNone
	if len(referenceConfidence) > 0 && len(currentConfidence) > 0 {
		psi, err := aigovdrift.CalculatePSI(referenceConfidence, currentConfidence, 10)
		if err == nil {
			confidencePSI = &psi
			confidenceLevel = aigovdrift.LevelForPSI(psi)
		}
	}

	currentVolume := int64(len(currentLogs))
	referenceVolume := int64(len(referenceLogs))
	volumeChange := aigovdrift.VolumeChangePct(referenceVolume, currentVolume)
	currentP95 := p95Latency(currentLogs)
	referenceP95 := p95Latency(referenceLogs)
	latencyChange := aigovdrift.LatencyChangePct(referenceP95, currentP95)
	currentAccuracy := accuracyFromFeedback(currentLogs)
	referenceAccuracy := accuracyFromFeedback(referenceLogs)
	accuracyChange := aigovdrift.AccuracyChange(referenceAccuracy, currentAccuracy)
	alerts := aigovdrift.BuildAlerts(outputLevel, confidenceLevel, volumeChange, latencyChange, accuracyChange)

	report := &aigovmodel.DriftReport{
		ID:                    uuid.New(),
		TenantID:              version.TenantID,
		ModelID:               version.ModelID,
		ModelVersionID:        version.ID,
		ModelSlug:             version.ModelSlug,
		Period:                period,
		PeriodStart:           currentStart,
		PeriodEnd:             now,
		OutputPSI:             outputPSI,
		OutputDriftLevel:      outputLevel,
		ConfidencePSI:         confidencePSI,
		ConfidenceDriftLevel:  confidenceLevel,
		CurrentVolume:         currentVolume,
		ReferenceVolume:       referenceVolume,
		VolumeChangePct:       volumeChange,
		CurrentP95LatencyMS:   currentP95,
		ReferenceP95LatencyMS: referenceP95,
		LatencyChangePct:      latencyChange,
		CurrentAccuracy:       currentAccuracy,
		ReferenceAccuracy:     referenceAccuracy,
		AccuracyChange:        accuracyChange,
		Alerts:                mustJSON(alerts),
		AlertCount:            len(alerts),
		CreatedAt:             now,
	}
	if err := s.repo.Create(ctx, report); err != nil {
		return nil, err
	}
	if s.metrics != nil && outputPSI != nil {
		s.metrics.DriftPSI.WithLabelValues(version.ModelSlug).Set(*outputPSI)
	}
	for _, alert := range alerts {
		if s.metrics != nil {
			s.metrics.DriftAlertsTotal.WithLabelValues(version.ModelSlug, alert.Severity).Inc()
		}
		s.publish(ctx, "com.clario360.ai.drift.alert", version.TenantID, map[string]any{
			"model_id":   version.ModelID,
			"alert_type": alert.Type,
			"severity":   alert.Severity,
		})
	}
	if outputPSI != nil && outputLevel == aigovmodel.DriftLevelSignificant {
		s.publish(ctx, "com.clario360.ai.drift.detected", version.TenantID, map[string]any{
			"model_id":     version.ModelID,
			"version_id":   version.ID,
			"drift_level":  outputLevel,
			"psi_value":    *outputPSI,
		})
	}
	return report, nil
}

func (s *DriftService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai drift event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai drift event")
	}
}

func extractOutputDistribution(logs []aigovmodel.PredictionLog) []float64 {
	values := make([]float64, 0, len(logs))
	for _, item := range logs {
		if score := numericPrediction(item.Prediction); score != nil {
			values = append(values, *score)
			continue
		}
		if item.Confidence != nil {
			values = append(values, *item.Confidence)
		}
	}
	return values
}

func extractConfidenceSeries(logs []aigovmodel.PredictionLog) []float64 {
	values := make([]float64, 0, len(logs))
	for _, item := range logs {
		if item.Confidence != nil {
			values = append(values, *item.Confidence)
		}
	}
	return values
}

func numericPrediction(payload json.RawMessage) *float64 {
	var number float64
	if err := json.Unmarshal(payload, &number); err == nil {
		return &number
	}
	var object map[string]any
	if err := json.Unmarshal(payload, &object); err != nil {
		return nil
	}
	for _, key := range []string{"score", "risk_score", "overall_score", "value", "priority_score"} {
		if raw, ok := object[key]; ok {
			switch typed := raw.(type) {
			case float64:
				return &typed
			case int:
				value := float64(typed)
				return &value
			}
		}
	}
	return nil
}

func p95Latency(logs []aigovmodel.PredictionLog) *float64 {
	if len(logs) == 0 {
		return nil
	}
	values := make([]int, 0, len(logs))
	for _, item := range logs {
		values = append(values, item.LatencyMS)
	}
	sort.Ints(values)
	idx := int(float64(len(values)-1) * 0.95)
	value := float64(values[idx])
	return &value
}

func accuracyFromFeedback(logs []aigovmodel.PredictionLog) *float64 {
	total := 0
	correct := 0
	for _, item := range logs {
		if item.FeedbackCorrect == nil {
			continue
		}
		total++
		if *item.FeedbackCorrect {
			correct++
		}
	}
	if total == 0 {
		return nil
	}
	value := float64(correct) / float64(total)
	return &value
}
