package contradiction

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/data/connector"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
	"github.com/clario360/platform/internal/data/contradiction/strategies"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

type configDecryptor interface {
	Decrypt(ciphertext []byte, keyID string) ([]byte, error)
}

type DetectionStrategy interface {
	Type() string
	Detect(ctx context.Context, pair cruntime.ModelPair, connA, connB connector.Connector) ([]cruntime.RawContradiction, error)
}

type Detector struct {
	strategies       []DetectionStrategy
	entityLinker     *EntityLinker
	connRegistry     *connector.ConnectorRegistry
	sourceRepo       *repository.SourceRepository
	modelRepo        *repository.ModelRepository
	contraRepo       *repository.ContradictionRepository
	decryptor        configDecryptor
	producer         *events.Producer
	logger           zerolog.Logger
	predictionLogger *aigovmiddleware.PredictionLogger
}

func NewDetector(
	connRegistry *connector.ConnectorRegistry,
	sourceRepo *repository.SourceRepository,
	modelRepo *repository.ModelRepository,
	contraRepo *repository.ContradictionRepository,
	decryptor configDecryptor,
	producer *events.Producer,
	logger zerolog.Logger,
) *Detector {
	return &Detector{
		strategies: []DetectionStrategy{
			strategies.NewLogicalStrategy(),
			strategies.NewSemanticStrategy(),
			strategies.NewTemporalStrategy(),
			strategies.NewAnalyticalStrategy(),
		},
		entityLinker: NewEntityLinker(),
		connRegistry: connRegistry,
		sourceRepo:   sourceRepo,
		modelRepo:    modelRepo,
		contraRepo:   contraRepo,
		decryptor:    decryptor,
		producer:     producer,
		logger:       logger,
	}
}

func (d *Detector) SetPredictionLogger(predictionLogger *aigovmiddleware.PredictionLogger) {
	d.predictionLogger = predictionLogger
}

