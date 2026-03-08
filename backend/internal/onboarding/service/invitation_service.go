package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

const invitationTTL = 7 * 24 * time.Hour

type InvitationService struct {
	invitationRepo *onboardingrepo.InvitationRepository
	onboardingRepo *onboardingrepo.OnboardingRepository
	userRepo       iamrepo.UserRepository
	roleRepo       iamrepo.RoleRepository
	sessionRepo    iamrepo.SessionRepository
	jwtMgr         *auth.JWTManager
	producer       *events.Producer
	emailSender    EmailSender
	logger         zerolog.Logger
	metrics        *Metrics
	bcryptCost     int
	refreshTTL     time.Duration
}

func NewInvitationService(
	invitationRepo *onboardingrepo.InvitationRepository,
	onboardingRepo *onboardingrepo.OnboardingRepository,
	userRepo iamrepo.UserRepository,
	roleRepo iamrepo.RoleRepository,
	sessionRepo iamrepo.SessionRepository,
	jwtMgr *auth.JWTManager,
	producer *events.Producer,
	emailSender EmailSender,
	logger zerolog.Logger,
	metrics *Metrics,
	bcryptCost int,
	refreshTTL time.Duration,
) *InvitationService {
	return &InvitationService{
		invitationRepo: invitationRepo,
		onboardingRepo: onboardingRepo,
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		sessionRepo:    sessionRepo,
		jwtMgr:         jwtMgr,
		producer:       producer,
		emailSender:    emailSender,
		logger:         logger.With().Str("service", "onboarding_invitation").Logger(),
		metrics:        metrics,
		bcryptCost:     bcryptCost,
		refreshTTL:     refreshTTL,
	}
}

