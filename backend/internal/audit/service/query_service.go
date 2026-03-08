package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/dto"
	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
)

// QueryService handles audit log query operations.
type QueryService struct {
	repo    *repository.AuditRepository
	masking *MaskingService
	logger  zerolog.Logger
}

// NewQueryService creates a new QueryService.
func NewQueryService(repo *repository.AuditRepository, masking *MaskingService, logger zerolog.Logger) *QueryService {
	return &QueryService{
		repo:    repo,
		masking: masking,
		logger:  logger,
	}
}

// Query executes a filtered, paginated audit log query.
func (s *QueryService) Query(ctx context.Context, params *dto.QueryParams, callerRoles []string) (*dto.PaginatedResult, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("list").Observe(time.Since(start).Seconds())
	}()

	filter := repository.QueryFilter{
		TenantID:     params.TenantID,
		UserID:       params.UserID,
		Service:      params.Service,
		Action:       params.Action,
		ResourceType: params.ResourceType,
		ResourceID:   params.ResourceID,
		DateFrom:     params.DateFrom,
		DateTo:       params.DateTo,
		Search:       params.Search,
		Severity:     params.Severity,
		Sort:         params.Sort,
		Order:        params.Order,
		Limit:        params.PerPage,
		Offset:       params.Offset(),
	}

	entries, total, err := s.repo.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply PII masking based on caller role
	masked := s.masking.MaskEntries(entries, callerRoles)

	metrics.QueryResults.WithLabelValues("list").Add(float64(len(masked)))

	return &dto.PaginatedResult{
		Data: masked,
		Meta: dto.NewPagination(params.Page, params.PerPage, total),
	}, nil
}

// GetByID retrieves a single audit entry by ID.
func (s *QueryService) GetByID(ctx context.Context, tenantID, id string, callerRoles []string) (*model.AuditEntry, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	entry, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	masked := s.masking.MaskEntry(entry, callerRoles)
	return &masked, nil
}

// GetStats returns aggregated statistics for a tenant.
func (s *QueryService) GetStats(ctx context.Context, tenantID string, dateFrom, dateTo time.Time) (*model.AuditStats, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("stats").Observe(time.Since(start).Seconds())
	}()

	return s.repo.GetStats(ctx, tenantID, dateFrom, dateTo)
}

// GetTimeline returns activity timeline for a specific resource.
func (s *QueryService) GetTimeline(ctx context.Context, tenantID, resourceID string, page, perPage int, callerRoles []string) (*dto.PaginatedResult, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("timeline").Observe(time.Since(start).Seconds())
	}()

	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	entries, total, err := s.repo.GetTimeline(ctx, tenantID, resourceID, perPage, offset)
	if err != nil {
		return nil, err
	}

	masked := s.masking.MaskEntries(entries, callerRoles)

	return &dto.PaginatedResult{
		Data: masked,
		Meta: dto.NewPagination(page, perPage, total),
	}, nil
}
