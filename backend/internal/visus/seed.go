package visus

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	visusconfig "github.com/clario360/platform/internal/visus/config"
	"github.com/clario360/platform/internal/visus/kpi"
	"github.com/clario360/platform/internal/visus/model"
	reportpkg "github.com/clario360/platform/internal/visus/report"
	"github.com/clario360/platform/internal/visus/repository"
)

var visusSeedReference = time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)

func SeedDemoData(ctx context.Context, app *Application, cfg *visusconfig.Config, logger zerolog.Logger) (uuid.UUID, error) {
	if app == nil || app.Store == nil {
		return uuid.Nil, fmt.Errorf("visus application is not initialized")
	}

	tenantID, err := uuid.Parse(cfg.DemoTenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse VISUS_DEMO_TENANT_ID: %w", err)
	}
	userID, err := uuid.Parse(cfg.DemoUserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse VISUS_DEMO_USER_ID: %w", err)
	}

	kpisByName, err := seedDefaultKPIs(ctx, app, tenantID, userID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := seedDashboards(ctx, app, tenantID, userID, kpisByName); err != nil {
		return uuid.Nil, err
	}
	if err := seedSnapshots(ctx, app, tenantID, kpisByName); err != nil {
		return uuid.Nil, err
	}
	if err := seedAlerts(ctx, app, tenantID, userID, kpisByName); err != nil {
		return uuid.Nil, err
	}
	if err := seedReports(ctx, app, tenantID, userID); err != nil {
		return uuid.Nil, err
	}

	logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("kpis", len(kpisByName)).
		Int("dashboards", 2).
		Int("snapshots", 360).
		Int("alerts", 15).
		Int("reports", 3).
		Msg("seeded visus demo dataset")

	return tenantID, nil
}

func seedDefaultKPIs(ctx context.Context, app *Application, tenantID, userID uuid.UUID) (map[string]model.KPIDefinition, error) {
	existing, _, err := app.Store.KPIs.List(ctx, tenantID, 1, 100)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]model.KPIDefinition, len(existing))
	for _, item := range existing {
		byName[item.Name] = item
	}
	for _, def := range kpi.DefaultDefinitions(tenantID, userID) {
		if _, ok := byName[def.Name]; ok {
			continue
		}
		created, err := app.Store.KPIs.Create(ctx, &def)
		if err != nil {
			return nil, fmt.Errorf("seed kpi %q: %w", def.Name, err)
		}
		byName[created.Name] = *created
	}
	return byName, nil
}

func seedDashboards(ctx context.Context, app *Application, tenantID, userID uuid.UUID, kpisByName map[string]model.KPIDefinition) error {
	existing, _, err := app.Store.Dashboards.ListAccessible(ctx, tenantID, &userID, 1, 20)
	if err != nil {
		return err
	}
	byName := make(map[string]model.Dashboard, len(existing))
	for _, item := range existing {
		byName[item.Name] = item
	}

	execDash, err := ensureDashboard(ctx, app, byName, &model.Dashboard{
		TenantID:    tenantID,
		Name:        "Executive Overview",
		Description: "Cross-suite executive dashboard with KPIs, alerts, and operational trends.",
		GridColumns: 12,
		Visibility:  model.DashboardVisibilityOrganization,
		IsDefault:   true,
		IsSystem:    true,
		Tags:        []string{"system", "executive"},
		Metadata:    map[string]any{"seeded": true},
		CreatedBy:   userID,
	})
	if err != nil {
		return err
	}
	securityDash, err := ensureDashboard(ctx, app, byName, &model.Dashboard{
		TenantID:    tenantID,
		Name:        "Security Command",
		Description: "Security-specific executive lens spanning risk, alerts, MITRE coverage, and trends.",
		GridColumns: 12,
		Visibility:  model.DashboardVisibilityOrganization,
		IsDefault:   false,
		IsSystem:    true,
		Tags:        []string{"system", "security"},
		Metadata:    map[string]any{"seeded": true},
		CreatedBy:   userID,
	})
	if err != nil {
		return err
	}

	if err := seedDashboardWidgets(ctx, app, *execDash, executiveOverviewWidgets(*execDash, kpisByName)); err != nil {
		return err
	}
	if err := seedDashboardWidgets(ctx, app, *securityDash, securityCommandWidgets(*securityDash, kpisByName)); err != nil {
		return err
	}
	return nil
}

