package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

const (
	defaultOAuthCodeTTL = 60 * time.Second
	oauthCodePrefix     = "oauth:code:"
)

// OAuthClient represents a registered OIDC client.
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	RedirectURIs []string
	Scopes       []string
	RequirePKCE  bool
}

type OAuthAuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type OAuthTokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	CodeVerifier string
	RefreshToken string
}

type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

type OAuthUserInfo struct {
	Subject           string   `json:"sub"`
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Role              string   `json:"role"`
	Roles             []string `json:"roles"`
	TenantID          string   `json:"tenant_id"`
	TenantName        string   `json:"tenant_name"`
	Permissions       []string `json:"permissions"`
	UserID            string   `json:"user_id"`
}

type OAuthAuthorizeResult struct {
	RedirectURL      string
	LoginRedirectURL string
}

type OAuthError struct {
	Status  int
	Code    string
	Message string
}

func (e *OAuthError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type storedAuthorizationCode struct {
	ClientID      string   `json:"client_id"`
	RedirectURI   string   `json:"redirect_uri"`
	Scope         []string `json:"scope"`
	CodeChallenge string   `json:"code_challenge"`
	UserID        string   `json:"user_id"`
	TenantID      string   `json:"tenant_id"`
}

type oidcIDClaims struct {
	jwt.RegisteredClaims
	Email             string   `json:"email"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
	Role              string   `json:"role"`
	Roles             []string `json:"roles"`
	TenantID          string   `json:"tenant_id"`
	TenantName        string   `json:"tenant_name"`
	Permissions       []string `json:"permissions"`
	UserID            string   `json:"user_id"`
}

type OAuthService struct {
	jwtMgr     *auth.JWTManager
	authSvc    *AuthService
	userRepo   repository.UserRepository
	tenantRepo repository.TenantRepository
	redis      *redis.Client
	logger     zerolog.Logger

	issuer   string
	loginURL string
	codeTTL  time.Duration
	clients  map[string]OAuthClient
}

func NewOAuthService(
	jwtMgr *auth.JWTManager,
	authSvc *AuthService,
	userRepo repository.UserRepository,
	tenantRepo repository.TenantRepository,
	redisClient *redis.Client,
	issuer string,
	loginURL string,
	clients []OAuthClient,
	logger zerolog.Logger,
) *OAuthService {
	clientMap := make(map[string]OAuthClient, len(clients))
	for _, client := range clients {
		if len(client.Scopes) == 0 {
			client.Scopes = []string{"openid", "profile", "email"}
		}
		if !client.RequirePKCE {
			client.RequirePKCE = true
		}
		clientMap[client.ClientID] = client
	}

	if issuer == "" {
		issuer = strings.TrimRight(jwtMgr.Issuer(), "/")
	}

	return &OAuthService{
		jwtMgr:     jwtMgr,
		authSvc:    authSvc,
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
		redis:      redisClient,
		logger:     logger,
		issuer:     strings.TrimRight(issuer, "/"),
		loginURL:   loginURL,
		codeTTL:    defaultOAuthCodeTTL,
		clients:    clientMap,
	}
}

func (s *OAuthService) Authorize(ctx context.Context, req OAuthAuthorizeRequest, accessToken string) (*OAuthAuthorizeResult, error) {
	client, err := s.validateAuthorizeRequest(req)
	if err != nil {
		return nil, err
	}

	if accessToken == "" {
		return &OAuthAuthorizeResult{LoginRedirectURL: s.buildLoginRedirect(req)}, nil
	}

	claims, err := s.jwtMgr.ValidateAccessToken(accessToken)
	if err != nil {
		s.logger.Debug().Err(err).Str("client_id", req.ClientID).Msg("oauth authorize access token invalid")
		return &OAuthAuthorizeResult{LoginRedirectURL: s.buildLoginRedirect(req)}, nil
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "UNAUTHENTICATED", Message: "authenticated user not found"}
	}

	if user.Status != model.UserStatusActive {
		return nil, &OAuthError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: "user account is not active"}
	}

	code, err := randomBase64URL(32)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to generate authorization code"}
	}

	scope := normalizeScopes(req.Scope, client.Scopes)
	payload := storedAuthorizationCode{
		ClientID:      client.ClientID,
		RedirectURI:   req.RedirectURI,
		Scope:         scope,
		CodeChallenge: req.CodeChallenge,
		UserID:        user.ID,
		TenantID:      user.TenantID,
	}

	if err := s.storeAuthorizationCode(ctx, code, payload); err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to persist authorization code"}
	}

	redirectURL, err := appendQuery(req.RedirectURI, map[string]string{
		"code":  code,
		"state": req.State,
	})
	if err != nil {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REDIRECT_URI", Message: "redirect_uri is invalid"}
	}

	return &OAuthAuthorizeResult{RedirectURL: redirectURL}, nil
}

func (s *OAuthService) ExchangeToken(ctx context.Context, req OAuthTokenRequest, ip, userAgent string) (*OAuthTokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return s.exchangeAuthorizationCode(ctx, req, ip, userAgent)
	case "refresh_token":
		return s.exchangeRefreshToken(ctx, req, ip, userAgent)
	default:
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "UNSUPPORTED_GRANT_TYPE", Message: "only authorization_code and refresh_token grants are supported"}
	}
}

func (s *OAuthService) UserInfo(ctx context.Context, accessToken string) (*OAuthUserInfo, error) {
	if accessToken == "" {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "UNAUTHENTICATED", Message: "bearer token is required"}
	}

	claims, err := s.jwtMgr.ValidateAccessToken(accessToken)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_TOKEN", Message: "token is invalid or expired"}
	}

	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_TOKEN", Message: "user not found"}
	}

	tenant, err := s.tenantRepo.GetByID(ctx, user.TenantID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_TOKEN", Message: "tenant not found"}
	}

	return buildUserInfo(user, tenant), nil
}

func (s *OAuthService) DiscoveryDocument() map[string]any {
	return map[string]any{
		"issuer":                                s.issuer,
		"authorization_endpoint":                s.issuer + "/api/v1/auth/oauth/authorize",
		"token_endpoint":                        s.issuer + "/api/v1/auth/oauth/token",
		"userinfo_endpoint":                     s.issuer + "/api/v1/auth/oauth/userinfo",
		"jwks_uri":                              s.issuer + "/.well-known/jwks.json",
		"scopes_supported":                      []string{"openid", "profile", "email", "roles"},
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": s.tokenEndpointAuthMethods(),
	}
}

func (s *OAuthService) JWKS() (map[string]any, error) {
	publicKey := s.jwtMgr.PublicKey()
	if publicKey == nil {
		return nil, fmt.Errorf("jwt manager public key unavailable")
	}

	return map[string]any{
		"keys": []map[string]any{buildJWK(publicKey)},
	}, nil
}

func (s *OAuthService) exchangeAuthorizationCode(ctx context.Context, req OAuthTokenRequest, ip, userAgent string) (*OAuthTokenResponse, error) {
	if req.Code == "" || req.RedirectURI == "" || req.ClientID == "" || req.CodeVerifier == "" {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "code, redirect_uri, client_id, and code_verifier are required"}
	}

	client, ok := s.clients[req.ClientID]
	if !ok {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_CLIENT", Message: "unregistered client_id"}
	}
	if err := validateClientAuthentication(client, req.ClientSecret); err != nil {
		return nil, err
	}

	payload, err := s.loadAuthorizationCode(ctx, req.Code)
	if err != nil {
		return nil, err
	}

	if payload.ClientID != client.ClientID {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_GRANT", Message: "authorization code does not belong to this client"}
	}
	if payload.RedirectURI != req.RedirectURI {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_GRANT", Message: "redirect_uri does not match the authorization request"}
	}
	if verifyPKCE(req.CodeVerifier, payload.CodeChallenge) != nil {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_GRANT", Message: "code_verifier does not match the stored PKCE challenge"}
	}
	if err := s.deleteAuthorizationCode(ctx, req.Code); err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to invalidate authorization code"}
	}

	user, err := s.userRepo.GetByID(ctx, payload.UserID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_GRANT", Message: "user not found for authorization code"}
	}
	tenant, err := s.tenantRepo.GetByID(ctx, payload.TenantID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_GRANT", Message: "tenant not found for authorization code"}
	}

	return s.issueOIDCTokens(ctx, client, user, tenant, payload.Scope, ip, userAgent)
}

func (s *OAuthService) exchangeRefreshToken(ctx context.Context, req OAuthTokenRequest, ip, userAgent string) (*OAuthTokenResponse, error) {
	if req.RefreshToken == "" || req.ClientID == "" {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "refresh_token and client_id are required"}
	}

	client, ok := s.clients[req.ClientID]
	if !ok {
		return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_CLIENT", Message: "unregistered client_id"}
	}
	if err := validateClientAuthentication(client, req.ClientSecret); err != nil {
		return nil, err
	}

	authResp, err := s.authSvc.RefreshToken(ctx, &dto.RefreshRequest{RefreshToken: req.RefreshToken}, ip, userAgent)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_GRANT", Message: "refresh token is invalid or expired"}
	}

	user, err := s.userRepo.GetByID(ctx, authResp.User.ID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_GRANT", Message: "user not found for refresh token"}
	}
	tenant, err := s.tenantRepo.GetByID(ctx, user.TenantID)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_GRANT", Message: "tenant not found for refresh token"}
	}

	idToken, err := s.buildIDToken(client.ClientID, user, tenant)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to sign id_token"}
	}

	return &OAuthTokenResponse{
		AccessToken:  authResp.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Until(authResp.ExpiresAt).Seconds()),
		RefreshToken: authResp.RefreshToken,
		IDToken:      idToken,
		Scope:        strings.Join(normalizeScopes("openid profile email roles", client.Scopes), " "),
	}, nil
}

func (s *OAuthService) issueOIDCTokens(
	ctx context.Context,
	client OAuthClient,
	user *model.User,
	tenant *model.Tenant,
	scope []string,
	ip string,
	userAgent string,
) (*OAuthTokenResponse, error) {
	authResp, err := s.authSvc.IssueTokens(ctx, user, ip, userAgent)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to issue access token"}
	}

	idToken, err := s.buildIDToken(client.ClientID, user, tenant)
	if err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to sign id_token"}
	}

	expiresIn := int(time.Until(authResp.ExpiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = int(s.jwtMgr.AccessTokenTTL().Seconds())
	}

	return &OAuthTokenResponse{
		AccessToken:  authResp.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: authResp.RefreshToken,
		IDToken:      idToken,
		Scope:        strings.Join(scope, " "),
	}, nil
}

func (s *OAuthService) buildIDToken(clientID string, user *model.User, tenant *model.Tenant) (string, error) {
	now := time.Now().UTC()
	claims := oidcIDClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.ID,
			Audience:  []string{clientID},
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtMgr.AccessTokenTTL())),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        mustRandomJWTID(),
		},
		Email:             user.Email,
		Name:              user.FullName(),
		PreferredUsername: user.Email,
		Role:              primaryRole(user),
		Roles:             user.RoleSlugs(),
		TenantID:          tenant.ID,
		TenantName:        tenant.Name,
		Permissions:       user.AllPermissions(),
		UserID:            user.ID,
	}
	return s.jwtMgr.SignClaims(claims)
}

func (s *OAuthService) validateAuthorizeRequest(req OAuthAuthorizeRequest) (OAuthClient, error) {
	if req.ResponseType != "code" {
		return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "UNSUPPORTED_RESPONSE_TYPE", Message: "response_type must be code"}
	}
	if req.ClientID == "" {
		return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_CLIENT", Message: "client_id is required"}
	}
	client, ok := s.clients[req.ClientID]
	if !ok {
		return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_CLIENT", Message: "unregistered client_id"}
	}
	if req.RedirectURI == "" || !matchesRedirectURI(client.RedirectURIs, req.RedirectURI) {
		return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REDIRECT_URI", Message: "redirect_uri is not registered for this client"}
	}
	if strings.TrimSpace(req.State) == "" {
		return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "state is required"}
	}
	if client.RequirePKCE {
		if req.CodeChallenge == "" {
			return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "code_challenge is required"}
		}
		if req.CodeChallengeMethod != "S256" {
			return OAuthClient{}, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "only S256 PKCE challenges are supported"}
		}
	}
	return client, nil
}

func (s *OAuthService) tokenEndpointAuthMethods() []string {
	methods := []string{"none"}
	for _, client := range s.clients {
		if client.ClientSecret != "" {
			return []string{"none", "client_secret_basic", "client_secret_post"}
		}
	}
	return methods
}

func (s *OAuthService) buildLoginRedirect(req OAuthAuthorizeRequest) string {
	if s.loginURL == "" {
		return ""
	}

	authorizeURL := s.issuer + "/api/v1/auth/oauth/authorize"
	params := url.Values{}
	params.Set("response_type", req.ResponseType)
	params.Set("client_id", req.ClientID)
	params.Set("redirect_uri", req.RedirectURI)
	params.Set("scope", req.Scope)
	params.Set("state", req.State)
	params.Set("code_challenge", req.CodeChallenge)
	params.Set("code_challenge_method", req.CodeChallengeMethod)

	loginURL, err := appendQuery(s.loginURL, map[string]string{
		"redirect": authorizeURL + "?" + params.Encode(),
	})
	if err != nil {
		return s.loginURL
	}
	return loginURL
}

func (s *OAuthService) storeAuthorizationCode(ctx context.Context, code string, payload storedAuthorizationCode) error {
	if s.redis == nil {
		return fmt.Errorf("redis is required for oauth authorization codes")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.redis.Set(ctx, oauthCodePrefix+code, data, s.codeTTL).Err()
}

func (s *OAuthService) loadAuthorizationCode(ctx context.Context, code string) (*storedAuthorizationCode, error) {
	if s.redis == nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "redis is required for oauth authorization codes"}
	}
	raw, err := s.redis.Get(ctx, oauthCodePrefix+code).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, &OAuthError{Status: http.StatusBadRequest, Code: "INVALID_GRANT", Message: "authorization code is invalid, expired, or already used"}
		}
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to load authorization code"}
	}
	var payload storedAuthorizationCode
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, &OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "authorization code payload is corrupt"}
	}
	return &payload, nil
}

func (s *OAuthService) deleteAuthorizationCode(ctx context.Context, code string) error {
	if s.redis == nil {
		return fmt.Errorf("redis is required for oauth authorization codes")
	}
	return s.redis.Del(ctx, oauthCodePrefix+code).Err()
}

func buildUserInfo(user *model.User, tenant *model.Tenant) *OAuthUserInfo {
	return &OAuthUserInfo{
		Subject:           user.ID,
		Email:             user.Email,
		Name:              user.FullName(),
		PreferredUsername: user.Email,
		Role:              primaryRole(user),
		Roles:             user.RoleSlugs(),
		TenantID:          tenant.ID,
		TenantName:        tenant.Name,
		Permissions:       user.AllPermissions(),
		UserID:            user.ID,
	}
}

func primaryRole(user *model.User) string {
	roles := user.RoleSlugs()
	if len(roles) == 0 {
		return "viewer"
	}
	return roles[0]
}

func normalizeScopes(raw string, allowed []string) []string {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, scope := range allowed {
		allowedSet[scope] = struct{}{}
	}

	parts := strings.Fields(strings.TrimSpace(raw))
	if len(parts) == 0 {
		parts = []string{"openid", "profile", "email"}
	}

	scopes := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, scope := range parts {
		if _, ok := allowedSet[scope]; !ok {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		return []string{"openid"}
	}
	return scopes
}

func verifyPKCE(codeVerifier, codeChallenge string) error {
	sum := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	if computed != codeChallenge {
		return fmt.Errorf("pkce mismatch")
	}
	return nil
}

func validateClientAuthentication(client OAuthClient, presentedSecret string) error {
	if client.ClientSecret == "" {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(client.ClientSecret), []byte(presentedSecret)) != 1 {
		return &OAuthError{Status: http.StatusUnauthorized, Code: "INVALID_CLIENT", Message: "client authentication failed"}
	}
	return nil
}

func matchesRedirectURI(registered []string, candidate string) bool {
	for _, allowed := range registered {
		if strings.TrimSpace(allowed) == candidate {
			return true
		}
	}
	return false
}

func buildJWK(key *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": rsaKeyID(key),
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}

func rsaKeyID(key *rsa.PublicKey) string {
	hash := sha256.Sum256(append(key.N.Bytes(), big.NewInt(int64(key.E)).Bytes()...))
	return base64.RawURLEncoding.EncodeToString(hash[:8])
}

func appendQuery(rawURL string, values map[string]string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, value := range values {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func randomBase64URL(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func mustRandomJWTID() string {
	value, err := randomBase64URL(18)
	if err != nil {
		return fmt.Sprintf("jti-%d", time.Now().UnixNano())
	}
	return value
}
