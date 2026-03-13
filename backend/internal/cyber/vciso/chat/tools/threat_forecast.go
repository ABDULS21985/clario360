package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type ThreatForecastTool struct {
	baseTool
}

func NewThreatForecastTool(deps *Dependencies) *ThreatForecastTool {
	return &ThreatForecastTool{baseTool: newBaseTool(deps)}
}

func (t *ThreatForecastTool) Name() string { return "get_threat_forecast" }

func (t *ThreatForecastTool) Description() string {
	return "get predicted alert volume, emerging attack techniques, and campaign trends"
}

func (t *ThreatForecastTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *ThreatForecastTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.PredictEngine == nil {
		return nil, fmt.Errorf("%w: predictive engine", errToolUnavailable)
	}
	forecastType := strings.TrimSpace(strings.ToLower(params["forecast_type"]))
	horizon := timeHorizonDays(params["time_horizon"], 7)
	switch forecastType {
	case "", "alert_volume":
		response, err := t.deps.PredictEngine.ForecastAlertVolume(ctx, tenantID, horizon)
		if err != nil {
			return nil, err
		}
		total := mapFloat64(response.Forecast.Summary, "predicted_total")
		text := fmt.Sprintf("Expected alert volume over the next %d days is %.0f alerts (P50), with confidence bounds from %.0f to %.0f. %s",
			horizon,
			total,
			response.ConfidenceInterval.P10,
			response.ConfidenceInterval.P90,
			response.ExplanationText,
		)
		return &ToolResult{
			Text:     text,
			Data:     response,
			DataType: "forecast",
			Actions: []chatmodel.SuggestedAction{
				messageAction("Show technique trends", "Show attack technique trends for the next 30 days"),
			},
		}, nil
	case "technique_trend":
		response, err := t.deps.PredictEngine.PredictTechniqueTrends(ctx, tenantID, horizon)
		if err != nil {
			return nil, err
		}
		lines := []string{fmt.Sprintf("Top predicted technique movements for the next %d days:", horizon), ""}
		entities := make([]chatmodel.EntityReference, 0, len(response.Items))
		for idx, item := range response.Items {
			lines = append(lines, fmt.Sprintf("%d. %s — %s (forecast %.1f, growth %.2f)", idx+1, item.TechniqueID, item.Trend, item.Forecast.P50, item.GrowthRate))
			entities = append(entities, entityRef("technique", item.TechniqueID, item.TechniqueID, idx))
			if idx == 4 {
				break
			}
		}
		return makeListResult(strings.Join(lines, "\n"), response, []chatmodel.SuggestedAction{
			messageAction("Show campaign clusters", "Are these alerts part of a coordinated attack?"),
		}, entities), nil
	case "campaign_detection":
		response, err := t.deps.PredictEngine.DetectCampaigns(ctx, tenantID, horizon)
		if err != nil {
			return nil, err
		}
		lines := []string{fmt.Sprintf("Detected %d candidate campaign clusters in the last %d days.", len(response.Items), horizon), ""}
		for idx, item := range response.Items {
			lines = append(lines, fmt.Sprintf("%d. %s — %d alerts, stage %s, confidence %.0f%%", idx+1, item.ClusterID, len(item.AlertIDs), item.Stage, item.Confidence.P50*100))
			if idx == 4 {
				break
			}
		}
		return &ToolResult{
			Text:     strings.Join(lines, "\n"),
			Data:     response,
			DataType: "forecast",
			Actions: []chatmodel.SuggestedAction{
				messageAction("Show alert forecast", "How many alerts should we expect this week?"),
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported forecast_type %q", forecastType)
	}
}

func timeHorizonDays(value string, fallback int) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "7", "7_days":
		return 7
	case "30", "30_days":
		return 30
	case "90", "90_days":
		return 90
	default:
		return fallback
	}
}
