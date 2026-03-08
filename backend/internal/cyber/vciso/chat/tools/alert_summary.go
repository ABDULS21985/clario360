package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type AlertSummaryTool struct {
	baseTool
}

func NewAlertSummaryTool(deps *Dependencies) *AlertSummaryTool {
	return &AlertSummaryTool{baseTool: newBaseTool(deps)}
}

func (t *AlertSummaryTool) Name() string { return "alert_summary" }

func (t *AlertSummaryTool) Description() string { return "view security alert counts and recent alerts" }

func (t *AlertSummaryTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *AlertSummaryTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.AlertService == nil {
		return nil, fmt.Errorf("%w: alert service", errToolUnavailable)
	}
	count := t.parseCount(params, 5, 25)
	start, end := t.parseStartEnd(params, 7)
	var (
		startPtr *time.Time
		endPtr   *time.Time
	)
	if !start.IsZero() {
		startPtr = &start
	}
	if !end.IsZero() {
		endPtr = &end
	}
	severities := csvValues(params["severity"])
	statuses := csvValues(params["status"])
	listParams := alertListParams(severities, statuses, startPtr, endPtr, count)
	actor := t.actorFromContext(ctx, userID)

	result, err := t.deps.AlertService.ListAlerts(ctx, tenantID, listParams, actor)
	if err != nil {
		return nil, err
	}
	total, err := t.deps.AlertService.Count(ctx, tenantID, listParams, actor)
	if err != nil {
		return nil, err
	}
	stats, err := t.deps.AlertService.Stats(ctx, tenantID, actor)
	if err != nil {
		return nil, err
	}

	severityDesc := "recent"
	if len(severities) > 0 {
		severityDesc = normalizeSeveritySet(strings.Join(severities, ","))
	}
	lines := []string{
		fmt.Sprintf("You have **%d** %s %s between **%s** and **%s**. Here are the most recent:", total, severityDesc, maybePlural(total, "alert", "alerts"), start.Format("2006-01-02"), end.Format("2006-01-02")),
		"",
	}
	entities := make([]chatmodel.EntityReference, 0, len(result.Data))
	alerts := make([]map[string]any, 0, len(result.Data))
	for idx, alert := range result.Data {
		lines = append(lines, fmt.Sprintf("%d. %s **%s** — Confidence: %s, Status: %s (%s)", idx+1, formatSeverityIcon(string(alert.Severity)), alert.Title, confidencePercent(alert.ConfidenceScore), alert.Status, friendlyTimeAgo(t.now(), alert.CreatedAt)))
		entities = append(entities, entityRef("alert", alert.ID.String(), alert.Title, idx))
		alerts = append(alerts, map[string]any{
			"id":         alert.ID,
			"title":      alert.Title,
			"severity":   alert.Severity,
			"confidence": alert.ConfidenceScore,
			"status":     alert.Status,
			"created_at": alert.CreatedAt,
		})
	}
	if len(result.Data) == 0 {
		lines = append(lines, "No alerts matched those filters.")
	}

	criticalCount := 0
	for _, item := range stats.BySeverity {
		if strings.EqualFold(item.Name, "critical") {
			criticalCount = item.Count
			break
		}
	}
	lines = append(lines, "", fmt.Sprintf("%d are currently open. %d are critical severity.", stats.OpenCount, criticalCount))

	actions := []chatmodel.SuggestedAction{
		navigateAction("View all alerts", "/cyber/alerts"),
		messageAction("Show alert trend", "How has risk changed this week?"),
	}
	if len(result.Data) > 0 {
		actions = append(actions, messageAction("Investigate first alert", fmt.Sprintf("Investigate alert %s", result.Data[0].ID.String())))
	}

	return makeListResult(strings.Join(lines, "\n"), map[string]any{
		"total":        total,
		"alerts":       alerts,
		"by_severity":  stats.BySeverity,
		"by_status":    stats.ByStatus,
		"open_count":   stats.OpenCount,
		"critical_count": criticalCount,
	}, actions, entities), nil
}
