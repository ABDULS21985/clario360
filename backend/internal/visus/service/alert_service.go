package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	visusalert "github.com/clario360/platform/internal/visus/alert"
	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type AlertService struct {
	alerts     *repository.AlertRepository
	generator  *visusalert.Generator
	correlator *visusalert.Correlator
	escalator  *visusalert.Escalator
	publisher  Publisher
	metrics    *visusmetrics.Metrics
	logger     zerolog.Logger
}

func NewAlertService(alerts *repository.AlertRepository, generator *visusalert.Generator, correlator *visusalert.Correlator, escalator *visusalert.Escalator, publisher Publisher, metrics *visusmetrics.Metrics, logger zerolog.Logger) *AlertService {
	return &AlertService{
		alerts:     alerts,
		generator:  generator,
		correlator: correlator,
		escalator:  escalator,
		publisher:  publisher,
		metrics:    metrics,
		logger:     logger.With().Str("component", "visus_alert_service").Logger(),
	}
}

func (s *AlertService) List(ctx context.Context, tenantID uuid.UUID, filters repository.AlertListFilters, page, perPage int, sortCol, sortDir string) ([]model.ExecutiveAlert, int, error) {
	items, total, err := s.alerts.List(ctx, tenantID, filters, page, perPage, sortCol, sortDir)
	if err == nil {
		s.syncMetrics(ctx, tenantID)
	}
	return items, total, err
}

func (s *AlertService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.ExecutiveAlert, error) {
	return s.alerts.Get(ctx, tenantID, id)
}

func (s *AlertService) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status model.AlertStatus, actorID *uuid.UUID, notes, dismissReason *string) (*model.ExecutiveAlert, error) {
	item, err := s.alerts.UpdateStatus(ctx, tenantID, id, status, actorID, notes, dismissReason)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.publisher, tenantID, "visus.alert.status_changed", map[string]any{"id": item.ID, "new_status": item.Status, "actioned_by": actorID})
	s.syncMetrics(ctx, tenantID)
	return item, nil
}

func (s *AlertService) Count(ctx context.Context, tenantID uuid.UUID) (int, error) {
	return s.alerts.CountUnactioned(ctx, tenantID)
}

func (s *AlertService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.AlertStats, error) {
	return s.alerts.Stats(ctx, tenantID)
}

func (s *AlertService) Create(ctx context.Context, alert *model.ExecutiveAlert) (*model.ExecutiveAlert, error) {
	created, err := s.generator.CreateAlert(ctx, alert)
	if err == nil {
		s.syncMetrics(ctx, alert.TenantID)
	}
	return created, err
}

func (s *AlertService) RunCorrelation(ctx context.Context, tenantID uuid.UUID, actorID *uuid.UUID) error {
	patterns, err := s.correlator.DetectPatterns(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, pattern := range patterns {
		_, err := s.generator.CreateAlert(ctx, &model.ExecutiveAlert{
			TenantID:        tenantID,
			Title:           pattern.Title,
			Description:     pattern.Description,
			Category:        pattern.Category,
			Severity:        pattern.Severity,
			SourceSuite:     pattern.SourceSuite,
			SourceType:      pattern.SourceType,
			SourceEntityID:  pattern.SourceEntityID,
			DedupKey:        &pattern.DedupKey,
			OccurrenceCount: 1,
			FirstSeenAt:     time.Now().UTC(),
			LastSeenAt:      time.Now().UTC(),
			Metadata:        pattern.Metadata,
		})
		if err != nil {
			return err
		}
	}
	return s.escalator.EscalateStaleCritical(ctx, tenantID, actorID)
}

func (s *AlertService) syncMetrics(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil || s.metrics.ExecutiveAlertsTotal == nil {
		return
	}
	stats, err := s.alerts.Stats(ctx, tenantID)
	if err != nil {
		return
	}
	for category, count := range stats.ByCategory {
		s.metrics.ExecutiveAlertsTotal.WithLabelValues(tenantID.String(), category, "", "").Set(float64(count))
	}
}
