package dto

import (
	"encoding/json"

	"github.com/google/uuid"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

type RegisterModelRequest struct {
	Name        string               `json:"name"`
	Slug        string               `json:"slug"`
	Description string               `json:"description"`
	ModelType   aigovmodel.ModelType `json:"model_type"`
	Suite       aigovmodel.ModelSuite `json:"suite"`
	OwnerUserID *uuid.UUID           `json:"owner_user_id"`
	OwnerTeam   string               `json:"owner_team"`
	RiskTier    aigovmodel.RiskTier  `json:"risk_tier"`
	Tags        []string             `json:"tags"`
	Metadata    json.RawMessage      `json:"metadata"`
}

type UpdateModelRequest struct {
	Name        *string               `json:"name"`
	Description *string               `json:"description"`
	OwnerUserID *uuid.UUID            `json:"owner_user_id"`
	OwnerTeam   *string               `json:"owner_team"`
	RiskTier    *aigovmodel.RiskTier  `json:"risk_tier"`
	Status      *aigovmodel.ModelStatus `json:"status"`
	Tags        *[]string             `json:"tags"`
	Metadata    *json.RawMessage      `json:"metadata"`
}

type CreateVersionRequest struct {
	Description          string                       `json:"description"`
	ArtifactType         aigovmodel.ArtifactType      `json:"artifact_type"`
	ArtifactConfig       json.RawMessage              `json:"artifact_config"`
	ExplainabilityType   aigovmodel.ExplainabilityType `json:"explainability_type"`
	ExplanationTemplate  *string                      `json:"explanation_template"`
	TrainingDataDesc     *string                      `json:"training_data_desc"`
	TrainingDataHash     *string                      `json:"training_data_hash"`
	TrainingMetrics      json.RawMessage              `json:"training_metrics"`
}

type ListModelsResponse struct {
	Items []aigovmodel.ModelWithVersions `json:"items"`
}
