package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type TrendAnalysisTool struct {
	baseTool
}

func NewTrendAnalysisTool(deps *Dependencies) *TrendAnalysisTool {
	return &TrendAnalysisTool{baseTool: newBaseTool(deps)}
}

func (t *TrendAnalysisTool) Name() string { return "trend_analysis" }

func (t *TrendAnalysisTool) Description() string {
	return "analyze security trends and how metrics have changed"
}

func (t *TrendAnalysisTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *TrendAnalysisTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.RiskService == nil || t.deps.AlertService == nil {
		return nil, fmt.Errorf("%w: trend dependencies", errToolUnavailable)
	}
	start, end := t.parseStartEnd(params, 30)
	days := int(end.Sub(start).Hours()/24) + 1
	if days < 2 {
		days = 7
	}
	trend, err := t.deps.RiskService.Trend(ctx, tenantID, days)
	if err != nil {
		return nil, err
	}
	actor := t.actorFromContext(ctx, userID)
	alertCount, err := t.deps.AlertService.Count(ctx, tenantID, alertListParams(nil, nil, &start, &end, 1), actor)
	if err != nil {
		return nil, err
	}
	lines := []string{
		fmt.Sprintf("Security trends from **%s** to **%s**:", start.Format("2006-01-02"), end.Format("2006-01-02")),
	}
	if len(trend) >= 2 {
		delta := trend[len(trend)-1].OverallScore - trend[0].OverallScore
		lines = append(lines, "", fmt.Sprintf("Risk score moved from **%.1f** to **%.1f** (%+.1f).", trend[0].OverallScore, trend[len(trend)-1].OverallScore, delta))
	}
	lines = append(lines, fmt.Sprintf("Alert volume in the same window: **%d**.", alertCount))
	return &ToolResult{
		Text: strings.Join(lines, "\n"),
		Data: map[string]any{
			"risk_trend":   trend,
			"alert_count":  alertCount,
			"window_start": start,
			"window_end":   end,
		},
		DataType: "chart",
		Actions: []chatmodel.SuggestedAction{
			messageAction("Show critical alerts", "Show critical alerts"),
			messageAction("What should I focus on?", "What should I focus on today?"),
		},
		Entities: nil,
	}, nil
}
