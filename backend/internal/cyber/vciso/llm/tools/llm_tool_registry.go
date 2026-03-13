package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type Registry struct {
	items map[string]Tool
}

func NewRegistry(deps *chattools.Dependencies) *Registry {
	legacy := chattools.NewRegistry(deps)
	items := map[string]Tool{}
	register := func(tool Tool) {
		items[tool.Name()] = tool
	}

	register(wrapLegacy("get_risk_score", "Get the organization's current security risk score, grade, and top risk contributors.", legacy.Get("risk_score"), nil, nil, false))
	register(wrapLegacy("get_alerts", "Get security alerts filtered by severity, time range, and status.", legacy.Get("alert_summary"), func(args map[string]any) map[string]string {
		params := map[string]string{
			"severity": stringArg(args, "severity"),
			"status":   stringArg(args, "status"),
			"count":    strconvItoa(intArg(args, "limit", 5)),
		}
		applyTimeRange(params, stringArg(args, "time_range"))
		return params
	}, requiredSchema(map[string]any{
		"severity":   enumString("critical", "high", "medium", "low"),
		"time_range": stringProp(timeRangeDescription()),
		"status":     enumString("open", "acknowledged", "resolved", "all"),
		"limit":      intProp("Maximum alerts to return", 1, 50),
	}), false))
	register(wrapLegacy("get_alert_detail", "Get detailed information about a specific alert including explanation and affected assets.", legacy.Get("alert_detail"), func(args map[string]any) map[string]string {
		return map[string]string{"alert_id": stringArg(args, "alert_id")}
	}, requiredSchema(map[string]any{"alert_id": stringProp("Alert UUID or legacy ID")}, "alert_id"), false))
	register(wrapLegacy("get_asset_info", "Look up details about a specific asset (server, device, endpoint) by name or IP.", legacy.Get("asset_lookup"), func(args map[string]any) map[string]string {
		name, ip := detectIdentifier(stringArg(args, "identifier"))
		return map[string]string{"asset_name": name, "asset_ip": ip}
	}, requiredSchema(map[string]any{"identifier": stringProp("Hostname, IP address, or asset ID")}, "identifier"), false))
	register(wrapLegacy("get_vulnerabilities", "Get top vulnerabilities sorted by severity and exploitability.", legacy.Get("vulnerability_summary"), func(args map[string]any) map[string]string {
		return map[string]string{
			"severity": stringArg(args, "severity"),
			"count":    strconvItoa(intArg(args, "limit", 10)),
		}
	}, requiredSchema(map[string]any{
		"severity": enumString("critical", "high", "medium", "low"),
		"limit":    intProp("Maximum vulnerabilities to return", 1, 50),
	}), false))
	register(wrapLegacy("get_mitre_coverage", "Get MITRE ATT&CK framework detection coverage, showing covered and uncovered techniques.", legacy.Get("mitre_coverage"), func(args map[string]any) map[string]string {
		return map[string]string{"tactic": stringArg(args, "tactic")}
	}, requiredSchema(map[string]any{"tactic": stringProp("Optional MITRE tactic name")}), false))
	register(&RiskyUsersTool{deps: deps})
	register(wrapLegacy("get_pipeline_status", "Get data pipeline health status including failing, stalled, and healthy pipelines.", legacy.Get("pipeline_status"), func(args map[string]any) map[string]string {
		return map[string]string{"count": strconvItoa(intArg(args, "limit", 10))}
	}, requiredSchema(map[string]any{
		"status": enumString("failing", "stalled", "healthy", "all"),
		"limit":  intProp("Maximum pipelines to return", 1, 50),
	}), false))
	register(wrapLegacy("get_compliance_score", "Get compliance posture for a specific framework or all frameworks.", legacy.Get("compliance_score"), func(args map[string]any) map[string]string {
		return map[string]string{"framework": stringArg(args, "framework")}
	}, requiredSchema(map[string]any{"framework": stringProp("Optional framework filter")}), false))
	register(wrapLegacy("get_recommendations", "Get prioritized action items based on current security state across all suites.", legacy.Get("recommendation"), func(map[string]any) map[string]string {
		return map[string]string{}
	}, requiredSchema(map[string]any{}), false))
	register(wrapLegacy("build_dashboard", "Create a custom dashboard in Visus360 with auto-inferred widgets based on description.", legacy.Get("dashboard_builder"), func(args map[string]any) map[string]string {
		return map[string]string{"description": stringArg(args, "description")}
	}, requiredSchema(map[string]any{
		"description": stringProp("Natural language dashboard description"),
		"time_range":  stringProp(timeRangeDescription()),
	}, "description"), true))
	register(wrapLegacy("investigate_alert", "Run a full investigation on an alert: detail, explanation, affected assets, related alerts, UEBA, MITRE mapping.", legacy.Get("investigation"), func(args map[string]any) map[string]string {
		return map[string]string{"alert_id": stringArg(args, "alert_id")}
	}, requiredSchema(map[string]any{"alert_id": stringProp("Alert UUID or legacy ID")}, "alert_id"), false))
	register(wrapLegacy("get_trend_analysis", "Get trend data showing how a security metric has changed over a time period.", legacy.Get("trend_analysis"), func(args map[string]any) map[string]string {
		params := map[string]string{}
		applyTimeRange(params, stringArg(args, "time_range"))
		return params
	}, requiredSchema(map[string]any{
		"metric":     enumString("risk_score", "alert_count", "vulnerability_count", "compliance_score"),
		"time_range": stringProp(timeRangeDescription()),
	}, "metric"), false))
	register(wrapLegacy("start_remediation", "Initiate remediation workflow for a specific alert. Requires confirmation from user before execution.", legacy.Get("remediation"), func(args map[string]any) map[string]string {
		return map[string]string{"alert_id": stringArg(args, "alert_id")}
	}, requiredSchema(map[string]any{
		"alert_id": stringProp("Alert UUID or legacy ID"),
		"confirm":  boolProp("Must be true after explicit user confirmation"),
	}, "alert_id", "confirm"), true))
	register(wrapLegacy("generate_report", "Generate a formatted security report.", legacy.Get("report_generator"), func(args map[string]any) map[string]string {
		params := map[string]string{}
		applyTimeRange(params, stringArg(args, "time_range"))
		if strings.EqualFold(stringArg(args, "report_type"), "compliance") {
			params["framework"] = "all"
		}
		return params
	}, requiredSchema(map[string]any{
		"report_type": enumString("executive_summary", "weekly", "monthly", "incident", "compliance"),
		"time_range":  stringProp(timeRangeDescription()),
	}, "report_type"), false))
	register(wrapLegacy("get_threat_forecast", "Get predicted alert volume, emerging attack techniques, and campaign trends.", legacy.Get("get_threat_forecast"), func(args map[string]any) map[string]string {
		return map[string]string{
			"forecast_type": stringArg(args, "forecast_type"),
			"time_horizon":  stringArg(args, "time_horizon"),
		}
	}, requiredSchema(map[string]any{
		"forecast_type": enumString("alert_volume", "technique_trend", "campaign_detection"),
		"time_horizon":  enumString("7_days", "30_days", "90_days"),
	}, "forecast_type", "time_horizon"), false))
	register(wrapLegacy("get_asset_risk_prediction", "Get predicted probability of each asset being targeted, ranked by risk.", legacy.Get("get_asset_risk_prediction"), func(args map[string]any) map[string]string {
		return map[string]string{
			"limit":      strconvItoa(intArg(args, "limit", 10)),
			"asset_type": stringArg(args, "asset_type"),
		}
	}, requiredSchema(map[string]any{
		"limit":      intProp("Maximum assets to return", 1, 50),
		"asset_type": enumString("server", "endpoint", "database", "network", "all"),
	}), false))
	register(wrapLegacy("get_vulnerability_priority", "Get prioritized vulnerabilities ranked by predicted exploit probability.", legacy.Get("get_vulnerability_priority"), func(args map[string]any) map[string]string {
		return map[string]string{
			"limit":           strconvItoa(intArg(args, "limit", 20)),
			"min_probability": stringArg(args, "min_probability"),
		}
	}, requiredSchema(map[string]any{
		"limit":           intProp("Maximum vulnerabilities to return", 1, 100),
		"min_probability": numberProp("Minimum exploit probability threshold between 0 and 1"),
	}), false))
	register(wrapLegacy("get_insider_threat_forecast", "Get users whose behavioral risk scores are predicted to escalate.", legacy.Get("get_insider_threat_forecast"), func(args map[string]any) map[string]string {
		return map[string]string{
			"time_horizon": stringArg(args, "time_horizon"),
			"threshold":    strconvItoa(intArg(args, "threshold", 70)),
		}
	}, requiredSchema(map[string]any{
		"time_horizon": enumString("7_days", "30_days"),
		"threshold":    intProp("Minimum projected risk score", 0, 100),
	}, "time_horizon"), false))

	register(NewCrossSuiteQuery(deps))
	register(NewExecutiveBriefingTool(deps))
	register(NewWhatIfAnalysisTool(deps))
	register(NewRootCauseTool(deps))
	register(NewNaturalLanguageFilterTool(deps))
	register(NewConversationMemoryTool(deps))

	return &Registry{items: items}
}

