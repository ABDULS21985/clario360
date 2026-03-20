package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/aigovernance"
	"github.com/clario360/platform/internal/cyber/model"
)

func (s *AssetService) recordAssetClassificationPrediction(ctx context.Context, tenantID uuid.UUID, asset *model.Asset, criticality model.Criticality, ruleName string) {
	if s.predictionLogger == nil || asset == nil {
		return
	}

	confidence := 0.78
	if ruleName != "" && ruleName != "default" {
		confidence = 0.9
	}
	input := map[string]any{
		"asset_id":             asset.ID.String(),
		"asset_type":           asset.Type,
		"asset_name":           asset.Name,
		"existing_criticality": asset.Criticality,
		"tags":                 asset.Tags,
	}
	_, _ = s.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "cyber-asset-classifier",
		UseCase:      "asset_classification",
		EntityType:   "asset",
		EntityID:     &asset.ID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"asset_id":               asset.ID,
					"classified_criticality": criticality,
					"rule_name":              ruleName,
				},
				Confidence: confidence,
				Metadata: map[string]any{
					"matched_rules":      []string{ruleName},
					"matched_conditions": []string{string(asset.Type), asset.Name},
					"rule_weights": map[string]any{
						ruleName: 0.9,
					},
					"asset_type": asset.Type,
					"tags":       asset.Tags,
				},
			}, nil
		},
	})
}
