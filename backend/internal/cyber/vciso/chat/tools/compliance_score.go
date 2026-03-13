package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type ComplianceScoreTool struct {
	baseTool
}

func NewComplianceScoreTool(deps *Dependencies) *ComplianceScoreTool {
	return &ComplianceScoreTool{baseTool: newBaseTool(deps)}
}

func (t *ComplianceScoreTool) Name() string { return "compliance_score" }

func (t *ComplianceScoreTool) Description() string {
	return "check compliance status across regulatory frameworks"
}

func (t *ComplianceScoreTool) RequiredPermissions() []string {
	return []string{"lex:read", "acta:read"}
}

func (t *ComplianceScoreTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, params map[string]string) (*ToolResult, error) {
	var (
		components []map[string]any
		total      float64
		count      float64
		notes      []string
	)
	if t.deps != nil && t.deps.LexComplianceService != nil {
		score, err := t.deps.LexComplianceService.GetScore(ctx, tenantID)
		if err == nil {
			components = append(components, map[string]any{
				"name":          "Legal and contract compliance",
				"score":         score.Score,
				"open_alerts":   score.OpenAlerts,
				"calculated_at": score.CalculatedAt,
			})
			total += score.Score
			count++
		}
	}
	if t.deps != nil && t.deps.ActaComplianceService != nil {
		score, err := t.deps.ActaComplianceService.Score(ctx, tenantID)
		if err == nil {
			components = append(components, map[string]any{
				"name":  "Governance process compliance",
				"score": score,
			})
			total += score
			count++
		}
	}
	if len(components) == 0 {
		return nil, fmt.Errorf("%w: compliance services", errToolUnavailable)
	}
	framework := strings.TrimSpace(params["framework"])
	if framework != "" {
		notes = append(notes, fmt.Sprintf("Framework-specific mapping for **%s** is not stored as a standalone score yet, so this uses the tenant's current governed compliance posture.", framework))
	}
	overall := total / count
	lines := []string{
		fmt.Sprintf("Current compliance posture is **%.1f/100**.", overall),
		"",
		"**Component Scores:**",
	}
	for _, component := range components {
		lines = append(lines, fmt.Sprintf("- %s: **%.1f/100**", component["name"], component["score"]))
	}
	if len(notes) > 0 {
		lines = append(lines, "", strings.Join(notes, " "))
	}
	return &ToolResult{
		Text: strings.Join(lines, "\n"),
		Data: map[string]any{
			"score":      overall,
			"framework":  framework,
			"components": components,
			"notes":      notes,
		},
		DataType: "kpi",
		Actions: []chatmodel.SuggestedAction{
			navigateAction("Open compliance workspace", "/lex/compliance"),
			messageAction("Generate executive report", "Generate executive report"),
		},
		Entities: nil,
	}, nil
}
