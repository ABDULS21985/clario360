package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	visusmodel "github.com/clario360/platform/internal/visus/model"
)

type DashboardBuilderTool struct {
	baseTool
}

type widgetSpec struct {
	Title    string
	Type     visusmodel.WidgetType
	Config   map[string]any
	Width    int
	Height   int
	FullName string
}

func NewDashboardBuilderTool(deps *Dependencies) *DashboardBuilderTool {
	return &DashboardBuilderTool{baseTool: newBaseTool(deps)}
}

func (t *DashboardBuilderTool) Name() string { return "dashboard_builder" }

func (t *DashboardBuilderTool) Description() string { return "build a custom dashboard with specified metrics and charts" }

func (t *DashboardBuilderTool) RequiredPermissions() []string { return []string{"visus:write"} }

func (t *DashboardBuilderTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.VisusDashboardService == nil || t.deps.VisusWidgetService == nil || t.deps.VisusKPIRepo == nil {
		return nil, fmt.Errorf("%w: visus services", errToolUnavailable)
	}
	description := strings.TrimSpace(params["description"])
	if description == "" {
		description = "security overview"
	}
	kpis, err := t.deps.VisusKPIRepo.ListEnabled(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	kpiByName := make(map[string]uuid.UUID, len(kpis))
	for _, item := range kpis {
		kpiByName[item.Name] = item.ID
	}
	specs := inferDashboardWidgets(description, kpiByName)
	name := "vCISO: " + description
	if len(name) > 100 {
		name = name[:100]
	}
	dashboard, err := t.deps.VisusDashboardService.Create(ctx, &visusmodel.Dashboard{
		TenantID:    tenantID,
		Name:        name,
		Description: description,
		GridColumns: 12,
		Visibility:  visusmodel.DashboardVisibilityPrivate,
		SharedWith:  []uuid.UUID{},
		IsDefault:   false,
		IsSystem:    false,
		Tags:        []string{"vciso", "generated"},
		Metadata:    map[string]any{"generated_by": "vciso"},
		CreatedBy:   userID,
	})
	if err != nil {
		return nil, err
	}
	widgets := layoutWidgetSpecs(dashboard.ID, tenantID, specs)
	created := make([]map[string]any, 0, len(widgets))
	for _, widget := range widgets {
		item := widget
		if _, err := t.deps.VisusWidgetService.Create(ctx, &item); err != nil {
			return nil, err
		}
		created = append(created, map[string]any{"title": item.Title, "type": item.Type})
	}
	lines := []string{
		fmt.Sprintf("I've created a dashboard **%s** with %d widgets:", name, len(created)),
	}
	for _, item := range created {
		lines = append(lines, fmt.Sprintf("- %s (%s)", item["title"], item["type"]))
	}
	lines = append(lines, "", "You can open it now or continue refining it in chat.")
	return &ToolResult{
		Text: strings.Join(lines, "\n"),
		Data: map[string]any{
			"dashboard_id": dashboard.ID,
			"url":          "/visus/dashboards/" + dashboard.ID.String(),
			"widgets":      created,
		},
		DataType: "dashboard",
		Actions: []chatmodel.SuggestedAction{
			navigateAction("Open dashboard", "/visus/dashboards/"+dashboard.ID.String()),
			navigateAction("Share with team", "/visus/dashboards/"+dashboard.ID.String()+"/share"),
			messageAction("Add more widgets", "Build me a dashboard with alerts, risk, compliance, and MITRE coverage"),
		},
		Entities: []chatmodel.EntityReference{entityRef("dashboard", dashboard.ID.String(), dashboard.Name, 0)},
	}, nil
}

func inferDashboardWidgets(description string, kpiByName map[string]uuid.UUID) []widgetSpec {
	lower := strings.ToLower(description)
	specs := make([]widgetSpec, 0, 6)
	add := func(items ...widgetSpec) {
		for _, item := range items {
			duplicate := false
			for _, existing := range specs {
				if existing.Title == item.Title {
					duplicate = true
					break
				}
			}
			if !duplicate {
				specs = append(specs, item)
			}
		}
	}
	if strings.Contains(lower, "alert") {
		add(
			widgetSpec{Title: "Open Alerts By Severity", Type: visusmodel.WidgetTypePieChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/dashboard/severity-distribution", "data_path": "$.data.counts", "label_path": "name", "value_path": "count"}},
			widgetSpec{Title: "Alert Timeline", Type: visusmodel.WidgetTypeLineChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/dashboard/alerts-timeline", "data_path": "$.data.points", "x_axis": "bucket", "y_axis": []any{"count"}}},
		)
	}
	if strings.Contains(lower, "risk") {
		if id := kpiByName["Security Risk Score"]; id != uuid.Nil {
			add(widgetSpec{Title: "Security Risk Score Gauge", Type: visusmodel.WidgetTypeGauge, Width: 3, Height: 3, Config: map[string]any{"kpi_id": id.String()}})
		}
		add(widgetSpec{Title: "Risk Trend", Type: visusmodel.WidgetTypeLineChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/risk/score/trend?days=30", "data_path": "$.data", "x_axis": "time", "y_axis": []any{"overall_score"}}})
	}
	if strings.Contains(lower, "vulnerab") {
		add(widgetSpec{Title: "Vulnerability Aging", Type: visusmodel.WidgetTypeBarChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/vulnerabilities/aging", "data_path": "$.data.buckets", "x_axis": "label", "y_axis": []any{"total"}}})
	}
	if strings.Contains(lower, "asset") {
		add(widgetSpec{Title: "Top Attacked Assets", Type: visusmodel.WidgetTypeTable, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/dashboard/top-attacked-assets", "data_path": "$.data", "max_rows": 10}})
	}
	if strings.Contains(lower, "mitre") || strings.Contains(lower, "att&ck") {
		add(widgetSpec{Title: "MITRE Heatmap", Type: visusmodel.WidgetTypeHeatmap, Width: 12, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/dashboard/mitre-heatmap", "data_path": "$.data.cells", "x_axis": "tactic_name", "y_axis": "technique_name", "value_key": "alert_count"}})
	}
	if strings.Contains(lower, "compliance") {
		add(widgetSpec{Title: "Compliance By Committee", Type: visusmodel.WidgetTypeBarChart, Width: 6, Height: 3, Config: map[string]any{"suite": "acta", "data_source": "/dashboard", "data_path": "$.data.compliance_by_committee", "x_axis": "committee_name", "y_axis": []any{"score"}}})
	}
	if strings.Contains(lower, "ueba") || strings.Contains(lower, "user") {
		add(widgetSpec{Title: "Top Risky Entities", Type: visusmodel.WidgetTypeBarChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/ueba/risk-ranking", "data_path": "$.data", "x_axis": "entity_name", "y_axis": []any{"risk_score"}}})
	}
	if strings.Contains(lower, "pipeline") {
		add(widgetSpec{Title: "Pipeline Status", Type: visusmodel.WidgetTypeTable, Width: 12, Height: 3, Config: map[string]any{"suite": "data", "data_source": "/pipelines", "data_path": "$.data", "max_rows": 10}})
	}
	if len(specs) == 0 {
		if id := kpiByName["Security Risk Score"]; id != uuid.Nil {
			specs = append(specs, widgetSpec{Title: "Security Risk Score", Type: visusmodel.WidgetTypeGauge, Width: 3, Height: 3, Config: map[string]any{"kpi_id": id.String()}})
		}
		specs = append(specs,
			widgetSpec{Title: "Open Alerts By Severity", Type: visusmodel.WidgetTypePieChart, Width: 3, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/dashboard/severity-distribution", "data_path": "$.data.counts", "label_path": "name", "value_path": "count"}},
			widgetSpec{Title: "Alert Timeline", Type: visusmodel.WidgetTypeLineChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/dashboard/alerts-timeline", "data_path": "$.data.points", "x_axis": "bucket", "y_axis": []any{"count"}}},
			widgetSpec{Title: "Vulnerability Aging", Type: visusmodel.WidgetTypeBarChart, Width: 6, Height: 3, Config: map[string]any{"suite": "cyber", "data_source": "/vulnerabilities/aging", "data_path": "$.data.buckets", "x_axis": "label", "y_axis": []any{"total"}}},
			widgetSpec{Title: "Compliance By Committee", Type: visusmodel.WidgetTypeBarChart, Width: 6, Height: 3, Config: map[string]any{"suite": "acta", "data_source": "/dashboard", "data_path": "$.data.compliance_by_committee", "x_axis": "committee_name", "y_axis": []any{"score"}}},
		)
	}
	return specs
}

func layoutWidgetSpecs(dashboardID, tenantID uuid.UUID, specs []widgetSpec) []visusmodel.Widget {
	out := make([]visusmodel.Widget, 0, len(specs))
	x, y, rowHeight := 0, 0, 0
	for _, spec := range specs {
		if x+spec.Width > 12 {
			x = 0
			y += rowHeight
			rowHeight = 0
		}
		position := visusmodel.WidgetPosition{X: x, Y: y, W: spec.Width, H: spec.Height}
		out = append(out, visusmodel.Widget{
			DashboardID:            dashboardID,
			TenantID:               tenantID,
			Title:                  spec.Title,
			Type:                   spec.Type,
			Config:                 spec.Config,
			Position:               position,
			RefreshIntervalSeconds: 300,
		})
		x += spec.Width
		if spec.Height > rowHeight {
			rowHeight = spec.Height
		}
	}
	return out
}

