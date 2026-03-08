package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	"github.com/clario360/platform/internal/onboarding/verification"
)

func TestRegisterValidInput(t *testing.T) {
	onboardingRepo := newFakeOnboardingRepo()
	userRepo := newFakeUserRepo()
	roleRepo := newFakeRoleRepo()
	onboardingRepo.userRepo = userRepo
	onboardingRepo.roleRepo = roleRepo
	emailSender := &fakeEmailSender{}

	service := NewRegistrationService(
		onboardingRepo,
		userRepo,
		roleRepo,
		&fakeSessionRepo{},
		newServiceTestJWTManager(t),
		nil,
		nil,
		emailSender,
		nil,
		newDiscardLogger(),
		nil,
		bcrypt.MinCost,
		7*24*time.Hour,
	)

	resp, err := service.Register(context.Background(), onboardingdto.RegisterRequest{
		OrganizationName: "Acme Corp",
		AdminEmail:       "admin@acme.com",
		AdminFirstName:   "John",
		AdminLastName:    "Doe",
		AdminPassword:    "SecureP@ss123!",
		Country:          "SA",
		Industry:         "financial",
	}, "203.0.113.10", "unit-test")
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if resp.TenantID == "" {
		t.Fatal("expected tenant ID in response")
	}
	if resp.Email != "a***@acme.com" {
		t.Fatalf("expected masked email, got %q", resp.Email)
	}
	if len(onboardingRepo.createdRegistrations) != 1 {
		t.Fatalf("expected 1 registration to be created, got %d", len(onboardingRepo.createdRegistrations))
	}
	if len(emailSender.verificationEmails) != 1 {
		t.Fatalf("expected 1 verification email, got %d", len(emailSender.verificationEmails))
	}

	created := onboardingRepo.createdRegistrations[0]
	if created.AdminEmail != "admin@acme.com" {
		t.Fatalf("expected normalized admin email, got %q", created.AdminEmail)
	}
	if created.PasswordHash == "SecureP@ss123!" {
		t.Fatal("expected password to be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("SecureP@ss123!")); err != nil {
		t.Fatalf("password hash does not match original password: %v", err)
	}
	if verification.VerifyOTP(created.OTPHash, emailSender.verificationEmails[0].otp) == false {
		t.Fatal("expected stored OTP hash to validate the emailed OTP")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	onboardingRepo := newFakeOnboardingRepo()
	onboardingRepo.emailExists["admin@acme.com"] = true

	service := NewRegistrationService(
		onboardingRepo,
		newFakeUserRepo(),
		newFakeRoleRepo(),
		&fakeSessionRepo{},
		newServiceTestJWTManager(t),
		nil,
		nil,
		&fakeEmailSender{},
		nil,
		newDiscardLogger(),
		nil,
		bcrypt.MinCost,
		7*24*time.Hour,
	)

	_, err := service.Register(context.Background(), onboardingdto.RegisterRequest{
		OrganizationName: "Acme Corp",
		AdminEmail:       "admin@acme.com",
		AdminFirstName:   "John",
		AdminLastName:    "Doe",
		AdminPassword:    "SecureP@ss123!",
		Country:          "SA",
		Industry:         "financial",
	}, "", "")
	if !errors.Is(err, iammodel.ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestRegisterWeakPassword(t *testing.T) {
	service := NewRegistrationService(
		newFakeOnboardingRepo(),
		newFakeUserRepo(),
		newFakeRoleRepo(),
		&fakeSessionRepo{},
		newServiceTestJWTManager(t),
		nil,
		nil,
		&fakeEmailSender{},
		nil,
		newDiscardLogger(),
		nil,
		bcrypt.MinCost,
		7*24*time.Hour,
	)

	_, err := service.Register(context.Background(), onboardingdto.RegisterRequest{
		OrganizationName: "Acme Corp",
		AdminEmail:       "admin@acme.com",
		AdminFirstName:   "John",
		AdminLastName:    "Doe",
		AdminPassword:    "weak-password",
		Country:          "SA",
		Industry:         "financial",
	}, "", "")
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestRegisterDisposableEmail(t *testing.T) {
	service := NewRegistrationService(
		newFakeOnboardingRepo(),
		newFakeUserRepo(),
		newFakeRoleRepo(),
		&fakeSessionRepo{},
		newServiceTestJWTManager(t),
		nil,
		nil,
		&fakeEmailSender{},
		nil,
		newDiscardLogger(),
		nil,
		bcrypt.MinCost,
		7*24*time.Hour,
	)

	_, err := service.Register(context.Background(), onboardingdto.RegisterRequest{
		OrganizationName: "Acme Corp",
		AdminEmail:       "admin@mailinator.com",
		AdminFirstName:   "John",
		AdminLastName:    "Doe",
		AdminPassword:    "SecureP@ss123!",
		Country:          "SA",
		Industry:         "financial",
	}, "", "")
	if !errors.Is(err, iammodel.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
