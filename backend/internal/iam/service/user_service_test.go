package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
)

func newTestUserService(t *testing.T) (*UserService, *mockUserRepo, *mockRoleRepo) {
	t.Helper()
	userRepo := newMockUserRepo()
	roleRepo := newMockRoleRepo()
	sessionRepo := newMockSessionRepo()
	rdb := newTestRedis(t)
	logger := zerolog.Nop()

	svc := NewUserService(userRepo, roleRepo, sessionRepo, rdb, nil, logger, 4)
	return svc, userRepo, roleRepo
}

func seedTestUser(repo *mockUserRepo, tenantID, email, password string) *model.User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 4)
	user := &model.User{
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    "Test",
		LastName:     "User",
		Status:       model.UserStatusActive,
	}
	_ = repo.Create(context.Background(), user)
	return user
}

func TestUserService_GetByID(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "get@example.com", "StrongP@ss12345")

	resp, err := svc.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if resp.Email != "get@example.com" {
		t.Errorf("expected email get@example.com, got %s", resp.Email)
	}
	if resp.FullName != "Test User" {
		t.Errorf("expected full name 'Test User', got %s", resp.FullName)
	}
}

func TestUserService_GetByID_NotFound(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_Update(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "update@example.com", "StrongP@ss12345")

	newFirst := "Updated"
	resp, err := svc.Update(context.Background(), user.ID, &dto.UpdateUserRequest{
		FirstName: &newFirst,
	}, "admin-id")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if resp.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got %s", resp.FirstName)
	}
}

func TestUserService_Delete(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "delete@example.com", "StrongP@ss12345")

	err := svc.Delete(context.Background(), user.ID, "admin-id")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "pwd@example.com", "StrongP@ss12345")

	err := svc.ChangePassword(context.Background(), user.ID, &dto.ChangePasswordRequest{
		CurrentPassword: "StrongP@ss12345",
		NewPassword:     "NewStrongP@ss99!",
	})
	if err != nil {
		t.Fatalf("ChangePassword failed: %v", err)
	}

	// Verify new password works
	updated := repo.users[user.ID]
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("NewStrongP@ss99!")); err != nil {
		t.Error("new password hash doesn't match")
	}
}

func TestUserService_ChangePassword_WrongCurrent(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "pwd2@example.com", "StrongP@ss12345")

	err := svc.ChangePassword(context.Background(), user.ID, &dto.ChangePasswordRequest{
		CurrentPassword: "WrongOldP@ss!!1",
		NewPassword:     "NewStrongP@ss99!",
	})
	if err == nil {
		t.Fatal("expected error for wrong current password")
	}
	if !errors.Is(err, model.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestUserService_ChangePassword_WeakNew(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "pwd3@example.com", "StrongP@ss12345")

	err := svc.ChangePassword(context.Background(), user.ID, &dto.ChangePasswordRequest{
		CurrentPassword: "StrongP@ss12345",
		NewPassword:     "weak",
	})
	if err == nil {
		t.Fatal("expected error for weak new password")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestUserService_EnableMFA(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "mfa@example.com", "StrongP@ss12345")

	resp, err := svc.EnableMFA(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("EnableMFA failed: %v", err)
	}
	if resp.Secret == "" {
		t.Error("expected TOTP secret")
	}
	if resp.OTPURL == "" {
		t.Error("expected OTP URL")
	}
	if len(resp.RecoveryCodes) != recoveryCodeCount {
		t.Errorf("expected %d recovery codes, got %d", recoveryCodeCount, len(resp.RecoveryCodes))
	}

	// MFA secret stored but not yet enabled (two-step flow)
	updated := repo.users[user.ID]
	if updated.MFASecret == nil {
		t.Error("expected MFA secret to be set")
	}

	// Complete setup by verifying a valid TOTP code
	code, err := totp.GenerateCode(resp.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}
	if err := svc.VerifyMFASetup(context.Background(), user.ID, code); err != nil {
		t.Fatalf("VerifyMFASetup failed: %v", err)
	}

	// Now MFA should be enabled
	updated = repo.users[user.ID]
	if !updated.MFAEnabled {
		t.Error("expected user to have MFA enabled after VerifyMFASetup")
	}
}

func TestUserService_EnableMFA_AlreadyEnabled(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "mfa2@example.com", "StrongP@ss12345")

	// Enable and complete two-step setup
	resp, err := svc.EnableMFA(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("first EnableMFA failed: %v", err)
	}
	code, err := totp.GenerateCode(resp.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}
	if err := svc.VerifyMFASetup(context.Background(), user.ID, code); err != nil {
		t.Fatalf("VerifyMFASetup failed: %v", err)
	}

	// MFA is now enabled — second EnableMFA should conflict
	_, err = svc.EnableMFA(context.Background(), user.ID)
	if err == nil {
		t.Fatal("expected error when MFA already enabled")
	}
	if !errors.Is(err, model.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
	_ = repo // suppress unused warning
}

func TestUserService_UpdateStatus(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "status@example.com", "StrongP@ss12345")

	err := svc.UpdateStatus(context.Background(), user.ID, &dto.UpdateStatusRequest{
		Status: "suspended",
	}, "admin-id")
	if err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}
}

