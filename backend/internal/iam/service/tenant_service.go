package service

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

type TenantService struct {
	tenantRepo repository.TenantRepository
	roleRepo   repository.RoleRepository
	producer   *events.Producer
	logger     zerolog.Logger
}

func NewTenantService(
	tenantRepo repository.TenantRepository,
	roleRepo repository.RoleRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *TenantService {
	return &TenantService{
		tenantRepo: tenantRepo,
		roleRepo:   roleRepo,
		producer:   producer,
		logger:     logger,
	}
}

func (s *TenantService) List(ctx context.Context, page, perPage int) ([]dto.TenantResponse, int, error) {
	tenants, total, err := s.tenantRepo.List(ctx, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	return dto.TenantsToResponse(tenants), total, nil
}

func (s *TenantService) GetByID(ctx context.Context, tenantID string) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := dto.TenantToResponse(tenant)
	return &resp, nil
}

func (s *TenantService) Create(ctx context.Context, req *dto.CreateTenantRequest) (*dto.TenantResponse, error) {
	// Check slug uniqueness
	_, err := s.tenantRepo.GetBySlug(ctx, req.Slug)
	if err == nil {
		return nil, fmt.Errorf("tenant slug %s: %w", req.Slug, model.ErrConflict)
	}

	tier := model.SubscriptionTier(req.SubscriptionTier)
	if tier == "" {
		tier = model.TierFree
	}

	tenant := &model.Tenant{
		Name:             req.Name,
		Slug:             req.Slug,
		Domain:           req.Domain,
		Settings:         req.Settings,
		Status:           model.TenantStatusActive,
		SubscriptionTier: tier,
	}

	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("creating tenant: %w", err)
	}

	// Seed system roles for the new tenant
	if err := s.roleRepo.SeedSystemRoles(ctx, tenant.ID); err != nil {
		s.logger.Error().Err(err).Str("tenant_id", tenant.ID).Msg("failed to seed system roles")
	}

	s.publishEvent(ctx, "tenant.created", tenant.ID)

	resp := dto.TenantToResponse(tenant)
	return &resp, nil
}

func (s *TenantService) Update(ctx context.Context, tenantID string, req *dto.UpdateTenantRequest) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Domain != nil {
		tenant.Domain = req.Domain
	}
	if req.Status != nil {
		tenant.Status = model.TenantStatus(*req.Status)
	}
	if req.SubscriptionTier != nil {
		tenant.SubscriptionTier = model.SubscriptionTier(*req.SubscriptionTier)
	}
	if req.Settings != nil {
		tenant.Settings = req.Settings
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	s.publishEvent(ctx, "tenant.updated", tenant.ID)

	resp := dto.TenantToResponse(tenant)
	return &resp, nil
}

func (s *TenantService) publishEvent(ctx context.Context, eventType, tenantID string) {
	if s.producer == nil {
		return
	}
	payload := map[string]any{}
	if tenantID != "" {
		payload["tenant_id"] = tenantID
	}

	evt, err := events.NewEvent(normalizeIAMEventType(eventType), "iam-service", tenantID, payload)
	if err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to create event")
		return
	}
	if err := s.producer.Publish(ctx, "platform.iam.events", evt); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}
