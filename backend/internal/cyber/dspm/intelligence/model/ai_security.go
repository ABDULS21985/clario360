package model

import (
	"time"

	"github.com/google/uuid"
)

// AIUsageType classifies how data is used in AI/ML systems.
type AIUsageType string

const (
	AIUsageTrainingData     AIUsageType = "training_data"
	AIUsageEvaluationData   AIUsageType = "evaluation_data"
	AIUsageInferenceInput   AIUsageType = "inference_input"
	AIUsageRAGKnowledgeBase AIUsageType = "rag_knowledge_base"
	AIUsagePromptContext    AIUsageType = "prompt_context"
	AIUsageFeatureStore     AIUsageType = "feature_store"
	AIUsageEmbeddingSource  AIUsageType = "embedding_source"
)

// AIRiskLevel categorizes AI data usage risk.
type AIRiskLevel string

const (
	AIRiskLow      AIRiskLevel = "low"
	AIRiskMedium   AIRiskLevel = "medium"
	AIRiskHigh     AIRiskLevel = "high"
	AIRiskCritical AIRiskLevel = "critical"
)

// AnonymizationLevel describes the level of data anonymization.
type AnonymizationLevel string

const (
	AnonymizationNone                AnonymizationLevel = "none"
	AnonymizationPseudonymized       AnonymizationLevel = "pseudonymized"
	AnonymizationAnonymized          AnonymizationLevel = "anonymized"
	AnonymizationDifferentialPrivacy AnonymizationLevel = "differential_privacy"
)

// AIUsageStatus tracks the lifecycle of an AI data usage record.
type AIUsageStatus string

const (
	AIUsageStatusActive      AIUsageStatus = "active"
	AIUsageStatusInactive    AIUsageStatus = "inactive"
	AIUsageStatusBlocked     AIUsageStatus = "blocked"
	AIUsageStatusUnderReview AIUsageStatus = "under_review"
)

// AIRiskFactor is a single factor contributing to the AI risk score.
type AIRiskFactor struct {
	Factor      string  `json:"factor"`
	Weight      float64 `json:"weight"`
	Description string  `json:"description"`
}

// AIDataUsage records a single data asset's usage in AI/ML.
type AIDataUsage struct {
	ID                 uuid.UUID          `json:"id"`
	TenantID           uuid.UUID          `json:"tenant_id"`
	DataAssetID        uuid.UUID          `json:"data_asset_id"`
	DataAssetName      string             `json:"data_asset_name"`
	DataClassification string             `json:"data_classification"`
	ContainsPII        bool               `json:"contains_pii"`
	PIITypes           []string           `json:"pii_types"`
	UsageType          AIUsageType        `json:"usage_type"`
	ModelID            *uuid.UUID         `json:"model_id,omitempty"`
	ModelName          string             `json:"model_name,omitempty"`
	ModelSlug          string             `json:"model_slug,omitempty"`
	PipelineID         string             `json:"pipeline_id,omitempty"`
	PipelineName       string             `json:"pipeline_name,omitempty"`
	AIRiskScore        float64            `json:"ai_risk_score"`
	AIRiskLevel        AIRiskLevel        `json:"ai_risk_level"`
	RiskFactors        []AIRiskFactor     `json:"risk_factors"`
	ConsentVerified    bool               `json:"consent_verified"`
	DataMinimization   bool               `json:"data_minimization"`
	AnonymizationLevel AnonymizationLevel `json:"anonymization_level"`
	RetentionCompliant bool               `json:"retention_compliant"`
	Status             AIUsageStatus      `json:"status"`
	FirstDetectedAt    time.Time          `json:"first_detected_at"`
	LastDetectedAt     time.Time          `json:"last_detected_at"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// ModelDataAssessment is the governance report for a single AI model.
type ModelDataAssessment struct {
	ModelSlug             string        `json:"model_slug"`
	ModelName             string        `json:"model_name"`
	TrainingDataCount     int           `json:"training_data_count"`
	PIITrainingData       int           `json:"pii_training_data"`
	ConsentCoverage       float64       `json:"consent_coverage"`
	AnonymizationCoverage float64       `json:"anonymization_coverage"`
	RiskScore             float64       `json:"risk_score"`
	DataUsages            []AIDataUsage `json:"data_usages"`
	Recommendations       []string      `json:"recommendations"`
}

// AISecurityDashboard aggregates AI data security metrics.
type AISecurityDashboard struct {
	TotalAILinkedAssets int            `json:"total_ai_linked_assets"`
	PIIInTrainingData   int            `json:"pii_in_training_data"`
	UnverifiedConsent   int            `json:"unverified_consent"`
	HighRiskUsageCount  int            `json:"high_risk_usage_count"`
	ModelsGoverned      int            `json:"models_governed"`
	AvgAIRiskScore      float64        `json:"avg_ai_risk_score"`
	RiskByLevel         map[string]int `json:"risk_by_level"`
	UsageByType         map[string]int `json:"usage_by_type"`
}
