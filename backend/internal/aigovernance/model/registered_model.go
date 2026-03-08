package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ModelType string

const (
	ModelTypeRuleBased       ModelType = "rule_based"
	ModelTypeStatistical     ModelType = "statistical"
	ModelTypeMLClassifier    ModelType = "ml_classifier"
	ModelTypeMLRegressor     ModelType = "ml_regressor"
	ModelTypeNLPExtractor    ModelType = "nlp_extractor"
	ModelTypeAnomalyDetector ModelType = "anomaly_detector"
	ModelTypeScorer          ModelType = "scorer"
	ModelTypeRecommender     ModelType = "recommender"
)

type ModelSuite string

const (
	SuiteCyber    ModelSuite = "cyber"
	SuiteData     ModelSuite = "data"
	SuiteActa     ModelSuite = "acta"
	SuiteLex      ModelSuite = "lex"
	SuiteVisus    ModelSuite = "visus"
	SuitePlatform ModelSuite = "platform"
)

type RiskTier string

const (
	RiskTierLow      RiskTier = "low"
	RiskTierMedium   RiskTier = "medium"
	RiskTierHigh     RiskTier = "high"
	RiskTierCritical RiskTier = "critical"
)

type ModelStatus string

const (
	ModelStatusActive     ModelStatus = "active"
	ModelStatusDeprecated ModelStatus = "deprecated"
	ModelStatusRetired    ModelStatus = "retired"
)

type RegisteredModel struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name        string          `json:"name" db:"name"`
	Slug        string          `json:"slug" db:"slug"`
	Description string          `json:"description" db:"description"`
	ModelType   ModelType       `json:"model_type" db:"model_type"`
	Suite       ModelSuite      `json:"suite" db:"suite"`
	OwnerUserID *uuid.UUID      `json:"owner_user_id,omitempty" db:"owner_user_id"`
	OwnerTeam   string          `json:"owner_team,omitempty" db:"owner_team"`
	RiskTier    RiskTier        `json:"risk_tier" db:"risk_tier"`
	Status      ModelStatus     `json:"status" db:"status"`
	Tags        []string        `json:"tags" db:"tags"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	CreatedBy   uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

type ModelWithVersions struct {
	Model             *RegisteredModel `json:"model"`
	ProductionVersion *ModelVersion    `json:"production_version,omitempty"`
	ShadowVersion     *ModelVersion    `json:"shadow_version,omitempty"`
}
