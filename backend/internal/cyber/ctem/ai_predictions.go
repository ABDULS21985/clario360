package ctem

import (
	"context"

	"github.com/clario360/platform/internal/aigovernance"
	"github.com/clario360/platform/internal/cyber/model"
)

func (e *CTEMEngine) recordPrioritizationPrediction(ctx context.Context, assessment *model.CTEMAssessment, findings []*model.CTEMFinding) {
	if e.predictionLogger == nil || assessment == nil || len(findings) == 0 {
		return
	}

	topFinding := findings[0]
	input := map[string]any{
		"assessment_id": assessment.ID.String(),
		"finding_count": len(findings),
		"asset_count":   assessment.ResolvedAssetCount,
	}
	_, _ = e.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     assessment.TenantID,
		ModelSlug:    "cyber-ctem-prioritizer",
		UseCase:      "ctem_prioritization",
		EntityType:   "ctem_assessment",
		EntityID:     &assessment.ID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"assessment_id":      assessment.ID,
					"finding_count":      len(findings),
					"top_finding_id":     topFinding.ID,
					"top_finding_title":  topFinding.Title,
					"top_priority_score": topFinding.PriorityScore,
				},
				Confidence: 0.9,
				Metadata: map[string]any{
					"overall_score": topFinding.PriorityScore,
					"component_scores": map[string]any{
						"impact":         topFinding.BusinessImpactScore,
						"exploitability": topFinding.ExploitabilityScore,
					},
					"component_weights": map[string]any{
						"impact":         0.55,
						"exploitability": 0.45,
					},
					"finding_count": len(findings),
					"top_finding":   topFinding.Title,
				},
			}, nil
		},
	})
}
