package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type PredictionLog struct {
	ID                         uuid.UUID       `json:"id" db:"id"`
	TenantID                   uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	ModelID                    uuid.UUID       `json:"model_id" db:"model_id"`
	ModelVersionID             uuid.UUID       `json:"model_version_id" db:"model_version_id"`
	ModelSlug                  string          `json:"model_slug,omitempty" db:"model_slug"`
	ModelVersionNumber         int             `json:"model_version_number,omitempty" db:"model_version_number"`
	InputHash                  string          `json:"input_hash" db:"input_hash"`
	InputSummary               json.RawMessage `json:"input_summary,omitempty" db:"input_summary"`
	Prediction                 json.RawMessage `json:"prediction" db:"prediction"`
	Confidence                 *float64        `json:"confidence,omitempty" db:"confidence"`
	ExplanationStructured      json.RawMessage `json:"explanation_structured" db:"explanation_structured"`
	ExplanationText            string          `json:"explanation_text" db:"explanation_text"`
	ExplanationFactors         json.RawMessage `json:"explanation_factors" db:"explanation_factors"`
	Suite                      string          `json:"suite" db:"suite"`
	UseCase                    string          `json:"use_case" db:"use_case"`
	EntityType                 string          `json:"entity_type,omitempty" db:"entity_type"`
	EntityID                   *uuid.UUID      `json:"entity_id,omitempty" db:"entity_id"`
	IsShadow                   bool            `json:"is_shadow" db:"is_shadow"`
	ShadowProductionVersionID  *uuid.UUID      `json:"shadow_production_version_id,omitempty" db:"shadow_production_version_id"`
	ShadowDivergence           json.RawMessage `json:"shadow_divergence,omitempty" db:"shadow_divergence"`
	FeedbackCorrect            *bool           `json:"feedback_correct,omitempty" db:"feedback_correct"`
	FeedbackBy                 *uuid.UUID      `json:"feedback_by,omitempty" db:"feedback_by"`
	FeedbackAt                 *time.Time      `json:"feedback_at,omitempty" db:"feedback_at"`
	FeedbackNotes              *string         `json:"feedback_notes,omitempty" db:"feedback_notes"`
	FeedbackCorrectedOutput    json.RawMessage `json:"feedback_corrected_output,omitempty" db:"feedback_corrected_output"`
	LatencyMS                  int             `json:"latency_ms" db:"latency_ms"`
	CreatedAt                  time.Time       `json:"created_at" db:"created_at"`
}

type PredictionStats struct {
	ModelID         uuid.UUID `json:"model_id"`
	ModelSlug       string    `json:"model_slug"`
	Suite           string    `json:"suite"`
	UseCase         string    `json:"use_case"`
	Total           int64     `json:"total"`
	ShadowTotal     int64     `json:"shadow_total"`
	AvgConfidence   *float64  `json:"avg_confidence,omitempty"`
	AvgLatencyMS    *float64  `json:"avg_latency_ms,omitempty"`
	CorrectFeedback int64     `json:"correct_feedback"`
	WrongFeedback   int64     `json:"wrong_feedback"`
}

type PerformancePoint struct {
	PeriodStart time.Time `json:"period_start"`
	Volume      int64     `json:"volume"`
	AvgLatency  *float64  `json:"avg_latency_ms,omitempty"`
	Accuracy    *float64  `json:"accuracy,omitempty"`
}