func ensureDashboard(ctx context.Context, app *Application, existing map[string]model.Dashboard, dashboard *model.Dashboard) (*model.Dashboard, error) {
	if current, ok := existing[dashboard.Name]; ok {
		return &current, nil
	}
	if dashboard.IsDefault {
		if err := app.Store.Dashboards.ClearDefault(ctx, dashboard.TenantID, nil); err != nil {
			return nil, err
		}
	}
	return app.Store.Dashboards.Create(ctx, dashboard)
}

func seedDashboardWidgets(ctx context.Context, app *Application, dashboard model.Dashboard, widgets []model.Widget) error {
	existing, err := app.Store.Widgets.ListByDashboard(ctx, dashboard.TenantID, dashboard.ID)
	if err != nil {
		return err
	}
	if len(existing) >= len(widgets) {
		return nil
	}
	existingByTitle := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		existingByTitle[item.Title] = struct{}{}
	}
	for _, widget := range widgets {
		if _, ok := existingByTitle[widget.Title]; ok {
			continue
		}
		copyWidget := widget
		if _, err := app.WidgetService.Create(ctx, &copyWidget); err != nil {
			return fmt.Errorf("seed widget %q: %w", widget.Title, err)
		}
	}
	return nil
}

func seedSnapshots(ctx context.Context, app *Application, tenantID uuid.UUID, kpisByName map[string]model.KPIDefinition) error {
	latest, err := app.Store.KPISnapshots.ListLatestByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if len(latest) >= len(kpisByName) {
		return nil
	}

	thresholds := kpi.NewThresholdEvaluator()
	definitions := make([]model.KPIDefinition, 0, len(kpisByName))
	for _, item := range kpisByName {
		definitions = append(definitions, item)
	}

	for _, def := range definitions {
		var previous *float64
		var latestValue float64
		var latestStatus model.KPIStatus
		var latestTime time.Time
		for day := 0; day < 30; day++ {
			periodEnd := visusSeedReference.AddDate(0, 0, -(29 - day))
			periodStart := periodEnd.Add(-24 * time.Hour)
			value := round2(seedKPIValue(def.Name, day))
			status := thresholds.Evaluate(&def, value)

			var delta *float64
			var deltaPercent *float64
			if previous != nil {
				currentDelta := round2(value - *previous)
				delta = &currentDelta
				currentPercent := 0.0
				if *previous != 0 {
					currentPercent = round2(((value - *previous) / *previous) * 100)
				}
				deltaPercent = &currentPercent
			}

			snapshot := &model.KPISnapshot{
				TenantID:      tenantID,
				KPIID:         def.ID,
				Value:         value,
				PreviousValue: previous,
				Delta:         delta,
				DeltaPercent:  deltaPercent,
				Status:        status,
				PeriodStart:   periodStart,
				PeriodEnd:     periodEnd,
				FetchSuccess:  true,
				CreatedAt:     periodEnd,
			}
			if _, err := app.Store.KPISnapshots.Create(ctx, snapshot); err != nil {
				return fmt.Errorf("seed kpi snapshot %q day %d: %w", def.Name, day, err)
			}
			current := value
			previous = &current
			latestValue = value
			latestStatus = status
			latestTime = periodEnd
		}
		if err := app.Store.KPIs.UpdateSnapshotState(ctx, tenantID, def.ID, latestTime, latestValue, latestStatus); err != nil {
			return err
		}
	}
	return nil
}

