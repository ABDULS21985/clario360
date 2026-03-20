package service

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

func TestAuthorize_ValidClient(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()

	pair, err := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}

	result, err := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	if result.RedirectURL == "" {
		t.Fatalf("expected redirect URL")
	}
	redirect, err := url.Parse(result.RedirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	if redirect.Query().Get("code") == "" {
		t.Fatalf("expected authorization code in redirect URL")
	}
	if redirect.Query().Get("state") != "state-123" {
		t.Fatalf("expected state to be preserved")
	}
}

func TestAuthorize_UnregisteredClient(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})

	req := validAuthorizeRequest()
	req.ClientID = "unknown"
	_, err := svc.Authorize(context.Background(), req, pair.AccessToken)
	assertOAuthErrorCode(t, err, "INVALID_CLIENT")
}

func TestAuthorize_InvalidRedirectURI(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})

	req := validAuthorizeRequest()
	req.RedirectURI = "https://evil.example.com/callback"
	_, err := svc.Authorize(context.Background(), req, pair.AccessToken)
	assertOAuthErrorCode(t, err, "INVALID_REDIRECT_URI")
}

func TestAuthorize_MissingPKCE(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})

	req := validAuthorizeRequest()
	req.CodeChallenge = ""
	_, err := svc.Authorize(context.Background(), req, pair.AccessToken)
	assertOAuthErrorCode(t, err, "INVALID_REQUEST")
}

func TestTokenExchange_ValidCode(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})

	result, err := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	if err != nil {
		t.Fatalf("Authorize failed: %v", err)
	}
	code := mustRedirectCode(t, result.RedirectURL)

	resp, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		CodeVerifier: "verifier-123",
	}, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("ExchangeToken failed: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.IDToken == "" {
		t.Fatalf("expected access, refresh, and id tokens")
	}
	if resp.TokenType != "Bearer" {
		t.Fatalf("expected Bearer token type, got %s", resp.TokenType)
	}
}

func TestTokenExchange_ExpiredCode(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	result, _ := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	code := mustRedirectCode(t, result.RedirectURL)

	redisSrv.FastForward(61 * time.Second)
	_, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		CodeVerifier: "verifier-123",
	}, "", "")
	assertOAuthErrorCode(t, err, "INVALID_GRANT")
}

func TestTokenExchange_UsedCode(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	result, _ := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	code := mustRedirectCode(t, result.RedirectURL)

	if _, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		CodeVerifier: "verifier-123",
	}, "", ""); err != nil {
		t.Fatalf("first exchange failed: %v", err)
	}
	_, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		CodeVerifier: "verifier-123",
	}, "", "")
	assertOAuthErrorCode(t, err, "INVALID_GRANT")
}

func TestTokenExchange_InvalidPKCE(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	result, _ := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	code := mustRedirectCode(t, result.RedirectURL)

	_, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		CodeVerifier: "wrong-verifier",
	}, "", "")
	assertOAuthErrorCode(t, err, "INVALID_GRANT")
}

func TestTokenExchange_InvalidRedirectURI(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	result, _ := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	code := mustRedirectCode(t, result.RedirectURL)

	_, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.example.com/other",
		ClientID:     "jupyterhub",
		CodeVerifier: "verifier-123",
	}, "", "")
	assertOAuthErrorCode(t, err, "INVALID_GRANT")
}

func TestTokenExchange_ValidClientSecret(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	svc.clients["jupyterhub"] = OAuthClient{
		ClientID:     "jupyterhub",
		ClientSecret: "super-secret",
		RedirectURIs: []string{"https://notebooks.clario360.sa/hub/oauth_callback"},
		Scopes:       []string{"openid", "profile", "email", "roles"},
		RequirePKCE:  true,
	}
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	result, _ := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	code := mustRedirectCode(t, result.RedirectURL)

	resp, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		ClientSecret: "super-secret",
		CodeVerifier: "verifier-123",
	}, "", "")
	if err != nil {
		t.Fatalf("ExchangeToken failed: %v", err)
	}
	if resp.AccessToken == "" || resp.IDToken == "" {
		t.Fatalf("expected access and id tokens")
	}
}

