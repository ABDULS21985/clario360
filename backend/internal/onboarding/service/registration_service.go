package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/events"
	iammodel "github.com/clario360/platform/internal/iam/model"
	iamrepo "github.com/clario360/platform/internal/iam/repository"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
	"github.com/clario360/platform/internal/onboarding/verification"
)

const (
	registrationPurpose     = "registration"
	registrationIPWindow    = time.Hour
	registrationIPLimit     = int64(5)
	registrationEmailWindow = 24 * time.Hour
	registrationEmailLimit  = int64(1)
	verifyEmailRateWindow   = 10 * time.Minute
	verifyEmailRateLimit    = int64(20)
	resendOTPRateWindow     = 60 * time.Second
	resendOTPRateLimit      = int64(1)
	emailVerificationTTL    = 10 * time.Minute
)

type RegistrationService struct {
	onboardingRepo registrationOnboardingRepository
	userRepo       iamrepo.UserRepository
	roleRepo       iamrepo.RoleRepository
	sessionRepo    iamrepo.SessionRepository
	jwtMgr         *auth.JWTManager
	redis          *redis.Client
	producer       *events.Producer
	emailSender    EmailSender
	provisioner    provisionerRunner
	logger         zerolog.Logger
	metrics        *Metrics
	bcryptCost     int
	refreshTTL     time.Duration
}

func NewRegistrationService(
	onboardingRepo registrationOnboardingRepository,
	userRepo iamrepo.UserRepository,
	roleRepo iamrepo.RoleRepository,
	sessionRepo iamrepo.SessionRepository,
	jwtMgr *auth.JWTManager,
	redisClient *redis.Client,
	producer *events.Producer,
	emailSender EmailSender,
	provisioner provisionerRunner,
	logger zerolog.Logger,
	metrics *Metrics,
	bcryptCost int,
	refreshTTL time.Duration,
) *RegistrationService {
	return &RegistrationService{
		onboardingRepo: onboardingRepo,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		sessionRepo:    sessionRepo,
		jwtMgr:         jwtMgr,
		redis:          redisClient,
		producer:       producer,
		emailSender:    emailSender,
		provisioner:    provisioner,
		logger:         logger.With().Str("service", "onboarding_registration").Logger(),
		metrics:        metrics,
		bcryptCost:     bcryptCost,
		refreshTTL:     refreshTTL,
	}
}

