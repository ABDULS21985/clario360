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

type UpdateTenantRequest struct {
	Name             *string         `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Domain           *string         `json:"domain,omitempty"`
	Status           *string         `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended trial"`
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