func (d *Detector) RunScan(ctx context.Context, tenantID, triggeredBy uuid.UUID) (*model.ContradictionScan, error) {
	startedAt := time.Now().UTC()
	scan := &model.ContradictionScan{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Status:      "running",
		ByType:      json.RawMessage(`{}`),
		BySeverity:  json.RawMessage(`{}`),
		StartedAt:   startedAt,
		TriggeredBy: triggeredBy,
		CreatedAt:   startedAt,
	}
	if err := d.contraRepo.CreateScan(ctx, scan); err != nil {
		return nil, err
	}
	d.publish(ctx, "data.contradiction.scan_started", tenantID, map[string]any{"scan_id": scan.ID})

	models, _, err := d.modelRepo.List(ctx, tenantID, dto.ListModelsParams{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, err
	}
	sources := make(map[string]*model.DataSource)
	for _, item := range models {
		if item.SourceID == nil {
			continue
		}
		record, err := d.sourceRepo.Get(ctx, tenantID, *item.SourceID)
		if err != nil {
			continue
		}
		sources[item.SourceID.String()] = record.Source
	}
	pairs := d.entityLinker.Link(models, sources)
	scan.ModelsScanned = len(models)
	scan.ModelPairsCompared = len(pairs)

	byType := make(map[string]int)
	bySeverity := make(map[string]int)
	created := 0
	for _, pair := range pairs {
		rawItems, err := d.scanPair(ctx, pair)
		if err != nil {
			d.logger.Error().Err(err).Str("model_a", pair.ModelA.ID.String()).Str("model_b", pair.ModelB.ID.String()).Msg("scan model pair")
			continue
		}
		for _, raw := range rawItems {
			confidence := ComputeConfidence(raw, pair.ModelA, pair.ModelB, pair.SourceA, pair.SourceB)
			guidance, authoritative := GenerateGuidance(raw, pair.ModelA, pair.ModelB, pair.SourceA, pair.SourceB)
			contradictionItem := &model.Contradiction{
				ID:                  uuid.New(),
				TenantID:            tenantID,
				ScanID:              &scan.ID,
				Type:                raw.Type,
				Severity:            severityForRaw(raw),
				ConfidenceScore:     confidence,
				Title:               raw.Title,
				Description:         raw.Description,
				SourceA:             raw.SourceA,
				SourceB:             raw.SourceB,
				EntityKeyColumn:     stringPtr(pair.LinkColumn),
				EntityKeyValue:      stringPtr(raw.EntityKey),
				AffectedRecords:     raw.AffectedRecords,
				SampleRecords:       marshal(raw.SampleRecords),
				ResolutionGuidance:  guidance,
				AuthoritativeSource: authoritative,
				Status:              model.ContradictionStatusDetected,
				Metadata:            marshal(raw.Metadata),
				CreatedAt:           time.Now().UTC(),
				UpdatedAt:           time.Now().UTC(),
			}
			if err := d.contraRepo.Create(ctx, contradictionItem); err != nil {
				return nil, err
			}
			created++
			byType[string(contradictionItem.Type)]++
			bySeverity[string(contradictionItem.Severity)]++
			d.publish(ctx, "data.contradiction.detected", tenantID, map[string]any{
				"id":       contradictionItem.ID,
				"type":     contradictionItem.Type,
				"severity": contradictionItem.Severity,
				"title":    contradictionItem.Title,
			})
		}
	}
	completedAt := time.Now().UTC()
	durationMs := completedAt.Sub(startedAt).Milliseconds()
	scan.Status = "completed"
	scan.ContradictionsFound = created
	scan.CompletedAt = &completedAt
	scan.DurationMs = &durationMs
	scan.ByType = marshal(byType)
	scan.BySeverity = marshal(bySeverity)
	if err := d.contraRepo.UpdateScan(ctx, scan); err != nil {
		return nil, err
	}
	d.publish(ctx, "data.contradiction.scan_completed", tenantID, map[string]any{
		"scan_id":              scan.ID,
		"contradictions_found": created,
		"by_type":              byType,
		"by_severity":          bySeverity,
	})
	d.recordPrediction(ctx, scan, models, pairs, byType, bySeverity, created, durationMs)
	return scan, nil
}

func (d *Detector) recordPrediction(ctx context.Context, scan *model.ContradictionScan, models []*model.DataModel, pairs []cruntime.ModelPair, byType, bySeverity map[string]int, contradictionCount int, durationMS int64) {
	if d.predictionLogger == nil || scan == nil {
		return
	}

	matchedConditions := make([]string, 0, len(byType))
	ruleWeights := make(map[string]any, len(byType))
	for contradictionType, count := range byType {
		matchedConditions = append(matchedConditions, fmt.Sprintf("%s:%d", contradictionType, count))
		ruleWeights[contradictionType] = float64(count) / float64(maxInt(1, contradictionCount))
	}
	if len(matchedConditions) == 0 {
		matchedConditions = append(matchedConditions, "no_contradictions_detected")
	}

	input := map[string]any{
		"scan_id":        scan.ID.String(),
		"models_scanned": len(models),
		"model_pairs":    len(pairs),
		"contradictions": contradictionCount,
	}
	confidence := 0.88
	if contradictionCount == 0 {
		confidence = 0.93
	}

	_, _ = d.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     scan.TenantID,
		ModelSlug:    "data-contradiction-detector",
		UseCase:      "contradiction_detection",
		EntityType:   "contradiction_scan",
		EntityID:     &scan.ID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"scan_id":              scan.ID,
					"contradictions_found": contradictionCount,
					"by_type":              byType,
					"by_severity":          bySeverity,
					"duration_ms":          durationMS,
				},
				Confidence: confidence,
				Metadata: map[string]any{
					"matched_rules":      []string{"logical", "semantic", "temporal", "analytical"},
					"matched_conditions": matchedConditions,
					"rule_weights":       ruleWeights,
					"models_scanned":     len(models),
					"model_pairs":        len(pairs),
					"duration_ms":        durationMS,
				},
			}, nil
		},
	})
}

func (d *Detector) scanPair(ctx context.Context, pair cruntime.ModelPair) ([]cruntime.RawContradiction, error) {
	connA, err := d.openSourceConnector(ctx, pair.SourceA)
	if err != nil {
		return nil, err
	}
	defer connA.Close()
	connB, err := d.openSourceConnector(ctx, pair.SourceB)
	if err != nil {
		return nil, err
	}
	defer connB.Close()

	result := make([]cruntime.RawContradiction, 0)
	for _, strategy := range d.strategies {
		items, err := strategy.Detect(ctx, pair, connA, connB)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
	}
	return result, nil
}

func (d *Detector) openSourceConnector(ctx context.Context, source *model.DataSource) (connector.Connector, error) {
	record, err := d.sourceRepo.Get(ctx, source.TenantID, source.ID)
	if err != nil {
		return nil, err
	}
	decrypted, err := d.decryptor.Decrypt(record.EncryptedConfig, record.Source.EncryptionKeyID)
	if err != nil {
		return nil, err
	}
	return d.connRegistry.Create(source.Type, json.RawMessage(decrypted))
}

func severityForRaw(raw cruntime.RawContradiction) model.QualitySeverity {
	switch {
	case raw.AffectedRecords > 100:
		return model.QualitySeverityCritical
	case raw.AffectedRecords > 25:
		return model.QualitySeverityHigh
	case raw.AffectedRecords > 5:
		return model.QualitySeverityMedium
	default:
		return model.QualitySeverityLow
	}
}

func (d *Detector) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if d.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "data-service", tenantID.String(), payload)
	if err != nil {
		return
	}
	_ = d.producer.Publish(ctx, "data.contradiction.events", event)
}

func marshal(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
