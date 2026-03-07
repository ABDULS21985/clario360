package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/kpi"
	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type KPIService struct {
	kpis         *repository.KPIRepository
	snapshots    *repository.KPISnapshotRepository
	engine       *kpi.KPIEngine
	publisher    Publisher
	metrics      *visusmetrics.Metrics
	logger       zerolog.Logger
}

func NewKPIService(kpis *repository.KPIRepository, snapshots *repository.KPISnapshotRepository, engine *kpi.KPIEngine, publisher Publisher, metrics *visusmetrics.Metrics, logger zerolog.Logger) *KPIService {
	return &KPIService{
		kpis:      kpis,
		snapshots: snapshots,
		engine:    engine,
		publisher: publisher,
		metrics:   metrics,
		logger:    logger.With().Str("component", "visus_kpi_service").Logger(),
	}
}

func (s *KPIService) Create(ctx context.Context, item *model.KPIDefinition) (*model.KPIDefinition, error) {
	if err := validateKPI(item); err != nil {
		return nil, err
	}
	created, err := s.kpis.Create(ctx, item)
	if err != nil {
		return nil, err
	}
	s.syncMetrics(ctx, item.TenantID)
	_ = publishEvent(ctx, s.publisher, item.TenantID, "visus.kpi.created", map[string]any{"id": created.ID, "name": created.Name, "suite": created.Suite})
	return created, nil
}

func (s *KPIService) List(ctx context.Context, tenantID uuid.UUID, page, perPage int) ([]model.KPIDefinition, int, error) {
	items, total, err := s.kpis.List(ctx, tenantID, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	for idx := range items {
		latest, _ := s.snapshots.LatestByKPI(ctx, tenantID, items[idx].ID)
		items[idx].LatestSnapshot = latest
	}
	s.syncMetrics(ctx, tenantID)
	return items, total, nil
}

func (s *KPIService) Get(ctx context.Context, tenantID, id uuid.UUID) (*model.KPIDefinition, []model.KPISnapshot, error) {
	item, err := s.kpis.Get(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	history, err := s.snapshots.ListByKPI(ctx, tenantID, id, model.KPIQuery{Limit: 30})
	if err != nil {
		return nil, nil, err
	}
	if latest, err := s.snapshots.LatestByKPI(ctx, tenantID, id); err == nil {
		item.LatestSnapshot = latest
	}
	return item, history, nil
}

func (s *KPIService) Update(ctx context.Context, item *model.KPIDefinition) (*model.KPIDefinition, error) {
	if err := validateKPI(item); err != nil {
		return nil, err
	}
	updated, err := s.kpis.Update(ctx, item)
	if err != nil {
		return nil, err
	}
	s.syncMetrics(ctx, item.TenantID)
	_ = publishEvent(ctx, s.publisher, item.TenantID, "visus.kpi.updated", map[string]any{"id": updated.ID, "name": updated.Name, "changed_fields": []string{"definition"}})
	return updated, nil
}

func (s *KPIService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.kpis.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.syncMetrics(ctx, tenantID)
	return nil
}

func (s *KPIService) History(ctx context.Context, tenantID, id uuid.UUID, start, end *time.Time, limit int) ([]model.KPISnapshot, error) {
	return s.snapshots.ListByKPI(ctx, tenantID, id, model.KPIQuery{Start: start, End: end, Limit: limit})
}

func (s *KPIService) TriggerSnapshot(ctx context.Context, tenantID uuid.UUID) error {
	return s.engine.TakeSnapshots(ctx, tenantID)
}

func (s *KPIService) Summary(ctx context.Context, tenantID uuid.UUID) ([]model.KPIDefinition, error) {
	items, _, err := s.List(ctx, tenantID, 1, 500)
	return items, err
}

func validateKPI(item *model.KPIDefinition) error {
	if item == nil {
		return fmt.Errorf("%w: kpi is required", ErrValidation)
	}
	if err := requireName(item.Name); err != nil {
		return err
	}
	if item.QueryEndpoint == "" || item.ValuePath == "" {
		return fmt.Errorf("%w: query_endpoint and value_path are required", ErrValidation)
	}
	if item.Direction == "" {
		item.Direction = model.KPIDirectionLowerIsBetter
	}
	if item.CalculationType == "" {
		item.CalculationType = model.KPICalcDirect
	}
	if item.SnapshotFrequency == "" {
		item.SnapshotFrequency = model.KPIFrequencyHour
	}
	if item.QueryParams == nil {
		item.QueryParams = map[string]any{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return nil
}

func (s *KPIService) syncMetrics(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil || s.metrics.KPIsTotal == nil {
		return
	}
	counts, err := s.kpis.CountBySuite(ctx, tenantID)
	if err != nil {
		return
	}
	for _, suite := range []string{"cyber", "data", "acta", "lex", "platform", "custom"} {
		for _, enabled := range []string{"true", "false"} {
			value := 0
			if counts[suite] != nil {
				value = counts[suite][enabled]
			}
			s.metrics.KPIsTotal.WithLabelValues(tenantID.String(), suite, enabled).Set(float64(value))
		}
	}
}
