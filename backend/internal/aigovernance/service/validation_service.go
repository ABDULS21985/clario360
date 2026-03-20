package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	aigovmetrics "github.com/clario360/platform/internal/aigovernance/metrics"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/events"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const minimumValidationSamples = 50

type ReplaySample struct {
	PredictionID      *uuid.UUID
	InputHash         string
	InputSummary      json.RawMessage
	PredictedOutput   json.RawMessage
	Confidence        float64
	PredictedPositive bool
	ActualPositive    bool
	Severity          string
	RuleType          string
	Explanation       string
}

type LiveReplayRunner interface {
	Replay(ctx context.Context, tenantID uuid.UUID, version *aigovmodel.ModelVersion, req aigovdto.ValidateRequest) ([]ReplaySample, error)
}

type ValidationService struct {
	registryRepo     validationRegistryRepository
	predictionRepo   validationPredictionRepository
	resultRepo       validationResultRepository
	producer         *events.Producer
	metrics          *Metrics
	liveReplayRunner LiveReplayRunner
	logger           zerolog.Logger
}

type validationExample struct {
	predictionID      *uuid.UUID
	inputHash         string
	inputSummary      json.RawMessage
	predictedOutput   json.RawMessage
	confidence        float64
	predictedPositive bool
	actualPositive    bool
	severity          string
	ruleType          string
	explanation       string
}

type validationRegistryRepository interface {
	GetVersion(ctx context.Context, tenantID, modelID, versionID uuid.UUID) (*aigovmodel.ModelVersion, error)
	GetCurrentProductionVersion(ctx context.Context, tenantID, modelID uuid.UUID) (*aigovmodel.ModelVersion, error)
	UpdateVersionValidationMetrics(ctx context.Context, tenantID, versionID uuid.UUID, trainingMetrics json.RawMessage, accuracy, falsePositiveRate, falseNegativeRate float64) error
}

type validationPredictionRepository interface {
	ListByVersionAndWindow(ctx context.Context, tenantID, versionID uuid.UUID, start, end time.Time, isShadow *bool) ([]aigovmodel.PredictionLog, error)
	ListLatestByVersionAndInputHashes(ctx context.Context, tenantID, versionID uuid.UUID, hashes []string) ([]aigovmodel.PredictionLog, error)
}

type validationResultRepository interface {
	Create(ctx context.Context, item *aigovmodel.ValidationResult) error
	LatestByVersion(ctx context.Context, tenantID, versionID uuid.UUID) (*aigovmodel.ValidationResult, error)
	HistoryByVersion(ctx context.Context, tenantID, versionID uuid.UUID, limit int) ([]aigovmodel.ValidationResult, error)
}

func NewValidationService(registryRepo validationRegistryRepository, predictionRepo validationPredictionRepository, resultRepo validationResultRepository, producer *events.Producer, metrics *Metrics, liveReplayRunner LiveReplayRunner, logger zerolog.Logger) *ValidationService {
	return &ValidationService{
		registryRepo:     registryRepo,
		predictionRepo:   predictionRepo,
		resultRepo:       resultRepo,
		producer:         producer,
		metrics:          metrics,
		liveReplayRunner: liveReplayRunner,
		logger:           logger.With().Str("component", "ai_validation_service").Logger(),
	}
}

func (s *ValidationService) Preview(ctx context.Context, tenantID, modelID, versionID uuid.UUID, req aigovdto.ValidateRequest) (*aigovdto.ValidationPreviewResponse, error) {
	version, err := s.registryRepo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}
	examples, warnings, err := s.gatherDataset(ctx, version, req)
	if err != nil {
		return nil, err
	}
	positive, negative := countLabels(examples)
	return &aigovdto.ValidationPreviewResponse{
		DatasetType:   req.DatasetType,
		DatasetSize:   len(examples),
		PositiveCount: positive,
		NegativeCount: negative,
		Warnings:      append(warnings, statisticalWarnings(len(examples))...),
	}, nil
}