func seedAlerts(ctx context.Context, app *Application, tenantID, userID uuid.UUID, kpisByName map[string]model.KPIDefinition) error {
	_, total, err := app.Store.Alerts.List(ctx, tenantID, repository.AlertListFilters{}, 1, 1)
	if err != nil {
		return err
	}
	if total >= 15 {
		return nil
	}

	statuses := []model.AlertStatus{
		model.AlertStatusNew, model.AlertStatusNew, model.AlertStatusNew, model.AlertStatusNew, model.AlertStatusNew,
		model.AlertStatusViewed, model.AlertStatusViewed, model.AlertStatusViewed, model.AlertStatusViewed,
		model.AlertStatusAcknowledged, model.AlertStatusAcknowledged, model.AlertStatusAcknowledged,
		model.AlertStatusActioned, model.AlertStatusActioned,
		model.AlertStatusDismissed,
	}

	seeded := buildSeedAlerts(tenantID, userID, kpisByName)
	for idx, alert := range seeded {
		alert.Status = statuses[idx]
		alert.CreatedAt = visusSeedReference.Add(-time.Duration((len(seeded)-idx)*3) * time.Hour)
		alert.UpdatedAt = alert.CreatedAt
		applyAlertLifecycle(&alert, userID)
		if _, err := app.Store.Alerts.Create(ctx, &alert); err != nil {
			return fmt.Errorf("seed alert %q: %w", alert.Title, err)
		}
	}
	return nil
}

func seedReports(ctx context.Context, app *Application, tenantID, userID uuid.UUID) error {
	existing, _, err := app.Store.Reports.List(ctx, tenantID, 1, 20)
	if err != nil {
		return err
	}
	reportByName := make(map[string]model.ReportDefinition, len(existing))
	for _, item := range existing {
		reportByName[item.Name] = item
	}

	definitions := []*model.ReportDefinition{
		{
			TenantID:    tenantID,
			Name:        "Monthly Executive Report",
			Description: "Monthly executive summary spanning all suites.",
			ReportType:  model.ReportTypeExecutiveSummary,
			Sections:    []string{"security_posture", "data_intelligence", "governance", "legal", "kpi_summary"},
			Period:      "30d",
			Schedule:    stringPtr("0 8 1 * *"),
			Recipients:  []uuid.UUID{userID},
			AutoSend:    true,
			CreatedBy:   userID,
		},
		{
			TenantID:    tenantID,
			Name:        "Security Posture Report",
			Description: "Weekly executive summary focused on cyber risk and response performance.",
			ReportType:  model.ReportTypeSecurityPosture,
			Sections:    []string{"security_posture", "kpi_summary"},
			Period:      "7d",
			Schedule:    stringPtr("0 8 * * 1"),
			Recipients:  []uuid.UUID{userID},
			AutoSend:    false,
			CreatedBy:   userID,
		},
		{
			TenantID:    tenantID,
			Name:        "Quarterly Governance Report",
			Description: "Quarterly governance and action-item posture review.",
			ReportType:  model.ReportTypeGovernance,
			Sections:    []string{"governance", "kpi_summary"},
			Period:      "quarterly",
			Schedule:    stringPtr("0 8 1 1,4,7,10 *"),
			Recipients:  []uuid.UUID{userID},
			AutoSend:    false,
			CreatedBy:   userID,
		},
	}

	for _, def := range definitions {
		if _, ok := reportByName[def.Name]; ok {
			continue
		}
		nextRun, err := reportpkg.NextRunForReport(def.Schedule, visusSeedReference)
		if err != nil {
			return err
		}
		def.NextRunAt = nextRun
		created, err := app.Store.Reports.Create(ctx, def)
		if err != nil {
			return fmt.Errorf("seed report %q: %w", def.Name, err)
		}
		reportByName[created.Name] = *created
	}

	if err := seedReportSnapshots(ctx, app, reportByName["Monthly Executive Report"], monthlyReportDates(), userID); err != nil {
		return err
	}
	if err := seedReportSnapshots(ctx, app, reportByName["Security Posture Report"], securityReportDates(), userID); err != nil {
		return err
	}
	return nil
}

