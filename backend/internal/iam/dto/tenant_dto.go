package dto

import (
	"encoding/json"
	"time"

	"github.com/clario360/platform/internal/iam/model"
)

// ---------- Requests ----------

type CreateTenantRequest struct {
	Name             string          `json:"name" validate:"required,min=1,max=255"`
	Slug             string          `json:"slug" validate:"required,min=1,max=100"`
	Domain           *string         `json:"domain,omitempty"`
	SubscriptionTier string          `json:"subscription_tier" validate:"omitempty,oneof=free starter professional enterprise"`
	Settings         json.RawMessage `json:"settings,omitempty"`
}

// ProvisionTenantRequest creates a tenant and its initial owner user.
type ProvisionTenantRequest struct {
	Name             string          `json:"name" validate:"required,min=1,max=255"`
	Slug             string          `json:"slug" validate:"required,min=1,max=100"`
	SubscriptionTier string          `json:"subscription_tier" validate:"omitempty,oneof=free starter professional enterprise"`
	OwnerEmail       string          `json:"owner_email" validate:"required,email"`
	OwnerName        string          `json:"owner_name" validate:"required,min=1,max=255"`
	// OwnerPassword is optional. If omitted, a secure random password is generated and returned.
	OwnerPassword string          `json:"owner_password,omitempty"`
	Settings      json.RawMessage `json:"settings,omitempty"`
}

// ProvisionTenantResponse extends TenantResponse with the owner's initial password.
// TempPassword is only non-empty on initial provisioning and must be stored securely.
type ProvisionTenantResponse struct {
	TenantResponse
	TempPassword string `json:"temp_password"`
}

type UpdateTenantRequest struct {
	Name             *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Domain           *string         `json:"domain,omitempty"`
	Status           *string         `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended trial onboarding deprovisioned"`
	SubscriptionTier *string         `json:"subscription_tier,omitempty" validate:"omitempty,oneof=free starter professional enterprise"`
	Settings         json.RawMessage `json:"settings,omitempty"`
}

// ---------- Responses ----------

type TenantResponse struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Slug             string          `json:"slug"`
	Domain           *string         `json:"domain,omitempty"`
	Settings         json.RawMessage `json:"settings"`
	Status           string          `json:"status"`
	SubscriptionTier string          `json:"subscription_tier"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

func TenantToResponse(t *model.Tenant) TenantResponse {
	return TenantResponse{
		ID:               t.ID,
		Name:             t.Name,
		Slug:             t.Slug,
		Domain:           t.Domain,
		Settings:         t.Settings,
		Status:           string(t.Status),
		SubscriptionTier: string(t.SubscriptionTier),
		CreatedAt:        t.CreatedAt,
		UpdatedAt:        t.UpdatedAt,
	}
}

func TenantsToResponse(tenants []model.Tenant) []TenantResponse {
	resp := make([]TenantResponse, len(tenants))
	for i := range tenants {
		resp[i] = TenantToResponse(&tenants[i])
	}
	return resp
}

// TenantUsageResponse contains aggregated usage statistics for a tenant.
type TenantUsageResponse struct {
	TenantID         string                    `json:"tenant_id"`
	Period           string                    `json:"period"`
	ActiveUsers      int                       `json:"active_users"`
	APICalls         int                       `json:"api_calls"`
	StorageUsedBytes int64                     `json:"storage_used_bytes"`
	BandwidthBytes   int64                     `json:"bandwidth_bytes"`
	SuiteUsage       map[string]SuiteUsageItem `json:"suite_usage"`
}

// SuiteUsageItem contains per-suite usage data.
type SuiteUsageItem struct {
	Suite        string  `json:"suite"`
	APICalls     int     `json:"api_calls"`
	ActiveUsers  int     `json:"active_users"`
	LastAccessed *string `json:"last_accessed"`
}