func (s *ValidationService) Validate(ctx context.Context, tenantID, modelID, versionID uuid.UUID, req aigovdto.ValidateRequest) (*aigovmodel.ValidationResult, error) {
	start := time.Now().UTC()
	version, err := s.registryRepo.GetVersion(ctx, tenantID, modelID, versionID)
	if err != nil {
		return nil, err
	}

	examples, warnings, err := s.gatherDataset(ctx, version, req)
	if err != nil {
		return nil, err
	}
	if len(examples) < minimumValidationSamples {
		return nil, fmt.Errorf("insufficient labeled data. Need at least 50 samples.")
	}
	warnings = append(warnings, statisticalWarnings(len(examples))...)

	binary := make([]aigovmetrics.BinarySample, 0, len(examples))
	scored := make([]aigovmetrics.ScoredSample, 0, len(examples))
	for _, example := range examples {
		binary = append(binary, aigovmetrics.BinarySample{
			PredictedPositive: example.predictedPositive,
			ActualPositive:    example.actualPositive,
		})
		scored = append(scored, aigovmetrics.ScoredSample{
			Score:          example.confidence,
			ActualPositive: example.actualPositive,
		})
	}
	matrix := aigovmetrics.CalculateConfusionMatrix(binary)
	rocCurve, auc := aigovmetrics.BuildROCCurve(scored, req.ConfidenceThresholds)
	summary := aigovmetrics.Summary(matrix, auc)

	bySeverity := summarizeByDimension(examples, func(item validationExample) string { return item.severity })
	byRuleType := summarizeByDimension(examples, func(item validationExample) string { return item.ruleType })
	productionMetrics, deltas := s.productionComparison(ctx, tenantID, modelID, versionID, summary)
	recommendation, reason := recommend(summary)

	positive, negative := countLabels(examples)
	result := &aigovmodel.ValidationResult{
		ID:                   uuid.New(),
		TenantID:             tenantID,
		ModelID:              modelID,
		VersionID:            versionID,
		DatasetType:          req.DatasetType,
		DatasetSize:          len(examples),
		PositiveCount:        positive,
		NegativeCount:        negative,
		TruePositives:        matrix.TP,
		FalsePositives:       matrix.FP,
		TrueNegatives:        matrix.TN,
		FalseNegatives:       matrix.FN,
		Precision:            summary.Precision,
		Recall:               summary.Recall,
		F1Score:              summary.F1Score,
		FalsePositiveRate:    summary.FalsePositiveRate,
		Accuracy:             summary.Accuracy,
		AUC:                  auc,
		ROCCurve:             rocCurve,
		ProductionMetrics:    productionMetrics,
		Deltas:               deltas,
		BySeverity:           bySeverity,
		ByRuleType:           byRuleType,
		FPSamples:            sampleErrors(examples, true, false),
		FNSamples:            sampleErrors(examples, false, true),
		Recommendation:       recommendation,
		RecommendationReason: reason,
		Warnings:             warnings,
		ValidatedAt:          time.Now().UTC(),
		DurationMs:           int(time.Since(start).Milliseconds()),
	}
	if err := s.resultRepo.Create(ctx, result); err != nil {
		return nil, err
	}
	if err := s.registryRepo.UpdateVersionValidationMetrics(ctx, tenantID, versionID, mergedTrainingMetrics(version.TrainingMetrics, result), summary.Accuracy, summary.FalsePositiveRate, summary.FalseNegativeRate); err != nil {
		s.logger.Warn().Err(err).Str("version_id", versionID.String()).Msg("failed to update model version validation metrics")
	}
	s.publish(ctx, "com.clario360.ai.model.validated", tenantID, map[string]any{
		"model_id":       modelID,
		"version_id":     versionID,
		"recommendation": result.Recommendation,
		"precision":      result.Precision,
		"recall":         result.Recall,
		"fpr":            result.FalsePositiveRate,
		"auc":            result.AUC,
	})
	return result, nil
}

