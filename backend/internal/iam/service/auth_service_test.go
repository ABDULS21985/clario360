package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

// ---- Mock Repositories ----

type mockUserRepo struct {
	users       map[string]*model.User
	emailIndex  map[string]*model.User // key: tenantID:email
	tenantCount map[string]int
	createErr   error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users:       make(map[string]*model.User),
		emailIndex:  make(map[string]*model.User),
		tenantCount: make(map[string]int),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	user.ID = "user-" + user.Email
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	m.emailIndex[user.TenantID+":"+user.Email] = user
	m.tenantCount[user.TenantID]++
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, tenantID, email string) (*model.User, error) {
	u, ok := m.emailIndex[tenantID+":"+email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) List(ctx context.Context, tenantID string, filter repository.UserFilter) ([]model.User, int, error) {
	return nil, 0, nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, id, deletedBy string) error {
	return nil
}

func (m *mockUserRepo) UpdateStatus(ctx context.Context, id string, status model.UserStatus, updatedBy string) error {
	return nil
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, id, hash string) error {
	if u, ok := m.users[id]; ok {
		u.PasswordHash = hash
	}
	return nil
}

func (m *mockUserRepo) UpdateMFA(ctx context.Context, id string, enabled bool, secret *string) error {
	if u, ok := m.users[id]; ok {
		u.MFAEnabled = enabled
		u.MFASecret = secret
	}
	return nil
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, id string) error {
	return nil
}

func (m *mockUserRepo) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	return m.tenantCount[tenantID], nil
}

type mockSessionRepo struct {
	sessions map[string]*model.Session
	byHash   map[string]*model.Session
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{
		sessions: make(map[string]*model.Session),
		byHash:   make(map[string]*model.Session),
	}
}

func (m *mockSessionRepo) Create(ctx context.Context, session *model.Session) error {
	session.ID = "sess-" + session.UserID
	session.CreatedAt = time.Now()
	m.sessions[session.ID] = session
	m.byHash[session.RefreshTokenHash] = session
	return nil
}

