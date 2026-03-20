package model

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type TenantStatus string

const (
	TenantStatusActive        TenantStatus = "active"
	TenantStatusInactive      TenantStatus = "inactive"
	TenantStatusSuspended     TenantStatus = "suspended"
	TenantStatusTrial         TenantStatus = "trial"
	TenantStatusOnboarding    TenantStatus = "onboarding"
	TenantStatusDeprovisioned TenantStatus = "deprovisioned"
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
	DeprovisionedAt  *time.Time
	DeprovisionedBy  *string
	RetainUntil      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TenantSettings is the typed representation of the Settings JSONB column.
// All fields are optional (existing tenants may have empty settings).
type TenantSettings struct {
	MaxUsers              *int              `json:"max_users,omitempty"`
	MaxStorageGB          *int              `json:"max_storage_gb,omitempty"`
	EnabledSuites         []string          `json:"enabled_suites,omitempty"`
	MFARequired           *bool             `json:"mfa_required,omitempty"`
	SessionTimeoutMinutes *int              `json:"session_timeout_minutes,omitempty"`
	PasswordPolicy        *PasswordPolicy   `json:"password_policy,omitempty"`
	IPWhitelist           []string          `json:"ip_whitelist,omitempty"`
	CustomDomain          *string           `json:"custom_domain,omitempty"`
	Branding              *BrandingSettings `json:"branding,omitempty"`
}

// PasswordPolicy defines tenant-level password rules.
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireLowercase bool `json:"require_lowercase"`
	RequireNumbers   bool `json:"require_numbers"`
	RequireSpecial   bool `json:"require_special"`
	MaxAgeDays       int  `json:"max_age_days"`
	HistoryCount     int  `json:"history_count"`
}

// BrandingSettings defines tenant UI customization.
type BrandingSettings struct {
	LogoURL      *string `json:"logo_url"`
	PrimaryColor string  `json:"primary_color"`
	AccentColor  string  `json:"accent_color"`
	CompanyName  string  `json:"company_name"`
}

// validSuites is the allowed set of suite identifiers.
var validSuites = map[string]bool{
	"cyber": true, "data": true, "acta": true,
	"lex": true, "visus": true, "platform": true,
}

// ValidateSettingsJSON validates the structure and field values of a tenant settings JSON blob.
// Returns nil if the settings are valid or empty.
func ValidateSettingsJSON(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}

	var s TenantSettings
	if err := json.Unmarshal(raw, &s); err != nil {
		return fmt.Errorf("invalid settings JSON: %w", err)
	}

	if s.MaxUsers != nil && *s.MaxUsers < 0 {
		return fmt.Errorf("max_users must be non-negative")
	}
	if s.MaxStorageGB != nil && *s.MaxStorageGB < 0 {
		return fmt.Errorf("max_storage_gb must be non-negative")
	}
	if s.SessionTimeoutMinutes != nil && *s.SessionTimeoutMinutes < 1 {
		return fmt.Errorf("session_timeout_minutes must be at least 1")
	}
	for _, suite := range s.EnabledSuites {
		if !validSuites[suite] {
			return fmt.Errorf("invalid suite %q in enabled_suites", suite)
		}
	}
	for _, ip := range s.IPWhitelist {
		if net.ParseIP(ip) == nil {
			// Try CIDR
			if _, _, err := net.ParseCIDR(ip); err != nil {
				return fmt.Errorf("invalid IP address or CIDR %q in ip_whitelist", ip)
			}
		}
	}
	if s.PasswordPolicy != nil {
		if s.PasswordPolicy.MinLength < 0 {
			return fmt.Errorf("password_policy.min_length must be non-negative")
		}
		if s.PasswordPolicy.MaxAgeDays < 0 {
			return fmt.Errorf("password_policy.max_age_days must be non-negative")
		}
		if s.PasswordPolicy.HistoryCount < 0 {
			return fmt.Errorf("password_policy.history_count must be non-negative")
		}
	}

	return nil
}