func (s *ValidationService) Latest(ctx context.Context, tenantID, versionID uuid.UUID) (*aigovmodel.ValidationResult, error) {
	return s.resultRepo.LatestByVersion(ctx, tenantID, versionID)
}

func (s *ValidationService) History(ctx context.Context, tenantID, versionID uuid.UUID, limit int) ([]aigovmodel.ValidationResult, error) {
	return s.resultRepo.HistoryByVersion(ctx, tenantID, versionID, limit)
}

func (s *ValidationService) gatherDataset(ctx context.Context, version *aigovmodel.ModelVersion, req aigovdto.ValidateRequest) ([]validationExample, []string, error) {
	switch req.DatasetType {
	case aigovmodel.ValidationDatasetHistorical:
		logs, err := s.predictionRepo.ListByVersionAndWindow(ctx, version.TenantID, version.ID, sinceForPeriod(req.TimeRange), time.Now().UTC(), boolPtr(false))
		if err != nil {
			return nil, nil, err
		}
		examples := make([]validationExample, 0, len(logs))
		for _, log := range logs {
			example, ok := validationExampleFromLog(log, nil)
			if ok {
				examples = append(examples, example)
			}
		}
		return examples, nil, nil
	case aigovmodel.ValidationDatasetCustom:
		if len(req.CustomData) == 0 {
			return nil, nil, fmt.Errorf("custom_data is required for custom dataset")
		}
		hashes := make([]string, 0, len(req.CustomData))
		for _, item := range req.CustomData {
			hash := strings.TrimSpace(item.InputHash)
			if hash == "" {
				continue
			}
			hashes = append(hashes, hash)
		}
		logs, err := s.predictionRepo.ListLatestByVersionAndInputHashes(ctx, version.TenantID, version.ID, hashes)
		if err != nil {
			return nil, nil, err
		}
		byHash := make(map[string]aigovmodel.PredictionLog, len(logs))
		for _, log := range logs {
			byHash[log.InputHash] = log
		}
		examples := make([]validationExample, 0, len(req.CustomData))
		missing := 0
		for _, item := range req.CustomData {
			log, ok := byHash[strings.TrimSpace(item.InputHash)]
			if !ok {
				missing++
				continue
			}
			actualPositive := item.ExpectedLabel == aigovmodel.ValidationLabelThreat
			example, built := validationExampleFromLog(log, &actualPositive)
			if built {
				examples = append(examples, example)
			}
		}
		warnings := []string{}
		if missing > 0 {
			warnings = append(warnings, fmt.Sprintf("%d custom samples did not match a stored prediction log and were excluded.", missing))
		}
		return examples, warnings, nil
	case aigovmodel.ValidationDatasetLiveReplay:
		if s.liveReplayRunner == nil {
			return nil, nil, fmt.Errorf("live replay is not configured for this deployment")
		}
		replayed, err := s.liveReplayRunner.Replay(ctx, version.TenantID, version, req)
		if err != nil {
			return nil, nil, err
		}
		examples := make([]validationExample, 0, len(replayed))
		for _, item := range replayed {
			examples = append(examples, validationExample{
				predictionID:      item.PredictionID,
				inputHash:         item.InputHash,
				inputSummary:      item.InputSummary,
				predictedOutput:   item.PredictedOutput,
				confidence:        item.Confidence,
				predictedPositive: item.PredictedPositive,
				actualPositive:    item.ActualPositive,
				severity:          normalizeSeverity(item.Severity),
				ruleType:          normalizeRuleType(item.RuleType),
				explanation:       strings.TrimSpace(item.Explanation),
			})
		}
		return examples, nil, nil
	default:
		return nil, nil, fmt.Errorf("invalid dataset_type %q", req.DatasetType)
	}
}

