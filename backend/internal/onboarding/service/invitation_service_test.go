package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
)

func newInvitationServiceForTest(t *testing.T) (*InvitationService, *fakeInvitationRepo, *fakeOnboardingRepo, *fakeUserRepo, *fakeRoleRepo, *fakeEmailSender, uuid.UUID) {
	t.Helper()

	tenantID := uuid.New()
	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	onboardingRepo := newFakeOnboardingRepo()
	invitationRepo := newFakeInvitationRepo()
	emailSender := &fakeEmailSender{}

	onboardingRepo.userRepo = userRepo
	onboardingRepo.roleRepo = roleRepo
	onboardingRepo.invitationRepo = invitationRepo
	onboardingRepo.tenantIdentities[tenantID] = fakeTenantIdentity{
		name:   "Acme Corp",
		slug:   "acme-corp-a1b2",
		status: iammodel.TenantStatusOnboarding,
	}

	roleRepo.addRole(tenantID.String(), "security-analyst", "Security Analyst", []string{"cyber:read"})
	roleRepo.addRole(tenantID.String(), "data-steward", "Data Steward", []string{"data:read"})

	service := NewInvitationService(
		invitationRepo,
		onboardingRepo,
		userRepo,
		roleRepo,
		&fakeSessionRepo{},
		newServiceTestJWTManager(t),
		nil,
		emailSender,
		newDiscardLogger(),
		nil,
		bcrypt.MinCost,
		7*24*time.Hour,
	)

	return service, invitationRepo, onboardingRepo, userRepo, roleRepo, emailSender, tenantID
}

func TestInviteValidBatch(t *testing.T) {
	service, invitationRepo, _, _, _, emailSender, tenantID := newInvitationServiceForTest(t)

	invitations, err := service.CreateBatch(context.Background(), tenantID, uuid.New(), "Jane Admin", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "alice@acme.com", RoleSlug: "security-analyst", Message: "Welcome"},
			{Email: "bob@acme.com", RoleSlug: "security-analyst"},
			{Email: "carol@acme.com", RoleSlug: "data-steward"},
		},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	if len(invitations) != 3 {
		t.Fatalf("expected 3 invitations, got %d", len(invitations))
	}
	if len(emailSender.invitationEmails) != 3 {
		t.Fatalf("expected 3 invitation emails, got %d", len(emailSender.invitationEmails))
	}

	stored, err := invitationRepo.ListByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListByTenant returned error: %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("expected 3 stored invitations, got %d", len(stored))
	}
}

func TestInviteDuplicateEmail(t *testing.T) {
	service, invitationRepo, _, _, _, _, tenantID := newInvitationServiceForTest(t)

	invitations, err := service.CreateBatch(context.Background(), tenantID, uuid.New(), "Jane Admin", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "alice@acme.com", RoleSlug: "security-analyst"},
			{Email: "alice@acme.com", RoleSlug: "security-analyst"},
		},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	if len(invitations) != 1 {
		t.Fatalf("expected 1 invitation after dedupe, got %d", len(invitations))
	}

	stored, err := invitationRepo.ListByTenant(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("ListByTenant returned error: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected 1 stored invitation, got %d", len(stored))
	}
}

func TestInviteMaxPending(t *testing.T) {
	service, invitationRepo, _, _, _, _, tenantID := newInvitationServiceForTest(t)
	for i := 0; i < 50; i++ {
		invitation := &onboardingmodel.Invitation{
			ID:            uuid.New(),
			TenantID:      tenantID,
			Email:         "user" + uuid.NewString() + "@acme.com",
			RoleSlug:      "security-analyst",
			TokenHash:     "hash",
			TokenPrefix:   "prefix123",
			Status:        onboardingmodel.InvitationStatusPending,
			InvitedBy:     uuid.New(),
			InvitedByName: "Jane Admin",
			ExpiresAt:     time.Now().Add(invitationTTL),
		}
		if err := invitationRepo.Create(context.Background(), invitation); err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
	}

	_, err := service.CreateBatch(context.Background(), tenantID, uuid.New(), "Jane Admin", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "overflow@acme.com", RoleSlug: "security-analyst"},
		},
	})
	if !errors.Is(err, iammodel.ErrAccountLocked) {
		t.Fatalf("expected account locked error, got %v", err)
	}
}