func (m *mockSessionRepo) GetByTokenHash(ctx context.Context, hash string) (*model.Session, error) {
	s, ok := m.byHash[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}

func (m *mockSessionRepo) GetByUserID(ctx context.Context, userID string) ([]model.Session, error) {
	return nil, nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, id string) error {
	if s, ok := m.sessions[id]; ok {
		delete(m.byHash, s.RefreshTokenHash)
		delete(m.sessions, id)
	}
	return nil
}

func (m *mockSessionRepo) DeleteByUserID(ctx context.Context, userID string) error {
	return nil
}

func (m *mockSessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

type mockRoleRepo struct {
	roles map[string]*model.Role
	slugs map[string]*model.Role // key: tenantID:slug
}

func newMockRoleRepo() *mockRoleRepo {
	return &mockRoleRepo{
		roles: make(map[string]*model.Role),
		slugs: make(map[string]*model.Role),
	}
}

func (m *mockRoleRepo) Create(ctx context.Context, role *model.Role) error {
	role.ID = "role-" + role.Slug
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()
	m.roles[role.ID] = role
	m.slugs[role.TenantID+":"+role.Slug] = role
	return nil
}

func (m *mockRoleRepo) GetByID(ctx context.Context, id string) (*model.Role, error) {
	r, ok := m.roles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}

func (m *mockRoleRepo) GetBySlug(ctx context.Context, tenantID, slug string) (*model.Role, error) {
	r, ok := m.slugs[tenantID+":"+slug]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}

func (m *mockRoleRepo) List(ctx context.Context, tenantID string) ([]model.Role, error) {
	return nil, nil
}

func (m *mockRoleRepo) Update(ctx context.Context, role *model.Role) error {
	return nil
}

func (m *mockRoleRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockRoleRepo) AssignToUser(ctx context.Context, userID, roleID, tenantID, assignedBy string) error {
	return nil
}

func (m *mockRoleRepo) RemoveFromUser(ctx context.Context, userID, roleID string) error {
	return nil
}

func (m *mockRoleRepo) GetUserRoles(ctx context.Context, userID string) ([]model.Role, error) {
	return nil, nil
}

func (m *mockRoleRepo) ListUserIDsByRole(ctx context.Context, tenantID, roleSlug string) ([]string, error) {
	return nil, nil
}

func (m *mockRoleRepo) SeedSystemRoles(ctx context.Context, tenantID string) error {
	for _, sr := range model.SystemRoles {
		role := sr
		role.TenantID = tenantID
		_ = m.Create(ctx, &role)
	}
	return nil
}

type mockTenantRepo struct {
	tenants map[string]*model.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{tenants: make(map[string]*model.Tenant)}
}

func (m *mockTenantRepo) Create(ctx context.Context, tenant *model.Tenant) error {
	tenant.ID = "tenant-" + tenant.Slug
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepo) GetByID(ctx context.Context, id string) (*model.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t, nil
}

func (m *mockTenantRepo) GetBySlug(ctx context.Context, slug string) (*model.Tenant, error) {
	return nil, model.ErrNotFound
}

func (m *mockTenantRepo) List(ctx context.Context, page, perPage int) ([]model.Tenant, int, error) {
	return nil, 0, nil
}

func (m *mockTenantRepo) Update(ctx context.Context, tenant *model.Tenant) error {
	return nil
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}
	rdb.FlushDB(ctx)
	t.Cleanup(func() {
		rdb.FlushDB(ctx)
		rdb.Close()
	})
	return rdb
}

func newTestAuthService(t *testing.T) (*AuthService, *mockUserRepo, *mockSessionRepo, *mockRoleRepo, *mockTenantRepo) {
	t.Helper()
	userRepo := newMockUserRepo()
	sessionRepo := newMockSessionRepo()
	roleRepo := newMockRoleRepo()
	tenantRepo := newMockTenantRepo()

	rdb := newTestRedis(t)

	jwtMgr, err := auth.NewJWTManager(config.AuthConfig{
		JWTIssuer:       "test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      4, // low cost for fast tests
	})
	if err != nil {
		t.Fatalf("failed to create JWT manager: %v", err)
	}

	logger := zerolog.Nop()

	svc := NewAuthService(
		userRepo, sessionRepo, roleRepo, tenantRepo,
		jwtMgr, rdb, nil, logger, 4, 7*24*time.Hour,
	)

	// Seed a test tenant
	tenant := &model.Tenant{
		Name:   "Test Tenant",
		Slug:   "test",
		Status: model.TenantStatusActive,
	}
	_ = tenantRepo.Create(context.Background(), tenant)

	return svc, userRepo, sessionRepo, roleRepo, tenantRepo
}

func TestRegister_Success(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	resp, err := svc.Register(context.Background(), &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "user@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if resp.User.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", resp.User.Email)
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("expected token type Bearer, got %s", resp.TokenType)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "user@example.com",
		Password:  "weak",
		FirstName: "John",
		LastName:  "Doe",
	})
	if err == nil {
		t.Fatal("expected error for weak password")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	req := &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "dup@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	}

	_, err := svc.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	_, err = svc.Register(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	if !errors.Is(err, model.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestRegister_InvalidTenant(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	_, err := svc.Register(context.Background(), &dto.RegisterRequest{
		TenantID:  "nonexistent-tenant",
		Email:     "user@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	})
	if err == nil {
		t.Fatal("expected error for invalid tenant")
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register first
	_, err := svc.Register(ctx, &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "login@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Login
	resp, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "login@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	authResp, ok := resp.(*dto.AuthResponse)
	if !ok {
		t.Fatal("expected AuthResponse type")
	}
	if authResp.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "wrong@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	})

	_, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "wrong@example.com",
		Password: "WrongPassword!!1",
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if !errors.Is(err, model.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_NonexistentUser(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	_, err := svc.Login(context.Background(), &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "ghost@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if !errors.Is(err, model.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "refresh@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	})

	loginResp, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "refresh@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	authResp := loginResp.(*dto.AuthResponse)

	newResp, err := svc.RefreshToken(ctx, &dto.RefreshRequest{
		RefreshToken: authResp.RefreshToken,
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	if newResp.AccessToken == "" {
		t.Error("expected new access token")
	}
	if newResp.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	_, err := svc.RefreshToken(context.Background(), &dto.RefreshRequest{
		RefreshToken: "invalid-token",
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}

func TestLogout_Success(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "logout@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "John",
		LastName:  "Doe",
	})

	loginResp, _ := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "logout@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")

	authResp := loginResp.(*dto.AuthResponse)

	err := svc.Logout(ctx, &dto.LogoutRequest{RefreshToken: authResp.RefreshToken})
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// Refresh should fail after logout
	_, err = svc.RefreshToken(ctx, &dto.RefreshRequest{
		RefreshToken: authResp.RefreshToken,
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error refreshing after logout")
	}
}

func TestForgotPassword_KnownUser(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register a user first
	_, err := svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "forgot@example.com",
		Password: "StrongP@ss12345", FirstName: "Forgot", LastName: "User",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = svc.ForgotPassword(ctx, &dto.ForgotPasswordRequest{
		TenantID: "tenant-test",
		Email:    "forgot@example.com",
	})
	if err != nil {
		t.Fatalf("ForgotPassword failed: %v", err)
	}
}

func TestForgotPassword_UnknownEmail(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	// Should not return error (prevent enumeration)
	err := svc.ForgotPassword(context.Background(), &dto.ForgotPasswordRequest{
		TenantID: "tenant-test",
		Email:    "nobody@example.com",
	})
	if err != nil {
		t.Fatalf("ForgotPassword should not error for unknown email: %v", err)
	}
}

func TestResetPassword_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	err := svc.ResetPassword(context.Background(), &dto.ResetPasswordRequest{
		Token:       "nonexistent-token",
		NewPassword: "NewStrongP@ss99!",
	})
	if err == nil {
		t.Fatal("expected error for invalid reset token")
	}
	if !errors.Is(err, model.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestResetPassword_WeakPassword(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	err := svc.ResetPassword(context.Background(), &dto.ResetPasswordRequest{
		Token:       "some-token",
		NewPassword: "weak",
	})
	if err == nil {
		t.Fatal("expected error for weak password")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestLogin_AccountLockout(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register
	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "lockout@example.com",
		Password: "StrongP@ss12345", FirstName: "Lock", LastName: "Out",
	})

	// Attempt login with wrong password 5 times
	for i := 0; i < loginLockoutMax; i++ {
		_, _ = svc.Login(ctx, &dto.LoginRequest{
			TenantID: "tenant-test",
			Email:    "lockout@example.com",
			Password: "WrongPassword!!1",
		}, "127.0.0.1", "test-agent")
	}

	// Next attempt should be locked out
	_, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "lockout@example.com",
		Password: "StrongP@ss12345", // correct password
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error when account is locked")
	}
	if !errors.Is(err, model.ErrAccountLocked) {
		t.Errorf("expected ErrAccountLocked, got %v", err)
	}
}

func TestLogin_InactiveAccount(t *testing.T) {
	svc, userRepo, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register
	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "inactive@example.com",
		Password: "StrongP@ss12345", FirstName: "In", LastName: "Active",
	})

	// Suspend the user
	u := userRepo.emailIndex["tenant-test:inactive@example.com"]
	u.Status = model.UserStatusSuspended

	// Login should fail
	_, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "inactive@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error for inactive account")
	}
	if !errors.Is(err, model.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestLogout_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	// Logout with invalid token should not error (silent ignore)
	err := svc.Logout(context.Background(), &dto.LogoutRequest{
		RefreshToken: "nonexistent-token",
	})
	if err != nil {
		t.Fatalf("Logout with invalid token should silently succeed: %v", err)
	}
}

func TestResetPasswordForUser_Success(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register
	resp, err := svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "resetuser@example.com",
		Password: "StrongP@ss12345", FirstName: "Reset", LastName: "User",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	err = svc.ResetPasswordForUser(ctx, resp.User.ID, "NewStrongP@ss99!")
	if err != nil {
		t.Fatalf("ResetPasswordForUser failed: %v", err)
	}
}

func TestResetPasswordForUser_WeakPassword(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	resp, _ := svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "resetweak@example.com",
		Password: "StrongP@ss12345", FirstName: "Reset", LastName: "Weak",
	})

	err := svc.ResetPasswordForUser(ctx, resp.User.ID, "short")
	if err == nil {
		t.Fatal("expected error for weak password")
	}
	if !errors.Is(err, model.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid", "StrongP@ss12345", false},
		{"too short", "Short@1", true},
		{"no uppercase", "strongp@ss12345", true},
		{"no lowercase", "STRONGP@SS12345", true},
		{"no digit", "StrongP@ssword!", true},
		{"no special", "StrongPass12345", true},
		{"exactly 12 chars", "StrongP@ss12", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePassword(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestVerifyMFA_Success(t *testing.T) {
	svc, userRepo, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register a user
	_, err := svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "mfaverify@example.com",
		Password: "StrongP@ss12345", FirstName: "MFA", LastName: "Verify",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Enable MFA on the user
	user := userRepo.emailIndex["tenant-test:mfaverify@example.com"]
	secret := "JBSWY3DPEHPK3PXP" // well-known test TOTP secret
	user.MFAEnabled = true
	user.MFASecret = &secret

	// Login should return MFA challenge
	resp, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "mfaverify@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	mfaResp, ok := resp.(*dto.MFARequiredResponse)
	if !ok {
		t.Fatal("expected MFARequiredResponse type")
	}
	if !mfaResp.MFARequired {
		t.Error("expected mfa_required to be true")
	}
	if mfaResp.MFAToken == "" {
		t.Error("expected mfa_token")
	}

	// Generate valid TOTP code
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}

	authResp, err := svc.VerifyMFA(ctx, &dto.VerifyMFARequest{
		MFAToken: mfaResp.MFAToken,
		Code:     code,
	})
	if err != nil {
		t.Fatalf("VerifyMFA failed: %v", err)
	}
	if authResp.AccessToken == "" {
		t.Error("expected access token after MFA verification")
	}
}

func TestVerifyMFA_InvalidCode(t *testing.T) {
	svc, userRepo, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "mfabad@example.com",
		Password: "StrongP@ss12345", FirstName: "MFA", LastName: "Bad",
	})

	user := userRepo.emailIndex["tenant-test:mfabad@example.com"]
	secret := "JBSWY3DPEHPK3PXP"
	user.MFAEnabled = true
	user.MFASecret = &secret

	resp, _ := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "mfabad@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")

	mfaResp := resp.(*dto.MFARequiredResponse)

	_, err := svc.VerifyMFA(ctx, &dto.VerifyMFARequest{
		MFAToken: mfaResp.MFAToken,
		Code:     "000000",
	})
	if err == nil {
		t.Fatal("expected error for invalid MFA code")
	}
	if !errors.Is(err, model.ErrInvalidMFA) {
		t.Errorf("expected ErrInvalidMFA, got %v", err)
	}
}

func TestVerifyMFA_InvalidToken(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)

	_, err := svc.VerifyMFA(context.Background(), &dto.VerifyMFARequest{
		MFAToken: "nonexistent-token",
		Code:     "123456",
	})
	if err == nil {
		t.Fatal("expected error for invalid MFA token")
	}
	if !errors.Is(err, model.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestVerifyMFA_RecoveryCode(t *testing.T) {
	svc, userRepo, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "mfarecovery@example.com",
		Password: "StrongP@ss12345", FirstName: "MFA", LastName: "Recovery",
	})

	user := userRepo.emailIndex["tenant-test:mfarecovery@example.com"]
	secret := "JBSWY3DPEHPK3PXP"
	user.MFAEnabled = true
	user.MFASecret = &secret

	// Store a known recovery code in Redis (bcrypt hashed)
	recoveryCode := "testrecovery01"
	codeHash, _ := bcrypt.GenerateFromPassword([]byte(recoveryCode), bcrypt.MinCost)
	svc.redis.SAdd(ctx, recoveryPrefix+user.ID, string(codeHash))

	resp, _ := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "mfarecovery@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")

	mfaResp := resp.(*dto.MFARequiredResponse)

	authResp, err := svc.VerifyMFA(ctx, &dto.VerifyMFARequest{
		MFAToken: mfaResp.MFAToken,
		Code:     recoveryCode,
	})
	if err != nil {
		t.Fatalf("VerifyMFA with recovery code failed: %v", err)
	}
	if authResp.AccessToken == "" {
		t.Error("expected access token after recovery code verification")
	}
}

func TestResetPassword_FullFlow(t *testing.T) {
	svc, _, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	// Register a user
	regResp, err := svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "resetflow@example.com",
		Password: "StrongP@ss12345", FirstName: "Reset", LastName: "Flow",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Trigger forgot password
	err = svc.ForgotPassword(ctx, &dto.ForgotPasswordRequest{
		TenantID: "tenant-test",
		Email:    "resetflow@example.com",
	})
	if err != nil {
		t.Fatalf("ForgotPassword failed: %v", err)
	}

	// Find the reset token in Redis (scan for resetTokenPrefix keys)
	var resetToken string
	iter := svc.redis.Scan(ctx, 0, resetTokenPrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		val, _ := svc.redis.Get(ctx, key).Result()
		if val == regResp.User.ID {
			// The key is resetTokenPrefix + hash, we need the original token
			// Since we can't reverse the hash, seed a known token instead
			resetToken = key
			break
		}
	}

	if resetToken == "" {
		t.Fatal("expected to find a reset token in Redis")
	}

	// We can't easily get the original token (it's hashed), so test ResetPassword
	// by directly seeding a known token
	knownToken := "known-test-reset-token-hex-value"
	h := sha256.Sum256([]byte(knownToken))
	tokenHash := hex.EncodeToString(h[:])
	svc.redis.Set(ctx, resetTokenPrefix+tokenHash, regResp.User.ID, resetTokenTTL)

	err = svc.ResetPassword(ctx, &dto.ResetPasswordRequest{
		Token:       knownToken,
		NewPassword: "NewStrongP@ss99!",
	})
	if err != nil {
		t.Fatalf("ResetPassword failed: %v", err)
	}

	// Verify new password works by logging in
	resp, err := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "resetflow@example.com",
		Password: "NewStrongP@ss99!",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Login with new password failed: %v", err)
	}
	authResp, ok := resp.(*dto.AuthResponse)
	if !ok {
		t.Fatal("expected AuthResponse type")
	}
	if authResp.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestRefreshToken_SuspendedUser(t *testing.T) {
	svc, userRepo, _, _, _ := newTestAuthService(t)
	ctx := context.Background()

	_, _ = svc.Register(ctx, &dto.RegisterRequest{
		TenantID: "tenant-test", Email: "suspended@example.com",
		Password: "StrongP@ss12345", FirstName: "Susp", LastName: "User",
	})

	loginResp, _ := svc.Login(ctx, &dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "suspended@example.com",
		Password: "StrongP@ss12345",
	}, "127.0.0.1", "test-agent")

	authResp := loginResp.(*dto.AuthResponse)

	// Suspend the user after login
	user := userRepo.emailIndex["tenant-test:suspended@example.com"]
	user.Status = model.UserStatusSuspended

	// Refresh should fail for suspended user
	_, err := svc.RefreshToken(ctx, &dto.RefreshRequest{
		RefreshToken: authResp.RefreshToken,
	}, "127.0.0.1", "test-agent")
	if err == nil {
		t.Fatal("expected error refreshing for suspended user")
	}
	if !errors.Is(err, model.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}
