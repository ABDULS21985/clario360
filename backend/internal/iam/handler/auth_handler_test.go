package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
	"github.com/clario360/platform/internal/iam/service"
)

func newTestRouter(t *testing.T) (http.Handler, *service.AuthService) {
	t.Helper()

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379", DB: 15})
	bgCtx := context.Background()
	if err := rdb.Ping(bgCtx).Err(); err != nil {
		t.Skip("redis not available, skipping test")
	}
	rdb.FlushDB(bgCtx)
	t.Cleanup(func() {
		rdb.FlushDB(bgCtx)
		rdb.Close()
	})

	jwtMgr := auth.NewJWTManager(config.AuthConfig{
		JWTSecret:       "handler-test-secret-key-32bytes!",
		JWTIssuer:       "test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      4,
	})

	logger := zerolog.Nop()

	// Create in-memory repos
	userRepo := newMemUserRepo()
	sessionRepo := newMemSessionRepo()
	roleRepo := newMemRoleRepo()
	tenantRepo := newMemTenantRepo()

	// Seed tenant
	_ = tenantRepo.Create(bgCtx, &model.Tenant{
		Name:   "Test",
		Slug:   "test",
		Status: model.TenantStatusActive,
	})

	authSvc := service.NewAuthService(
		userRepo, sessionRepo, roleRepo, tenantRepo,
		jwtMgr, rdb, nil, logger, 4, 7*24*time.Hour,
	)

	h := NewAuthHandler(authSvc, logger)
	return h.Routes(), authSvc
}

func TestAuthHandler_Register(t *testing.T) {
	router, _ := newTestRouter(t)

	body, _ := json.Marshal(dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "handler@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "Handler",
		LastName:  "Test",
	})

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token in response")
	}
	if resp.User.Email != "handler@example.com" {
		t.Errorf("expected email handler@example.com, got %s", resp.User.Email)
	}
}