func TestTokenExchange_InvalidClientSecret(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	svc.clients["jupyterhub"] = OAuthClient{
		ClientID:     "jupyterhub",
		ClientSecret: "super-secret",
		RedirectURIs: []string{"https://notebooks.clario360.sa/hub/oauth_callback"},
		Scopes:       []string{"openid", "profile", "email", "roles"},
		RequirePKCE:  true,
	}
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})
	result, _ := svc.Authorize(context.Background(), validAuthorizeRequest(), pair.AccessToken)
	code := mustRedirectCode(t, result.RedirectURL)

	_, err := svc.ExchangeToken(context.Background(), OAuthTokenRequest{
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  "https://notebooks.clario360.sa/hub/oauth_callback",
		ClientID:     "jupyterhub",
		ClientSecret: "wrong-secret",
		CodeVerifier: "verifier-123",
	}, "", "")
	assertOAuthErrorCode(t, err, "INVALID_CLIENT")
}

func TestUserInfo_ValidToken(t *testing.T) {
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := svc.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})

	info, err := svc.UserInfo(context.Background(), pair.AccessToken)
	if err != nil {
		t.Fatalf("UserInfo failed: %v", err)
	}
	if info.Email != user.Email {
		t.Fatalf("expected email %s, got %s", user.Email, info.Email)
	}
	if info.TenantID != user.TenantID {
		t.Fatalf("expected tenant %s, got %s", user.TenantID, info.TenantID)
	}
}

func TestUserInfo_ExpiredToken(t *testing.T) {
	expiredMgr, err := auth.NewJWTManager(config.AuthConfig{
		JWTIssuer:       "clario360-test",
		AccessTokenTTL:  -1 * time.Minute,
		RefreshTokenTTL: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}
	svc, _, user, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()
	pair, _ := expiredMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, []string{"security-analyst"})

	_, err = svc.UserInfo(context.Background(), pair.AccessToken)
	assertOAuthErrorCode(t, err, "INVALID_TOKEN")
}

func TestDiscovery(t *testing.T) {
	svc, _, _, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()

	doc := svc.DiscoveryDocument()
	if doc["authorization_endpoint"] == "" {
		t.Fatalf("expected authorization endpoint")
	}
	if doc["userinfo_endpoint"] == "" {
		t.Fatalf("expected userinfo endpoint")
	}
	methods, ok := doc["token_endpoint_auth_methods_supported"].([]string)
	if !ok || len(methods) == 0 {
		t.Fatalf("expected token endpoint auth methods")
	}
}

func validAuthorizeRequest() OAuthAuthorizeRequest {
	return OAuthAuthorizeRequest{
		ResponseType:        "code",
		ClientID:            "jupyterhub",
		RedirectURI:         "https://notebooks.clario360.sa/hub/oauth_callback",
		Scope:               "openid profile email roles",
		State:               "state-123",
		CodeChallenge:       "Ds3NpaREu9I2EYq6l0l3ZkFyv_Gt5O4EpGD6cZlY0Kg",
		CodeChallengeMethod: "S256",
	}
}

func mustRedirectCode(t *testing.T, redirectURL string) string {
	t.Helper()
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL: %v", err)
	}
	code := parsed.Query().Get("code")
	if code == "" {
		t.Fatalf("redirect URL missing code")
	}
	return code
}

func assertOAuthErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %s, got nil", code)
	}
	oauthErr, ok := err.(*OAuthError)
	if !ok {
		t.Fatalf("expected OAuthError, got %T", err)
	}
	if oauthErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, oauthErr.Code)
	}
}

