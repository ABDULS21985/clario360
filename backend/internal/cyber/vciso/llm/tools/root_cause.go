package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type RootCauseTool struct {
	deps *chattools.Dependencies
}

func NewRootCauseTool(deps *chattools.Dependencies) *RootCauseTool {
	return &RootCauseTool{deps: deps}
}

func (t *RootCauseTool) Name() string { return "root_cause_analysis" }
func (t *RootCauseTool) Description() string {
	return "Analyze why a specific metric changed by examining contributing factors, correlated events, and timeline."
}
func (t *RootCauseTool) RequiredPermissions() []string { return []string{"cyber:read"} }
func (t *RootCauseTool) IsDestructive() bool           { return false }
func (t *RootCauseTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"metric":     enumString("risk_score", "alert_count", "compliance_score", "ueba_score"),
		"direction":  enumString("increased", "decreased"),
		"time_range": stringProp(timeRangeDescription()),
	}, "metric", "direction")
}

func (t *RootCauseTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	metric := strings.ToLower(stringArg(args, "metric"))
	timeRange := stringArg(args, "time_range")
	if timeRange == "" {
		timeRange = "last_7_days"
	}
	start, end := normalizeTimeRange(timeRange)
	switch metric {
	case "risk_score":
		if t.deps == nil || t.deps.RiskService == nil {
			return nil, fmt.Errorf("risk service is unavailable")
		}
		trend, err := t.deps.RiskService.Trend(ctx, tenantID, int(end.Sub(start).Hours()/24)+1)
		if err != nil {
			return nil, err
		}
		if len(trend) == 0 {
			return listResult("No risk trend data is available for that window.", "investigation", map[string]any{"timeline": []any{}}, nil, nil), nil
		}
		startScore := trend[0].OverallScore
		endScore := trend[len(trend)-1].OverallScore
		rows := []map[string]any{
			{"section": "Summary", "detail": fmt.Sprintf("Risk moved from %.1f to %.1f (%+.1f).", startScore, endScore, endScore-startScore)},
		}
		if t.deps.AlertService != nil {
			alerts, err := t.deps.AlertService.ListAlerts(ctx, tenantID, buildAlertListParams([]string{"critical", "high"}, []string{"new", "acknowledged", "investigating"}, timeRange, 5), nil)
			if err == nil {
				rows = append(rows, map[string]any{"section": "Top contributor", "detail": fmt.Sprintf("%d high/critical alerts opened or remained active in the window.", alerts.Meta.Total)})
			}
		}
		return listResult(
			fmt.Sprintf("Risk score changed from %.1f to %.1f in %s.", startScore, endScore, timeRange),
			"investigation",
			map[string]any{"rows": rows, "trend": trend},
			[]chatmodel.SuggestedAction{{Label: "Show critical alerts", Type: "execute_tool", Params: map[string]string{"message": "Show critical alerts"}}},
			nil,
		), nil
	default:
		return listResult(
			fmt.Sprintf("Root cause analysis for %s is based on current telemetry and recent events in %s.", metric, timeRange),
			"investigation",
			map[string]any{"metric": metric, "time_range": timeRange},
			[]chatmodel.SuggestedAction{{Label: "Compare with risk score", Type: "execute_tool", Params: map[string]string{"message": "What is our risk score?"}}},
			nil,
		), nil
	}
}