func TestAuthHandler_Register_BadRequest(t *testing.T) {
	router, _ := newTestRouter(t)

	// Missing required fields
	body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login(t *testing.T) {
	router, _ := newTestRouter(t)

	// Register first
	regBody, _ := json.Marshal(dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "login@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "Login",
		LastName:  "Test",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	router.ServeHTTP(regW, regReq)

	if regW.Code != http.StatusCreated {
		t.Fatalf("register failed: %d %s", regW.Code, regW.Body.String())
	}

	// Login
	loginBody, _ := json.Marshal(dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "login@example.com",
		Password: "StrongP@ss12345",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", loginW.Code, loginW.Body.String())
	}
}

func TestAuthHandler_Login_WrongPassword(t *testing.T) {
	router, _ := newTestRouter(t)

	// Register
	regBody, _ := json.Marshal(dto.RegisterRequest{
		TenantID:  "tenant-test",
		Email:     "wrong@example.com",
		Password:  "StrongP@ss12345",
		FirstName: "Wrong",
		LastName:  "Pass",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	router.ServeHTTP(regW, regReq)

	// Login with wrong password
	loginBody, _ := json.Marshal(dto.LoginRequest{
		TenantID: "tenant-test",
		Email:    "wrong@example.com",
		Password: "WrongPassword!!1",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReq)

	if loginW.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", loginW.Code)
	}
}

func TestAuthHandler_ForgotPassword(t *testing.T) {
	router, _ := newTestRouter(t)

	body, _ := json.Marshal(dto.ForgotPasswordRequest{Email: "anyone@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Always returns 200 to prevent enumeration
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	router, _ := newTestRouter(t)

	// Register + Login
	regBody, _ := json.Marshal(dto.RegisterRequest{
		TenantID: "tenant-test", Email: "logout@example.com",
		Password: "StrongP@ss12345", FirstName: "Logout", LastName: "Test",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	router.ServeHTTP(regW, regReq)

	var authResp dto.AuthResponse
	_ = json.NewDecoder(regW.Body).Decode(&authResp)

	// Logout
	logoutBody, _ := json.Marshal(dto.LogoutRequest{RefreshToken: authResp.RefreshToken})
	logoutReq := httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(logoutBody))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutW := httptest.NewRecorder()
	router.ServeHTTP(logoutW, logoutReq)

	if logoutW.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", logoutW.Code)
	}
}

// ---- In-memory repos for handler tests ----

type memUserRepo struct {
	users       map[string]*model.User
	emailIndex  map[string]*model.User
	tenantCount map[string]int
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{
		users:       make(map[string]*model.User),
		emailIndex:  make(map[string]*model.User),
		tenantCount: make(map[string]int),
	}
}

func (m *memUserRepo) Create(_ context.Context, user *model.User) error {
	user.ID = "user-" + user.Email
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	m.users[user.ID] = user
	m.emailIndex[user.TenantID+":"+user.Email] = user
	m.tenantCount[user.TenantID]++
	return nil
}

func (m *memUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *memUserRepo) GetByEmail(_ context.Context, tenantID, email string) (*model.User, error) {
	u, ok := m.emailIndex[tenantID+":"+email]
	if !ok {
		return nil, model.ErrNotFound
	}
	return u, nil
}

func (m *memUserRepo) List(_ context.Context, _ string, _ repository.UserFilter) ([]model.User, int, error) {
	return nil, 0, nil
}
func (m *memUserRepo) Update(_ context.Context, user *model.User) error {
	m.users[user.ID] = user
	return nil
}
func (m *memUserRepo) SoftDelete(_ context.Context, id, _ string) error { return nil }
func (m *memUserRepo) UpdateStatus(_ context.Context, id string, status model.UserStatus, _ string) error {
	return nil
}
func (m *memUserRepo) UpdatePassword(_ context.Context, id, hash string) error {
	if u, ok := m.users[id]; ok {
		u.PasswordHash = hash
	}
	return nil
}
func (m *memUserRepo) UpdateMFA(_ context.Context, id string, enabled bool, secret *string) error {
	if u, ok := m.users[id]; ok {
		u.MFAEnabled = enabled
		u.MFASecret = secret
	}
	return nil
}
func (m *memUserRepo) UpdateLastLogin(_ context.Context, _ string) error { return nil }
func (m *memUserRepo) CountByTenant(_ context.Context, tenantID string) (int, error) {
	return m.tenantCount[tenantID], nil
}

type memSessionRepo struct {
	sessions map[string]*model.Session
	byHash   map[string]*model.Session
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{
		sessions: make(map[string]*model.Session),
		byHash:   make(map[string]*model.Session),
	}
}

func (m *memSessionRepo) Create(_ context.Context, s *model.Session) error {
	s.ID = "sess-" + s.UserID
	s.CreatedAt = time.Now()
	m.sessions[s.ID] = s
	m.byHash[s.RefreshTokenHash] = s
	return nil
}
func (m *memSessionRepo) GetByTokenHash(_ context.Context, hash string) (*model.Session, error) {
	s, ok := m.byHash[hash]
	if !ok {
		return nil, model.ErrNotFound
	}
	return s, nil
}
func (m *memSessionRepo) GetByUserID(_ context.Context, _ string) ([]model.Session, error) {
	return nil, nil
}
func (m *memSessionRepo) Delete(_ context.Context, id string) error {
	if s, ok := m.sessions[id]; ok {
		delete(m.byHash, s.RefreshTokenHash)
		delete(m.sessions, id)
	}
	return nil
}
func (m *memSessionRepo) DeleteByUserID(_ context.Context, _ string) error { return nil }
func (m *memSessionRepo) DeleteExpired(_ context.Context) (int64, error)   { return 0, nil }

type memRoleRepo struct {
	roles map[string]*model.Role
	slugs map[string]*model.Role
}

func newMemRoleRepo() *memRoleRepo {
	return &memRoleRepo{roles: make(map[string]*model.Role), slugs: make(map[string]*model.Role)}
}

func (m *memRoleRepo) Create(_ context.Context, r *model.Role) error {
	r.ID = "role-" + r.Slug
	r.CreatedAt = time.Now()
	r.UpdatedAt = time.Now()
	m.roles[r.ID] = r
	m.slugs[r.TenantID+":"+r.Slug] = r
	return nil
}
func (m *memRoleRepo) GetByID(_ context.Context, id string) (*model.Role, error) {
	r, ok := m.roles[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}
func (m *memRoleRepo) GetBySlug(_ context.Context, tenantID, slug string) (*model.Role, error) {
	r, ok := m.slugs[tenantID+":"+slug]
	if !ok {
		return nil, model.ErrNotFound
	}
	return r, nil
}
func (m *memRoleRepo) List(_ context.Context, _ string) ([]model.Role, error) { return nil, nil }
func (m *memRoleRepo) Update(_ context.Context, _ *model.Role) error          { return nil }
func (m *memRoleRepo) Delete(_ context.Context, _ string) error                { return nil }
func (m *memRoleRepo) AssignToUser(_ context.Context, _, _, _, _ string) error { return nil }
func (m *memRoleRepo) RemoveFromUser(_ context.Context, _, _ string) error     { return nil }
func (m *memRoleRepo) GetUserRoles(_ context.Context, _ string) ([]model.Role, error) {
	return nil, nil
}
func (m *memRoleRepo) SeedSystemRoles(ctx context.Context, tenantID string) error {
	for _, sr := range model.SystemRoles {
		role := sr
		role.TenantID = tenantID
		_ = m.Create(ctx, &role)
	}
	return nil
}

type memTenantRepo struct {
	tenants map[string]*model.Tenant
}

func newMemTenantRepo() *memTenantRepo {
	return &memTenantRepo{tenants: make(map[string]*model.Tenant)}
}

func (m *memTenantRepo) Create(_ context.Context, t *model.Tenant) error {
	t.ID = "tenant-" + t.Slug
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	m.tenants[t.ID] = t
	return nil
}
func (m *memTenantRepo) GetByID(_ context.Context, id string) (*model.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	return t, nil
}
func (m *memTenantRepo) GetBySlug(_ context.Context, _ string) (*model.Tenant, error) {
	return nil, model.ErrNotFound
}
func (m *memTenantRepo) List(_ context.Context, _, _ int) ([]model.Tenant, int, error) {
	return nil, 0, nil
}
func (m *memTenantRepo) Update(_ context.Context, _ *model.Tenant) error { return nil }
