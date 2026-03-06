package model

import (
	"encoding/json"
	"time"
)

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusTrial     TenantStatus = "trial"
)

type SubscriptionTier string

const (
	TierFree         SubscriptionTier = "free"
	TierStarter      SubscriptionTier = "starter"
	TierProfessional SubscriptionTier = "professional"
	TierEnterprise   SubscriptionTier = "enterprise"
)

type Tenant struct {
	ID               string
	Name             string
	Slug             string
	Domain           *string
	Settings         json.RawMessage
	Status           TenantStatus
	SubscriptionTier SubscriptionTier
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
