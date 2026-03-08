package seeder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	visusmodel "github.com/clario360/platform/internal/visus/model"
	visusrepo "github.com/clario360/platform/internal/visus/repository"
	visusservice "github.com/clario360/platform/internal/visus/service"
)

type DashboardSeeder struct {
	pool *pgxpool.Pool
	log  zerolog.Logger
}

func NewDashboardSeeder(pool *pgxpool.Pool, logger zerolog.Logger) *DashboardSeeder {
	return &DashboardSeeder{pool: pool, log: logger.With().Str("component", "dashboard_seeder").Logger()}
}

func (s *DashboardSeeder) Seed(ctx context.Context, tenantID, adminUserID uuid.UUID) error {
	store := visusrepo.NewStore(s.pool, s.log)
	dashboardService := visusservice.NewDashboardService(store.Dashboards, store.Widgets, nil, nil, s.log)
	widgetService := visusservice.NewWidgetService(store.Dashboards, store.Widgets, store.KPIs, store.KPISnapshots, store.Alerts, nil, nil, s.log)

	existing, _, err := store.Dashboards.ListAccessible(ctx, tenantID, nil, 1, 50)
	if err != nil {
		return err
	}
	var dashboardID uuid.UUID
	for _, item := range existing {
		if item.Name == "Executive Overview" && item.IsSystem {
			dashboardID = item.ID
			break
		}
	}
	if dashboardID == uuid.Nil {
		created, err := dashboardService.Create(ctx, &visusmodel.Dashboard{
			TenantID:    tenantID,
			Name:        "Executive Overview",
			Description: "Cross-suite executive overview dashboard.",
			GridColumns: 12,
			Visibility:  visusmodel.DashboardVisibilityOrganization,
			IsDefault:   true,
			IsSystem:    true,
			Tags:        []string{"system", "onboarding"},
			Metadata:    map[string]any{"seeded_by": "tenant_onboarding"},
			CreatedBy:   adminUserID,
		})
		if err != nil {
			return err
		}
		dashboardID = created.ID
	}

	kpis, _, err := store.KPIs.List(ctx, tenantID, 1, 100)
	if err != nil {
		return err
	}
	kpiIDs := map[string]string{}
	for _, kpi := range kpis {
		kpiIDs[kpi.Name] = kpi.ID.String()
	}

	widgets := []visusmodel.Widget{
		newWidget(tenantID, dashboardID, "Risk Score", visusmodel.WidgetTypeGauge, 0, 0, 3, 2, map[string]any{"kpi_id": kpiIDs["Security Risk Score"]}),
		newWidget(tenantID, dashboardID, "Critical Alerts", visusmodel.WidgetTypeKPICard, 3, 0, 3, 2, map[string]any{"kpi_id": kpiIDs["Open Critical Alerts"]}),
		newWidget(tenantID, dashboardID, "Quality Score", visusmodel.WidgetTypeKPICard, 6, 0, 3, 2, map[string]any{"kpi_id": kpiIDs["Data Quality Score"]}),
		newWidget(tenantID, dashboardID, "Compliance Score", visusmodel.WidgetTypeKPICard, 9, 0, 3, 2, map[string]any{"kpi_id": kpiIDs["Governance Compliance"]}),
		newWidget(tenantID, dashboardID, "Alert Trend", visusmodel.WidgetTypeLineChart, 0, 2, 6, 3, map[string]any{"suite": "cyber", "data_source": "/dashboard", "data_path": "$.data.alert_timeline"}),
		newWidget(tenantID, dashboardID, "Pipeline Status", visusmodel.WidgetTypeBarChart, 6, 2, 6, 3, map[string]any{"suite": "data", "data_source": "/dashboard", "data_path": "$.data.pipeline_success_trend"}),
		newWidget(tenantID, dashboardID, "Expiring Contracts", visusmodel.WidgetTypeTable, 0, 5, 6, 3, map[string]any{"suite": "lex", "data_source": "/dashboard", "data_path": "$.data.expiring_contracts", "max_rows": 8}),
		newWidget(tenantID, dashboardID, "Overdue Actions", visusmodel.WidgetTypeTable, 6, 5, 6, 3, map[string]any{"suite": "acta", "data_source": "/dashboard", "data_path": "$.data.overdue_actions", "max_rows": 8}),
	}

	currentWidgets, err := store.Widgets.ListByDashboard(ctx, tenantID, dashboardID)
	if err != nil {
		return err
	}
	existingTitles := map[string]struct{}{}
	for _, widget := range currentWidgets {
		existingTitles[widget.Title] = struct{}{}
	}
	for _, widget := range widgets {
		if _, ok := existingTitles[widget.Title]; ok {
			continue
		}
		copyWidget := widget
		if _, err := widgetService.Create(ctx, &copyWidget); err != nil {
			return fmt.Errorf("seed widget %s: %w", widget.Title, err)
		}
	}
	return nil
}

func newWidget(tenantID, dashboardID uuid.UUID, title string, widgetType visusmodel.WidgetType, x, y, w, h int, config map[string]any) visusmodel.Widget {
	return visusmodel.Widget{
		TenantID:    tenantID,
		DashboardID: dashboardID,
		Title:       title,
		Type:        widgetType,
		Config:      config,
		Position: visusmodel.WidgetPosition{
			X: x,
			Y: y,
			W: w,
			H: h,
		},
		RefreshIntervalSeconds: 60,
	}
}