func newOAuthServiceForTest(t *testing.T) (*OAuthService, *redis.Client, *model.User, *miniredis.Miniredis) {
	t.Helper()

	redisSrv := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: redisSrv.Addr()})
	jwtMgr, err := auth.NewJWTManager(config.AuthConfig{
		JWTIssuer:       "clario360-test",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewJWTManager failed: %v", err)
	}

	user := &model.User{
		ID:        "user-1",
		TenantID:  "tenant-1",
		Email:     "analyst@example.com",
		FirstName: "Analyst",
		LastName:  "User",
		Status:    model.UserStatusActive,
		Roles: []model.Role{
			{ID: "role-1", Slug: "security-analyst", Permissions: []string{"cyber:read"}},
		},
	}
	tenant := &model.Tenant{
		ID:   "tenant-1",
		Name: "Acme",
		Slug: "acme",
	}
	userRepo := &fakeUserRepo{users: map[string]*model.User{user.ID: user}}
	tenantRepo := &fakeTenantRepo{tenants: map[string]*model.Tenant{tenant.ID: tenant}}
	sessionRepo := &fakeSessionRepo{sessions: map[string]*model.Session{}}
	roleRepo := &fakeRoleRepo{}
	authSvc := NewAuthService(userRepo, sessionRepo, roleRepo, tenantRepo, jwtMgr, rdb, nil, zerolog.New(ioDiscard{}), 4, 24*time.Hour)
	svc := NewOAuthService(
		jwtMgr,
		authSvc,
		userRepo,
		tenantRepo,
		rdb,
		"https://api.clario360.sa",
		"https://app.clario360.sa/login",
		[]OAuthClient{
			{
				ClientID:     "jupyterhub",
				RedirectURIs: []string{"https://notebooks.clario360.sa/hub/oauth_callback"},
				Scopes:       []string{"openid", "profile", "email", "roles"},
				RequirePKCE:  true,
			},
		},
		zerolog.New(ioDiscard{}),
	)
	return svc, rdb, user, redisSrv
}

type fakeUserRepo struct {
	users map[string]*model.User
}

