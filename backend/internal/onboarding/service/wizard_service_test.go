package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

// ─── fake wizard repository ───────────────────────────────────────────────────

// fakeWizardRepo is a lightweight in-memory implementation of wizardOnboardingRepository
// that requires no external services. All mutations are applied in-place so tests can
// inspect the resulting WizardProgress without a real database.
type fakeWizardRepo struct {
	state map[uuid.UUID]*onboardingmodel.OnboardingStatus
	// err, if non-nil, is returned by every method.
	err error
}

func newFakeWizardRepo(tenantID uuid.UUID) *fakeWizardRepo {
	status := &onboardingmodel.OnboardingStatus{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		AdminUserID:        uuid.New(),
		AdminEmail:         "admin@test.example",
		CurrentStep:        1,
		StepsCompleted:     []int{},
		ActiveSuites:       []string{"cyber", "data", "visus"},
		ProvisioningStatus: onboardingmodel.OnboardingProvisioningPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	return &fakeWizardRepo{
		state: map[uuid.UUID]*onboardingmodel.OnboardingStatus{tenantID: status},
	}
}

func (f *fakeWizardRepo) get(tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error) {
	if f.err != nil {
		return nil, f.err
	}
	s, ok := f.state[tenantID]
	if !ok {
		return nil, iammodel.ErrNotFound
	}
	return s, nil
}

func (f *fakeWizardRepo) progress(s *onboardingmodel.OnboardingStatus) *onboardingmodel.WizardProgress {
	return &onboardingmodel.WizardProgress{
		TenantID:           s.TenantID,
		CurrentStep:        s.CurrentStep,
		CurrentStepLabel:   toWizardStepLabel(s.CurrentStep),
		StepsCompleted:     s.StepsCompleted,
		WizardCompleted:    s.WizardCompleted,
		EmailVerified:      s.EmailVerified,
		OrganizationName:   s.OrgName,
		Industry:           s.OrgIndustry,
		Country:            s.OrgCountry,
		City:               s.OrgCity,
		OrganizationSize:   s.OrgSize,
		LogoFileID:         s.LogoFileID,
		PrimaryColor:       s.PrimaryColor,
		AccentColor:        s.AccentColor,
		ActiveSuites:       s.ActiveSuites,
		ProvisioningStatus: s.ProvisioningStatus,
	}
}

func (f *fakeWizardRepo) GetOnboardingByTenantID(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.OnboardingStatus, error) {
	return f.get(tenantID)
}

func (f *fakeWizardRepo) UpdateOrganization(ctx context.Context, tenantID uuid.UUID, orgName string, industry onboardingmodel.OrgIndustry, country string, city *string, size onboardingmodel.OrgSize) (*onboardingmodel.WizardProgress, error) {
	s, err := f.get(tenantID)
	if err != nil {
		return nil, err
	}
	s.OrgName = &orgName
	s.OrgIndustry = &industry
	s.OrgCountry = country
	s.OrgCity = city
	s.OrgSize = &size
	if s.CurrentStep < 2 {
		s.CurrentStep = 2
		s.StepsCompleted = append(s.StepsCompleted, 1)
	}
	return f.progress(s), nil
}

func (f *fakeWizardRepo) UpdateBranding(ctx context.Context, tenantID uuid.UUID, logoFileID *uuid.UUID, primaryColor, accentColor *string) (*onboardingmodel.WizardProgress, error) {
	s, err := f.get(tenantID)
	if err != nil {
		return nil, err
	}
	if logoFileID != nil {
		s.LogoFileID = logoFileID
	}
	if primaryColor != nil {
		s.PrimaryColor = primaryColor
	}
	if accentColor != nil {
		s.AccentColor = accentColor
	}
	if s.CurrentStep < 3 {
		s.CurrentStep = 3
		s.StepsCompleted = append(s.StepsCompleted, 2)
	}
	return f.progress(s), nil
}

func (f *fakeWizardRepo) MarkTeamStepCompleted(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error) {
	s, err := f.get(tenantID)
	if err != nil {
		return nil, err
	}
	if s.CurrentStep < 4 {
		s.CurrentStep = 4
		s.StepsCompleted = append(s.StepsCompleted, 3)
	}
	return f.progress(s), nil
}

func (f *fakeWizardRepo) UpdateSuites(ctx context.Context, tenantID uuid.UUID, activeSuites []string) (*onboardingmodel.WizardProgress, error) {
	s, err := f.get(tenantID)
	if err != nil {
		return nil, err
	}
	s.ActiveSuites = activeSuites
	if s.CurrentStep < 5 {
		s.CurrentStep = 5
		s.StepsCompleted = append(s.StepsCompleted, 4)
	}
	return f.progress(s), nil
}

func (f *fakeWizardRepo) CompleteWizard(ctx context.Context, tenantID uuid.UUID) (*onboardingmodel.WizardProgress, error) {
	s, err := f.get(tenantID)
	if err != nil {
		return nil, err
	}
	s.WizardCompleted = true
	now := time.Now()
	s.WizardCompletedAt = &now
	return f.progress(s), nil
}

// ─── builder helpers ──────────────────────────────────────────────────────────

func newTestWizardService(t *testing.T, repo *fakeWizardRepo) *WizardService {
	t.Helper()
	return NewWizardService(repo, nil, nil, nil, newDiscardLogger(), nil)
}

// newTestWizardServiceWithInvitations builds a WizardService that can send real
// invitation batches via an InvitationService backed entirely by in-memory fakes.
func newTestWizardServiceWithInvitations(t *testing.T, tenantID uuid.UUID, repo *fakeWizardRepo) (*WizardService, *fakeEmailSender) {
	t.Helper()

	invRepo := newFakeInvitationRepo()
	onbRepo := newFakeOnboardingRepo()
	// Seed a tenant identity so GetTenantIdentity succeeds inside CreateBatch.
	onbRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Test Org",
		slug:   "test-org",
		status: iammodel.TenantStatusOnboarding,
	}
	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	// Seed the "viewer" role so CreateBatch can look it up by slug.
	roleRepo.addRole(tenantID.String(), "viewer", "Viewer", []string{"*:read"})

	sessionRepo := &fakeSessionRepo{}
	jwtMgr := newServiceTestJWTManager(t)
	emailSender := &fakeEmailSender{}

	invSvc := NewInvitationService(
		invRepo,
		onbRepo,
		userRepo,
		roleRepo,
		sessionRepo,
		jwtMgr,
		nil, // events producer — nil is safe
		emailSender,
		newDiscardLogger(),
		nil, // metrics — nil is safe
		bcrypt.MinCost,
		7*24*time.Hour,
	)

	svc := NewWizardService(repo, nil, invSvc, nil, newDiscardLogger(), nil)
	return svc, emailSender
}