func validationExampleFromLog(log aigovmodel.PredictionLog, explicitActual *bool) (validationExample, bool) {
	predictedPositive, _ := inferBinaryLabel(log.Prediction, log.Confidence)
	actualPositive, ok := false, false
	switch {
	case explicitActual != nil:
		actualPositive = *explicitActual
		ok = true
	default:
		actualPositive, ok = deriveActualLabel(log, predictedPositive)
	}
	if !ok {
		return validationExample{}, false
	}
	confidence := 0.0
	if log.Confidence != nil {
		confidence = *log.Confidence
	} else if predictedPositive {
		confidence = 1
	}
	predictionID := log.ID
	return validationExample{
		predictionID:      &predictionID,
		inputHash:         log.InputHash,
		inputSummary:      log.InputSummary,
		predictedOutput:   log.Prediction,
		confidence:        confidence,
		predictedPositive: predictedPositive,
		actualPositive:    actualPositive,
		severity:          normalizeSeverity(firstNonEmpty(extractSeverity(log.InputSummary), extractSeverity(log.Prediction), extractSeverity(log.ExplanationStructured))),
		ruleType:          normalizeRuleType(firstNonEmpty(extractRuleType(log.InputSummary), extractRuleType(log.Prediction), extractRuleType(log.ExplanationStructured), log.UseCase)),
		explanation:       strings.TrimSpace(log.ExplanationText),
	}, true
}

func deriveActualLabel(log aigovmodel.PredictionLog, predictedPositive bool) (bool, bool) {
	if value, ok := inferBinaryLabel(log.FeedbackCorrectedOutput, nil); ok {
		return value, true
	}
	if log.FeedbackCorrect == nil {
		return false, false
	}
	if *log.FeedbackCorrect {
		return predictedPositive, true
	}
	return !predictedPositive, true
}

func inferBinaryLabel(raw json.RawMessage, confidence *float64) (bool, bool) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		if confidence == nil {
			return false, false
		}
		return *confidence >= 0.5, true
	}
	var boolean bool
	if err := json.Unmarshal(raw, &boolean); err == nil {
		return boolean, true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		if label, ok := classifyString(text); ok {
			return label, true
		}
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number >= 0.5, true
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		if confidence == nil {
			return false, false
		}
		return *confidence >= 0.5, true
	}
	if label, ok := inferBinaryValue(payload); ok {
		return label, true
	}
	if confidence == nil {
		return false, false
	}
	return *confidence >= 0.5, true
}

func inferBinaryValue(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		return classifyString(typed)
	case float64:
		return typed >= 0.5, true
	case map[string]any:
		for _, key := range []string{"matched", "match", "anomaly_detected", "threat", "is_threat", "malicious", "detected", "positive", "alert"} {
			if raw, ok := typed[key]; ok {
				return inferBinaryValue(raw)
			}
		}
		for _, key := range []string{"predicted_label", "label", "classification", "class", "result", "verdict", "status"} {
			if raw, ok := typed[key]; ok {
				return inferBinaryValue(raw)
			}
		}
		for _, key := range []string{"probability", "score", "risk_score", "overall_score", "confidence"} {
			if raw, ok := typed[key]; ok {
				return inferBinaryValue(raw)
			}
		}
	case []any:
		for _, item := range typed {
			if label, ok := inferBinaryValue(item); ok {
				return label, true
			}
		}
	}
	return false, false
}

func classifyString(value string) (bool, bool) {
	switch normalizeStringValue(value) {
	case "threat", "malicious", "match", "matched", "positive", "anomaly", "detected", "alert", "suspicious", "critical", "high", "medium":
		return true, true
	case "benign", "negative", "clean", "normal", "safe", "no", "false", "low", "none", "no_match", "not_detected":
		return false, true
	default:
		return false, false
	}
}

func countLabels(examples []validationExample) (int, int) {
	positive := 0
	for _, item := range examples {
		if item.actualPositive {
			positive++
		}
	}
	return positive, len(examples) - positive
}

