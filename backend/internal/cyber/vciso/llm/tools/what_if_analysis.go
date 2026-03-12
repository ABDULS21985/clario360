package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type WhatIfAnalysisTool struct {
	deps *chattools.Dependencies
}

func NewWhatIfAnalysisTool(deps *chattools.Dependencies) *WhatIfAnalysisTool {
	return &WhatIfAnalysisTool{deps: deps}
}

func (t *WhatIfAnalysisTool) Name() string { return "what_if_analysis" }
func (t *WhatIfAnalysisTool) Description() string {
	return "Run a hypothetical scenario analysis and calculate projected impact on risk score, compliance, and alert volume."
}
func (t *WhatIfAnalysisTool) RequiredPermissions() []string { return []string{"cyber:read"} }
func (t *WhatIfAnalysisTool) IsDestructive() bool           { return false }
func (t *WhatIfAnalysisTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"scenario":      stringProp("Natural language scenario"),
		"scenario_type": enumString("defer_remediation", "disable_control", "add_asset", "remove_asset", "change_policy"),
		"target_ids":    arrayOfStrings("Optional target IDs", 0),
		"duration_days": intProp("Scenario duration in days", 1, 365),
	}, "scenario", "scenario_type")
}

func (t *WhatIfAnalysisTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	if t.deps == nil || t.deps.RiskService == nil {
		return nil, fmt.Errorf("risk service is unavailable")
	}
	score, err := t.deps.RiskService.GetCurrentScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	scenarioType := strings.ToLower(stringArg(args, "scenario_type"))
	durationDays := intArg(args, "duration_days", 14)
	targetIDs := stringSliceArg(args, "target_ids")

	riskDelta := 0.0
	complianceDelta := 0.0
	escalationProbability := 0.05
	switch scenarioType {
	case "defer_remediation":
		for _, targetID := range targetIDs {
			if t.deps.AlertService == nil {
				continue
			}
			alertID, err := uuid.Parse(strings.TrimSpace(targetID))
			if err != nil {
				continue
			}
			alert, err := t.deps.AlertService.GetAlert(ctx, tenantID, alertID, nil)
			if err != nil {
				continue
			}
			riskDelta += severityWeight(string(alert.Severity)) * float64(durationDays) * 0.5
			complianceDelta -= severityWeight(string(alert.Severity)) * 0.4
			escalationProbability += 0.05 * severityWeight(string(alert.Severity))
		}
	case "disable_control":
		riskDelta = float64(durationDays) * 0.3
		complianceDelta = -6
		escalationProbability = 0.3
	case "add_asset":
		riskDelta = 2 + float64(durationDays)*0.05
		escalationProbability = 0.12
	case "remove_asset":
		riskDelta = -1.5
	case "change_policy":
		riskDelta = 1.2
		complianceDelta = -2.5
	}
	projectedRisk := maxFloat(score.OverallScore+riskDelta, 0)
	if projectedRisk > 100 {
		projectedRisk = 100
	}
	rows := []map[string]any{
		{"metric": "Risk score", "current": fmt.Sprintf("%.1f/100", score.OverallScore), "projected": fmt.Sprintf("%.1f/100", projectedRisk), "delta": fmt.Sprintf("%+.1f", riskDelta)},
		{"metric": "Compliance posture", "current": "current baseline", "projected": fmt.Sprintf("%+.1f points", complianceDelta), "delta": fmt.Sprintf("%+.1f", complianceDelta)},
		{"metric": "Escalation probability", "current": "baseline", "projected": fmt.Sprintf("%.0f%%", escalationProbability*100), "delta": "projected"},
	}
	return listResult(
		fmt.Sprintf("Scenario `%s` projects the largest impact on risk score (%+.1f).", scenarioType, riskDelta),
		"table",
		map[string]any{"rows": rows, "scenario": stringArg(args, "scenario")},
		[]chatmodel.SuggestedAction{{Label: "Review critical alerts", Type: "execute_tool", Params: map[string]string{"message": "Show critical alerts"}}},
		nil,
	), nil
}