func (s *RegistrationService) Register(ctx context.Context, req onboardingdto.RegisterRequest, ip, userAgent string) (*onboardingdto.RegisterResponse, error) {
	email := normalizeEmail(req.AdminEmail)
	if err := validateRegistrationInput(email, req.AdminPassword, req.OrganizationName, req.Country); err != nil {
		if s.metrics != nil && s.metrics.registrationsTotal != nil {
			s.metrics.registrationsTotal.WithLabelValues("failed").Inc()
		}
		return nil, err
	}
	if err := validateIndustry(req.Industry); err != nil {
		if s.metrics != nil && s.metrics.registrationsTotal != nil {
			s.metrics.registrationsTotal.WithLabelValues("failed").Inc()
		}
		return nil, err
	}

	if ok, _, err := consumeThrottle(ctx, s.redis, throttleKey("onboarding:registration:ip", ip), registrationIPLimit, registrationIPWindow); err == nil && !ok {
		return nil, fmt.Errorf("registration rate limit exceeded: %w", iammodel.ErrAccountLocked)
	}
	if ok, _, err := consumeThrottle(ctx, s.redis, throttleKey("onboarding:registration:email", email), registrationEmailLimit, registrationEmailWindow); err == nil && !ok {
		return nil, fmt.Errorf("if this email is available, a verification code has been sent: %w", iammodel.ErrConflict)
	}

	emailExists, err := s.onboardingRepo.EmailExists(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check existing email: %w", err)
	}
	if emailExists {
		if s.metrics != nil && s.metrics.registrationsTotal != nil {
			s.metrics.registrationsTotal.WithLabelValues("failed").Inc()
		}
		return nil, fmt.Errorf("if this email is available, a verification code has been sent: %w", iammodel.ErrConflict)
	}

	orgExists, err := s.onboardingRepo.OrganizationNameExists(ctx, req.OrganizationName)
	if err != nil {
		return nil, fmt.Errorf("check existing organization: %w", err)
	}
	if orgExists {
		if s.metrics != nil && s.metrics.registrationsTotal != nil {
			s.metrics.registrationsTotal.WithLabelValues("failed").Inc()
		}
		return nil, fmt.Errorf("organization name already taken: %w", iammodel.ErrConflict)
	}

	slug, err := generateTenantSlug(req.OrganizationName)
	if err != nil {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.AdminPassword), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash registration password: %w", err)
	}
	otp, err := verification.GenerateOTP(6)
	if err != nil {
		return nil, err
	}
	otpHash, err := verification.HashOTP(otp)
	if err != nil {
		return nil, err
	}

	tenantID := uuid.New()
	adminUserID := uuid.New()
	referralSource := strings.TrimSpace(req.ReferralSource)
	var referralPtr *string
	if referralSource != "" {
		referralPtr = &referralSource
	}

	createErr := s.onboardingRepo.CreateRegistration(ctx, onboardingrepo.CreateRegistrationParams{
		TenantID:        tenantID,
		TenantName:      strings.TrimSpace(req.OrganizationName),
		TenantSlug:      slug,
		AdminUserID:     adminUserID,
		AdminEmail:      email,
		FirstName:       strings.TrimSpace(req.AdminFirstName),
		LastName:        strings.TrimSpace(req.AdminLastName),
		PasswordHash:    string(passwordHash),
		Country:         normalizeCountry(req.Country),
		Industry:        onboardingmodel.OrgIndustry(strings.TrimSpace(strings.ToLower(req.Industry))),
		ReferralSource:  referralPtr,
		OTPHash:         otpHash,
		OTPExpiresAt:    time.Now().Add(emailVerificationTTL),
		IPAddress:       stringPtr(ip),
		UserAgent:       stringPtr(userAgent),
		RolePermissions: rolePermissionsBySlug("tenant-admin"),
	})
	if createErr != nil {
		return nil, fmt.Errorf("create registration: %w", createErr)
	}

	if err := s.emailSender.SendVerificationEmail(ctx, email, req.OrganizationName, strings.TrimSpace(req.AdminFirstName), otp); err != nil {
		s.logger.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("email_hash", hashPII(email)).
			Msg("verification email delivery failed after registration")
	}

	if s.metrics != nil && s.metrics.registrationsTotal != nil {
		s.metrics.registrationsTotal.WithLabelValues("started").Inc()
	}

	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.registration.started",
		tenantID,
		&adminUserID,
		map[string]any{
			"tenant_id":    tenantID.String(),
			"admin_email":  maskedEventEmail(email),
			"country":      normalizeCountry(req.Country),
			"email_hash":   hashPII(email),
			"ip_hash":      hashPII(ip),
			"organization": req.OrganizationName,
		},
		s.logger,
	)

	s.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("email_hash", hashPII(email)).
		Str("ip_hash", hashPII(ip)).
		Str("country", normalizeCountry(req.Country)).
		Msg("tenant registration created")

	return &onboardingdto.RegisterResponse{
		TenantID:        tenantID.String(),
		Email:           maskEmail(email),
		Message:         "Verification email sent.",
		VerificationTTL: int(emailVerificationTTL.Seconds()),
	}, nil
}