func summarizeByDimension(examples []validationExample, selector func(validationExample) string) map[string]aigovmodel.MetricsSummary {
	grouped := make(map[string][]aigovmetrics.BinarySample)
	for _, item := range examples {
		key := strings.TrimSpace(selector(item))
		if key == "" {
			key = "unclassified"
		}
		grouped[key] = append(grouped[key], aigovmetrics.BinarySample{
			PredictedPositive: item.predictedPositive,
			ActualPositive:    item.actualPositive,
		})
	}
	out := make(map[string]aigovmodel.MetricsSummary, len(grouped))
	for key, samples := range grouped {
		out[key] = aigovmetrics.Summary(aigovmetrics.CalculateConfusionMatrix(samples), 0)
	}
	return out
}

func sampleErrors(examples []validationExample, predictedPositive, actualPositive bool) []aigovmodel.PredictionSample {
	filtered := make([]validationExample, 0, len(examples))
	for _, item := range examples {
		if item.predictedPositive == predictedPositive && item.actualPositive == actualPositive {
			filtered = append(filtered, item)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].confidence > filtered[j].confidence
	})
	limit := 10
	if len(filtered) < limit {
		limit = len(filtered)
	}
	samples := make([]aigovmodel.PredictionSample, 0, limit)
	for idx := 0; idx < limit; idx++ {
		item := filtered[idx]
		samples = append(samples, aigovmodel.PredictionSample{
			PredictionID:    item.predictionID,
			InputHash:       item.inputHash,
			InputSummary:    item.inputSummary,
			PredictedOutput: item.predictedOutput,
			PredictedLabel:  labelForBool(item.predictedPositive),
			ExpectedLabel:   labelForBool(item.actualPositive),
			Confidence:      item.confidence,
			Severity:        item.severity,
			RuleType:        item.ruleType,
			Explanation:     item.explanation,
		})
	}
	return samples
}

func (s *ValidationService) productionComparison(ctx context.Context, tenantID, modelID, versionID uuid.UUID, candidate aigovmodel.MetricsSummary) (*aigovmodel.MetricsSummary, map[string]float64) {
	productionVersion, err := s.registryRepo.GetCurrentProductionVersion(ctx, tenantID, modelID)
	if err != nil || productionVersion.ID == versionID {
		return nil, nil
	}
	productionResult, err := s.resultRepo.LatestByVersion(ctx, tenantID, productionVersion.ID)
	if err != nil {
		return nil, nil
	}
	summary := &aigovmodel.MetricsSummary{
		DatasetSize:       productionResult.DatasetSize,
		PositiveCount:     productionResult.PositiveCount,
		NegativeCount:     productionResult.NegativeCount,
		TruePositives:     productionResult.TruePositives,
		FalsePositives:    productionResult.FalsePositives,
		TrueNegatives:     productionResult.TrueNegatives,
		FalseNegatives:    productionResult.FalseNegatives,
		Precision:         productionResult.Precision,
		Recall:            productionResult.Recall,
		F1Score:           productionResult.F1Score,
		FalsePositiveRate: productionResult.FalsePositiveRate,
		Accuracy:          productionResult.Accuracy,
		AUC:               productionResult.AUC,
	}
	return summary, aigovmetrics.CompareMetrics(candidate, *summary)
}

func recommend(summary aigovmodel.MetricsSummary) (aigovmodel.ValidationRecommendation, string) {
	switch {
	case summary.Precision >= 0.85 && summary.Recall >= 0.80 && summary.FalsePositiveRate <= 0.05 && summary.AUC >= 0.90:
		return aigovmodel.ValidationRecommendationPromote, "all promotion thresholds are satisfied"
	case summary.Precision < 0.70:
		return aigovmodel.ValidationRecommendationReject, fmt.Sprintf("precision is below threshold (%.3f < 0.700)", summary.Precision)
	case summary.Recall < 0.60:
		return aigovmodel.ValidationRecommendationReject, fmt.Sprintf("recall is below threshold (%.3f < 0.600)", summary.Recall)
	case summary.FalsePositiveRate > 0.15:
		return aigovmodel.ValidationRecommendationReject, fmt.Sprintf("false positive rate exceeds threshold (%.3f > 0.150)", summary.FalsePositiveRate)
	case summary.AUC < 0.75:
		return aigovmodel.ValidationRecommendationReject, fmt.Sprintf("AUC is below threshold (%.3f < 0.750)", summary.AUC)
	default:
		return aigovmodel.ValidationRecommendationKeepTesting, "metrics are close to promotion targets but at least one threshold still needs improvement"
	}
}

