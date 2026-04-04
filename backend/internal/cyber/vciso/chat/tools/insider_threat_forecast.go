package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type InsiderThreatForecastTool struct {
	baseTool
}

func NewInsiderThreatForecastTool(deps *Dependencies) *InsiderThreatForecastTool {
	return &InsiderThreatForecastTool{baseTool: newBaseTool(deps)}
}

func (t *InsiderThreatForecastTool) Name() string { return "get_insider_threat_forecast" }

func (t *InsiderThreatForecastTool) Description() string {
	return "get users whose behavioral risk scores are predicted to escalate"
}

func (t *InsiderThreatForecastTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *InsiderThreatForecastTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.PredictEngine == nil {
		return nil, fmt.Errorf("%w: predictive engine", errToolUnavailable)
	}
	horizon := timeHorizonDays(params["time_horizon"], 30)
	threshold := 70
	if raw := strings.TrimSpace(params["threshold"]); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			threshold = parsed
		}
	}
	response, err := t.deps.PredictEngine.ForecastInsiderThreats(ctx, tenantID, horizon, threshold)
	if err != nil {
		return nil, err
	}
	lines := []string{fmt.Sprintf("These users are on a trajectory to exceed risk threshold %d in the next %d days:", threshold, horizon), ""}
	entities := make([]chatmodel.EntityReference, 0, len(response.Items))
	rows := make([]map[string]any, 0, len(response.Items))
	for idx, item := range response.Items {
		name := item.EntityName
		if strings.TrimSpace(name) == "" {
			name = item.EntityID
		}
		timing := "threshold not reached"
		if item.DaysToThreshold != nil {
			timing = fmt.Sprintf("~%d days", *item.DaysToThreshold)
		}
		lines = append(lines, fmt.Sprintf("%d. %s — current %.1f, projected %.1f, threshold in %s", idx+1, name, item.CurrentRisk, item.ProjectedRisk, timing))
		entities = append(entities, entityRef("user", item.EntityID, name, idx))
		rows = append(rows, map[string]any{
			"entity_id":         item.EntityID,
			"entity_name":       name,
			"current_risk":      item.CurrentRisk,
			"projected_risk":    item.ProjectedRisk,
			"days_to_threshold": item.DaysToThreshold,
			"accelerating":      item.Accelerating,
		})
	}
	if len(rows) == 0 {
		lines = append(lines, "No users currently cross the configured threshold.")
	}
	return makeListResult(strings.Join(lines, "\n"), map[string]any{"items": rows, "meta": response.GenericPredictionResponse}, []chatmodel.SuggestedAction{
		navigateAction("Open UEBA dashboard", "/cyber/ueba"),
	}, entities), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