func (r *Registry) Get(name string) Tool {
	if r == nil {
		return nil
	}
	return r.items[name]
}

func (r *Registry) List() []Tool {
	if r == nil {
		return nil
	}
	items := make([]Tool, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	return items
}

func wrapLegacy(name, description string, tool chattools.Tool, transform func(map[string]any) map[string]string, schema map[string]any, destructive bool) Tool {
	if tool == nil {
		return &legacyToolAdapter{name: name, description: description, schema: schema, destructive: destructive}
	}
	return &legacyToolAdapter{
		name:        name,
		description: description,
		permissions: tool.RequiredPermissions(),
		schema:      schema,
		destructive: destructive,
		delegate:    tool,
		transform:   transform,
	}
}

type RiskyUsersTool struct {
	deps *chattools.Dependencies
}

func (t *RiskyUsersTool) Name() string { return "get_risky_users" }
func (t *RiskyUsersTool) Description() string {
	return "Get users with highest behavioral risk scores (UEBA) including anomaly details."
}
func (t *RiskyUsersTool) RequiredPermissions() []string { return []string{"cyber:read"} }
func (t *RiskyUsersTool) IsDestructive() bool           { return false }
func (t *RiskyUsersTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"limit":     intProp("Maximum entities to return", 1, 50),
		"threshold": intProp("Minimum risk score", 0, 100),
	})
}
func (t *RiskyUsersTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	if t.deps == nil || t.deps.UEBAService == nil {
		return nil, fmt.Errorf("ueba service is unavailable")
	}
	limit := intArg(args, "limit", 10)
	threshold := float64(intArg(args, "threshold", 0))
	items, err := t.deps.UEBAService.GetRiskRanking(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(items))
	entities := make([]chatmodel.EntityReference, 0, len(items))
	for idx, item := range items {
		if item.RiskScore < threshold {
			continue
		}
		rows = append(rows, map[string]any{
			"entity_id":      item.EntityID,
			"entity_name":    item.EntityName,
			"entity_type":    item.EntityType,
			"risk_score":     item.RiskScore,
			"risk_level":     item.RiskLevel,
			"alert_count_7d": item.AlertCount7D,
		})
		entities = append(entities, entity("user", item.EntityID, item.EntityName, idx))
	}
	sortRowsByScore(rows, "risk_score")
	return listResult(
		summarizeRows(rows),
		"table",
		map[string]any{"items": rows},
		[]chatmodel.SuggestedAction{{Label: "Open UEBA dashboard", Type: "navigate", Params: map[string]string{"url": "/cyber/ueba"}}},
		entities,
	), nil
}

func stringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func intProp(description string, minimum, maximum int) map[string]any {
	return map[string]any{"type": "integer", "description": description, "minimum": minimum, "maximum": maximum}
}

func numberProp(description string) map[string]any {
	return map[string]any{"type": "integer", "description": description}
}

func boolProp(description string) map[string]any {
	return map[string]any{"type": "boolean", "description": description}
}

func enumString(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

func arrayOfStrings(description string, minItems int) map[string]any {
	schema := map[string]any{
		"type":        "array",
		"description": description,
		"items":       map[string]any{"type": "string"},
	}
	if minItems > 0 {
		schema["minItems"] = minItems
	}
	return schema
}

func strconvItoa(value int) string {
	return strconv.Itoa(value)
}
