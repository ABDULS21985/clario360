package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/events"
	iammodel "github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	"github.com/clario360/platform/internal/onboarding/repository"
	"github.com/clario360/platform/internal/onboarding/verification"
)

// OnboardingService handles registration, email verification, and wizard steps.
type OnboardingService struct {
	repo      *repository.OnboardingRepository
	provRepo  *repository.ProvisioningRepository
	producer  *events.Producer
	logger    zerolog.Logger
	bcryptCost int
}

// NewOnboardingService creates a new OnboardingService.
func NewOnboardingService(
	repo *repository.OnboardingRepository,
	provRepo *repository.ProvisioningRepository,
	producer *events.Producer,
	logger zerolog.Logger,
	bcryptCost int,
) *OnboardingService {
	return &OnboardingService{
		repo:       repo,
		provRepo:   provRepo,
		producer:   producer,
		logger:     logger.With().Str("svc", "onboarding").Logger(),
		bcryptCost: bcryptCost,
	}
}

// Register creates a new tenant + admin user and initiates email verification.
func (s *OnboardingService) Register(ctx context.Context, req *dto.RegisterRequest, ipAddress, userAgent *string) (*dto.RegisterResponse, error) {
	// Check for duplicate email
	emailExists, err := s.repo.EmailExists(ctx, req.AdminEmail)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if emailExists {
		return nil, ErrEmailTaken
	}

	// Check for duplicate org name
	orgExists, err := s.repo.OrganizationNameExists(ctx, req.OrganizationName)
	if err != nil {
		return nil, fmt.Errorf("check org name: %w", err)
	}
	if orgExists {
		return nil, ErrOrganizationNameTaken
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Generate OTP
	otp, err := verification.GenerateOTP(6)
	if err != nil {
		return nil, fmt.Errorf("generate otp: %w", err)
	}
	otpHash, err := verification.HashOTP(otp)
	if err != nil {
		return nil, fmt.Errorf("hash otp: %w", err)
	}

	tenantID := uuid.New()
	adminUserID := uuid.New()
	slug := slugify(req.OrganizationName)

	var referralSource *string
	if req.ReferralSource != "" {
		referralSource = &req.ReferralSource
	}

	industry := onboardingmodel.OrgIndustry(req.Industry)
	if _, ok := onboardingmodel.ValidOrgIndustries[industry]; !ok {
		industry = onboardingmodel.OrgIndustryOther
	}

	adminPermissions := iammodel.SystemRoles[1].Permissions // "Tenant Admin" permissions

	params := repository.CreateRegistrationParams{
		TenantID:        tenantID,
		TenantName:      strings.TrimSpace(req.OrganizationName),
		TenantSlug:      slug,
		AdminUserID:     adminUserID,
		AdminEmail:      strings.ToLower(strings.TrimSpace(req.AdminEmail)),
		FirstName:       strings.TrimSpace(req.AdminFirstName),
		LastName:        strings.TrimSpace(req.AdminLastName),
		PasswordHash:    string(hash),
		Country:         req.Country,
		Industry:        industry,
		ReferralSource:  referralSource,
		OTPHash:         otpHash,
		OTPExpiresAt:    otpExpiry(),
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
		RolePermissions: adminPermissions,
	}

	if err := s.repo.CreateRegistration(ctx, params); err != nil {
		return nil, fmt.Errorf("create registration: %w", err)
	}

	// Publish email notification event
	s.publishOTPEvent(ctx, tenantID.String(), req.AdminEmail, otp)

	return &dto.RegisterResponse{
		TenantID:        tenantID.String(),
		Email:           strings.ToLower(strings.TrimSpace(req.AdminEmail)),
		Message:         "Registration successful. Please check your email for the verification code.",
		VerificationTTL: 600,
	}, nil
}

// VerifyEmail validates the OTP and activates the user account.
func (s *OnboardingService) VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error {
	ev, err := s.repo.GetLatestEmailVerification(ctx, req.Email, "registration")
	if err != nil {
		return ErrInvalidOTP
	}

	if ev.Verified {
		return ErrAlreadyVerified
	}

	if ev.LockedAt != nil {
		return ErrOTPLocked
	}

	if isExpired(ev.ExpiresAt) {
		return ErrOTPExpired
	}

	if !verification.VerifyOTP(ev.OTPHash, req.OTP) {
		_, _ = s.repo.IncrementVerificationAttempts(ctx, ev.ID)
		return ErrInvalidOTP
	}

	if err := s.repo.MarkEmailVerificationVerified(ctx, ev.ID); err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}

	_, err = s.repo.ActivateRegistration(ctx, req.Email)
	if err != nil {
		return fmt.Errorf("activate registration: %w", err)
	}

	return nil
}

