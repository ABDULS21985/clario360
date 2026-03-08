package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/aigovernance/repository"
	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	aishadow "github.com/clario360/platform/internal/aigovernance/shadow"
	"github.com/clario360/platform/internal/events"
)

type cacheEntry struct {
	version   *aigovmodel.ModelVersion
	expiresAt time.Time
}

type PredictionLogger struct {
	registryCache  sync.Map
	predictionCh   chan *aigovmodel.PredictionLog
	shadowCh       chan *aishadow.ExecutionTask
	explanationSvc *aigovservice.ExplanationService
	predictionRepo *repository.PredictionLogRepository
	registryRepo   *repository.ModelRegistryRepository
	producer       *events.Producer
	metrics        *aigovservice.Metrics
	logger         zerolog.Logger
	shadowExecutor *aishadow.Executor
}

func NewPredictionLogger(ctx context.Context, explanationSvc *aigovservice.ExplanationService, predictionRepo *repository.PredictionLogRepository, registryRepo *repository.ModelRegistryRepository, producer *events.Producer, metrics *aigovservice.Metrics, logger zerolog.Logger) *PredictionLogger {
	pl := &PredictionLogger{
		predictionCh:   make(chan *aigovmodel.PredictionLog, 10000),
		shadowCh:       make(chan *aishadow.ExecutionTask, 1000),
		explanationSvc: explanationSvc,
		predictionRepo: predictionRepo,
		registryRepo:   registryRepo,
		producer:       producer,
		metrics:        metrics,
		logger:         logger.With().Str("component", "ai_prediction_logger").Logger(),
	}
	pl.shadowExecutor = aishadow.NewExecutor(explanationSvc, logger)
	go pl.predictionLogWriter(ctx)
	go pl.shadowWorker(ctx)
	return pl
}

