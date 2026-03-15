package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

type TenantService struct {
	tenantRepo repository.TenantRepository
	roleRepo   repository.RoleRepository
	userRepo   repository.UserRepository
	pool       *pgxpool.Pool
	producer   *events.Producer
	logger     zerolog.Logger
	bcryptCost int
}

func NewTenantService(
	tenantRepo repository.TenantRepository,
	roleRepo repository.RoleRepository,
	userRepo repository.UserRepository,
	pool *pgxpool.Pool,
	producer *events.Producer,
	logger zerolog.Logger,
	bcryptCost int,
) *TenantService {
	if bcryptCost < 10 {
		bcryptCost = 12
	}
	return &TenantService{
		tenantRepo: tenantRepo,
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		pool:       pool,
		producer:   producer,
		logger:     logger,
		bcryptCost: bcryptCost,
	}
}

func (s *TenantService) List(ctx context.Context, page, perPage int, search, status, tier, sort, order string) ([]dto.TenantResponse, int, error) {
	tenants, total, err := s.tenantRepo.List(ctx, page, perPage, repository.TenantListParams{
		Search:           search,
		Status:           status,
		SubscriptionTier: tier,
		Sort:             sort,
		Order:            order,
	})
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

// Provision creates a tenant with an initial owner user assigned the tenant-admin role.
func (s *TenantService) Provision(ctx context.Context, req *dto.ProvisionTenantRequest) (*dto.TenantResponse, error) {
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

	// Create the owner user with a temporary password.
	tempPassword, err := generateTempPassword()
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate temp password")
		// Continue — tenant is created, owner just won't have an account yet.
	} else {
		hash, err := bcrypt.GenerateFromPassword([]byte(tempPassword), s.bcryptCost)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to hash temp password")
		} else {
			parts := splitName(req.OwnerName)
			owner := &model.User{
				TenantID:     tenant.ID,
				Email:        req.OwnerEmail,
				PasswordHash: string(hash),
				FirstName:    parts[0],
				LastName:     parts[1],
				Status:       model.UserStatusActive,
			}
			if err := s.userRepo.Create(ctx, owner); err != nil {
				s.logger.Error().Err(err).
					Str("tenant_id", tenant.ID).
					Str("email", req.OwnerEmail).
					Msg("failed to create owner user")
			} else {
				// Assign tenant-admin role.
				adminRole, err := s.roleRepo.GetBySlug(ctx, tenant.ID, "tenant-admin")
				if err != nil {
					s.logger.Error().Err(err).Str("tenant_id", tenant.ID).Msg("tenant-admin role not found")
				} else {
					if err := s.roleRepo.AssignToUser(ctx, owner.ID, adminRole.ID, tenant.ID, "system"); err != nil {
						s.logger.Error().Err(err).Str("user_id", owner.ID).Msg("failed to assign tenant-admin role")
					}
				}
			}
		}
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
		// Server-side merge: merge incoming settings on top of existing ones
		// to prevent concurrent overwrites of unrelated fields.
		merged, err := mergeSettingsJSON(tenant.Settings, req.Settings)
		if err != nil {
			return nil, fmt.Errorf("merging settings: %w", err)
		}
		tenant.Settings = merged
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	s.publishEvent(ctx, "tenant.updated", tenant.ID)

	resp := dto.TenantToResponse(tenant)
	return &resp, nil
}

// GetUsage returns aggregated usage statistics for a tenant.
func (s *TenantService) GetUsage(ctx context.Context, tenantID string) (*dto.TenantUsageResponse, error) {
	// Verify tenant exists.
	if _, err := s.tenantRepo.GetByID(ctx, tenantID); err != nil {
		return nil, err
	}

	usage := &dto.TenantUsageResponse{
		TenantID:   tenantID,
		Period:     "current",
		SuiteUsage: make(map[string]dto.SuiteUsageItem),
	}

	// Active users count.
	if s.userRepo != nil {
		count, err := s.userRepo.CountByTenant(ctx, tenantID)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("failed to count users")
		} else {
			usage.ActiveUsers = count
		}
	}

	// API calls from audit_logs (best-effort).
	if s.pool != nil {
		var apiCalls int
		err := s.pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`, tenantID,
		).Scan(&apiCalls)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("failed to count audit logs")
		} else {
			usage.APICalls = apiCalls
		}

		// Storage used from files table (best-effort).
		var storageBytes *int64
		err = s.pool.QueryRow(ctx,
			`SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE tenant_id = $1`, tenantID,
		).Scan(&storageBytes)
		if err != nil {
			s.logger.Warn().Err(err).Str("tenant_id", tenantID).Msg("failed to sum storage")
		} else if storageBytes != nil {
			usage.StorageUsedBytes = *storageBytes
		}
	}

	return usage, nil
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

// mergeSettingsJSON performs a shallow merge of incoming JSON on top of existing JSON.
func mergeSettingsJSON(existing, incoming json.RawMessage) (json.RawMessage, error) {
	base := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &base); err != nil {
			// If existing settings are invalid, start fresh.
			base = map[string]any{}
		}
	}

	overlay := map[string]any{}
	if err := json.Unmarshal(incoming, &overlay); err != nil {
		return nil, fmt.Errorf("invalid settings JSON: %w", err)
	}

	for k, v := range overlay {
		base[k] = v
	}

	return json.Marshal(base)
}

// generateTempPassword creates a random 24-character hex password.
func generateTempPassword() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// splitName splits a full name into first and last name parts.
func splitName(fullName string) [2]string {
	parts := strings.SplitN(strings.TrimSpace(fullName), " ", 2)
	if len(parts) == 1 {
		return [2]string{parts[0], ""}
	}
	return [2]string{parts[0], parts[1]}
}