// ─── SaveOrganization ─────────────────────────────────────────────────────────

func TestWizardService_SaveOrganization_Valid(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	resp, err := svc.SaveOrganization(context.Background(), tenantID, onboardingdto.OrganizationDetailsRequest{
		OrganizationName: "Acme Corp",
		Industry:         "technology",
		Country:          "US",
		City:             "San Francisco",
		OrganizationSize: "51-200",
	})
	if err != nil {
		t.Fatalf("SaveOrganization failed: %v", err)
	}
	if resp.CurrentStep != 2 {
		t.Errorf("expected current_step=2, got %d", resp.CurrentStep)
	}
	if len(resp.CompletedSteps) == 0 || resp.CompletedSteps[0] != 1 {
		t.Errorf("expected completed_steps to contain 1, got %v", resp.CompletedSteps)
	}
}

func TestWizardService_SaveOrganization_NameTooShort(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveOrganization(context.Background(), tenantID, onboardingdto.OrganizationDetailsRequest{
		OrganizationName: "X",
		Industry:         "technology",
		Country:          "US",
		OrganizationSize: "1-50",
	})
	if err == nil {
		t.Fatal("expected validation error for short org name, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestWizardService_SaveOrganization_InvalidCountry(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveOrganization(context.Background(), tenantID, onboardingdto.OrganizationDetailsRequest{
		OrganizationName: "Acme Corp",
		Industry:         "technology",
		Country:          "USA", // 3-letter code — invalid
		OrganizationSize: "1-50",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid country code, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestWizardService_SaveOrganization_InvalidIndustry(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveOrganization(context.Background(), tenantID, onboardingdto.OrganizationDetailsRequest{
		OrganizationName: "Acme Corp",
		Industry:         "unicorns", // not a valid industry
		Country:          "US",
		OrganizationSize: "1-50",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid industry, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestWizardService_SaveOrganization_InvalidSize(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveOrganization(context.Background(), tenantID, onboardingdto.OrganizationDetailsRequest{
		OrganizationName: "Acme Corp",
		Industry:         "technology",
		Country:          "US",
		OrganizationSize: "99999", // not a valid size
	})
	if err == nil {
		t.Fatal("expected validation error for invalid org size, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestWizardService_SaveOrganization_RepoError(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	boom := errors.New("db connection lost")
	repo.err = boom
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveOrganization(context.Background(), tenantID, onboardingdto.OrganizationDetailsRequest{
		OrganizationName: "Acme Corp",
		Industry:         "technology",
		Country:          "US",
		OrganizationSize: "1-50",
	})
	if !errors.Is(err, boom) {
		t.Errorf("expected repo error to propagate, got %v", err)
	}
}

// ─── SaveBranding ─────────────────────────────────────────────────────────────

func TestWizardService_SaveBranding_ColorsOnly(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	primary := "#006B3F"
	accent := "#C5A04E"
	resp, err := svc.SaveBranding(context.Background(), tenantID, nil, &primary, &accent)
	if err != nil {
		t.Fatalf("SaveBranding failed: %v", err)
	}
	if resp.CurrentStep != 3 {
		t.Errorf("expected current_step=3, got %d", resp.CurrentStep)
	}
	// Verify state was persisted in the fake repo.
	stored := repo.state[tenantID]
	if stored.PrimaryColor == nil || *stored.PrimaryColor != primary {
		t.Errorf("primary color not persisted: %v", stored.PrimaryColor)
	}
	if stored.AccentColor == nil || *stored.AccentColor != accent {
		t.Errorf("accent color not persisted: %v", stored.AccentColor)
	}
	if stored.LogoFileID != nil {
		t.Errorf("expected logo_file_id to remain nil when not supplied")
	}
}

func TestWizardService_SaveBranding_WithLogoFileID(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	logoID := uuid.New()
	_, err := svc.SaveBranding(context.Background(), tenantID, &logoID, nil, nil)
	if err != nil {
		t.Fatalf("SaveBranding with logo failed: %v", err)
	}
	stored := repo.state[tenantID]
	if stored.LogoFileID == nil || *stored.LogoFileID != logoID {
		t.Errorf("expected logo_file_id %s, got %v", logoID, stored.LogoFileID)
	}
}

func TestWizardService_SaveBranding_InvalidHexColor(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	bad := "not-a-color"
	_, err := svc.SaveBranding(context.Background(), tenantID, nil, &bad, nil)
	if err == nil {
		t.Fatal("expected validation error for invalid hex color, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// ─── SaveTeam ─────────────────────────────────────────────────────────────────

func TestWizardService_SaveTeam_NoInvitations(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	// invitationSvc is nil — CreateBatch must NOT be reached when invitations are empty.
	svc := newTestWizardService(t, repo)

	resp, err := svc.SaveTeam(context.Background(), tenantID, uuid.New(), "Admin User",
		onboardingdto.TeamStepRequest{Invitations: []onboardingdto.InvitationInput{}},
	)
	if err != nil {
		t.Fatalf("SaveTeam (no invitations) failed: %v", err)
	}
	if resp.InvitationsSent != 0 {
		t.Errorf("expected invitations_sent=0, got %d", resp.InvitationsSent)
	}
	if resp.CurrentStep != 4 {
		t.Errorf("expected current_step=4, got %d", resp.CurrentStep)
	}
}

func TestWizardService_SaveTeam_BlankEmailFiltered(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	// invitationSvc is nil but all emails are blank → CreateBatch is never called.
	svc := newTestWizardService(t, repo)

	resp, err := svc.SaveTeam(context.Background(), tenantID, uuid.New(), "Admin",
		onboardingdto.TeamStepRequest{
			Invitations: []onboardingdto.InvitationInput{
				{Email: "   ", RoleSlug: "viewer"},
				{Email: "", RoleSlug: "admin"},
			},
		},
	)
	if err != nil {
		t.Fatalf("SaveTeam (blank emails) failed: %v", err)
	}
	if resp.InvitationsSent != 0 {
		t.Errorf("expected invitations_sent=0 (blanks filtered), got %d", resp.InvitationsSent)
	}
}

func TestWizardService_SaveTeam_WithInvitations(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc, emailSender := newTestWizardServiceWithInvitations(t, tenantID, repo)

	inviterID := uuid.New()
	resp, err := svc.SaveTeam(context.Background(), tenantID, inviterID, "Admin User",
		onboardingdto.TeamStepRequest{
			Invitations: []onboardingdto.InvitationInput{
				{Email: "alice@example.com", RoleSlug: "viewer"},
			},
		},
	)
	if err != nil {
		t.Fatalf("SaveTeam (with invitations) failed: %v", err)
	}
	if resp.InvitationsSent != 1 {
		t.Errorf("expected invitations_sent=1, got %d", resp.InvitationsSent)
	}
	if len(emailSender.invitationEmails) != 1 {
		t.Errorf("expected 1 invitation email sent, got %d", len(emailSender.invitationEmails))
	}
	if emailSender.invitationEmails[0].email != "alice@example.com" {
		t.Errorf("unexpected invitation email recipient: %s", emailSender.invitationEmails[0].email)
	}
}

// ─── SaveSuites ───────────────────────────────────────────────────────────────

func TestWizardService_SaveSuites_Valid(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	resp, err := svc.SaveSuites(context.Background(), tenantID, onboardingdto.SuitesStepRequest{
		ActiveSuites: []string{"cyber", "data"},
	})
	if err != nil {
		t.Fatalf("SaveSuites failed: %v", err)
	}
	if resp.CurrentStep != 5 {
		t.Errorf("expected current_step=5, got %d", resp.CurrentStep)
	}
	stored := repo.state[tenantID]
	if len(stored.ActiveSuites) != 2 {
		t.Errorf("expected 2 active suites persisted, got %v", stored.ActiveSuites)
	}
}

func TestWizardService_SaveSuites_DeduplicatesEntries(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	resp, err := svc.SaveSuites(context.Background(), tenantID, onboardingdto.SuitesStepRequest{
		ActiveSuites: []string{"cyber", "cyber", "data"},
	})
	if err != nil {
		t.Fatalf("SaveSuites with duplicates failed: %v", err)
	}
	stored := repo.state[tenantID]
	if len(stored.ActiveSuites) != 2 {
		t.Errorf("expected duplicates deduplicated to 2, got %v", stored.ActiveSuites)
	}
	_ = resp
}

func TestWizardService_SaveSuites_EmptyList(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveSuites(context.Background(), tenantID, onboardingdto.SuitesStepRequest{
		ActiveSuites: []string{},
	})
	if err == nil {
		t.Fatal("expected validation error for empty suites list, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestWizardService_SaveSuites_UnknownSuite(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	_, err := svc.SaveSuites(context.Background(), tenantID, onboardingdto.SuitesStepRequest{
		ActiveSuites: []string{"cyber", "unknownSuite"},
	})
	if err == nil {
		t.Fatal("expected validation error for unknown suite, got nil")
	}
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// ─── Complete ─────────────────────────────────────────────────────────────────

func TestWizardService_Complete_SetsWizardCompleted(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	resp, err := svc.Complete(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp.Message != "Onboarding complete." {
		t.Errorf("unexpected message: %s", resp.Message)
	}
	stored := repo.state[tenantID]
	if !stored.WizardCompleted {
		t.Error("expected wizard_completed=true after Complete()")
	}
	if stored.WizardCompletedAt == nil {
		t.Error("expected wizard_completed_at to be set")
	}
}

func TestWizardService_Complete_Idempotent(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	svc := newTestWizardService(t, repo)

	// Call Complete twice — both calls must succeed.
	if _, err := svc.Complete(context.Background(), tenantID); err != nil {
		t.Fatalf("first Complete failed: %v", err)
	}
	if _, err := svc.Complete(context.Background(), tenantID); err != nil {
		t.Fatalf("second Complete failed: %v", err)
	}
	stored := repo.state[tenantID]
	if !stored.WizardCompleted {
		t.Error("expected wizard_completed=true after second call")
	}
}

func TestWizardService_Complete_RepoError(t *testing.T) {
	tenantID := uuid.New()
	repo := newFakeWizardRepo(tenantID)
	boom := errors.New("transaction aborted")
	repo.err = boom
	svc := newTestWizardService(t, repo)

	_, err := svc.Complete(context.Background(), tenantID)
	if !errors.Is(err, boom) {
		t.Errorf("expected repo error to propagate, got %v", err)
	}
}