func (pl *PredictionLogger) Predict(ctx context.Context, params aigovernance.PredictParams) (*aigovernance.PredictionResult, error) {
	if params.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required")
	}
	if strings.TrimSpace(params.ModelSlug) == "" {
		return nil, fmt.Errorf("model_slug is required")
	}
	if params.ModelFunc == nil {
		return nil, fmt.Errorf("model function is required")
	}
	productionVersion, err := pl.getVersion(ctx, params.TenantID, params.ModelSlug, aigovmodel.VersionStatusProduction)
	if err != nil {
		return nil, fmt.Errorf("resolve production version: %w", err)
	}

	start := time.Now()
	modelOutput, err := params.ModelFunc(ctx, params.Input)
	latency := time.Since(start)
	if err != nil {
		return nil, err
	}
	explanation, err := pl.explanationSvc.Explain(ctx, productionVersion, params.Input, modelOutput)
	if err != nil {
		return nil, err
	}
	if pl.metrics != nil {
		pl.metrics.PredictionLatencySeconds.WithLabelValues(params.ModelSlug, string(productionVersion.ModelSuite)).Observe(latency.Seconds())
		pl.metrics.PredictionsTotal.WithLabelValues(params.ModelSlug, string(productionVersion.ModelSuite), "false").Inc()
		pl.metrics.PredictionConfidence.WithLabelValues(params.ModelSlug).Observe(modelOutput.Confidence)
	}

	inputHash, err := aigovernance.HashJSON(params.Input)
	if err != nil {
		return nil, fmt.Errorf("hash prediction input: %w", err)
	}
	inputSummary := params.InputSummary
	if len(inputSummary) == 0 {
		inputSummary = aigovernance.SummarizeInput(params.Input)
	}
	logEntry := &aigovmodel.PredictionLog{
		ID:                     uuid.New(),
		TenantID:               params.TenantID,
		ModelID:                productionVersion.ModelID,
		ModelVersionID:         productionVersion.ID,
		ModelSlug:              productionVersion.ModelSlug,
		ModelVersionNumber:     productionVersion.VersionNumber,
		InputHash:              inputHash,
		InputSummary:           mustJSON(inputSummary),
		Prediction:             mustJSON(modelOutput.Output),
		Confidence:             floatPtr(modelOutput.Confidence),
		ExplanationStructured:  mustJSON(explanation.Structured),
		ExplanationText:        explanation.HumanReadable,
		ExplanationFactors:     mustJSON(explanation.Factors),
		Suite:                  string(productionVersion.ModelSuite),
		UseCase:                params.UseCase,
		EntityType:             params.EntityType,
		EntityID:               params.EntityID,
		IsShadow:               false,
		LatencyMS:              int(latency.Milliseconds()),
		CreatedAt:              time.Now().UTC(),
	}
	select {
	case pl.predictionCh <- logEntry:
		pl.observeQueueDepth()
	default:
		pl.logger.Warn().Str("model_slug", params.ModelSlug).Msg("prediction log queue full, dropping log entry")
		if pl.metrics != nil {
			pl.metrics.PredictionLogsDropped.Inc()
		}
	}

	if params.ShadowModelFunc != nil {
		if shadowVersion, err := pl.getShadowVersion(ctx, params.TenantID, params.ModelSlug); err == nil && shadowVersion != nil {
			task := &aishadow.ExecutionTask{
				TenantID:          params.TenantID,
				ShadowVersion:     shadowVersion,
				ProductionVersion: productionVersion,
				Params:            params,
				ProductionResult:  modelOutput,
				InputHash:         inputHash,
				InputSummary:      inputSummary,
			}
			select {
			case pl.shadowCh <- task:
				if pl.metrics != nil {
					pl.metrics.ShadowExecutionsTotal.WithLabelValues(params.ModelSlug).Inc()
				}
				pl.observeQueueDepth()
			default:
				pl.logger.Warn().Str("model_slug", params.ModelSlug).Msg("shadow execution queue full, skipping shadow execution")
			}
		}
	}

	return &aigovernance.PredictionResult{
		Output:      modelOutput.Output,
		Confidence:  modelOutput.Confidence,
		Explanation: explanation,
		ModelID:     productionVersion.ModelID,
		VersionID:   productionVersion.ID,
		LatencyMS:   int(latency.Milliseconds()),
	}, nil
}

func (pl *PredictionLogger) InvalidateModel(slug string) {
	pl.registryCache.Range(func(key, _ any) bool {
		if strings.Contains(fmt.Sprint(key), ":"+slug+":") {
			pl.registryCache.Delete(key)
		}
		return true
	})
}

func (pl *PredictionLogger) predictionLogWriter(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	batch := make([]*aigovmodel.PredictionLog, 0, 100)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := pl.flushBatch(ctx, batch); err != nil {
			pl.logger.Error().Err(err).Int("batch_size", len(batch)).Msg("failed to persist ai prediction batch")
		}
		batch = batch[:0]
		pl.observeQueueDepth()
	}
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case item := <-pl.predictionCh:
			if item != nil {
				batch = append(batch, item)
			}
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (pl *PredictionLogger) shadowWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-pl.shadowCh:
			if task == nil {
				continue
			}
			result, err := pl.shadowExecutor.Execute(ctx, task)
			if err != nil {
				pl.logger.Error().Err(err).Str("model_slug", task.Params.ModelSlug).Msg("shadow execution failed")
				continue
			}
			select {
			case pl.predictionCh <- result.Log:
				pl.observeQueueDepth()
			default:
				pl.logger.Warn().Str("model_slug", task.Params.ModelSlug).Msg("prediction log queue full, dropping shadow result")
				if pl.metrics != nil {
					pl.metrics.PredictionLogsDropped.Inc()
				}
			}
			if result.Divergence != nil {
				pl.publishShadowDivergence(ctx, task, result.Divergence)
			}
		}
	}
}

