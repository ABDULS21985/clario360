package risk

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/aigovernance"
	"github.com/clario360/platform/internal/cyber/model"
)

func (rs *RiskScorer) recordPrediction(ctx context.Context, tenantID uuid.UUID, score *model.OrganizationRiskScore) {
	if rs.predictionLogger == nil || score == nil {
		return
	}

	input := map[string]any{
		"tenant_id":            tenantID.String(),
		"total_assets":         score.Context.TotalAssets,
		"open_alerts":          score.Context.TotalOpenAlerts,
		"open_vulnerabilities": score.Context.TotalOpenVulns,
		"active_threats":       score.Context.TotalActiveThreats,
	}
	tenantEntityID := tenantID
	_, _ = rs.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     tenantID,
		ModelSlug:    "cyber-risk-scorer",
		UseCase:      "risk_scoring",
		EntityType:   "organization",
		EntityID:     &tenantEntityID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output:     score,
				Confidence: 0.92,
				Metadata: map[string]any{
					"overall_score": score.OverallScore,
					"component_scores": map[string]any{
						"vulnerability": score.Components.VulnerabilityRisk.Score,
						"threat":        score.Components.ThreatExposure.Score,
						"configuration": score.Components.ConfigurationRisk.Score,
						"surface":       score.Components.AttackSurfaceRisk.Score,
						"compliance":    score.Components.ComplianceGapRisk.Score,
					},
					"component_weights": map[string]any{
						"vulnerability": score.Components.VulnerabilityRisk.Weight,
						"threat":        score.Components.ThreatExposure.Weight,
						"configuration": score.Components.ConfigurationRisk.Weight,
						"surface":       score.Components.AttackSurfaceRisk.Weight,
						"compliance":    score.Components.ComplianceGapRisk.Weight,
					},
					"grade": score.Grade,
				},
			}, nil
		},
	})
}
