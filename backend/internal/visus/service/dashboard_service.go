package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type DashboardService struct {
	dashboards *repository.DashboardRepository
	widgets    *repository.WidgetRepository
	publisher  Publisher
	metrics    *visusmetrics.Metrics
	logger     zerolog.Logger
}

func NewDashboardService(dashboards *repository.DashboardRepository, widgets *repository.WidgetRepository, publisher Publisher, metrics *visusmetrics.Metrics, logger zerolog.Logger) *DashboardService {
	return &DashboardService{
		dashboards: dashboards,
		widgets:    widgets,
		publisher:  publisher,
		metrics:    metrics,
		logger:     logger.With().Str("component", "visus_dashboard_service").Logger(),
	}
}

func (s *DashboardService) Create(ctx context.Context, dashboard *model.Dashboard) (*model.Dashboard, error) {
	if err := validateDashboard(dashboard); err != nil {
		return nil, err
	}
	if dashboard.IsDefault {
		if err := s.dashboards.ClearDefault(ctx, dashboard.TenantID, nil); err != nil {
			return nil, err
		}
	}
	created, err := s.dashboards.Create(ctx, dashboard)
	if err != nil {
		return nil, err
	}
	s.syncMetrics(ctx, dashboard.TenantID)
	_ = publishEvent(ctx, s.publisher, dashboard.TenantID, "visus.dashboard.created", map[string]any{
		"id":         created.ID,
		"name":       created.Name,
		"created_by": created.CreatedBy,
	})
	return created, nil
}

func (s *DashboardService) List(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, page, perPage int) ([]model.Dashboard, int, error) {
	items, total, err := s.dashboards.ListAccessible(ctx, tenantID, userID, page, perPage)
	if err == nil {
		s.syncMetrics(ctx, tenantID)
	}
	return items, total, err
}

func (s *DashboardService) Get(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, id uuid.UUID) (*model.Dashboard, error) {
	dashboard, err := s.dashboards.GetByID(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	widgets, err := s.widgets.ListByDashboard(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	dashboard.Widgets = widgets
	return dashboard, nil
}

func (s *DashboardService) Update(ctx context.Context, dashboard *model.Dashboard) (*model.Dashboard, error) {
	if err := validateDashboard(dashboard); err != nil {
		return nil, err
	}
	existing, err := s.dashboards.GetByID(ctx, dashboard.TenantID, nil, dashboard.ID)
	if err != nil {
		return nil, err
	}
	if existing.IsSystem && dashboard.Name != existing.Name {
		return nil, fmt.Errorf("%w: system dashboards cannot be renamed", ErrForbidden)
	}
	if dashboard.IsDefault {
		if err := s.dashboards.ClearDefault(ctx, dashboard.TenantID, &dashboard.ID); err != nil {
			return nil, err
		}
	}
	updated, err := s.dashboards.Update(ctx, dashboard)
	if err != nil {
		return nil, err
	}
	s.syncMetrics(ctx, dashboard.TenantID)
	_ = publishEvent(ctx, s.publisher, dashboard.TenantID, "visus.dashboard.updated", map[string]any{
		"id":             updated.ID,
		"name":           updated.Name,
		"changed_fields": []string{"name", "description", "visibility", "shared_with", "metadata", "tags"},
	})
	return updated, nil
}

func (s *DashboardService) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	existing, err := s.dashboards.GetByID(ctx, tenantID, nil, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return fmt.Errorf("%w: system dashboards cannot be deleted", ErrForbidden)
	}
	if err := s.dashboards.SoftDelete(ctx, tenantID, id); err != nil {
		return err
	}
	s.syncMetrics(ctx, tenantID)
	return nil
}

func (s *DashboardService) Duplicate(ctx context.Context, tenantID, userID, id uuid.UUID) (*model.Dashboard, error) {
	existing, err := s.Get(ctx, tenantID, &userID, id)
	if err != nil {
		return nil, err
	}
	copyDashboard := *existing
	copyDashboard.ID = uuid.Nil
	copyDashboard.Name = existing.Name + " Copy"
	copyDashboard.Visibility = model.DashboardVisibilityPrivate
	copyDashboard.SharedWith = []uuid.UUID{}
	copyDashboard.IsDefault = false
	copyDashboard.IsSystem = false
	copyDashboard.CreatedBy = userID
	created, err := s.Create(ctx, &copyDashboard)
	if err != nil {
		return nil, err
	}
	for _, widget := range existing.Widgets {
		copyWidget := widget
		copyWidget.ID = uuid.Nil
		copyWidget.DashboardID = created.ID
		if _, err := s.widgets.Create(ctx, &copyWidget); err != nil {
			return nil, err
		}
	}
	return s.Get(ctx, tenantID, &userID, created.ID)
}

func (s *DashboardService) Share(ctx context.Context, tenantID, id uuid.UUID, visibility model.DashboardVisibility, sharedWith []uuid.UUID) (*model.Dashboard, error) {
	existing, err := s.dashboards.GetByID(ctx, tenantID, nil, id)
	if err != nil {
		return nil, err
	}
	existing.Visibility = visibility
	existing.SharedWith = sharedWith
	updated, err := s.dashboards.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	_ = publishEvent(ctx, s.publisher, tenantID, "visus.dashboard.shared", map[string]any{
		"id":          updated.ID,
		"visibility":  updated.Visibility,
		"shared_with": updated.SharedWith,
	})
	return updated, nil
}

func validateDashboard(dashboard *model.Dashboard) error {
	if dashboard == nil {
		return fmt.Errorf("%w: dashboard is required", ErrValidation)
	}
	if err := requireName(dashboard.Name); err != nil {
		return err
	}
	if dashboard.GridColumns == 0 {
		dashboard.GridColumns = 12
	}
	if dashboard.GridColumns < 1 || dashboard.GridColumns > 12 {
		return fmt.Errorf("%w: grid_columns must be between 1 and 12", ErrValidation)
	}
	switch dashboard.Visibility {
	case model.DashboardVisibilityPrivate, model.DashboardVisibilityTeam, model.DashboardVisibilityOrganization, model.DashboardVisibilityPublic:
	default:
		return fmt.Errorf("%w: invalid visibility", ErrValidation)
	}
	if dashboard.Metadata == nil {
		dashboard.Metadata = map[string]any{}
	}
	if dashboard.Tags == nil {
		dashboard.Tags = []string{}
	}
	if dashboard.SharedWith == nil {
		dashboard.SharedWith = []uuid.UUID{}
	}
	return nil
}

func (s *DashboardService) syncMetrics(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil || s.metrics.DashboardsTotal == nil {
		return
	}
	counts, err := s.dashboards.CountByVisibility(ctx, tenantID)
	if err != nil {
		s.logger.Warn().Err(err).Str("tenant_id", tenantID.String()).Msg("failed to sync dashboard metrics")
		return
	}
	for _, visibility := range []string{"private", "team", "organization", "public"} {
		s.metrics.DashboardsTotal.WithLabelValues(tenantID.String(), visibility).Set(float64(counts[visibility]))
	}
}
