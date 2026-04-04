package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/model"
	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/report"
	"github.com/clario360/platform/internal/visus/repository"
)

type ReportService struct {
	reports    *repository.ReportRepository
	snapshots  *repository.ReportSnapshotRepository
	generator  *report.Generator
	metrics    *visusmetrics.Metrics
	logger     zerolog.Logger
}

func NewReportService(reports *repository.ReportRepository, snapshots *repository.ReportSnapshotRepository, generator *report.Generator, metrics *visusmetrics.Metrics, logger zerolog.Logger) *ReportService {
	return &ReportService{
		reports:   reports,
		snapshots: snapshots,
		generator: generator,
		metrics:   metrics,
		logger:    logger.With().Str("component", "visus_report_service").Logger(),
	}
}

func (s *ReportService) Create(ctx context.Context, item *model.ReportDefinition) (*model.ReportDefinition, error) {
	if err := validateReportDefinition(item); err != nil {
		return nil, err
	}
	nextRun, err := report.NextRunForReport(item.Schedule, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	item.NextRunAt = nextRun
	created, err := s.reports.Create(ctx, item)
	if err == nil {
		s.syncMetrics(ctx, item.TenantID)
	}
	return created, err
}

func (s *ReportService) List(ctx context.Context, tenantID uuid.UUID, page, perPage int, sortCol, sortDir, search, reportType string, autoSend *bool) ([]model.ReportDefinition, int, error) {
	items, total, err := s.reports.List(ctx, tenantID, page, perPage, sortCol, sortDir, search, reportType, autoSend)
	if err == nil {
		s.syncMetrics(ctx, tenantID)
	}
	return items, total, err
}

func (s *ReportService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ReportDefinition, error) {
	return s.reports.Get(ctx, tenantID, id)
}

func (s *ReportService) Update(ctx context.Context, item *model.ReportDefinition) (*model.ReportDefinition, error) {
	if err := validateReportDefinition(item); err != nil {
		return nil, err
	}
	nextRun, err := report.NextRunForReport(item.Schedule, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	item.NextRunAt = nextRun
	updated, err := s.reports.Update(ctx, item)
	if err == nil {
		s.syncMetrics(ctx, item.TenantID)
	}
	return updated, err
}

func (s *ReportService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.reports.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.syncMetrics(ctx, tenantID)
	return nil
}

func (s *ReportService) Generate(ctx context.Context, reportID uuid.UUID, triggeredBy *uuid.UUID) (*model.ReportSnapshot, error) {
	start := time.Now()
	snapshot, err := s.generator.Generate(ctx, reportID, triggeredBy)
	if err == nil && s.metrics != nil && s.metrics.ReportGenerationDurationSeconds != nil {
		s.metrics.ReportGenerationDurationSeconds.Observe(time.Since(start).Seconds())
	}
	return snapshot, err
}

func (s *ReportService) ListSnapshots(ctx context.Context, tenantID, reportID uuid.UUID) ([]model.ReportSnapshot, error) {
	return s.snapshots.ListByReport(ctx, tenantID, reportID)
}

func (s *ReportService) GetSnapshot(ctx context.Context, tenantID, reportID, snapshotID uuid.UUID) (*model.ReportSnapshot, error) {
	return s.snapshots.Get(ctx, tenantID, reportID, snapshotID)
}

func (s *ReportService) LatestSnapshot(ctx context.Context, tenantID, reportID uuid.UUID) (*model.ReportSnapshot, error) {
	return s.snapshots.LatestByReport(ctx, tenantID, reportID)
}

func validateReportDefinition(item *model.ReportDefinition) error {
	if item == nil {
		return fmt.Errorf("%w: report is required", ErrValidation)
	}
	if err := requireName(item.Name); err != nil {
		return err
	}
	if len(item.Sections) == 0 {
		return fmt.Errorf("%w: at least one section is required", ErrValidation)
	}
	if item.Period == "" {
		item.Period = "30d"
	}
	if item.Recipients == nil {
		item.Recipients = []uuid.UUID{}
	}
	return nil
}

func (s *ReportService) syncMetrics(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil || s.metrics.ReportsTotal == nil {
		return
	}
	counts, err := s.reports.CountByType(ctx, tenantID)
	if err != nil {
		return
	}
	for _, typ := range []string{"executive_summary", "security_posture", "data_intelligence", "governance", "legal", "custom"} {
		s.metrics.ReportsTotal.WithLabelValues(tenantID.String(), typ).Set(float64(counts[typ]))
	}
}
