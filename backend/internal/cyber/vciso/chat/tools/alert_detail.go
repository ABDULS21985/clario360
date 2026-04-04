package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type AlertDetailTool struct {
	baseTool
}

func NewAlertDetailTool(deps *Dependencies) *AlertDetailTool {
	return &AlertDetailTool{baseTool: newBaseTool(deps)}
}

func (t *AlertDetailTool) Name() string { return "alert_detail" }

func (t *AlertDetailTool) Description() string {
	return "get detailed information about a specific alert"
}

func (t *AlertDetailTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *AlertDetailTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.AlertService == nil {
		return nil, fmt.Errorf("%w: alert service", errToolUnavailable)
	}
	alertID, err := t.requireCyberAlertID(ctx, tenantID, params["alert_id"])
	if err != nil {
		return nil, err
	}
	actor := t.actorFromContext(ctx, userID)
	alert, err := t.deps.AlertService.GetAlert(ctx, tenantID, alertID, actor)
	if err != nil {
		return nil, err
	}
	related, _ := t.deps.AlertService.Related(ctx, tenantID, alertID, actor)
	lines := []string{
		fmt.Sprintf("## %s %s", formatSeverityIcon(string(alert.Severity)), alert.Title),
		"",
		fmt.Sprintf("**Severity:** %s", alert.Severity),
		fmt.Sprintf("**Confidence:** %s", confidencePercent(alert.ConfidenceScore)),
		fmt.Sprintf("**Status:** %s", alert.Status),
		fmt.Sprintf("**Detected:** %s", alert.CreatedAt.Format(time.RFC3339)),
		"",
		"### What Happened",
		alert.Explanation.Summary,
	}
	if len(alert.Explanation.ConfidenceFactors) > 0 {
		lines = append(lines, "", "### Confidence Factors")
		for _, factor := range alert.Explanation.ConfidenceFactors {
			lines = append(lines, fmt.Sprintf("- %s (%+.1f): %s", factor.Factor, factor.Impact, factor.Description))
		}
	}
	if len(alert.Explanation.MatchedConditions) > 0 {
		lines = append(lines, "", "### Matched Conditions")
		for _, condition := range alert.Explanation.MatchedConditions {
			lines = append(lines, "- "+condition)
		}
	}
	if len(related) > 0 {
		lines = append(lines, "", fmt.Sprintf("### Related Alerts (%d)", len(related)))
		for _, item := range related {
			lines = append(lines, fmt.Sprintf("- %s %s (%s)", formatSeverityIcon(string(item.Severity)), item.Title, item.Status))
		}
	}
	actions := []chatmodel.SuggestedAction{
		messageAction("Investigate this alert", fmt.Sprintf("Investigate alert %s", alert.ID.String())),
		messageAction("Start remediation", fmt.Sprintf("Start remediation for alert %s", alert.ID.String())),
		navigateAction("View in dashboard", "/cyber/alerts/"+alert.ID.String()),
	}
	return &ToolResult{
		Text:     strings.Join(lines, "\n"),
		Data:     map[string]any{"alert": alert, "related_alerts": related},
		DataType: "investigation",
		Actions:  actions,
		Entities: []chatmodel.EntityReference{entityRef("alert", alert.ID.String(), alert.Title, 0)},
	}, nil
}
