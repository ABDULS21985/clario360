package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type MITRECoverageTool struct {
	baseTool
}

func NewMITRECoverageTool(deps *Dependencies) *MITRECoverageTool {
	return &MITRECoverageTool{baseTool: newBaseTool(deps)}
}

func (t *MITRECoverageTool) Name() string { return "mitre_coverage" }

func (t *MITRECoverageTool) Description() string {
	return "check MITRE ATT&CK detection coverage and gaps"
}

func (t *MITRECoverageTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *MITRECoverageTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, _ map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.RuleService == nil {
		return nil, fmt.Errorf("%w: rule service", errToolUnavailable)
	}
	coverage, err := t.deps.RuleService.Coverage(ctx, tenantID, t.actorFromContext(ctx, userID))
	if err != nil {
		return nil, err
	}
	total := len(coverage)
	covered := 0
	gaps := make([]map[string]any, 0, 5)
	for _, item := range coverage {
		if item.HasDetection {
			covered++
			continue
		}
		if len(gaps) < 5 {
			gaps = append(gaps, map[string]any{
				"id":     item.TechniqueID,
				"name":   item.TechniqueName,
				"tactic": item.TacticIDs,
			})
		}
	}
	percent := 0.0
	if total > 0 {
		percent = (float64(covered) / float64(total)) * 100
	}
	lines := []string{
		fmt.Sprintf("Current MITRE ATT&CK coverage is **%.1f%%**.", percent),
		fmt.Sprintf("Covered techniques: **%d/%d**.", covered, total),
	}
	if len(gaps) > 0 {
		lines = append(lines, "", "Top gaps:")
		for _, item := range gaps {
			lines = append(lines, fmt.Sprintf("- **%s** (%s)", item["name"], item["id"]))
		}
	}
	return &ToolResult{
		Text: strings.Join(lines, "\n"),
		Data: map[string]any{
			"coverage_percent": percent,
			"covered":          covered,
			"total":            total,
			"gaps":             gaps,
		},
		DataType: "chart",
		Actions: []chatmodel.SuggestedAction{
			navigateAction("View MITRE coverage", "/cyber"),
			messageAction("Show recommendations", "What should I focus on today?"),
		},
		Entities: []chatmodel.EntityReference{},
	}, nil
}