func seedReportSnapshots(ctx context.Context, app *Application, reportDef model.ReportDefinition, generatedDates []time.Time, userID uuid.UUID) error {
	if reportDef.ID == uuid.Nil {
		return nil
	}
	existing, err := app.Store.ReportSnapshots.ListByReport(ctx, reportDef.TenantID, reportDef.ID)
	if err != nil {
		return err
	}
	if len(existing) >= len(generatedDates) {
		return nil
	}
	for _, generatedAt := range generatedDates {
		start, end, err := reportpkg.ResolvePeriod(&reportDef, generatedAt)
		if err != nil {
			return err
		}
		sections := seedNarrativeSections(reportDef.ReportType, generatedAt)
		narrative := reportpkg.GenerateNarrative(sections, [2]time.Time{start, end})
		duration := int64(225)
		snapshot := &model.ReportSnapshot{
			TenantID:         reportDef.TenantID,
			ReportID:         reportDef.ID,
			ReportData:       map[string]any{"sections": sections, "generated_for": reportDef.Name},
			Narrative:        &narrative,
			FileFormat:       model.ReportFileJSON,
			PeriodStart:      start,
			PeriodEnd:        end,
			SectionsIncluded: append([]string(nil), reportDef.Sections...),
			GenerationTimeMS: &duration,
			SuiteFetchErrors: map[string]string{},
			GeneratedBy:      &userID,
			GeneratedAt:      generatedAt,
		}
		if _, err := app.Store.ReportSnapshots.Create(ctx, snapshot); err != nil {
			return fmt.Errorf("seed report snapshot %q %s: %w", reportDef.Name, generatedAt.Format(time.RFC3339), err)
		}
		nextRun, err := reportpkg.NextRunForReport(reportDef.Schedule, generatedAt)
		if err != nil {
			return err
		}
		if err := app.Store.Reports.UpdateGeneration(ctx, reportDef.TenantID, reportDef.ID, generatedAt, nextRun); err != nil {
			return err
		}
	}
	return nil
}

func executiveOverviewWidgets(dashboard model.Dashboard, kpis map[string]model.KPIDefinition) []model.Widget {
	return []model.Widget{
		newWidget(dashboard, "Security Risk Score", model.WidgetTypeKPICard, 0, 0, 3, 2, map[string]any{"kpi_id": kpis["Security Risk Score"].ID.String(), "show_trend": true, "show_target": true}),
		newWidget(dashboard, "Data Quality Score", model.WidgetTypeKPICard, 3, 0, 3, 2, map[string]any{"kpi_id": kpis["Data Quality Score"].ID.String(), "show_trend": true, "show_target": true}),
		newWidget(dashboard, "Governance Compliance", model.WidgetTypeKPICard, 6, 0, 3, 2, map[string]any{"kpi_id": kpis["Governance Compliance"].ID.String(), "show_trend": true, "show_target": true}),
		newWidget(dashboard, "High Risk Contracts", model.WidgetTypeKPICard, 9, 0, 3, 2, map[string]any{"kpi_id": kpis["High Risk Contracts"].ID.String(), "show_trend": true, "show_target": true}),
		newWidget(dashboard, "Executive Alert Feed", model.WidgetTypeAlertFeed, 9, 2, 3, 4, map[string]any{"alert_sources": []string{"cyber", "data", "acta", "lex"}, "max_alerts": 8, "severity_filter": []string{"critical", "high"}}),
		newWidget(dashboard, "Security Risk Gauge", model.WidgetTypeGauge, 0, 2, 4, 3, map[string]any{"kpi_id": kpis["Security Risk Score"].ID.String()}),
		newWidget(dashboard, "Data Quality Trend", model.WidgetTypeLineChart, 4, 2, 5, 3, map[string]any{"suite": "data", "data_source": "/dashboard", "data_path": "$.data.quality_trend", "x_axis": "date", "y_axis": []any{"score"}}),
		newWidget(dashboard, "Pipeline Success Rate", model.WidgetTypeAreaChart, 0, 5, 6, 3, map[string]any{"suite": "data", "data_source": "/dashboard", "data_path": "$.data.pipeline_success_trend", "x_axis": "date", "y_axis": []any{"success_rate"}}),
		newWidget(dashboard, "Governance Compliance By Committee", model.WidgetTypeBarChart, 6, 5, 3, 3, map[string]any{"suite": "acta", "data_source": "/dashboard", "data_path": "$.data.compliance_by_committee", "x_axis": "committee", "y_axis": []any{"score"}}),
		newWidget(dashboard, "Contract Expiry Timeline", model.WidgetTypeTable, 9, 6, 3, 2, map[string]any{"suite": "lex", "data_source": "/dashboard", "data_path": "$.data.expiring_contracts", "columns": []any{map[string]any{"key": "title", "label": "Contract"}, map[string]any{"key": "expiry_date", "label": "Expiry"}, map[string]any{"key": "risk_level", "label": "Risk"}}, "max_rows": 5}),
		newWidget(dashboard, "Executive Notes", model.WidgetTypeText, 0, 8, 12, 1, map[string]any{"content": "Last updated: auto-refresh every 60s. This dashboard degrades gracefully when one suite is unavailable."}),
	}
}

