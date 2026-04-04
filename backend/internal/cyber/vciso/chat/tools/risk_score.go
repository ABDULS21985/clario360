package tools

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type RiskScoreTool struct {
	baseTool
}

func NewRiskScoreTool(deps *Dependencies) *RiskScoreTool {
	return &RiskScoreTool{baseTool: newBaseTool(deps)}
}

func (t *RiskScoreTool) Name() string { return "risk_score" }

func (t *RiskScoreTool) Description() string {
	return "view the organization's security risk score"
}

func (t *RiskScoreTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *RiskScoreTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, _ map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.RiskService == nil {
		return nil, fmt.Errorf("%w: risk service", errToolUnavailable)
	}
	score, err := t.deps.RiskService.GetCurrentScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	text := joinLines(
		fmt.Sprintf("Your current security risk score is **%.1f/100** (Grade **%s**).", score.OverallScore, score.Grade),
		"",
		"**Score Breakdown:**",
		fmt.Sprintf("- Vulnerability: %.1f/100", score.Components.VulnerabilityRisk.Score),
		fmt.Sprintf("- Threat Exposure: %.1f/100", score.Components.ThreatExposure.Score),
		fmt.Sprintf("- Configuration: %.1f/100", score.Components.ConfigurationRisk.Score),
		fmt.Sprintf("- Attack Surface: %.1f/100", score.Components.AttackSurfaceRisk.Score),
		fmt.Sprintf("- Compliance: %.1f/100", score.Components.ComplianceGapRisk.Score),
		"",
		fmt.Sprintf("The score has **%s** since the previous baseline (%+.1f points).", riskDirection(score.TrendDelta), score.TrendDelta),
	)
	data := map[string]any{
		"score": score.OverallScore,
		"grade": score.Grade,
		"breakdown": []map[string]any{
			{"name": "Vulnerability", "score": score.Components.VulnerabilityRisk.Score},
			{"name": "Threat Exposure", "score": score.Components.ThreatExposure.Score},
			{"name": "Configuration", "score": score.Components.ConfigurationRisk.Score},
			{"name": "Attack Surface", "score": score.Components.AttackSurfaceRisk.Score},
			{"name": "Compliance", "score": score.Components.ComplianceGapRisk.Score},
		},
		"trend_delta":     score.TrendDelta,
		"trend_direction": riskDirection(score.TrendDelta),
		"calculated_at":   score.CalculatedAt,
	}
	return &ToolResult{
		Text:     text,
		Data:     data,
		DataType: "kpi",
		Actions: []chatmodel.SuggestedAction{
			navigateAction("View full risk breakdown", "/cyber/risk"),
			messageAction("Show recommendations", "What should I focus on today?"),
			messageAction("Show top vulnerabilities", "Show top vulnerabilities"),
		},
		Entities: []chatmodel.EntityReference{},
	}, nil
}
