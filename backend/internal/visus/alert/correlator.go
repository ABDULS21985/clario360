package alert

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
	"github.com/clario360/platform/internal/visus/model"
)

type CorrelatedAlert struct {
	Title          string
	Description    string
	Category       model.AlertCategory
	Severity       model.AlertSeverity
	SourceSuite    string
	SourceType     string
	DedupKey       string
	Metadata       map[string]any
	SourceEntityID *uuid.UUID
}

type correlatorKPIRepository interface {
	List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]model.KPIDefinition, int, error)
}

type correlatorSnapshotRepository interface {
	ListByKPI(ctx context.Context, tenantID, kpiID uuid.UUID, query model.KPIQuery) ([]model.KPISnapshot, error)
}

type correlatorAlertRepository interface {
	CountCriticalSuites(ctx context.Context, tenantID uuid.UUID) ([]string, error)
	ListRecentBySource(ctx context.Context, tenantID uuid.UUID, sourceSuite string, since time.Time, severity *string) ([]model.ExecutiveAlert, error)
}

type correlatorSuiteClient interface {
	Fetch(ctx context.Context, suite, endpoint string, tenantID uuid.UUID, target interface{}) aggregator.FetchMetadata
}

type Correlator struct {
	kpis        correlatorKPIRepository
	snapshots   correlatorSnapshotRepository
	alerts      correlatorAlertRepository
	suiteClient correlatorSuiteClient
}

func NewCorrelator(kpis correlatorKPIRepository, snapshots correlatorSnapshotRepository, alerts correlatorAlertRepository, suiteClient correlatorSuiteClient) *Correlator {
	return &Correlator{
		kpis:        kpis,
		snapshots:   snapshots,
		alerts:      alerts,
		suiteClient: suiteClient,
	}
}

func (c *Correlator) DetectPatterns(ctx context.Context, tenantID uuid.UUID) ([]CorrelatedAlert, error) {
	out := make([]CorrelatedAlert, 0)
	kpis, _, err := c.kpis.List(ctx, tenantID, 1, 500)
	if err != nil {
		return nil, err
	}
	for _, kpi := range kpis {
		history, histErr := c.snapshots.ListByKPI(ctx, tenantID, kpi.ID, model.KPIQuery{Limit: 3})
		if histErr != nil || len(history) < 3 {
			continue
		}
		if degrading(kpi.Direction, history) {
			out = append(out, CorrelatedAlert{
				Title:          fmt.Sprintf("%s has been degrading for 3 consecutive days", kpi.Name),
				Description:    fmt.Sprintf("%s moved in the wrong direction for three consecutive snapshots.", kpi.Name),
				Category:       categoryFromSuite(kpi.Suite),
				Severity:       model.AlertSeverityHigh,
				SourceSuite:    string(kpi.Suite),
				SourceType:     "pattern",
				DedupKey:       "pattern:degrading:" + kpi.ID.String(),
				SourceEntityID: &kpi.ID,
				Metadata:       map[string]any{"pattern": "degrading_trend", "kpi_id": kpi.ID, "kpi_name": kpi.Name},
			})
		}
	}

	criticalSuites, err := c.alerts.CountCriticalSuites(ctx, tenantID)
	if err == nil && len(criticalSuites) >= 2 {
		key := "pattern:multi_suite:" + strings.Join(criticalSuites, ",")
		out = append(out, CorrelatedAlert{
			Title:       "Multiple suites reporting critical issues",
			Description: fmt.Sprintf("Multiple suites reporting critical issues: %s. Potential systemic problem.", strings.Join(criticalSuites, ", ")),
			Category:    model.AlertCategoryStrategic,
			Severity:    model.AlertSeverityCritical,
			SourceSuite: "visus",
			SourceType:  "pattern",
			DedupKey:    key,
			Metadata:    map[string]any{"pattern": "multi_suite_issue", "suites": criticalSuites},
		})
	}

	dataAlerts, _ := c.alerts.ListRecentBySource(ctx, tenantID, "data", time.Now().UTC().Add(-time.Hour), nil)
	critical := string(model.AlertSeverityCritical)
	cyberAlerts, _ := c.alerts.ListRecentBySource(ctx, tenantID, "cyber", time.Now().UTC().Add(-time.Hour), &critical)
	if len(dataAlerts) > 0 && len(cyberAlerts) > 0 {
		out = append(out, CorrelatedAlert{
			Title:       "Pipeline failures and security alerts detected together",
			Description: "Data pipeline failures detected alongside security alerts. Possible data breach scenario.",
			Category:    model.AlertCategoryStrategic,
			Severity:    model.AlertSeverityCritical,
			SourceSuite: "visus",
			SourceType:  "pattern",
			DedupKey:    "pattern:pipeline_security",
			Metadata:    map[string]any{"pattern": "pipeline_security_correlation"},
		})
	}

	if c.suiteClient != nil {
		var payload map[string]any
		meta := c.suiteClient.Fetch(ctx, "acta", "/dashboard", tenantID, &payload)
		if meta.Status != "unavailable" {
			overdue := mustValue(payload, "$.data.kpis.overdue_action_items")
			upcoming := mustValue(payload, "$.data.kpis.upcoming_meetings_30d")
			if overdue > 5 && upcoming > 0 {
				out = append(out, CorrelatedAlert{
					Title:       "Governance deadline pressure",
					Description: fmt.Sprintf("Board meeting pressure detected with %.0f overdue action items and %.0f upcoming meetings.", overdue, upcoming),
					Category:    model.AlertCategoryGovernance,
					Severity:    model.AlertSeverityHigh,
					SourceSuite: "acta",
					SourceType:  "pattern",
					DedupKey:    "pattern:governance_deadline",
					Metadata:    map[string]any{"pattern": "governance_deadline_pressure", "overdue": overdue, "upcoming": upcoming},
				})
			}
		}
	}

	return out, nil
}

func degrading(direction model.KPIDirection, history []model.KPISnapshot) bool {
	if len(history) < 3 {
		return false
	}
	oldest := history[2].Value
	middle := history[1].Value
	latest := history[0].Value
	if direction == model.KPIDirectionHigherIsBetter {
		return oldest > middle && middle > latest
	}
	return oldest < middle && middle < latest
}

func mustValue(payload map[string]any, path string) float64 {
	value, err := aggregator.ExtractValue(payload, path)
	if err != nil {
		return 0
	}
	return value
}