func (s *InvitationService) CreateBatch(ctx context.Context, tenantID, invitedBy uuid.UUID, invitedByName string, req onboardingdto.BatchInviteRequest) ([]onboardingmodel.Invitation, error) {
	if len(req.Invitations) == 0 || len(req.Invitations) > 10 {
		return nil, fmt.Errorf("invitations batch must contain between 1 and 10 invitations: %w", iammodel.ErrValidation)
	}

	pendingCount, err := s.invitationRepo.CountPending(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if pendingCount >= 50 {
		return nil, fmt.Errorf("tenant already has the maximum number of pending invitations: %w", iammodel.ErrAccountLocked)
	}

	orgName, _, _, _, err := s.onboardingRepo.GetTenantIdentity(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	created := make([]onboardingmodel.Invitation, 0, len(req.Invitations))
	seenEmails := map[string]struct{}{}
	for _, item := range req.Invitations {
		email := normalizeEmail(item.Email)
		if _, duplicate := seenEmails[email]; duplicate {
			continue
		}
		seenEmails[email] = struct{}{}

		if !emailRegex.MatchString(email) {
			return nil, fmt.Errorf("invalid invitation email: %w", iammodel.ErrValidation)
		}
		if _, err := s.roleRepo.GetBySlug(ctx, tenantID.String(), strings.TrimSpace(item.RoleSlug)); err != nil {
			return nil, fmt.Errorf("invalid role %q: %w", item.RoleSlug, err)
		}
		if _, err := s.userRepo.GetByEmail(ctx, tenantID.String(), email); err == nil {
			return nil, fmt.Errorf("user %s already belongs to this tenant: %w", email, iammodel.ErrConflict)
		}
		if pendingCount+len(created) >= 50 {
			return nil, fmt.Errorf("tenant already has the maximum number of pending invitations: %w", iammodel.ErrAccountLocked)
		}

		token, tokenPrefix, err := verification.GenerateInviteToken()
		if err != nil {
			return nil, err
		}
		tokenHash, err := verification.HashInviteToken(token)
		if err != nil {
			return nil, err
		}
		message := strings.TrimSpace(item.Message)
		var messagePtr *string
		if message != "" {
			messagePtr = &message
		}

		invitation := onboardingmodel.Invitation{
			TenantID:      tenantID,
			Email:         email,
			RoleSlug:      strings.TrimSpace(item.RoleSlug),
			TokenHash:     tokenHash,
			TokenPrefix:   tokenPrefix,
			Status:        onboardingmodel.InvitationStatusPending,
			InvitedBy:     invitedBy,
			InvitedByName: invitedByName,
			ExpiresAt:     time.Now().Add(invitationTTL),
			Message:       messagePtr,
		}
		if err := s.invitationRepo.Create(ctx, &invitation); err != nil {
			return nil, err
		}

		role, _ := s.roleRepo.GetBySlug(ctx, tenantID.String(), invitation.RoleSlug)
		roleName := invitation.RoleSlug
		if role != nil {
			roleName = role.Name
		}
		if err := s.emailSender.SendInvitationEmail(ctx, email, orgName, invitedByName, roleName, token, messagePtr, invitation.ExpiresAt); err != nil {
			s.logger.Error().Err(err).Str("invitation_id", invitation.ID.String()).Msg("invitation email delivery failed")
		}
		if s.metrics != nil && s.metrics.invitationsTotal != nil {
			s.metrics.invitationsTotal.WithLabelValues("created").Inc()
		}
		created = append(created, invitation)
		publishOnboardingEvent(ctx, s.producer,
			"com.clario360.onboarding.invitation.created",
			tenantID,
			&invitedBy,
			map[string]any{
				"tenant_id": tenantID.String(),
				"email":     maskedEventEmail(email),
				"role":      invitation.RoleSlug,
			},
			s.logger,
		)
	}

	s.updateInvitationAcceptanceRate(ctx, tenantID)
	return created, nil
}

func (s *InvitationService) List(ctx context.Context, tenantID uuid.UUID) ([]onboardingmodel.Invitation, error) {
	return s.invitationRepo.ListByTenant(ctx, tenantID)
}

func (s *InvitationService) Cancel(ctx context.Context, tenantID, invitationID uuid.UUID) error {
	if err := s.invitationRepo.UpdateStatus(ctx, tenantID, invitationID, onboardingmodel.InvitationStatusCancelled); err != nil {
		return err
	}
	if s.metrics != nil && s.metrics.invitationsTotal != nil {
		s.metrics.invitationsTotal.WithLabelValues("cancelled").Inc()
	}
	s.updateInvitationAcceptanceRate(ctx, tenantID)
	return nil
}

func (s *InvitationService) Resend(ctx context.Context, tenantID, invitationID uuid.UUID) error {
	invitation, err := s.invitationRepo.GetByID(ctx, tenantID, invitationID)
	if err != nil {
		return err
	}
	orgName, _, _, _, err := s.onboardingRepo.GetTenantIdentity(ctx, tenantID)
	if err != nil {
		return err
	}
	rawToken, tokenPrefix, err := verification.GenerateInviteToken()
	if err != nil {
		return err
	}
	tokenHash, err := verification.HashInviteToken(rawToken)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(invitationTTL)
	if err := s.invitationRepo.Refresh(ctx, tenantID, invitationID, tokenHash, tokenPrefix, expiresAt); err != nil {
		return err
	}
	role, _ := s.roleRepo.GetBySlug(ctx, tenantID.String(), invitation.RoleSlug)
	roleName := invitation.RoleSlug
	if role != nil {
		roleName = role.Name
	}
	if err := s.emailSender.SendInvitationEmail(ctx, invitation.Email, orgName, invitation.InvitedByName, roleName, rawToken, invitation.Message, expiresAt); err != nil {
		s.logger.Error().Err(err).Str("invitation_id", invitationID.String()).Msg("resend invitation email failed")
	}
	return nil
}

func (s *InvitationService) ValidateToken(ctx context.Context, token string) (*onboardingmodel.InvitationDetails, error) {
	token = strings.TrimSpace(token)
	if len(token) < 8 {
		return nil, fmt.Errorf("invalid or expired invitation: %w", iammodel.ErrInvalidToken)
	}
	if err := s.invitationRepo.ExpirePastDue(ctx); err != nil {
		s.logger.Warn().Err(err).Msg("expire invitations sweep failed")
	}
	candidates, err := s.invitationRepo.ListPendingByPrefix(ctx, token[:8])
	if err != nil {
		return nil, err
	}
	for _, candidate := range candidates {
		if !verification.VerifyInviteToken(candidate.TokenHash, token) {
			continue
		}
		if candidate.ExpiresAt.Before(time.Now()) {
			_ = s.invitationRepo.UpdateStatus(ctx, candidate.TenantID, candidate.ID, onboardingmodel.InvitationStatusExpired)
			if s.metrics != nil && s.metrics.invitationsTotal != nil {
				s.metrics.invitationsTotal.WithLabelValues("expired").Inc()
			}
			return nil, fmt.Errorf("this invitation has expired: %w", iammodel.ErrInvalidToken)
		}
		role, _ := s.roleRepo.GetBySlug(ctx, candidate.TenantID.String(), candidate.RoleSlug)
		roleName := candidate.RoleSlug
		if role != nil {
			roleName = role.Name
		}
		orgName, _, _, _, err := s.onboardingRepo.GetTenantIdentity(ctx, candidate.TenantID)
		if err != nil {
			return nil, err
		}
		return &onboardingmodel.InvitationDetails{
			InvitationID:     candidate.ID,
			TenantID:         candidate.TenantID,
			Email:            candidate.Email,
			RoleSlug:         candidate.RoleSlug,
			RoleName:         roleName,
			OrganizationName: orgName,
			InviterName:      candidate.InvitedByName,
			ExpiresAt:        candidate.ExpiresAt,
			Message:          candidate.Message,
		}, nil
	}
	return nil, fmt.Errorf("invalid or expired invitation: %w", iammodel.ErrInvalidToken)
}

func (s *InvitationService) Accept(ctx context.Context, req onboardingdto.AcceptInviteRequest, ip, userAgent string) (*onboardingdto.AcceptInviteResponse, error) {
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}
	details, err := s.ValidateToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if _, err := s.userRepo.GetByEmail(ctx, details.TenantID.String(), details.Email); err == nil {
		return nil, fmt.Errorf("user already exists for invitation email: %w", iammodel.ErrConflict)
	}
	role, err := s.roleRepo.GetBySlug(ctx, details.TenantID.String(), details.RoleSlug)
	if err != nil {
		return nil, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash invited user password: %w", err)
	}
	userID := uuid.New()
	if err := s.onboardingRepo.CreateTenantUserWithRole(ctx, onboardingrepo.CreateTenantUserParams{
		UserID:       userID,
		TenantID:     details.TenantID,
		RoleID:       uuid.MustParse(role.ID),
		Email:        details.Email,
		FirstName:    strings.TrimSpace(req.FirstName),
		LastName:     strings.TrimSpace(req.LastName),
		PasswordHash: string(passwordHash),
	}); err != nil {
		return nil, err
	}
	if err := s.invitationRepo.MarkAccepted(ctx, details.InvitationID, userID); err != nil {
		return nil, err
	}
	user, err := s.userRepo.GetByID(ctx, userID.String())
	if err != nil {
		return nil, err
	}
	roles, err := s.roleRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	tokens, err := issueAuthTokens(ctx, user, s.sessionRepo, s.jwtMgr, s.refreshTTL, ip, userAgent)
	if err != nil {
		return nil, err
	}
	orgName, _, _, _, err := s.onboardingRepo.GetTenantIdentity(ctx, details.TenantID)
	if err == nil {
		_ = s.emailSender.SendWelcomeEmail(ctx, details.Email, orgName, req.FirstName)
	}
	if s.metrics != nil && s.metrics.invitationsTotal != nil {
		s.metrics.invitationsTotal.WithLabelValues("accepted").Inc()
	}
	s.updateInvitationAcceptanceRate(ctx, details.TenantID)
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.invitation.accepted",
		details.TenantID,
		&userID,
		map[string]any{
			"tenant_id": details.TenantID.String(),
			"email":     maskedEventEmail(details.Email),
			"user_id":   userID.String(),
		},
		s.logger,
	)
	return &onboardingdto.AcceptInviteResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    tokens.ExpiresAt.UTC().Format(time.RFC3339),
		TenantID:     details.TenantID.String(),
		Message:      "Invitation accepted.",
	}, nil
}

func (s *InvitationService) updateInvitationAcceptanceRate(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil || s.metrics.invitationAcceptanceRate == nil {
		return
	}
	invitations, err := s.invitationRepo.ListByTenant(ctx, tenantID)
	if err != nil || len(invitations) == 0 {
		return
	}
	accepted := 0
	for _, invitation := range invitations {
		if invitation.Status == onboardingmodel.InvitationStatusAccepted {
			accepted++
		}
	}
	s.metrics.invitationAcceptanceRate.WithLabelValues(tenantID.String()).Set(float64(accepted) / float64(len(invitations)))
}
