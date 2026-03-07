package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/visus/aggregator"
	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type WidgetService struct {
	dashboards   *repository.DashboardRepository
	widgets      *repository.WidgetRepository
	kpis         *repository.KPIRepository
	kpiSnapshots *repository.KPISnapshotRepository
	alerts       *repository.AlertRepository
	suiteClient  *aggregator.SuiteClient
	metrics      *visusmetrics.Metrics
	logger       zerolog.Logger
}

func NewWidgetService(dashboards *repository.DashboardRepository, widgets *repository.WidgetRepository, kpis *repository.KPIRepository, kpiSnapshots *repository.KPISnapshotRepository, alerts *repository.AlertRepository, suiteClient *aggregator.SuiteClient, metrics *visusmetrics.Metrics, logger zerolog.Logger) *WidgetService {
	return &WidgetService{
		dashboards:   dashboards,
		widgets:      widgets,
		kpis:         kpis,
		kpiSnapshots: kpiSnapshots,
		alerts:       alerts,
		suiteClient:  suiteClient,
		metrics:      metrics,
		logger:       logger.With().Str("component", "visus_widget_service").Logger(),
	}
}

func (s *WidgetService) Create(ctx context.Context, widget *model.Widget) (*model.Widget, error) {
	if err := s.ensureDashboard(ctx, widget.TenantID, widget.DashboardID); err != nil {
		return nil, err
	}
	if err := validateWidget(widget); err != nil {
		return nil, err
	}
	widgets, err := s.widgets.ListByDashboard(ctx, widget.TenantID, widget.DashboardID)
	if err != nil {
		return nil, err
	}
	widgets = append(widgets, *widget)
	if err := ValidateWidgetLayout(widgets); err != nil {
		return nil, err
	}
	created, err := s.widgets.Create(ctx, widget)
	if err == nil {
		s.syncMetrics(ctx, widget.TenantID)
	}
	return created, err
}

func (s *WidgetService) List(ctx context.Context, tenantID, dashboardID uuid.UUID) ([]model.Widget, error) {
	if err := s.ensureDashboard(ctx, tenantID, dashboardID); err != nil {
		return nil, err
	}
	items, err := s.widgets.ListByDashboard(ctx, tenantID, dashboardID)
	if err == nil {
		s.syncMetrics(ctx, tenantID)
	}
	return items, err
}

func (s *WidgetService) Update(ctx context.Context, widget *model.Widget) (*model.Widget, error) {
	if err := validateWidget(widget); err != nil {
		return nil, err
	}
	items, err := s.widgets.ListByDashboard(ctx, widget.TenantID, widget.DashboardID)
	if err != nil {
		return nil, err
	}
	replaced := make([]model.Widget, 0, len(items))
	for _, item := range items {
		if item.ID == widget.ID {
			replaced = append(replaced, *widget)
			continue
		}
		replaced = append(replaced, item)
	}
	if err := ValidateWidgetLayout(replaced); err != nil {
		return nil, err
	}
	return s.widgets.Update(ctx, widget)
}

func (s *WidgetService) Delete(ctx context.Context, tenantID, dashboardID, id uuid.UUID) error {
	if err := s.ensureDashboard(ctx, tenantID, dashboardID); err != nil {
		return err
	}
	if err := s.widgets.Delete(ctx, tenantID, dashboardID, id); err != nil {
		return err
	}
	s.syncMetrics(ctx, tenantID)
	return nil
}

func (s *WidgetService) UpdateLayout(ctx context.Context, tenantID, dashboardID uuid.UUID, positions map[uuid.UUID]model.WidgetPosition) error {
	items, err := s.widgets.ListByDashboard(ctx, tenantID, dashboardID)
	if err != nil {
		return err
	}
	index := make(map[uuid.UUID]model.Widget, len(items))
	for _, item := range items {
		index[item.ID] = item
	}
	layout := make([]model.Widget, 0, len(items))
	for id, pos := range positions {
		item, ok := index[id]
		if !ok {
			return fmt.Errorf("%w: widget %s does not belong to dashboard", ErrValidation, id)
		}
		item.Position = pos
		layout = append(layout, item)
		delete(index, id)
	}
	for _, item := range index {
		layout = append(layout, item)
	}
	if err := ValidateWidgetLayout(layout); err != nil {
		return err
	}
	return s.widgets.UpdateLayout(ctx, tenantID, dashboardID, positions)
}

