package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
)

type registrationOnboardingRepository interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	OrganizationNameExists(ctx context.Context, name string) (bool, error)
	CreateRegistration(ctx context.Context, params onboardingrepo.CreateRegistrationParams) error
	GetLatestEmailVerification(ctx context.Context, email, purpose string) (*onboardingmodel.EmailVerification, error)
	IncrementVerificationAttempts(ctx context.Context, verificationID uuid.UUID) (int, error)
	MarkEmailVerificationVerified(ctx context.Context, verificationID uuid.UUID) error
	ActivateRegistration(ctx context.Context, email string) (*onboardingrepo.ActivationResult, error)
	GetOnboardingByAdminEmail(ctx context.Context, email string) (*onboardingmodel.OnboardingStatus, error)
	CreateEmailVerification(ctx context.Context, email, otpHash string, expiresAt time.Time, ipAddress, userAgent *string) error
}

type invitationRepository interface {
	CountPending(ctx context.Context, tenantID uuid.UUID) (int, error)
	Create(ctx context.Context, invitation *onboardingmodel.Invitation) error
	ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.Invitation, error)
	GetByID(ctx context.Context, tenantID, invitationID uuid.UUID) (*onboardingmodel.Invitation, error)
	ListByPrefix(ctx context.Context, tokenPrefix string) ([]onboardingmodel.Invitation, error)
	UpdateStatus(ctx context.Context, tenantID, invitationID uuid.UUID, status onboardingmodel.InvitationStatus) error
	Refresh(ctx context.Context, tenantID, invitationID uuid.UUID, tokenHash, tokenPrefix string, expiresAt time.Time) error
	ExpirePastDue(ctx context.Context) error
}

type invitationOnboardingRepository interface {
	GetTenantIdentity(ctx context.Context, tenantID uuid.UUID) (name, slug string, status iammodel.TenantStatus, retainUntil *time.Time, err error)
	CreateTenantUserWithRole(ctx context.Context, params onboardingrepo.CreateTenantUserParams) error
}

type wizardOnboardingRepository interface {
	GetOnboardingByTenantID(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error)
	UpdateOrganization(ctx context.Context, tenantID uuid.UUID, orgName string, industry onboardingmodel.OrgIndustry, country string, city *string, size onboardingmodel.OrgSize) (*onboardingmodel.WizardProgress, error)
	UpdateBranding(ctx context.Context, tenantID uuid.UUID, logoFileID *uuid.UUID, primaryColor, accentColor *string) (*onboardingmodel.WizardProgress, error)
	MarkTeamStepCompleted(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error)
	UpdateSuites(ctx context.Context, tenantID uuid.UUID, activeSuites []string) (*onboardingmodel.WizardProgress, error)
	CompleteWizard(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error)
}

type provisioningStepsReader interface {
	ListSteps(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.ProvisioningStep, error)
}

type provisioningOnboardingRepository interface {
	GetOnboardingByTenantID(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error)
	GetTenantIdentity(ctx context.Context, tenantID uuid.UUID) (name, slug string, status iammodel.TenantStatus, retainUntil *time.Time, err error)
}

type provisioningStatusRepository interface {
	Initialize(ctx context.Context, tenantID, onboardingID uuid.UUID, stepNames []string) error
	ListSteps(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.ProvisioningStep, error)
	StartStep(ctx context.Context, tenantID uuid.UUID, stepNumber int) error
	CompleteStep(ctx context.Context, tenantID uuid.UUID, stepNumber int, metadata map[string]any) error
	FailStep(ctx context.Context, tenantID uuid.UUID, stepNumber int, errMessage string, metadata map[string]any) error
	MarkFailed(ctx context.Context, tenantID uuid.UUID, errMessage string) error
	MarkCompleted(ctx context.Context, tenantID uuid.UUID) error
	SetTenantStatus(ctx context.Context, tenantID uuid.UUID, status string) error
}

type tenantIdentityRepository interface {
	GetTenantIdentity(ctx context.Context, tenantID uuid.UUID) (name, slug string, status iammodel.TenantStatus, retainUntil *time.Time, err error)
}

type provisionerRunner interface {
	Provision(ctx context.Context, tenantID uuid.UUID) error
}
