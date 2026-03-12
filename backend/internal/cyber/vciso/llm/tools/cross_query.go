package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type CrossSuiteQuery struct {
	deps *chattools.Dependencies
}

func NewCrossSuiteQuery(deps *chattools.Dependencies) *CrossSuiteQuery {
	return &CrossSuiteQuery{deps: deps}
}

func (t *CrossSuiteQuery) Name() string { return "cross_suite_query" }
func (t *CrossSuiteQuery) Description() string {
	return "Query and compare data across multiple security suites (cyber, data, compliance, UEBA) in a single call."
}
func (t *CrossSuiteQuery) RequiredPermissions() []string {
	return []string{"cyber:read", "data:read", "lex:read", "acta:read"}
}
func (t *CrossSuiteQuery) IsDestructive() bool { return false }
func (t *CrossSuiteQuery) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"suites":     arrayOfStrings("Suites to compare", 2),
		"metric":     stringProp("What to compare, such as risk_level, alert_count, or health"),
		"time_range": stringProp(timeRangeDescription()),
	}, "suites", "metric")
}

func (t *CrossSuiteQuery) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	suites := stringSliceArg(args, "suites")
	if len(suites) < 2 {
		return nil, fmt.Errorf("at least two suites are required")
	}
	rows := make([]map[string]any, 0, len(suites))
	for _, suite := range suites {
		switch strings.ToLower(suite) {
		case "cyber":
			if t.deps != nil && t.deps.RiskService != nil {
				score, err := t.deps.RiskService.GetCurrentScore(ctx, tenantID)
				if err == nil && score != nil {
					rows = append(rows, map[string]any{
						"suite":     "Cyber",
						"status":    statusIcon(score.Grade),
						"score":     fmt.Sprintf("%.1f/100", score.OverallScore),
						"key_issue": firstContributor(score.TopContributors),
					})
				}
			}
		case "compliance":
			if t.deps != nil && t.deps.LexComplianceService != nil {
				score, err := t.deps.LexComplianceService.GetScore(ctx, tenantID)
				if err == nil && score != nil {
					rows = append(rows, map[string]any{
						"suite":     "Compliance",
						"status":    statusIcon("good"),
						"score":     fmt.Sprintf("%.1f/100", score.Score),
						"key_issue": fmt.Sprintf("%d open compliance alerts", score.OpenAlerts),
					})
				}
			}
		case "ueba":
			if t.deps != nil && t.deps.UEBAService != nil {
				items, err := t.deps.UEBAService.GetRiskRanking(ctx, tenantID, 3)
				if err == nil {
					issue := "No elevated entities"
					score := "0/100"
					if len(items) > 0 {
						issue = fmt.Sprintf("%d entity(ies) above 70", countHighRisk(items))
						score = fmt.Sprintf("%.1f/100", items[0].RiskScore)
					}
					rows = append(rows, map[string]any{
						"suite":     "UEBA",
						"status":    statusIcon("medium"),
						"score":     score,
						"key_issue": issue,
					})
				}
			}
		case "data", "pipelines":
			if t.deps != nil && t.deps.DataPipelineRepo != nil {
				items, _, err := t.deps.DataPipelineRepo.List(ctx, tenantID, chattools.ListPipelinesParamsCompat(5))
				if err == nil {
					failing := 0
					for _, item := range items {
						if strings.EqualFold(string(item.Status), "error") || (item.LastRunStatus != nil && strings.EqualFold(*item.LastRunStatus, "failed")) {
							failing++
						}
					}
					rows = append(rows, map[string]any{
						"suite":     "Pipelines",
						"status":    statusIcon(map[bool]string{true: "failing", false: "healthy"}[failing > 0]),
						"score":     fmt.Sprintf("%d/%d healthy", max(len(items)-failing, 0), len(items)),
						"key_issue": fmt.Sprintf("%d failing pipelines", failing),
					})
				}
			}
		case "assets":
			if t.deps != nil && t.deps.AssetService != nil {
				stats, err := t.deps.AssetService.GetStats(ctx, tenantID)
				if err == nil && stats != nil {
					rows = append(rows, map[string]any{
						"suite":     "Assets",
						"status":    statusIcon("medium"),
						"score":     fmt.Sprintf("%d total", stats.Total),
						"key_issue": fmt.Sprintf("%d assets with vulnerabilities", stats.AssetsWithVulns),
					})
				}
			}
		}
	}
	return listResult(
		fmt.Sprintf("Compared %d suites.", len(rows)),
		"table",
		map[string]any{"rows": rows},
		[]chatmodel.SuggestedAction{{Label: "Open cyber dashboard", Type: "navigate", Params: map[string]string{"url": "/cyber"}}},
		nil,
	), nil
}

func firstContributor(items []any) string {
	if len(items) == 0 {
		return "No major contributor"
	}
	return "Top contributor present"
}
