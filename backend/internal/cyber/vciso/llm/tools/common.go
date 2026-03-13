package tools

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type Tool interface {
	Name() string
	Description() string
	RequiredPermissions() []string
	Schema() map[string]any
	IsDestructive() bool
	Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, args map[string]any) (*chattools.ToolResult, error)
}

type legacyToolAdapter struct {
	name        string
	description string
	permissions []string
	schema      map[string]any
	destructive bool
	delegate    chattools.Tool
	transform   func(map[string]any) map[string]string
}

func (t *legacyToolAdapter) Name() string                  { return t.name }
func (t *legacyToolAdapter) Description() string           { return t.description }
func (t *legacyToolAdapter) RequiredPermissions() []string { return t.permissions }
func (t *legacyToolAdapter) Schema() map[string]any        { return t.schema }
func (t *legacyToolAdapter) IsDestructive() bool           { return t.destructive }
func (t *legacyToolAdapter) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	if t.delegate == nil {
		return nil, fmt.Errorf("tool %s is unavailable", t.name)
	}
	params := map[string]string{}
	if t.transform != nil {
		params = t.transform(args)
	}
	result, err := t.delegate.Execute(ctx, tenantID, userID, params)
	if err != nil {
		return nil, err
	}
	return cloneResult(result), nil
}

func cloneResult(result *chattools.ToolResult) *chattools.ToolResult {
	if result == nil {
		return nil
	}
	return &chattools.ToolResult{
		Text:     result.Text,
		Data:     result.Data,
		DataType: result.DataType,
		Actions:  append([]chatmodel.SuggestedAction(nil), result.Actions...),
		Entities: append([]chatmodel.EntityReference(nil), result.Entities...),
	}
}

func normalizeTimeRange(value string) (time.Time, time.Time) {
	now := time.Now().UTC()
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, now
	case "yesterday":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return end.Add(-24 * time.Hour), end
	case "this_week":
		offset := int(now.Weekday())
		if offset == 0 {
			offset = 7
		}
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(offset - 1))
		return start, now
	case "last_week":
		end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return end.AddDate(0, 0, -7), end
	case "last_24_hours":
		return now.Add(-24 * time.Hour), now
	case "last_7_days":
		return now.AddDate(0, 0, -7), now
	case "last_30_days":
		return now.AddDate(0, 0, -30), now
	case "last_90_days":
		return now.AddDate(0, 0, -90), now
	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, now
	case "last_quarter":
		return now.AddDate(0, -3, 0), now
	default:
		return now.AddDate(0, 0, -7), now
	}
}

func applyTimeRange(params map[string]string, timeRange string) {
	if strings.TrimSpace(timeRange) == "" {
		return
	}
	start, end := normalizeTimeRange(timeRange)
	params["start_time"] = start.Format(time.RFC3339)
	params["end_time"] = end.Format(time.RFC3339)
}

// func stringArg(args map[string]any, key string) string {
// 	value, ok := args[key]
// 	if !ok || value == nil {
// 		return ""
// 	}
// 	switch typed := value.(type) {
// 	case string:
// 		return strings.TrimSpace(typed)
// 	case fmt.Stringer:
// 		return strings.TrimSpace(typed.String())
// 	default:
// 		payload, _ := json.Marshal(typed)
// 		return strings.Trim(strings.TrimSpace(string(payload)), `"`)
// 	}
// }

// func intArg(args map[string]any, key string, fallback int) int {
// 	value, ok := args[key]
// 	if !ok || value == nil {
// 		return fallback
// 	}
// 	switch typed := value.(type) {
// 	case int:
// 		return typed
// 	case float64:
// 		return int(typed)
// 	case string:
// 		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
// 			return parsed
// 		}
// 	}
// 	return fallback
// }

func boolArg(args map[string]any, key string, fallback ...bool) bool {
	defaultValue := false
	if len(fallback) > 0 {
		defaultValue = fallback[0]
	}
	value, ok := args[key]
	if !ok || value == nil {
		return defaultValue
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		parsed, _ := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed
	default:
		return defaultValue
	}
}

func stringSliceArg(args map[string]any, key string) []string {
	value, ok := args[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return cleanStrings(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := stringArg(map[string]any{"v": item}, "v"); text != "" {
				items = append(items, text)
			}
		}
		return cleanStrings(items)
	case string:
		return cleanStrings(strings.Split(typed, ","))
	default:
		return nil
	}
}

func cleanStrings(items []string) []string {
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func listResult(text, dataType string, data any, actions []chatmodel.SuggestedAction, entities []chatmodel.EntityReference) *chattools.ToolResult {
	if actions == nil {
		actions = []chatmodel.SuggestedAction{}
	}
	if entities == nil {
		entities = []chatmodel.EntityReference{}
	}
	return &chattools.ToolResult{
		Text:     text,
		Data:     data,
		DataType: dataType,
		Actions:  actions,
		Entities: entities,
	}
}

func requiredSchema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func timeRangeDescription() string {
	return "Supported values: today, yesterday, this_week, last_week, last_24_hours, last_7_days, last_30_days, last_90_days, this_month, last_quarter"
}

func buildAlertListParams(severities, statuses []string, timeRange string, limit int) *cyberdto.AlertListParams {
	start, end := normalizeTimeRange(timeRange)
	params := &cyberdto.AlertListParams{
		Severities: severities,
		Statuses:   statuses,
		DateFrom:   &start,
		DateTo:     &end,
		Page:       1,
		PerPage:    limit,
		Sort:       "created_at",
		Order:      "desc",
	}
	params.SetDefaults()
	return params
}

func entity(name, id, label string, index int) chatmodel.EntityReference {
	return chatmodel.EntityReference{Type: name, ID: id, Name: label, Index: index}
}

func severityWeight(severity string) float64 {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 1
	}
}

func statusIcon(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy", "good", "passed", "resolved":
		return "✅"
	case "critical", "error", "failing":
		return "🔴"
	case "high", "warning", "stalled":
		return "🟠"
	case "medium":
		return "🟡"
	default:
		return "🔵"
	}
}

func summarizeRows(rows []map[string]any) string {
	if len(rows) == 0 {
		return "No rows returned."
	}
	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		for _, key := range []string{"name", "title", "suite", "entity_name"} {
			if value := stringArg(row, key); value != "" {
				labels = append(labels, value)
				break
			}
		}
		if len(labels) == 3 {
			break
		}
	}
	return fmt.Sprintf("%d rows returned%s", len(rows), map[bool]string{true: "", false: ": " + strings.Join(labels, ", ")}[len(labels) == 0])
}

func sortRowsByScore(rows []map[string]any, key string) {
	sort.SliceStable(rows, func(i, j int) bool {
		return numeric(rows[i][key]) > numeric(rows[j][key])
	})
}

func numeric(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		return 0
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func detectIdentifier(value string) (assetName, assetIP string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}
	if net.ParseIP(value) != nil {
		return "", value
	}
	return value, ""
}
