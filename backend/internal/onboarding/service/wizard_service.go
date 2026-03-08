package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

type WizardService struct {
	onboardingRepo wizardOnboardingRepository
	invitationSvc  *InvitationService
	producer       *events.Producer
	logger         zerolog.Logger
	metrics        *Metrics
}

func NewWizardService(
	onboardingRepo wizardOnboardingRepository,
	invitationSvc *InvitationService,
	producer *events.Producer,
	logger zerolog.Logger,
	metrics *Metrics,
) *WizardService {
	return &WizardService{
		onboardingRepo: onboardingRepo,
		invitationSvc:  invitationSvc,
		producer:       producer,
		logger:         logger.With().Str("service", "onboarding_wizard").Logger(),
		metrics:        metrics,
	}
}

func (s *WizardService) GetProgress(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error) {
	current, err := s.onboardingRepo.GetOnboardingByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &onboardingmodel.WizardProgress{
		TenantID:                current.TenantID,
		CurrentStep:             current.CurrentStep,
		StepsCompleted:          current.StepsCompleted,
		WizardCompleted:         current.WizardCompleted,
		EmailVerified:           current.EmailVerified,
		OrganizationName:        current.OrgName,
		Industry:                current.OrgIndustry,
		Country:                 current.OrgCountry,
		City:                    current.OrgCity,
		OrganizationSize:        current.OrgSize,
		LogoFileID:              current.LogoFileID,
		PrimaryColor:            current.PrimaryColor,
		AccentColor:             current.AccentColor,
		ActiveSuites:            current.ActiveSuites,
		ProvisioningStatus:      current.ProvisioningStatus,
		ProvisioningStartedAt:   current.ProvisioningStartedAt,
		ProvisioningCompletedAt: current.ProvisioningCompletedAt,
		ProvisioningError:       current.ProvisioningError,
	}, nil
}

func (s *WizardService) SaveOrganization(ctx context.Context, tenantID uuid.UUID, req onboardingdto.OrganizationDetailsRequest) (*onboardingdto.WizardStepResponse, error) {
	if err := validateOrganizationDetails(req.OrganizationName, req.Country); err != nil {
		return nil, err
	}
	if _, ok := onboardingmodel.ValidOrgIndustries[onboardingmodel.OrgIndustry(req.Industry)]; !ok {
		return nil, fmt.Errorf("invalid organization industry: %w", iammodel.ErrValidation)
	}
	if _, ok := onboardingmodel.ValidOrgSizes[onboardingmodel.OrgSize(req.OrganizationSize)]; !ok {
		return nil, fmt.Errorf("invalid organization size: %w", iammodel.ErrValidation)
	}
	var city *string
	if trimmed := strings.TrimSpace(req.City); trimmed != "" {
		city = &trimmed
	}
	progress, err := s.onboardingRepo.UpdateOrganization(
		ctx,
		tenantID,
		strings.TrimSpace(req.OrganizationName),
		onboardingmodel.OrgIndustry(req.Industry),
		normalizeCountry(req.Country),
		city,
		onboardingmodel.OrgSize(req.OrganizationSize),
	)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil && s.metrics.wizardStepCompletionsTotal != nil {
		s.metrics.wizardStepCompletionsTotal.WithLabelValues("organization").Inc()
	}
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.wizard.step_completed",
		tenantID,
		nil,
		map[string]any{"step_number": 1, "step_name": "organization"},
		s.logger,
	)
	return &onboardingdto.WizardStepResponse{
		Message:          "Organization details saved.",
		CurrentStep:      progress.CurrentStep,
		CompletedSteps:   progress.StepsCompleted,
		OrganizationName: req.OrganizationName,
	}, nil
}

