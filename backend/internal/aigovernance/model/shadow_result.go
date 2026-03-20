package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ShadowRecommendation string

const (
	ShadowRecommendationPromote    ShadowRecommendation = "promote"
	ShadowRecommendationKeepShadow ShadowRecommendation = "keep_shadow"
	ShadowRecommendationReject     ShadowRecommendation = "reject"
	ShadowRecommendationNeedsReview ShadowRecommendation = "needs_review"
)

type ShadowDivergence struct {
	PredictionID      uuid.UUID       `json:"prediction_id"`
	InputHash         string          `json:"input_hash"`
	UseCase           string          `json:"use_case"`
	EntityID          *uuid.UUID      `json:"entity_id,omitempty"`
	ProductionOutput  json.RawMessage `json:"production_output"`
	ShadowOutput      json.RawMessage `json:"shadow_output"`
	ProductionConfidence *float64     `json:"production_confidence,omitempty"`
	ShadowConfidence  *float64        `json:"shadow_confidence,omitempty"`
	Reason            string          `json:"reason"`
	CreatedAt         time.Time       `json:"created_at"`
}

type ShadowComparison struct {
	ID                   uuid.UUID            `json:"id" db:"id"`
	TenantID             uuid.UUID            `json:"tenant_id" db:"tenant_id"`
	ModelID              uuid.UUID            `json:"model_id" db:"model_id"`
	ProductionVersionID  uuid.UUID            `json:"production_version_id" db:"production_version_id"`
	ShadowVersionID      uuid.UUID            `json:"shadow_version_id" db:"shadow_version_id"`
	PeriodStart          time.Time            `json:"period_start" db:"period_start"`
	PeriodEnd            time.Time            `json:"period_end" db:"period_end"`
	TotalPredictions     int                  `json:"total_predictions" db:"total_predictions"`
	AgreementCount       int                  `json:"agreement_count" db:"agreement_count"`
	DisagreementCount    int                  `json:"disagreement_count" db:"disagreement_count"`
	AgreementRate        float64              `json:"agreement_rate" db:"agreement_rate"`
	ProductionMetrics    json.RawMessage      `json:"production_metrics" db:"production_metrics"`
	ShadowMetrics        json.RawMessage      `json:"shadow_metrics" db:"shadow_metrics"`
	MetricsDelta         json.RawMessage      `json:"metrics_delta" db:"metrics_delta"`
	DivergenceSamples    json.RawMessage      `json:"divergence_samples" db:"divergence_samples"`
	DivergenceByUseCase  json.RawMessage      `json:"divergence_by_use_case" db:"divergence_by_use_case"`
	Recommendation       ShadowRecommendation `json:"recommendation" db:"recommendation"`
	RecommendationReason string               `json:"recommendation_reason" db:"recommendation_reason"`
	RecommendationFactors json.RawMessage     `json:"recommendation_factors" db:"recommendation_factors"`
	CreatedAt            time.Time            `json:"created_at" db:"created_at"`
}