func securityCommandWidgets(dashboard model.Dashboard, kpis map[string]model.KPIDefinition) []model.Widget {
	return []model.Widget{
		newWidget(dashboard, "Security Risk Score Gauge", model.WidgetTypeGauge, 0, 0, 3, 3, map[string]any{"kpi_id": kpis["Security Risk Score"].ID.String()}),
		newWidget(dashboard, "Open Alerts By Severity", model.WidgetTypePieChart, 3, 0, 3, 3, map[string]any{"suite": "cyber", "data_source": "/dashboard", "data_path": "$.data.alert_breakdown", "label_path": "severity", "value_path": "count"}),
		newWidget(dashboard, "Alert Timeline", model.WidgetTypeLineChart, 6, 0, 6, 3, map[string]any{"suite": "cyber", "data_source": "/dashboard", "data_path": "$.data.alert_timeline", "x_axis": "date", "y_axis": []any{"count"}}),
		newWidget(dashboard, "MITRE Coverage", model.WidgetTypeStatusGrid, 0, 3, 4, 3, map[string]any{"items": []any{map[string]any{"label": "MITRE ATT&CK Coverage", "kpi_id": kpis["MITRE ATT&CK Coverage"].ID.String()}, map[string]any{"label": "Security Risk Score", "kpi_id": kpis["Security Risk Score"].ID.String()}, map[string]any{"label": "MTTR", "kpi_id": kpis["Mean Time to Respond"].ID.String()}}}),
		newWidget(dashboard, "Top Attacked Assets", model.WidgetTypeTable, 4, 3, 4, 3, map[string]any{"suite": "cyber", "data_source": "/dashboard", "data_path": "$.data.top_attacked_assets", "max_rows": 5}),
		newWidget(dashboard, "Threat Landscape", model.WidgetTypeHeatmap, 8, 3, 4, 3, map[string]any{"suite": "cyber", "data_source": "/dashboard", "data_path": "$.data.mitre_heatmap.cells", "x_axis": "tactic", "y_axis": "technique", "value_key": "value"}),
		newWidget(dashboard, "MTTR Trend", model.WidgetTypeSparkline, 0, 6, 12, 2, map[string]any{"kpi_id": kpis["Mean Time to Respond"].ID.String(), "points": 30}),
	}
}