func (s *WizardService) SaveBranding(ctx context.Context, tenantID uuid.UUID, logoFileID *uuid.UUID, primaryColor, accentColor *string) (*onboardingdto.WizardStepResponse, error) {
	if primaryColor != nil {
		if err := validateHexColor(*primaryColor); err != nil {
			return nil, err
		}
	}
	if accentColor != nil {
		if err := validateHexColor(*accentColor); err != nil {
			return nil, err
		}
	}
	progress, err := s.onboardingRepo.UpdateBranding(ctx, tenantID, logoFileID, primaryColor, accentColor)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil && s.metrics.wizardStepCompletionsTotal != nil {
		s.metrics.wizardStepCompletionsTotal.WithLabelValues("branding").Inc()
	}
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.wizard.step_completed",
		tenantID,
		nil,
		map[string]any{"step_number": 2, "step_name": "branding"},
		s.logger,
	)
	return &onboardingdto.WizardStepResponse{
		Message:        "Branding saved.",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}

func (s *WizardService) SaveTeam(ctx context.Context, tenantID, invitedBy uuid.UUID, invitedByName string, req onboardingdto.TeamStepRequest) (*onboardingdto.WizardStepResponse, error) {
	validInvites := make([]onboardingdto.InvitationInput, 0, len(req.Invitations))
	for _, item := range req.Invitations {
		if strings.TrimSpace(item.Email) == "" {
			continue
		}
		validInvites = append(validInvites, item)
	}
	sent := 0
	if len(validInvites) > 0 {
		invitations, err := s.invitationSvc.CreateBatch(ctx, tenantID, invitedBy, invitedByName, onboardingdto.BatchInviteRequest{Invitations: validInvites})
		if err != nil {
			return nil, err
		}
		sent = len(invitations)
	}
	progress, err := s.onboardingRepo.MarkTeamStepCompleted(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil && s.metrics.wizardStepCompletionsTotal != nil {
		s.metrics.wizardStepCompletionsTotal.WithLabelValues("team").Inc()
	}
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.wizard.step_completed",
		tenantID,
		&invitedBy,
		map[string]any{"step_number": 3, "step_name": "team"},
		s.logger,
	)
	return &onboardingdto.WizardStepResponse{
		Message:         "Team step saved.",
		CurrentStep:     progress.CurrentStep,
		CompletedSteps:  progress.StepsCompleted,
		InvitationsSent: sent,
	}, nil
}

func (s *WizardService) SaveSuites(ctx context.Context, tenantID uuid.UUID, req onboardingdto.SuitesStepRequest) (*onboardingdto.WizardStepResponse, error) {
	activeSuites, err := ensureActiveSuites(req.ActiveSuites)
	if err != nil {
		return nil, err
	}
	progress, err := s.onboardingRepo.UpdateSuites(ctx, tenantID, activeSuites)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil && s.metrics.wizardStepCompletionsTotal != nil {
		s.metrics.wizardStepCompletionsTotal.WithLabelValues("suites").Inc()
	}
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.wizard.step_completed",
		tenantID,
		nil,
		map[string]any{"step_number": 4, "step_name": "suites"},
		s.logger,
	)
	return &onboardingdto.WizardStepResponse{
		Message:        "Suites saved.",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}

func (s *WizardService) Complete(ctx context.Context, tenantID uuid.UUID) (*onboardingdto.WizardStepResponse, error) {
	progress, err := s.onboardingRepo.CompleteWizard(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if s.metrics != nil {
		if s.metrics.wizardStepCompletionsTotal != nil {
			s.metrics.wizardStepCompletionsTotal.WithLabelValues("complete").Inc()
		}
		if s.metrics.wizardCompletionsTotal != nil {
			s.metrics.wizardCompletionsTotal.WithLabelValues().Inc()
		}
	}
	publishOnboardingEvent(ctx, s.producer,
		"com.clario360.onboarding.wizard.completed",
		tenantID,
		nil,
		map[string]any{
			"step_number":      5,
			"step_name":        "complete",
			"suites_selected":  progress.ActiveSuites,
			"wizard_completed": true,
		},
		s.logger,
	)
	return &onboardingdto.WizardStepResponse{
		Message:        "Onboarding complete.",
		CurrentStep:    progress.CurrentStep,
		CompletedSteps: progress.StepsCompleted,
	}, nil
}