func (pl *PredictionLogger) flushBatch(ctx context.Context, batch []*aigovmodel.PredictionLog) error {
	if len(batch) == 0 {
		return nil
	}
	err := pl.predictionRepo.InsertBatch(ctx, batch)
	if err != nil {
		time.Sleep(100 * time.Millisecond)
		err = pl.predictionRepo.InsertBatch(ctx, batch)
	}
	if err != nil {
		return err
	}
	if pl.metrics != nil {
		pl.metrics.PredictionLogsWritten.Add(float64(len(batch)))
	}
	versionSet := make(map[uuid.UUID]uuid.UUID, len(batch))
	for _, item := range batch {
		versionSet[item.ModelVersionID] = item.TenantID
	}
	for versionID, tenantID := range versionSet {
		if err := pl.registryRepo.UpdateVersionAggregates(ctx, tenantID, versionID); err != nil {
			pl.logger.Warn().Err(err).Str("version_id", versionID.String()).Msg("failed to refresh ai model version aggregates")
		}
	}
	return nil
}

func (pl *PredictionLogger) getShadowVersion(ctx context.Context, tenantID uuid.UUID, slug string) (*aigovmodel.ModelVersion, error) {
	version, err := pl.getVersion(ctx, tenantID, slug, aigovmodel.VersionStatusShadow)
	if err == repository.ErrNotFound {
		return nil, nil
	}
	return version, err
}

func (pl *PredictionLogger) getVersion(ctx context.Context, tenantID uuid.UUID, slug string, status aigovmodel.VersionStatus) (*aigovmodel.ModelVersion, error) {
	cacheKey := fmt.Sprintf("%s:%s:%s", tenantID.String(), slug, status)
	if cached, ok := pl.registryCache.Load(cacheKey); ok {
		entry := cached.(*cacheEntry)
		if time.Now().UTC().Before(entry.expiresAt) {
			return entry.version, nil
		}
		pl.registryCache.Delete(cacheKey)
	}
	if pl.registryRepo == nil {
		return nil, fmt.Errorf("model registry repository is not configured")
	}
	var (
		version *aigovmodel.ModelVersion
		err     error
	)
	switch status {
	case aigovmodel.VersionStatusProduction:
		version, err = pl.registryRepo.GetProductionVersionBySlug(ctx, tenantID, slug)
	case aigovmodel.VersionStatusShadow:
		version, err = pl.registryRepo.GetShadowVersionBySlug(ctx, tenantID, slug)
	default:
		err = fmt.Errorf("unsupported cached status %s", status)
	}
	if err != nil {
		return nil, err
	}
	pl.registryCache.Store(cacheKey, &cacheEntry{
		version:   version,
		expiresAt: time.Now().UTC().Add(60 * time.Second),
	})
	return version, nil
}

func (pl *PredictionLogger) publishShadowDivergence(ctx context.Context, task *aishadow.ExecutionTask, divergence *aigovmodel.ShadowDivergence) {
	if pl.producer == nil {
		return
	}
	event, err := events.NewEvent("com.clario360.ai.shadow.divergence_detected", "iam-service", task.TenantID.String(), map[string]any{
		"model_id":        task.ShadowVersion.ModelID,
		"divergence_count": 1,
		"agreement_rate":  0,
		"reason":          divergence.Reason,
	})
	if err != nil {
		pl.logger.Warn().Err(err).Str("model_slug", task.Params.ModelSlug).Msg("failed to build shadow divergence event")
		return
	}
	if err := pl.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		pl.logger.Warn().Err(err).Str("model_slug", task.Params.ModelSlug).Msg("failed to publish shadow divergence event")
	}
}

func (pl *PredictionLogger) observeQueueDepth() {
	if pl.metrics == nil {
		return
	}
	pl.metrics.PredictionLogsQueued.Set(float64(len(pl.predictionCh)))
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func floatPtr(value float64) *float64 {
	return &value
}