func buildSeedAlerts(tenantID, userID uuid.UUID, kpis map[string]model.KPIDefinition) []model.ExecutiveAlert {
	newEventType := func(v string) *string { return &v }
	return []model.ExecutiveAlert{
		seedKPIAlert(tenantID, kpis["Security Risk Score"], model.KPIStatusCritical, 84),
		seedKPIAlert(tenantID, kpis["Contracts Expiring 30d"], model.KPIStatusCritical, 6),
		seedKPIAlert(tenantID, kpis["Open Contradictions"], model.KPIStatusWarning, 16),
		seedKPIAlert(tenantID, kpis["Overdue Action Items"], model.KPIStatusWarning, 8),
		seedKPIAlert(tenantID, kpis["High Risk Contracts"], model.KPIStatusWarning, 4),
		{
			TenantID:        tenantID,
			Title:           "Critical security event propagated",
			Description:     "A critical cyber alert was promoted into the executive console.",
			Category:        model.AlertCategoryRisk,
			Severity:        model.AlertSeverityCritical,
			SourceSuite:     "cyber",
			SourceType:      "event",
			SourceEventType: newEventType("cyber.alert.events"),
			DedupKey:        stringPtr("seed:cyber:critical:1"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-23 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-23 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Security control gap detected",
			Description:     "Coverage drift was detected in a priority security control area.",
			Category:        model.AlertCategoryRisk,
			Severity:        model.AlertSeverityHigh,
			SourceSuite:     "cyber",
			SourceType:      "event",
			SourceEventType: newEventType("cyber.alert.events"),
			DedupKey:        stringPtr("seed:cyber:critical:2"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-22 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-22 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Executive visibility on emerging threat",
			Description:     "An elevated threat pattern was escalated from the cyber suite.",
			Category:        model.AlertCategoryRisk,
			Severity:        model.AlertSeverityHigh,
			SourceSuite:     "cyber",
			SourceType:      "event",
			SourceEventType: newEventType("cyber.alert.events"),
			DedupKey:        stringPtr("seed:cyber:critical:3"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-21 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-21 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Pipeline failure spike",
			Description:     "Multiple data pipelines failed within the executive reporting window.",
			Category:        model.AlertCategoryOperational,
			Severity:        model.AlertSeverityHigh,
			SourceSuite:     "data",
			SourceType:      "event",
			SourceEventType: newEventType("data.pipeline.events"),
			DedupKey:        stringPtr("seed:data:event:1"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-20 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-20 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Data quality score dropped",
			Description:     "The data suite reported a material quality drop requiring executive attention.",
			Category:        model.AlertCategoryDataQuality,
			Severity:        model.AlertSeverityHigh,
			SourceSuite:     "data",
			SourceType:      "event",
			SourceEventType: newEventType("data.quality.events"),
			DedupKey:        stringPtr("seed:data:event:2"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-19 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-19 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Contradiction backlog growth",
			Description:     "Open contradictions increased faster than expected during the week.",
			Category:        model.AlertCategoryDataQuality,
			Severity:        model.AlertSeverityMedium,
			SourceSuite:     "data",
			SourceType:      "event",
			SourceEventType: newEventType("data.quality.events"),
			DedupKey:        stringPtr("seed:data:event:3"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-18 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-18 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Overdue action items increased",
			Description:     "Governance action items exceeded the expected backlog threshold.",
			Category:        model.AlertCategoryGovernance,
			Severity:        model.AlertSeverityHigh,
			SourceSuite:     "acta",
			SourceType:      "event",
			SourceEventType: newEventType("enterprise.acta.events"),
			DedupKey:        stringPtr("seed:acta:event:1"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-17 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-17 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Board packet risk",
			Description:     "A near-term governance meeting is at risk due to incomplete follow-up actions.",
			Category:        model.AlertCategoryGovernance,
			Severity:        model.AlertSeverityMedium,
			SourceSuite:     "acta",
			SourceType:      "event",
			SourceEventType: newEventType("enterprise.acta.events"),
			DedupKey:        stringPtr("seed:acta:event:2"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-16 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-16 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Contracts expiring soon",
			Description:     "Multiple contracts are approaching expiry inside the next 30 days.",
			Category:        model.AlertCategoryLegal,
			Severity:        model.AlertSeverityHigh,
			SourceSuite:     "lex",
			SourceType:      "event",
			SourceEventType: newEventType("enterprise.lex.events"),
			DedupKey:        stringPtr("seed:lex:event:1"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-15 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-15 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
		{
			TenantID:        tenantID,
			Title:           "Legal compliance alert",
			Description:     "A contractual compliance issue was raised from the legal suite.",
			Category:        model.AlertCategoryLegal,
			Severity:        model.AlertSeverityMedium,
			SourceSuite:     "lex",
			SourceType:      "event",
			SourceEventType: newEventType("enterprise.lex.events"),
			DedupKey:        stringPtr("seed:lex:event:2"),
			OccurrenceCount: 1,
			FirstSeenAt:     visusSeedReference.Add(-14 * time.Hour),
			LastSeenAt:      visusSeedReference.Add(-14 * time.Hour),
			Metadata:        map[string]any{"seeded": true},
		},
	}
}

func seedKPIAlert(tenantID uuid.UUID, def model.KPIDefinition, status model.KPIStatus, value float64) model.ExecutiveAlert {
	title := fmt.Sprintf("%s is %s", def.Name, status)
	description := fmt.Sprintf("%s reached %.2f %s and breached its %s threshold.", def.Name, value, def.Unit, status)
	dedupKey := fmt.Sprintf("seed:kpi:%s", def.ID)
	return model.ExecutiveAlert{
		TenantID:        tenantID,
		Title:           title,
		Description:     description,
		Category:        seedCategoryFromSuite(def.Suite),
		Severity:        seedSeverityFromStatus(status),
		SourceSuite:     string(def.Suite),
		SourceType:      "kpi_breach",
		SourceEntityID:  &def.ID,
		DedupKey:        &dedupKey,
		OccurrenceCount: 1,
		FirstSeenAt:     visusSeedReference.Add(-24 * time.Hour),
		LastSeenAt:      visusSeedReference.Add(-24 * time.Hour),
		LinkedKPIID:     &def.ID,
		Metadata:        map[string]any{"seeded": true, "value": value, "status": status},
	}
}

func applyAlertLifecycle(alert *model.ExecutiveAlert, userID uuid.UUID) {
	switch alert.Status {
	case model.AlertStatusViewed:
		viewedAt := alert.CreatedAt.Add(30 * time.Minute)
		alert.ViewedAt = &viewedAt
		alert.ViewedBy = &userID
	case model.AlertStatusAcknowledged:
		viewedAt := alert.CreatedAt.Add(30 * time.Minute)
		alert.ViewedAt = &viewedAt
		alert.ViewedBy = &userID
	case model.AlertStatusActioned:
		viewedAt := alert.CreatedAt.Add(30 * time.Minute)
		actionedAt := alert.CreatedAt.Add(2 * time.Hour)
		alert.ViewedAt = &viewedAt
		alert.ViewedBy = &userID
		alert.ActionedAt = &actionedAt
		alert.ActionedBy = &userID
		alert.ActionNotes = stringPtr("Seeded remediation action completed.")
	case model.AlertStatusDismissed:
		viewedAt := alert.CreatedAt.Add(30 * time.Minute)
		dismissedAt := alert.CreatedAt.Add(90 * time.Minute)
		alert.ViewedAt = &viewedAt
		alert.ViewedBy = &userID
		alert.DismissedAt = &dismissedAt
		alert.DismissedBy = &userID
		alert.DismissReason = stringPtr("Seeded alert dismissed after review.")
	}
}

func monthlyReportDates() []time.Time {
	return []time.Time{
		time.Date(2025, time.December, 7, 8, 0, 0, 0, time.UTC),
		time.Date(2026, time.January, 7, 8, 0, 0, 0, time.UTC),
		time.Date(2026, time.February, 7, 8, 0, 0, 0, time.UTC),
	}
}

func securityReportDates() []time.Time {
	return []time.Time{
		time.Date(2026, time.February, 28, 8, 0, 0, 0, time.UTC),
		time.Date(2026, time.March, 7, 8, 0, 0, 0, time.UTC),
	}
}

func seedNarrativeSections(reportType model.ReportType, generatedAt time.Time) map[string]interface{} {
	sections := map[string]interface{}{
		"security_posture": map[string]any{
			"available":       true,
			"risk_score":      74.0,
			"grade":           "B",
			"trend_word":      reportpkg.TrendWord(-6, false),
			"prev_risk_score": 80.0,
			"open_alerts":     14.0,
			"critical_alerts": 3.0,
			"mttr_hours":      5.2,
			"coverage":        58.0,
		},
		"data_intelligence": map[string]any{
			"available":           true,
			"quality_score":       88.0,
			"quality_grade":       "B+",
			"success_rate":        93.0,
			"failed_count":        4.0,
			"contradiction_count": 16.0,
		},
		"governance": map[string]any{
			"available":        true,
			"compliance_score": 83.0,
			"meeting_count":    3.0,
			"overdue_count":    8.0,
			"minutes_pending":  2.0,
		},
		"legal": map[string]any{
			"available":        true,
			"active_contracts": 42.0,
			"value":            12500000.0,
			"expiring_count":   6.0,
			"high_risk_count":  4.0,
		},
		"recommendations": map[string]any{
			"available": true,
			"items": []string{
				"Reduce open contradictions to return the data quality backlog to target levels.",
				"Resolve overdue governance action items before the next board packet deadline.",
				"Review contracts expiring in the next 30 days and prioritize high-risk renewals.",
			},
		},
	}
	if reportType == model.ReportTypeSecurityPosture {
		delete(sections, "data_intelligence")
		delete(sections, "governance")
		delete(sections, "legal")
	}
	if reportType == model.ReportTypeGovernance {
		delete(sections, "security_posture")
		delete(sections, "data_intelligence")
		delete(sections, "legal")
	}
	sections["generated_at"] = generatedAt
	return sections
}

func seedKPIValue(name string, day int) float64 {
	switch name {
	case "Security Risk Score":
		return 85 - float64(day)*0.7
	case "Open Critical Alerts":
		return 4 + math.Sin(float64(day)/3)
	case "Mean Time to Respond":
		return 8.5 - float64(day)*0.14
	case "MITRE ATT&CK Coverage":
		return 28 + float64(day)*1.1
	case "Data Quality Score":
		return 73 + float64(day)*0.55
	case "Pipeline Success Rate":
		return 91 + math.Sin(float64(day)/4)*1.5
	case "Open Contradictions":
		return 8 + float64(day)*0.28
	case "Dark Data Assets":
		return 18 + math.Sin(float64(day)/5)*1.2
	case "Governance Compliance":
		return 70 + float64(day)*0.48
	case "Overdue Action Items":
		return 4 + float64(day)*0.18
	case "Contracts Expiring 30d":
		return 2 + float64(day)*0.14
	case "High Risk Contracts":
		return 4 + math.Sin(float64(day)/6)*0.4
	default:
		return 0
	}
}

func seedCategoryFromSuite(suite model.KPISuite) model.AlertCategory {
	switch suite {
	case model.KPISuiteCyber:
		return model.AlertCategoryRisk
	case model.KPISuiteData:
		return model.AlertCategoryDataQuality
	case model.KPISuiteActa:
		return model.AlertCategoryGovernance
	case model.KPISuiteLex:
		return model.AlertCategoryLegal
	default:
		return model.AlertCategoryOperational
	}
}

func seedSeverityFromStatus(status model.KPIStatus) model.AlertSeverity {
	switch status {
	case model.KPIStatusCritical:
		return model.AlertSeverityCritical
	case model.KPIStatusWarning:
		return model.AlertSeverityHigh
	default:
		return model.AlertSeverityInfo
	}
}

func newWidget(dashboard model.Dashboard, title string, widgetType model.WidgetType, x, y, w, h int, config map[string]any) model.Widget {
	return model.Widget{
		TenantID:               dashboard.TenantID,
		DashboardID:            dashboard.ID,
		Title:                  title,
		Type:                   widgetType,
		Config:                 config,
		Position:               model.WidgetPosition{X: x, Y: y, W: w, H: h},
		RefreshIntervalSeconds: 60,
	}
}

func stringPtr(value string) *string {
	return &value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