func (s *RegistrationService) VerifyEmail(ctx context.Context, req onboardingdto.VerifyEmailRequest, ip, userAgent string) (*onboardingdto.VerifyEmailResponse, error) {
	email := normalizeEmail(req.Email)
	if ok, _, err := consumeThrottle(ctx, s.redis, throttleKey("onboarding:verify-email", email+":"+ip), verifyEmailRateLimit, verifyEmailRateWindow); err == nil && !ok {
		return nil, fmt.Errorf("too many verification attempts: %w", iammodel.ErrAccountLocked)
	}

	verificationRow, err := s.onboardingRepo.GetLatestEmailVerification(ctx, email, registrationPurpose)
	if err != nil {
		return nil, fmt.Errorf("load verification code: %w", iammodel.ErrUnauthorized)
	}
	if verificationRow.Verified {
		return nil, fmt.Errorf("verification code has already been used: %w", iammodel.ErrConflict)
	}
	if verificationRow.ExpiresAt.Before(time.Now()) {
		if s.metrics != nil && s.metrics.otpVerificationsTotal != nil {
			s.metrics.otpVerificationsTotal.WithLabelValues("expired").Inc()
		}
		return nil, fmt.Errorf("verification code expired. Please request a new code: %w", iammodel.ErrUnauthorized)
	}
	if verificationRow.LockedAt != nil || verificationRow.Attempts >= verificationRow.MaxAttempts {
		if s.metrics != nil && s.metrics.otpVerificationsTotal != nil {
			s.metrics.otpVerificationsTotal.WithLabelValues("locked").Inc()
		}
		return nil, fmt.Errorf("too many attempts. Please request a new code: %w", iammodel.ErrAccountLocked)
	}
	if !verification.VerifyOTP(verificationRow.OTPHash, req.OTP) {
		remaining, updateErr := s.onboardingRepo.IncrementVerificationAttempts(ctx, verificationRow.ID)
		if updateErr != nil {
			return nil, fmt.Errorf("update verification attempts: %w", updateErr)
		}
		if s.metrics != nil && s.metrics.otpVerificationsTotal != nil {
			s.metrics.otpVerificationsTotal.WithLabelValues("failure").Inc()
		}
		if remaining <= 0 {
			return nil, fmt.Errorf("too many attempts. Please request a new code: %w", iammodel.ErrAccountLocked)
		}
		return nil, fmt.Errorf("invalid verification code. %d attempts remaining: %w", remaining, iammodel.ErrUnauthorized)
	}

	if err := s.onboardingRepo.MarkEmailVerificationVerified(ctx, verificationRow.ID); err != nil {
		return nil, fmt.Errorf("mark verification successful: %w", err)
	}
	activation, err := s.onboardingRepo.ActivateRegistration(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("activate verified registration: %w", err)
	}

	user, err := s.userRepo.GetByID(ctx, activation.UserID.String())
	if err != nil {
		return nil, fmt.Errorf("load activated user: %w", err)
	}
	roles, err := s.roleRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("load activated user roles: %w", err)
	}
	user.Roles = roles

	tokens, err := issueAuthTokens(ctx, user, s.sessionRepo, s.jwtMgr, s.refreshTTL, ip, userAgent)
	if err != nil {
		return nil, err
	}

	if s.metrics != nil {
		if s.metrics.otpVerificationsTotal != nil {
			s.metrics.otpVerificationsTotal.WithLabelValues("success").Inc()
		}
		if s.metrics.registrationsTotal != nil {
			s.metrics.registrationsTotal.WithLabelValues("verified").Inc()
		}
	}

	userID := activation.UserID
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.email.verified",
		activation.TenantID,
		&userID,
		map[string]any{
			"tenant_id":   activation.TenantID.String(),
			"admin_email": maskedEventEmail(email),
		},
		s.logger,
	)

	if s.provisioner != nil {
		go func(tenantID uuid.UUID) {
			provisionCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := s.provisioner.Provision(provisionCtx, tenantID); err != nil {
				s.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("async provisioning failed")
			}
		}(activation.TenantID)
	}

	return &onboardingdto.VerifyEmailResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    tokens.ExpiresAt.UTC().Format(time.RFC3339),
		TenantID:     activation.TenantID.String(),
		Message:      "Email verified. Redirecting to setup.",
	}, nil
}

func (s *RegistrationService) ResendOTP(ctx context.Context, email, ip, userAgent string) error {
	normalizedEmail := normalizeEmail(email)
	if ok, _, err := consumeThrottle(ctx, s.redis, throttleKey("onboarding:resend-otp", normalizedEmail), resendOTPRateLimit, resendOTPRateWindow); err == nil && !ok {
		return fmt.Errorf("please wait before requesting another code: %w", iammodel.ErrAccountLocked)
	}
	onboardingRow, err := s.onboardingRepo.GetOnboardingByAdminEmail(ctx, normalizedEmail)
	if err != nil {
		return fmt.Errorf("registration not found: %w", iammodel.ErrNotFound)
	}
	if onboardingRow.EmailVerified {
		return fmt.Errorf("email is already verified: %w", iammodel.ErrConflict)
	}

	otp, err := verification.GenerateOTP(6)
	if err != nil {
		return err
	}
	otpHash, err := verification.HashOTP(otp)
	if err != nil {
		return err
	}
	if err := s.onboardingRepo.CreateEmailVerification(ctx, normalizedEmail, otpHash, time.Now().Add(emailVerificationTTL), stringPtr(ip), stringPtr(userAgent)); err != nil {
		return fmt.Errorf("create resend verification code: %w", err)
	}
	orgName := ""
	if onboardingRow.OrgName != nil {
		orgName = *onboardingRow.OrgName
	}
	if err := s.emailSender.SendVerificationEmail(ctx, normalizedEmail, orgName, "Administrator", otp); err != nil {
		s.logger.Error().Err(err).Str("email_hash", hashPII(normalizedEmail)).Msg("resend otp email failed")
	}
	return nil
}

func stringPtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}
