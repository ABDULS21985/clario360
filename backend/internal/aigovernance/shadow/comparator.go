package shadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type MetricsSummary struct {
	Accuracy      float64 `json:"accuracy"`
	AvgConfidence float64 `json:"avg_confidence"`
	AvgLatencyMS  float64 `json:"avg_latency_ms"`
	Total         int     `json:"total"`
}

func ComparePredictionLogs(production, shadow *aigovmodel.PredictionLog) (bool, *aigovmodel.ShadowDivergence) {
	if production == nil || shadow == nil {
		return false, nil
	}
	productionJSON := normalizeJSON(production.Prediction)
	shadowJSON := normalizeJSON(shadow.Prediction)
	confidenceDelta := confidenceDiff(production.Confidence, shadow.Confidence)
	if bytes.Equal(productionJSON, shadowJSON) && confidenceDelta < 0.15 {
		return true, nil
	}
	reason := "prediction output diverged"
	if bytes.Equal(productionJSON, shadowJSON) {
		reason = fmt.Sprintf("confidence diverged by %.2f", confidenceDelta)
	}
	return false, &aigovmodel.ShadowDivergence{
		PredictionID:         production.ID,
		InputHash:            production.InputHash,
		UseCase:              production.UseCase,
		EntityID:             production.EntityID,
		ProductionOutput:     production.Prediction,
		ShadowOutput:         shadow.Prediction,
		ProductionConfidence: production.Confidence,
		ShadowConfidence:     shadow.Confidence,
		Reason:               reason,
		CreatedAt:            time.Now().UTC(),
	}
}

func Recommend(agreementRate float64, production, shadow MetricsSummary) (aigovmodel.ShadowRecommendation, string, []map[string]any) {
	factors := []map[string]any{
		{"factor": "agreement_rate", "value": agreementRate},
		{"factor": "production_accuracy", "value": production.Accuracy},
		{"factor": "shadow_accuracy", "value": shadow.Accuracy},
		{"factor": "production_latency_ms", "value": production.AvgLatencyMS},
		{"factor": "shadow_latency_ms", "value": shadow.AvgLatencyMS},
	}
	switch {
	case agreementRate > 0.95 && shadowScore(shadow) >= shadowScore(production):
		return aigovmodel.ShadowRecommendationPromote, "shadow agrees with production and meets or exceeds current performance", factors
	case agreementRate >= 0.90 && agreementRate <= 0.95:
		return aigovmodel.ShadowRecommendationKeepShadow, "shadow performance is promising but needs a longer observation window", factors
	case agreementRate < 0.80:
		return aigovmodel.ShadowRecommendationReject, "shadow diverges too frequently from production", factors
	default:
		return aigovmodel.ShadowRecommendationNeedsReview, "shadow requires manual review before promotion", factors
	}
}

func DeltaMetrics(production, shadow MetricsSummary) map[string]float64 {
	return map[string]float64{
		"accuracy":       shadow.Accuracy - production.Accuracy,
		"avg_confidence": shadow.AvgConfidence - production.AvgConfidence,
		"avg_latency_ms": shadow.AvgLatencyMS - production.AvgLatencyMS,
	}
}

func normalizeJSON(value json.RawMessage) []byte {
	if len(value) == 0 {
		return []byte("{}")
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return value
	}
	payload, err := json.Marshal(decoded)
	if err != nil {
		return value
	}
	return payload
}

func confidenceDiff(left, right *float64) float64 {
	if left == nil || right == nil {
		return 0
	}
	return math.Abs(*left - *right)
}

func shadowScore(value MetricsSummary) float64 {
	return value.Accuracy + value.AvgConfidence - (value.AvgLatencyMS / 1000)
}

func NewSyntheticDivergence(inputHash, useCase string, entityID *uuid.UUID, productionOutput, shadowOutput json.RawMessage, productionConfidence, shadowConfidence *float64, reason string) *aigovmodel.ShadowDivergence {
	return &aigovmodel.ShadowDivergence{
		PredictionID:         uuid.New(),
		InputHash:            inputHash,
		UseCase:              useCase,
		EntityID:             entityID,
		ProductionOutput:     productionOutput,
		ShadowOutput:         shadowOutput,
		ProductionConfidence: productionConfidence,
		ShadowConfidence:     shadowConfidence,
		Reason:               reason,
		CreatedAt:            time.Now().UTC(),
	}
}