func (f *fakeUserRepo) Create(context.Context, *model.User) error { return nil }
func (f *fakeUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	user, ok := f.users[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	copy := *user
	return &copy, nil
}
func (f *fakeUserRepo) GetByEmail(_ context.Context, tenantID, email string) (*model.User, error) {
	for _, user := range f.users {
		if user.TenantID == tenantID && user.Email == email {
			copy := *user
			return &copy, nil
		}
	}
	return nil, model.ErrNotFound
}
func (f *fakeUserRepo) GetByEmailGlobal(_ context.Context, email string) (*model.User, error) {
	for _, user := range f.users {
		if user.Email == email {
			copy := *user
			return &copy, nil
		}
	}
	return nil, model.ErrNotFound
}
func (f *fakeUserRepo) List(context.Context, string, repository.UserFilter) ([]model.User, int, error) {
	return nil, 0, nil
}
func (f *fakeUserRepo) Update(context.Context, *model.User) error        { return nil }
func (f *fakeUserRepo) SoftDelete(context.Context, string, string) error { return nil }
func (f *fakeUserRepo) UpdateStatus(context.Context, string, model.UserStatus, string) error {
	return nil
}
func (f *fakeUserRepo) UpdatePassword(context.Context, string, string) error   { return nil }
func (f *fakeUserRepo) UpdateMFA(context.Context, string, bool, *string) error { return nil }
func (f *fakeUserRepo) UpdateLastLogin(context.Context, string) error          { return nil }
func (f *fakeUserRepo) CountByTenant(context.Context, string) (int, error)     { return len(f.users), nil }

type fakeTenantRepo struct {
	tenants map[string]*model.Tenant
}

func (f *fakeTenantRepo) Create(context.Context, *model.Tenant) error { return nil }
func (f *fakeTenantRepo) GetByID(_ context.Context, id string) (*model.Tenant, error) {
	tenant, ok := f.tenants[id]
	if !ok {
		return nil, model.ErrNotFound
	}
	copy := *tenant
	return &copy, nil
}
func (f *fakeTenantRepo) GetBySlug(_ context.Context, slug string) (*model.Tenant, error) {
	for _, tenant := range f.tenants {
		if tenant.Slug == slug {
			copy := *tenant
			return &copy, nil
		}
	}
	return nil, model.ErrNotFound
}
func (f *fakeTenantRepo) List(context.Context, int, int, repository.TenantListParams) ([]model.Tenant, int, error) {
	return nil, 0, nil
}
func (f *fakeTenantRepo) Update(context.Context, *model.Tenant) error { return nil }

type fakeSessionRepo struct {
	sessions map[string]*model.Session
}

func (f *fakeSessionRepo) Create(_ context.Context, session *model.Session) error {
	copy := *session
	copy.ID = "session-" + session.RefreshTokenHash
	f.sessions[session.RefreshTokenHash] = &copy
	session.ID = copy.ID
	return nil
}
func (f *fakeSessionRepo) GetByTokenHash(_ context.Context, tokenHash string) (*model.Session, error) {
	session, ok := f.sessions[tokenHash]
	if !ok {
		return nil, model.ErrNotFound
	}
	copy := *session
	return &copy, nil
}
func (f *fakeSessionRepo) GetByUserID(_ context.Context, userID string) ([]model.Session, error) {
	var out []model.Session
	for _, session := range f.sessions {
		if session.UserID == userID {
			out = append(out, *session)
		}
	}
	return out, nil
}
func (f *fakeSessionRepo) UpdateLastActive(_ context.Context, _ string) error { return nil }
func (f *fakeSessionRepo) Delete(_ context.Context, id string) error {
	for key, session := range f.sessions {
		if session.ID == id {
			delete(f.sessions, key)
			return nil
		}
	}
	return nil
}
func (f *fakeSessionRepo) DeleteByUserID(_ context.Context, userID string) error {
	for key, session := range f.sessions {
		if session.UserID == userID {
			delete(f.sessions, key)
		}
	}
	return nil
}
func (f *fakeSessionRepo) DeleteExpired(context.Context) (int64, error) { return 0, nil }

type fakeRoleRepo struct{}

func (f *fakeRoleRepo) Create(context.Context, *model.Role) error { return nil }
func (f *fakeRoleRepo) GetByID(context.Context, string) (*model.Role, error) {
	return nil, model.ErrNotFound
}
func (f *fakeRoleRepo) GetBySlug(context.Context, string, string) (*model.Role, error) {
	return nil, model.ErrNotFound
}
func (f *fakeRoleRepo) List(context.Context, string) ([]model.Role, error) { return nil, nil }
func (f *fakeRoleRepo) Update(context.Context, *model.Role) error          { return nil }
func (f *fakeRoleRepo) Delete(context.Context, string) error               { return nil }
func (f *fakeRoleRepo) AssignToUser(context.Context, string, string, string, string) error {
	return nil
}
func (f *fakeRoleRepo) RemoveFromUser(context.Context, string, string) error       { return nil }
func (f *fakeRoleRepo) GetUserRoles(context.Context, string) ([]model.Role, error) { return nil, nil }
func (f *fakeRoleRepo) ListUserIDsByRole(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (f *fakeRoleRepo) SeedSystemRoles(context.Context, string) error { return nil }

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

func TestJWKS(t *testing.T) {
	svc, _, _, redisSrv := newOAuthServiceForTest(t)
	defer redisSrv.Close()

	jwks, err := svc.JWKS()
	if err != nil {
		t.Fatalf("JWKS failed: %v", err)
	}
	payload, err := json.Marshal(jwks)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	if !strings.Contains(string(payload), "\"keys\"") {
		t.Fatalf("expected jwks keys payload")
	}
}