// ResendOTP generates a new OTP for the given email.
func (s *OnboardingService) ResendOTP(ctx context.Context, email string, ipAddress, userAgent *string) error {
	otp, err := verification.GenerateOTP(6)
	if err != nil {
		return fmt.Errorf("generate otp: %w", err)
	}
	otpHash, err := verification.HashOTP(otp)
	if err != nil {
		return fmt.Errorf("hash otp: %w", err)
	}

	if err := s.repo.CreateEmailVerification(ctx, email, otpHash, otpExpiry(), ipAddress, userAgent); err != nil {
		return fmt.Errorf("create verification: %w", err)
	}

	// Publish resend OTP event
	onboarding, err := s.repo.GetOnboardingByAdminEmail(ctx, email)
	if err == nil {
		s.publishOTPEvent(ctx, onboarding.TenantID.String(), email, otp)
	}

	return nil
}

// GetOnboardingStatus returns the onboarding status for a tenant.
func (s *OnboardingService) GetOnboardingStatus(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error) {
	return s.repo.GetOnboardingByTenantID(ctx, tenantID)
}

// UpdateOrganization saves step 1 wizard data.
func (s *OnboardingService) UpdateOrganization(ctx context.Context, tenantID uuid.UUID, req *dto.OrganizationDetailsRequest) (*dto.WizardStepResponse, error) {
	industry := onboardingmodel.OrgIndustry(req.Industry)
	if _, ok := onboardingmodel.ValidOrgIndustries[industry]; !ok {
		return nil, ErrInvalidIndustry
	}

	size := onboardingmodel.OrgSize(req.OrganizationSize)
	if _, ok := onboardingmodel.ValidOrgSizes[size]; !ok {
		return nil, ErrInvalidOrgSize
	}

	var city *string
	if req.City != "" {
		c := strings.TrimSpace(req.City)
		city = &c
	}

	progress, err := s.repo.UpdateOrganization(ctx, tenantID, strings.TrimSpace(req.OrganizationName), industry, req.Country, city, size)
	if err != nil {
		return nil, fmt.Errorf("update organization: %w", err)
	}

	name := ""
	if progress.OrganizationName != nil {
		name = *progress.OrganizationName
	}

	return &dto.WizardStepResponse{
		Message:          "Organization details saved",
		CurrentStep:      progress.CurrentStep,
		CompletedSteps:   progress.StepsCompleted,
		OrganizationName: name,
	}, nil
}

// UpdateBranding saves step 2 wizard data.
func (s *OnboardingService) UpdateBranding(ctx context.Context, tenantID uuid.UUID, req *dto.BrandingRequest) (*dto.WizardStepResponse, error) {
	var primary, accent *string
	if req.PrimaryColor != "" {
		primary = &req.PrimaryColor
	}
	if req.AccentColor != "" {
		accent = &req.AccentColor
	}

	progress, err := s.repo.UpdateBranding(ctx, tenantID, nil, primary, accent)
	if err != nil {
		return nil, fmt.Errorf("update branding: %w", err)
	}

	return &dto.WizardStepResponse{
		Message:        "Branding saved",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}

// MarkTeamStepCompleted marks the team invite step as done.
func (s *OnboardingService) MarkTeamStepCompleted(ctx context.Context, tenantID uuid.UUID) (*dto.WizardStepResponse, error) {
	progress, err := s.repo.MarkTeamStepCompleted(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("mark team step: %w", err)
	}

	return &dto.WizardStepResponse{
		Message:        "Team step completed",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}

// UpdateSuites saves step 4 suite selection.
func (s *OnboardingService) UpdateSuites(ctx context.Context, tenantID uuid.UUID, req *dto.SuitesStepRequest) (*dto.WizardStepResponse, error) {
	progress, err := s.repo.UpdateSuites(ctx, tenantID, req.ActiveSuites)
	if err != nil {
		return nil, fmt.Errorf("update suites: %w", err)
	}

	return &dto.WizardStepResponse{
		Message:        "Suite selection saved",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}

// CompleteWizard marks the wizard as done.
func (s *OnboardingService) CompleteWizard(ctx context.Context, tenantID uuid.UUID) (*dto.WizardStepResponse, error) {
	progress, err := s.repo.CompleteWizard(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("complete wizard: %w", err)
	}

	return &dto.WizardStepResponse{
		Message:        "Setup complete",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}

// GetProvisioningStatus returns the provisioning status for a tenant.
func (s *OnboardingService) GetProvisioningStatus(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.ProvisioningStatus, error) {
	return s.provRepo.GetStatus(ctx, tenantID)
}

// publishOTPEvent publishes an OTP notification event.
func (s *OnboardingService) publishOTPEvent(ctx context.Context, tenantID, email, otp string) {
	if s.producer == nil {
		return
	}
	evt, err := events.NewEvent(
		"platform.onboarding.otp.requested",
		"onboarding-service",
		tenantID,
		map[string]string{"email": email, "otp": otp},
	)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to build otp event")
		return
	}
	if err := s.producer.Publish(ctx, events.Topics.OnboardingEvents, evt); err != nil {
		s.logger.Warn().Err(err).Msg("failed to publish otp event")
	}
}

// slugify converts an org name to a URL-safe slug.
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var result strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			result.WriteRune(r)
		case r >= '0' && r <= '9':
			result.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			result.WriteByte('-')
		}
	}
	slug := strings.Trim(result.String(), "-")
	if slug == "" {
		slug = uuid.New().String()[:8]
	}
	return slug
}
