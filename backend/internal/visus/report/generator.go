package report

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/aggregator"
	reportsections "github.com/clario360/platform/internal/visus/report/sections"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

type Generator struct {
	reports         *repository.ReportRepository
	snapshots       *repository.ReportSnapshotRepository
	kpis            *repository.KPIRepository
	kpiSnapshots    *repository.KPISnapshotRepository
	suiteClient     *aggregator.SuiteClient
	publisher       Publisher
	logger          zerolog.Logger
	predictionLogger *aigovmiddleware.PredictionLogger
}

func NewGenerator(reports *repository.ReportRepository, snapshots *repository.ReportSnapshotRepository, kpis *repository.KPIRepository, kpiSnapshots *repository.KPISnapshotRepository, suiteClient *aggregator.SuiteClient, publisher Publisher, logger zerolog.Logger, predictionLogger *aigovmiddleware.PredictionLogger) *Generator {
	return &Generator{
		reports:      reports,
		snapshots:    snapshots,
		kpis:         kpis,
		kpiSnapshots: kpiSnapshots,
		suiteClient:  suiteClient,
		publisher:    publisher,
		logger:       logger.With().Str("component", "visus_report_generator").Logger(),
		predictionLogger: predictionLogger,
	}
}

func (g *Generator) Generate(ctx context.Context, reportID uuid.UUID, triggeredBy *uuid.UUID) (*model.ReportSnapshot, error) {
	reportDef, err := g.reports.GetByID(ctx, reportID)
	if err != nil {
		return nil, err
	}
	start, end, err := ResolvePeriod(reportDef, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if existing, err := g.snapshots.FindByPeriod(ctx, reportDef.TenantID, reportDef.ID, start, end); err == nil {
		return existing, nil
	}

	buildStart := time.Now()
	sectionData := make(map[string]interface{}, len(reportDef.Sections)+1)
	fetchErrors := make(map[string]string)
	for _, section := range reportDef.Sections {
		switch section {
		case "security_posture":
			data, fetchErr := reportsections.BuildSecurity(ctx, g.suiteClient, reportDef.TenantID)
			sectionData[section] = data
			if fetchErr != "" {
				fetchErrors["cyber"] = fetchErr
			}
		case "data_intelligence":
			data, fetchErr := reportsections.BuildDataIntelligence(ctx, g.suiteClient, reportDef.TenantID)
			sectionData[section] = data
			if fetchErr != "" {
				fetchErrors["data"] = fetchErr
			}
		case "governance":
			data, fetchErr := reportsections.BuildGovernance(ctx, g.suiteClient, reportDef.TenantID)
			sectionData[section] = data
			if fetchErr != "" {
				fetchErrors["acta"] = fetchErr
			}
		case "legal":
			data, fetchErr := reportsections.BuildLegal(ctx, g.suiteClient, reportDef.TenantID)
			sectionData[section] = data
			if fetchErr != "" {
				fetchErrors["lex"] = fetchErr
			}
		case "kpi_summary":
			data, fetchErr := reportsections.BuildKPISummary(ctx, g.kpis, g.kpiSnapshots, reportDef.TenantID, start, end)
			sectionData[section] = data
			if fetchErr != "" {
				fetchErrors["kpis"] = fetchErr
			}
		}
	}
	sectionData["recommendations"] = reportsections.BuildRecommendations(sectionData)
	g.recordRecommendationPrediction(ctx, reportDef, sectionData["recommendations"])
	narrative := GenerateNarrative(sectionData, [2]time.Time{start, end})
	durationMS := time.Since(buildStart).Milliseconds()
	snapshot := &model.ReportSnapshot{
		TenantID:         reportDef.TenantID,
		ReportID:         reportDef.ID,
		ReportData:       map[string]any{"sections": sectionData, "report": reportDef},
		Narrative:        &narrative,
		FileFormat:       model.ReportFileJSON,
		PeriodStart:      start,
		PeriodEnd:        end,
		SectionsIncluded: append([]string(nil), reportDef.Sections...),
		GenerationTimeMS: &durationMS,
		SuiteFetchErrors: fetchErrors,
		GeneratedBy:      triggeredBy,
	}
	created, err := g.snapshots.Create(ctx, snapshot)
	if err != nil {
		return nil, err
	}
	nextRun := (*time.Time)(nil)
	if reportDef.Schedule != nil {
		if calculated, calcErr := NextRun(*reportDef.Schedule, time.Now().UTC()); calcErr == nil {
			nextRun = &calculated
		}
	}
	if err := g.reports.UpdateGeneration(ctx, reportDef.TenantID, reportDef.ID, created.GeneratedAt, nextRun); err != nil {
		return nil, err
	}
	if g.publisher != nil {
		event, err := events.NewEvent("visus.report.generated", "visus-service", reportDef.TenantID.String(), map[string]any{
			"report_id":    reportDef.ID,
			"snapshot_id":  created.ID,
			"sections":     created.SectionsIncluded,
			"period_start": start.Format("2006-01-02"),
			"period_end":   end.Format("2006-01-02"),
		})
		if err == nil {
			_ = g.publisher.Publish(ctx, events.Topics.VisusEvents, event)
		}
	}
	return created, nil
}

func (g *Generator) recordRecommendationPrediction(ctx context.Context, reportDef *model.ReportDefinition, recommendationData any) {
	if g.predictionLogger == nil || reportDef == nil {
		return
	}
	entityID := reportDef.ID
	items := recommendationItems(recommendationData)
	_, _ = g.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     reportDef.TenantID,
		ModelSlug:    "visus-recommendation-engine",
		UseCase:      "executive_recommendations",
		EntityType:   "report",
		EntityID:     &entityID,
		Input: map[string]any{
			"report_id": reportDef.ID.String(),
			"sections":  reportDef.Sections,
		},
		InputSummary: map[string]any{
			"report_id": reportDef.ID.String(),
			"section_count": len(reportDef.Sections),
		},
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output:     recommendationData,
				Confidence: 0.90,
				Metadata: map[string]any{
					"matched_rules": recommendationRules(items),
					"recommendation_count": len(items),
				},
			}, nil
		},
	})
}

