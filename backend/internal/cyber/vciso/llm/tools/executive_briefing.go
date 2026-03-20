package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type ExecutiveBriefingTool struct {
	deps *chattools.Dependencies
}

func NewExecutiveBriefingTool(deps *chattools.Dependencies) *ExecutiveBriefingTool {
	return &ExecutiveBriefingTool{deps: deps}
}

func (t *ExecutiveBriefingTool) Name() string { return "executive_briefing" }
func (t *ExecutiveBriefingTool) Description() string {
	return "Generate a concise executive briefing covering overall posture, top risks, key metrics, and recommended actions."
}
func (t *ExecutiveBriefingTool) RequiredPermissions() []string {
	return []string{"cyber:read", "lex:read", "acta:read"}
}
func (t *ExecutiveBriefingTool) IsDestructive() bool { return false }
func (t *ExecutiveBriefingTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"format":      enumString("narrative", "bullet_points", "kpi_card"),
		"focus_areas": arrayOfStrings("Optional focus areas such as risk or compliance", 0),
		"max_length":  enumString("brief", "standard", "detailed"),
	})
}

func (t *ExecutiveBriefingTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	if t.deps == nil || t.deps.RiskService == nil {
		return nil, fmt.Errorf("risk service is unavailable")
	}
	score, err := t.deps.RiskService.GetCurrentScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	alertTotal := 0
	criticalTotal := 0
	if t.deps.AlertService != nil {
		alertParams := buildAlertListParams([]string{"critical"}, []string{"new", "acknowledged", "investigating"}, "this_week", 10)
		alerts, err := t.deps.AlertService.ListAlerts(ctx, tenantID, alertParams, nil)
		if err == nil && alerts != nil {
			alertTotal = alerts.Meta.Total
			criticalTotal = len(alerts.Data)
		}
	}
	complianceScore := 0.0
	if t.deps.LexComplianceService != nil {
		if compliance, err := t.deps.LexComplianceService.GetScore(ctx, tenantID); err == nil && compliance != nil {
			complianceScore = compliance.Score
		}
	}
	recommendation := "Reduce critical exposure by clearing the top unresolved alert backlog."
	if len(score.Recommendations) > 0 {
		recommendation = score.Recommendations[0].Title
	}

	format := strings.ToLower(stringArg(args, "format"))
	maxLength := strings.ToLower(stringArg(args, "max_length"))
	if format == "" {
		format = "narrative"
	}
	if maxLength == "" {
		maxLength = "standard"
	}

	switch format {
	case "bullet_points":
		items := []string{
			fmt.Sprintf("Overall posture: %.1f/100 (grade %s)", score.OverallScore, score.Grade),
			fmt.Sprintf("Critical alerts this week: %d", criticalTotal),
			fmt.Sprintf("Compliance posture: %.1f/100", complianceScore),
			"Recommended action: " + recommendation,
		}
		return listResult(strings.Join(items, "\n"), "list", map[string]any{"items": items}, []chatmodel.SuggestedAction{}, nil), nil
	case "kpi_card":
		return listResult("Executive KPIs prepared.", "kpi", map[string]any{
			"kpis": []map[string]any{
				{"label": "Risk score", "value": fmt.Sprintf("%.1f/100", score.OverallScore)},
				{"label": "Grade", "value": score.Grade},
				{"label": "Critical alerts", "value": criticalTotal},
				{"label": "Compliance", "value": fmt.Sprintf("%.1f/100", complianceScore)},
			},
		}, []chatmodel.SuggestedAction{}, nil), nil
	default:
		text := fmt.Sprintf("Your organization is operating at **%.1f/100 (%s)**. There are **%d critical alerts** active this week, compliance is **%.1f/100**, and the highest-value next move is to **%s**.", score.OverallScore, score.Grade, alertTotal, complianceScore, recommendation)
		if maxLength == "brief" {
			text = fmt.Sprintf("Posture is **%.1f/100 (%s)**. Critical alerts: **%d**. Compliance: **%.1f/100**. Next action: **%s**.", score.OverallScore, score.Grade, alertTotal, complianceScore, recommendation)
		}
		return listResult(text, "text", map[string]any{
			"risk_score":      score.OverallScore,
			"grade":           score.Grade,
			"critical_alerts": alertTotal,
			"compliance":      complianceScore,
			"recommendation":  recommendation,
		}, []chatmodel.SuggestedAction{{Label: "Open vCISO dashboard", Type: "navigate", Params: map[string]string{"url": "/cyber/vciso"}}}, nil), nil
	}
}
