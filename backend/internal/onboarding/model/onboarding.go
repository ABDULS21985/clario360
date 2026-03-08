package model

import (
	"time"

	"github.com/google/uuid"
)

type OrgIndustry string

const (
	OrgIndustryFinancial     OrgIndustry = "financial"
	OrgIndustryGovernment    OrgIndustry = "government"
	OrgIndustryHealthcare    OrgIndustry = "healthcare"
	OrgIndustryTechnology    OrgIndustry = "technology"
	OrgIndustryEnergy        OrgIndustry = "energy"
	OrgIndustryTelecom       OrgIndustry = "telecom"
	OrgIndustryEducation     OrgIndustry = "education"
	OrgIndustryRetail        OrgIndustry = "retail"
	OrgIndustryManufacturing OrgIndustry = "manufacturing"
	OrgIndustryOther         OrgIndustry = "other"
)

var ValidOrgIndustries = map[OrgIndustry]struct{}{
	OrgIndustryFinancial:     {},
	OrgIndustryGovernment:    {},
	OrgIndustryHealthcare:    {},
	OrgIndustryTechnology:    {},
	OrgIndustryEnergy:        {},
	OrgIndustryTelecom:       {},
	OrgIndustryEducation:     {},
	OrgIndustryRetail:        {},
	OrgIndustryManufacturing: {},
	OrgIndustryOther:         {},
}

type OrgSize string

const (
	OrgSize1To50     OrgSize = "1-50"
	OrgSize51To200   OrgSize = "51-200"
	OrgSize201To1000 OrgSize = "201-1000"
	OrgSize1000Plus  OrgSize = "1000+"
)

var ValidOrgSizes = map[OrgSize]struct{}{
	OrgSize1To50:     {},
	OrgSize51To200:   {},
	OrgSize201To1000: {},
	OrgSize1000Plus:  {},
}

type OnboardingProvisioningStatus string

const (
	OnboardingProvisioningPending      OnboardingProvisioningStatus = "pending"
	OnboardingProvisioningProvisioning OnboardingProvisioningStatus = "provisioning"
	OnboardingProvisioningCompleted    OnboardingProvisioningStatus = "completed"
	OnboardingProvisioningFailed       OnboardingProvisioningStatus = "failed"
)

type OnboardingStatus struct {
	ID                      uuid.UUID                    `json:"id"`
	TenantID                uuid.UUID                    `json:"tenant_id"`
	AdminUserID             uuid.UUID                    `json:"admin_user_id"`
	AdminEmail              string                       `json:"admin_email"`
	EmailVerified           bool                         `json:"email_verified"`
	EmailVerifiedAt         *time.Time                   `json:"email_verified_at,omitempty"`
	CurrentStep             int                          `json:"current_step"`
	StepsCompleted          []int                        `json:"steps_completed"`
	WizardCompleted         bool                         `json:"wizard_completed"`
	WizardCompletedAt       *time.Time                   `json:"wizard_completed_at,omitempty"`
	OrgName                 *string                      `json:"org_name,omitempty"`
	OrgIndustry             *OrgIndustry                 `json:"org_industry,omitempty"`
	OrgCountry              string                       `json:"org_country"`
	OrgCity                 *string                      `json:"org_city,omitempty"`
	OrgSize                 *OrgSize                     `json:"org_size,omitempty"`
	LogoFileID              *uuid.UUID                   `json:"logo_file_id,omitempty"`
	PrimaryColor            *string                      `json:"primary_color,omitempty"`
	AccentColor             *string                      `json:"accent_color,omitempty"`
	ActiveSuites            []string                     `json:"active_suites"`
	ProvisioningStatus      OnboardingProvisioningStatus `json:"provisioning_status"`
	ProvisioningStartedAt   *time.Time                   `json:"provisioning_started_at,omitempty"`
	ProvisioningCompletedAt *time.Time                   `json:"provisioning_completed_at,omitempty"`
	ProvisioningError       *string                      `json:"provisioning_error,omitempty"`
	ReferralSource          *string                      `json:"referral_source,omitempty"`
	CreatedAt               time.Time                    `json:"created_at"`
	UpdatedAt               time.Time                    `json:"updated_at"`
}

type WizardProgress struct {
	TenantID                uuid.UUID                    `json:"tenant_id"`
	CurrentStep             int                          `json:"current_step"`
	StepsCompleted          []int                        `json:"steps_completed"`
	WizardCompleted         bool                         `json:"wizard_completed"`
	EmailVerified           bool                         `json:"email_verified"`
	OrganizationName        *string                      `json:"organization_name,omitempty"`
	Industry                *OrgIndustry                 `json:"industry,omitempty"`
	Country                 string                       `json:"country"`
	City                    *string                      `json:"city,omitempty"`
	OrganizationSize        *OrgSize                     `json:"organization_size,omitempty"`
	LogoFileID              *uuid.UUID                   `json:"logo_file_id,omitempty"`
	PrimaryColor            *string                      `json:"primary_color,omitempty"`
	AccentColor             *string                      `json:"accent_color,omitempty"`
	ActiveSuites            []string                     `json:"active_suites"`
	ProvisioningStatus      OnboardingProvisioningStatus `json:"provisioning_status"`
	ProvisioningStartedAt   *time.Time                   `json:"provisioning_started_at,omitempty"`
	ProvisioningCompletedAt *time.Time                   `json:"provisioning_completed_at,omitempty"`
	ProvisioningError       *string                      `json:"provisioning_error,omitempty"`
}

type EmailVerification struct {
	ID          uuid.UUID  `json:"id"`
	Email       string     `json:"email"`
	OTPHash     string     `json:"-"`
	Purpose     string     `json:"purpose"`
	Verified    bool       `json:"verified"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	LockedAt    *time.Time `json:"locked_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	VerifiedAt  *time.Time `json:"verified_at,omitempty"`
	IPAddress   *string    `json:"ip_address,omitempty"`
	UserAgent   *string    `json:"user_agent,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