func recommendationItems(value any) []string {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	rawItems, ok := typed["items"].([]string)
	if ok {
		return rawItems
	}
	rawSlice, ok := typed["items"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(rawSlice))
	for _, item := range rawSlice {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func recommendationRules(items []string) []string {
	rules := make([]string, 0, len(items))
	for _, item := range items {
		switch {
		case strings.Contains(item, "KPI"), strings.Contains(item, "threshold"):
			rules = append(rules, "critical_kpi")
		case strings.Contains(item, "overdue"):
			rules = append(rules, "overdue_action")
		case strings.Contains(item, "contract"):
			rules = append(rules, "expiring_contract")
		default:
			rules = append(rules, "report_signal")
		}
	}
	return rules
}

func ResolvePeriod(report *model.ReportDefinition, now time.Time) (time.Time, time.Time, error) {
	end := dateFloor(now)
	switch report.Period {
	case "7d":
		return end.AddDate(0, 0, -7), end, nil
	case "14d":
		return end.AddDate(0, 0, -14), end, nil
	case "30d":
		return end.AddDate(0, 0, -30), end, nil
	case "90d":
		return end.AddDate(0, 0, -90), end, nil
	case "quarterly":
		month := ((int(end.Month())-1)/3)*3 + 1
		return time.Date(end.Year(), time.Month(month), 1, 0, 0, 0, 0, time.UTC), end, nil
	case "annual":
		return time.Date(end.Year(), 1, 1, 0, 0, 0, 0, time.UTC), end, nil
	case "custom":
		if report.CustomPeriodStart == nil || report.CustomPeriodEnd == nil {
			return time.Time{}, time.Time{}, fmt.Errorf("custom report period requires start and end date")
		}
		return dateFloor(*report.CustomPeriodStart), dateFloor(*report.CustomPeriodEnd), nil
	default:
		return end.AddDate(0, 0, -30), end, nil
	}
}

func dateFloor(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
