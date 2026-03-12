package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type NaturalLanguageFilterTool struct {
	deps *chattools.Dependencies
}

var relativeHoursPattern = regexp.MustCompile(`(?i)last\s+(\d+)\s+hours?`)

func NewNaturalLanguageFilterTool(deps *chattools.Dependencies) *NaturalLanguageFilterTool {
	return &NaturalLanguageFilterTool{deps: deps}
}

func (t *NaturalLanguageFilterTool) Name() string { return "natural_language_filter" }
func (t *NaturalLanguageFilterTool) Description() string {
	return "Translate a complex natural language filter into structured query parameters."
}
func (t *NaturalLanguageFilterTool) RequiredPermissions() []string { return []string{"cyber:read"} }
func (t *NaturalLanguageFilterTool) IsDestructive() bool           { return false }
func (t *NaturalLanguageFilterTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"entity_type": enumString("alerts", "assets", "vulnerabilities", "users"),
		"filter_text": stringProp("Complex filter text"),
	}, "entity_type", "filter_text")
}

func (t *NaturalLanguageFilterTool) Execute(_ context.Context, _ uuid.UUID, _ uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	filterText := strings.ToLower(stringArg(args, "filter_text"))
	params := map[string]any{
		"entity_type": strings.ToLower(stringArg(args, "entity_type")),
	}
	if strings.Contains(filterText, "critical") {
		params["severity"] = "critical"
	}
	if strings.Contains(filterText, "high") && params["severity"] == nil {
		params["severity"] = "high"
	}
	for _, team := range []string{"finance", "engineering", "sales", "hr"} {
		if strings.Contains(filterText, team) {
			params["asset_group"] = team
			break
		}
	}
	if matches := relativeHoursPattern.FindStringSubmatch(filterText); len(matches) == 2 {
		hours := intArg(map[string]any{"h": matches[1]}, "h", 48)
		params["time_range"] = map[string]any{
			"start": time.Now().UTC().Add(-1 * time.Duration(hours) * time.Hour).Format(time.RFC3339),
			"end":   time.Now().UTC().Format(time.RFC3339),
		}
	}
	if strings.Contains(filterText, "server") {
		params["asset_type"] = "server"
	}
	return listResult(
		fmt.Sprintf("Translated the filter into %d structured fields.", len(params)),
		"list",
		map[string]any{"filters": params},
		nil,
		nil,
	), nil
}
