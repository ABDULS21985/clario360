package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type UEBASummaryTool struct {
	baseTool
}

func NewUEBASummaryTool(deps *Dependencies) *UEBASummaryTool {
	return &UEBASummaryTool{baseTool: newBaseTool(deps)}
}

func (t *UEBASummaryTool) Name() string { return "ueba_summary" }

func (t *UEBASummaryTool) Description() string {
	return "view users and entities with anomalous behavioral patterns"
}

func (t *UEBASummaryTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *UEBASummaryTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.UEBAService == nil {
		return nil, fmt.Errorf("%w: ueba service", errToolUnavailable)
	}
	limit := t.parseCount(params, 5, 20)
	items, err := t.deps.UEBAService.GetRiskRanking(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	lines := []string{fmt.Sprintf("These are the riskiest UEBA entities right now (top %d):", len(items)), ""}
	entities := make([]chatmodel.EntityReference, 0, len(items))
	rows := make([]map[string]any, 0, len(items))
	for idx, item := range items {
		name := item.EntityName
		if strings.TrimSpace(name) == "" {
			name = item.EntityID
		}
		lines = append(lines, fmt.Sprintf("%d. %s **%s** — Risk **%.1f/100**, %s maturity, %d alerts in 7d", idx+1, formatSeverityIcon(item.RiskLevel), name, item.RiskScore, item.ProfileMaturity, item.AlertCount7D))
		entities = append(entities, entityRef("user", item.EntityID, name, idx))
		rows = append(rows, map[string]any{
			"entity_id":        item.EntityID,
			"entity_name":      name,
			"entity_type":      item.EntityType,
			"risk_score":       item.RiskScore,
			"risk_level":       item.RiskLevel,
			"alert_count_7d":   item.AlertCount7D,
			"profile_maturity": item.ProfileMaturity,
		})
	}
	if len(items) == 0 {
		lines = append(lines, "No high-risk entities are currently ranked.")
	}
	return makeListResult(strings.Join(lines, "\n"), map[string]any{"items": rows}, []chatmodel.SuggestedAction{
		navigateAction("Open UEBA dashboard", "/cyber/ueba"),
		messageAction("Show UEBA alerts", "Show critical alerts"),
	}, entities), nil
}

