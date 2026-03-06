package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

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

	jwtMgr := auth.NewJWTManager(config.AuthConfig{
		JWTSecret:       "test-secret-key-for-testing-only-32bytes!",
		JWTIssuer:       "test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      4, // low cost for fast tests
	})

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
