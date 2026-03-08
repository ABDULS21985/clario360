package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type VersionStatus string

const (
	VersionStatusDevelopment VersionStatus = "development"
	VersionStatusStaging     VersionStatus = "staging"
	VersionStatusShadow      VersionStatus = "shadow"
	VersionStatusProduction  VersionStatus = "production"
	VersionStatusRetired     VersionStatus = "retired"
	VersionStatusFailed      VersionStatus = "failed"
	VersionStatusRolledBack  VersionStatus = "rolled_back"
)

type ArtifactType string

const (
	ArtifactTypeGoFunction        ArtifactType = "go_function"
	ArtifactTypeRuleSet           ArtifactType = "rule_set"
	ArtifactTypeStatisticalConfig ArtifactType = "statistical_config"
	ArtifactTypeTemplateConfig    ArtifactType = "template_config"
	ArtifactTypeSerializedModel   ArtifactType = "serialized_model"
)

type ExplainabilityType string

const (
	ExplainabilityRuleTrace            ExplainabilityType = "rule_trace"
	ExplainabilityFeatureImportance    ExplainabilityType = "feature_importance"
	ExplainabilityStatisticalDeviation ExplainabilityType = "statistical_deviation"
	ExplainabilityTemplateBased        ExplainabilityType = "template_based"
)

type ModelVersion struct {
	ID                        uuid.UUID           `json:"id" db:"id"`
	TenantID                  uuid.UUID           `json:"tenant_id" db:"tenant_id"`
	ModelID                   uuid.UUID           `json:"model_id" db:"model_id"`
	ModelSlug                 string              `json:"model_slug,omitempty" db:"model_slug"`
	ModelName                 string              `json:"model_name,omitempty" db:"model_name"`
	ModelType                 ModelType           `json:"model_type,omitempty" db:"model_type"`
	ModelSuite                ModelSuite          `json:"suite,omitempty" db:"suite"`
	ModelRiskTier             RiskTier            `json:"risk_tier,omitempty" db:"risk_tier"`
	VersionNumber             int                 `json:"version_number" db:"version_number"`
	Status                    VersionStatus       `json:"status" db:"status"`
	Description               string              `json:"description" db:"description"`
	ArtifactType              ArtifactType        `json:"artifact_type" db:"artifact_type"`
	ArtifactConfig            json.RawMessage     `json:"artifact_config" db:"artifact_config"`
	ArtifactHash              string              `json:"artifact_hash" db:"artifact_hash"`
	ExplainabilityType        ExplainabilityType  `json:"explainability_type" db:"explainability_type"`
	ExplanationTemplate       *string             `json:"explanation_template,omitempty" db:"explanation_template"`
	TrainingDataDesc          *string             `json:"training_data_desc,omitempty" db:"training_data_desc"`
	TrainingDataHash          *string             `json:"training_data_hash,omitempty" db:"training_data_hash"`
	TrainingMetrics           json.RawMessage     `json:"training_metrics" db:"training_metrics"`
	PredictionCount           int64               `json:"prediction_count" db:"prediction_count"`
	AvgLatencyMS              *float64            `json:"avg_latency_ms,omitempty" db:"avg_latency_ms"`
	AvgConfidence             *float64            `json:"avg_confidence,omitempty" db:"avg_confidence"`
	AccuracyMetric            *float64            `json:"accuracy_metric,omitempty" db:"accuracy_metric"`
	FalsePositiveRate         *float64            `json:"false_positive_rate,omitempty" db:"false_positive_rate"`
	FalseNegativeRate         *float64            `json:"false_negative_rate,omitempty" db:"false_negative_rate"`
	FeedbackCount             int                 `json:"feedback_count" db:"feedback_count"`
	PromotedToStagingAt       *time.Time          `json:"promoted_to_staging_at,omitempty" db:"promoted_to_staging_at"`
	PromotedToShadowAt        *time.Time          `json:"promoted_to_shadow_at,omitempty" db:"promoted_to_shadow_at"`
	PromotedToProductionAt    *time.Time          `json:"promoted_to_production_at,omitempty" db:"promoted_to_production_at"`
	PromotedBy                *uuid.UUID          `json:"promoted_by,omitempty" db:"promoted_by"`
	RetiredAt                 *time.Time          `json:"retired_at,omitempty" db:"retired_at"`
	RetiredBy                 *uuid.UUID          `json:"retired_by,omitempty" db:"retired_by"`
	RetirementReason          *string             `json:"retirement_reason,omitempty" db:"retirement_reason"`
	RolledBackAt              *time.Time          `json:"rolled_back_at,omitempty" db:"rolled_back_at"`
	RolledBackBy              *uuid.UUID          `json:"rolled_back_by,omitempty" db:"rolled_back_by"`
	RollbackReason            *string             `json:"rollback_reason,omitempty" db:"rollback_reason"`
	ReplacedVersionID         *uuid.UUID          `json:"replaced_version_id,omitempty" db:"replaced_version_id"`
	CreatedBy                 uuid.UUID           `json:"created_by" db:"created_by"`
	CreatedAt                 time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at" db:"updated_at"`
	LatestShadowComparisonID  *uuid.UUID          `json:"latest_shadow_comparison_id,omitempty"`
	LatestShadowRecommendation *ShadowRecommendation `json:"latest_shadow_recommendation,omitempty"`
}

type LifecycleHistoryEntry struct {
	VersionID     uuid.UUID      `json:"version_id"`
	VersionNumber int            `json:"version_number"`
	FromStatus    *VersionStatus `json:"from_status,omitempty"`
	ToStatus      VersionStatus  `json:"to_status"`
	ChangedBy     *uuid.UUID     `json:"changed_by,omitempty"`
	Reason        string         `json:"reason,omitempty"`
	ChangedAt     time.Time      `json:"changed_at"`
}