func TestUserService_List(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	// Empty tenant should return empty list
	users, total, err := svc.List(context.Background(), "tenant-1", 1, 20, "", "")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestUserService_List_WithFilters(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	// List with search and status filters (empty results expected, but exercises the code path)
	users, total, err := svc.List(context.Background(), "tenant-1", 1, 10, "test", "active")
	if err != nil {
		t.Fatalf("List with filters failed: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if len(users) != 0 {
		t.Errorf("expected 0 users, got %d", len(users))
	}
}

func TestUserService_DisableMFA(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "disablemfa@example.com", "StrongP@ss12345")

	// Enable MFA and complete two-step setup
	mfaResp, err := svc.EnableMFA(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("EnableMFA failed: %v", err)
	}
	setupCode, err := totp.GenerateCode(mfaResp.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate setup TOTP code: %v", err)
	}
	if err := svc.VerifyMFASetup(context.Background(), user.ID, setupCode); err != nil {
		t.Fatalf("VerifyMFASetup failed: %v", err)
	}

	// Generate a valid TOTP code to disable MFA
	code, err := totp.GenerateCode(mfaResp.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}

	err = svc.DisableMFA(context.Background(), user.ID, &dto.DisableMFARequest{
		Code: code,
	})
	if err != nil {
		t.Fatalf("DisableMFA failed: %v", err)
	}

	// Verify MFA is now disabled
	updated := repo.users[user.ID]
	if updated.MFAEnabled {
		t.Error("expected MFA to be disabled")
	}
}

func TestUserService_DisableMFA_InvalidCode(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "disablemfabad@example.com", "StrongP@ss12345")

	_, err := svc.EnableMFA(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("EnableMFA failed: %v", err)
	}

	err = svc.DisableMFA(context.Background(), user.ID, &dto.DisableMFARequest{
		Code: "000000",
	})
	if err == nil {
		t.Fatal("expected error for invalid TOTP code")
	}
}

func TestUserService_DisableMFA_NotEnabled(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "nomfa@example.com", "StrongP@ss12345")

	err := svc.DisableMFA(context.Background(), user.ID, &dto.DisableMFARequest{
		Code: "000000",
	})
	if err == nil {
		t.Fatal("expected error when MFA is not enabled")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestUserService_GetByID_WithUpdate(t *testing.T) {
	svc, repo, _ := newTestUserService(t)
	user := seedTestUser(repo, "tenant-1", "combo@example.com", "StrongP@ss12345")

	// Update then get
	newFirst := "Updated"
	newLast := "Name"
	_, err := svc.Update(context.Background(), user.ID, &dto.UpdateUserRequest{
		FirstName: &newFirst,
		LastName:  &newLast,
	}, "admin-id")
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	resp, err := svc.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}
	if resp.FirstName != "Updated" {
		t.Errorf("expected first name 'Updated', got %s", resp.FirstName)
	}
	if resp.LastName != "Name" {
		t.Errorf("expected last name 'Name', got %s", resp.LastName)
	}
}

func TestUserService_Delete_NotFound(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	err := svc.Delete(context.Background(), "nonexistent", "admin-id")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserService_UpdateStatus_NotFound(t *testing.T) {
	svc, _, _ := newTestUserService(t)

	err := svc.UpdateStatus(context.Background(), "nonexistent", &dto.UpdateStatusRequest{
		Status: "suspended",
	}, "admin-id")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