func statisticalWarnings(datasetSize int) []string {
	if datasetSize >= 200 {
		return nil
	}
	return []string{"Results may not be statistically significant with < 200 samples."}
}

func mergedTrainingMetrics(existing json.RawMessage, result *aigovmodel.ValidationResult) json.RawMessage {
	out := map[string]any{}
	if len(existing) > 0 && string(existing) != "null" {
		_ = json.Unmarshal(existing, &out)
	}
	out["validation"] = map[string]any{
		"dataset_type":          result.DatasetType,
		"dataset_size":          result.DatasetSize,
		"precision":             result.Precision,
		"recall":                result.Recall,
		"f1_score":              result.F1Score,
		"false_positive_rate":   result.FalsePositiveRate,
		"accuracy":              result.Accuracy,
		"auc":                   result.AUC,
		"recommendation":        result.Recommendation,
		"recommendation_reason": result.RecommendationReason,
		"validated_at":          result.ValidatedAt,
	}
	return mustJSON(out)
}

func (s *ValidationService) publish(ctx context.Context, eventType string, tenantID uuid.UUID, payload map[string]any) {
	if s.producer == nil {
		return
	}
	event, err := events.NewEvent(eventType, "iam-service", tenantID.String(), payload)
	if err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to build ai validation event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.AIEvents, event); err != nil {
		s.logger.Warn().Err(err).Str("event_type", eventType).Msg("failed to publish ai validation event")
	}
}

func labelForBool(value bool) aigovmodel.ValidationLabel {
	if value {
		return aigovmodel.ValidationLabelThreat
	}
	return aigovmodel.ValidationLabelBenign
}

func normalizeSeverity(value string) string {
	switch normalizeStringValue(value) {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "unclassified"
	}
}

func normalizeRuleType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return normalizeStringValue(value)
}

func extractSeverity(raw json.RawMessage) string {
	value, err := decodePayload(raw)
	if err != nil {
		return ""
	}
	if severity, ok := recursiveStringLookup(value, map[string]struct{}{
		"severity":                       {},
		"risk_level":                     {},
		"priority":                       {},
		"criticality":                    {},
		"highest_vulnerability_severity": {},
	}); ok {
		return severity
	}
	if severity, ok := recursiveSeverityValue(value); ok {
		return severity
	}
	return ""
}

func extractRuleType(raw json.RawMessage) string {
	value, err := decodePayload(raw)
	if err != nil {
		return ""
	}
	ruleType, _ := recursiveStringLookup(value, map[string]struct{}{
		"rule_type":           {},
		"detection_rule_type": {},
		"model_type":          {},
		"use_case":            {},
	})
	return ruleType
}

func decodePayload(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil, fmt.Errorf("empty payload")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

func recursiveStringLookup(value any, keys map[string]struct{}) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if _, ok := keys[key]; ok {
				switch candidate := item.(type) {
				case string:
					if strings.TrimSpace(candidate) != "" {
						return candidate, true
					}
				}
			}
			if candidate, ok := recursiveStringLookup(item, keys); ok {
				return candidate, true
			}
		}
	case []any:
		for _, item := range typed {
			if candidate, ok := recursiveStringLookup(item, keys); ok {
				return candidate, true
			}
		}
	}
	return "", false
}

func recursiveSeverityValue(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		if normalized := normalizeSeverity(typed); normalized != "unclassified" {
			return normalized, true
		}
	case map[string]any:
		for _, item := range typed {
			if severity, ok := recursiveSeverityValue(item); ok {
				return severity, true
			}
		}
	case []any:
		for _, item := range typed {
			if severity, ok := recursiveSeverityValue(item); ok {
				return severity, true
			}
		}
	}
	return "", false
}

func normalizeStringValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