func TestAcceptValidToken(t *testing.T) {
	service, invitationRepo, _, userRepo, roleRepo, emailSender, tenantID := newInvitationServiceForTest(t)

	invitations, err := service.CreateBatch(context.Background(), tenantID, uuid.New(), "Jane Admin", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "alice@acme.com", RoleSlug: "security-analyst"},
		},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	if len(emailSender.invitationEmails) != 1 {
		t.Fatalf("expected 1 invitation email, got %d", len(emailSender.invitationEmails))
	}

	resp, err := service.Accept(context.Background(), onboardingdto.AcceptInviteRequest{
		Token:     emailSender.invitationEmails[0].rawToken,
		FirstName: "Alice",
		LastName:  "Smith",
		Password:  "SecureP@ss123!",
	}, "198.51.100.5", "unit-test")
	if err != nil {
		t.Fatalf("Accept returned error: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	user, err := userRepo.GetByEmail(context.Background(), tenantID.String(), "alice@acme.com")
	if err != nil {
		t.Fatalf("GetByEmail returned error: %v", err)
	}
	if user.Status != iammodel.UserStatusActive {
		t.Fatalf("expected invited user to be active, got %s", user.Status)
	}
	roles, err := roleRepo.GetUserRoles(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetUserRoles returned error: %v", err)
	}
	if len(roles) != 1 || roles[0].Slug != "security-analyst" {
		t.Fatalf("expected security-analyst role, got %+v", roles)
	}

	stored, err := invitationRepo.GetByID(context.Background(), tenantID, invitations[0].ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if stored.Status != onboardingmodel.InvitationStatusAccepted {
		t.Fatalf("expected invitation to be accepted, got %s", stored.Status)
	}
}

func TestAcceptExpiredToken(t *testing.T) {
	service, invitationRepo, _, _, _, emailSender, tenantID := newInvitationServiceForTest(t)

	invitations, err := service.CreateBatch(context.Background(), tenantID, uuid.New(), "Jane Admin", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "alice@acme.com", RoleSlug: "security-analyst"},
		},
	})
	if err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}
	invitation, err := invitationRepo.GetByID(context.Background(), tenantID, invitations[0].ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	invitation.ExpiresAt = time.Now().Add(-time.Hour)

	_, err = service.Accept(context.Background(), onboardingdto.AcceptInviteRequest{
		Token:     emailSender.invitationEmails[0].rawToken,
		FirstName: "Alice",
		LastName:  "Smith",
		Password:  "SecureP@ss123!",
	}, "", "")
	if !errors.Is(err, onboardingmodel.ErrExpiredInvitation) {
		t.Fatalf("expected expired invitation error, got %v", err)
	}
}

func TestAcceptUsedToken(t *testing.T) {
	service, _, _, _, _, emailSender, tenantID := newInvitationServiceForTest(t)

	if _, err := service.CreateBatch(context.Background(), tenantID, uuid.New(), "Jane Admin", onboardingdto.BatchInviteRequest{
		Invitations: []onboardingdto.InvitationInput{
			{Email: "alice@acme.com", RoleSlug: "security-analyst"},
		},
	}); err != nil {
		t.Fatalf("CreateBatch returned error: %v", err)
	}

	req := onboardingdto.AcceptInviteRequest{
		Token:     emailSender.invitationEmails[0].rawToken,
		FirstName: "Alice",
		LastName:  "Smith",
		Password:  "SecureP@ss123!",
	}
	if _, err := service.Accept(context.Background(), req, "", ""); err != nil {
		t.Fatalf("first Accept returned error: %v", err)
	}
	if _, err := service.Accept(context.Background(), req, "", ""); !errors.Is(err, iammodel.ErrConflict) {
		t.Fatalf("expected conflict on reused invitation, got %v", err)
	}
}