func (s *WidgetService) GetWidgetData(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	if widget == nil {
		return nil, fmt.Errorf("%w: widget is required", ErrValidation)
	}
	start := time.Now()
	defer func() {
		if s.metrics != nil && s.metrics.WidgetDataFetchDurationSeconds != nil {
			s.metrics.WidgetDataFetchDurationSeconds.WithLabelValues(string(widget.Type)).Observe(time.Since(start).Seconds())
		}
	}()

	switch widget.Type {
	case model.WidgetTypeKPICard:
		return s.resolveKPICard(ctx, tenantID, widget)
	case model.WidgetTypeGauge:
		return s.resolveGauge(ctx, tenantID, widget)
	case model.WidgetTypeSparkline:
		return s.resolveSparkline(ctx, tenantID, widget)
	case model.WidgetTypeTrendIndicator:
		return s.resolveTrendIndicator(ctx, tenantID, widget)
	case model.WidgetTypeAlertFeed:
		return s.resolveAlertFeed(ctx, tenantID, widget)
	case model.WidgetTypeText:
		return map[string]any{"content": stringFromConfig(widget.Config, "content")}, nil
	case model.WidgetTypeStatusGrid:
		return s.resolveStatusGrid(ctx, tenantID, widget)
	case model.WidgetTypeLineChart, model.WidgetTypeAreaChart, model.WidgetTypeBarChart:
		return s.resolveSeriesWidget(ctx, tenantID, widget)
	case model.WidgetTypePieChart:
		return s.resolvePieChart(ctx, tenantID, widget)
	case model.WidgetTypeTable:
		return s.resolveTable(ctx, tenantID, widget)
	case model.WidgetTypeHeatmap:
		return s.resolveHeatmap(ctx, tenantID, widget)
	default:
		return nil, fmt.Errorf("%w: unsupported widget type", ErrValidation)
	}
}

func ValidateWidgetLayout(widgets []model.Widget) error {
	for _, widget := range widgets {
		if widget.Position.X < 0 || widget.Position.Y < 0 {
			return fmt.Errorf("%w: widget %s has negative coordinates", ErrValidation, widget.ID)
		}
		if widget.Position.W < 1 || widget.Position.W > 12 || widget.Position.H < 1 || widget.Position.H > 8 {
			return fmt.Errorf("%w: widget %s has invalid size", ErrValidation, widget.ID)
		}
		if widget.Position.X+widget.Position.W > 12 {
			return fmt.Errorf("%w: widget %s exceeds grid boundary", ErrValidation, widget.ID)
		}
	}
	for i := 0; i < len(widgets); i++ {
		for j := i + 1; j < len(widgets); j++ {
			if overlaps(widgets[i].Position, widgets[j].Position) {
				return fmt.Errorf("%w: widgets %s and %s overlap", ErrValidation, widgets[i].ID, widgets[j].ID)
			}
		}
	}
	return nil
}

func overlaps(a, b model.WidgetPosition) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X && a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

func validateWidget(widget *model.Widget) error {
	if widget == nil {
		return fmt.Errorf("%w: widget is required", ErrValidation)
	}
	if strings.TrimSpace(widget.Title) == "" {
		return fmt.Errorf("%w: widget title is required", ErrValidation)
	}
	if err := ValidateWidgetLayout([]model.Widget{*widget}); err != nil {
		return err
	}
	if widget.Config == nil {
		widget.Config = map[string]any{}
	}
	if widget.RefreshIntervalSeconds <= 0 {
		widget.RefreshIntervalSeconds = 60
	}
	return nil
}

func (s *WidgetService) ensureDashboard(ctx context.Context, tenantID, dashboardID uuid.UUID) error {
	_, err := s.dashboards.GetByID(ctx, tenantID, nil, dashboardID)
	return err
}
