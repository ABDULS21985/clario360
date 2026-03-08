package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
	"github.com/clario360/platform/pkg/crypto"
)

const (
	mfaPendingPrefix   = "mfa:pending:"
	mfaPendingTTL      = 5 * time.Minute
	loginLockoutPrefix = "login:lockout:"
	loginLockoutMax    = 5
	loginLockoutTTL    = 15 * time.Minute
	resetTokenPrefix   = "reset:token:"
	resetTokenTTL      = 1 * time.Hour
	recoveryPrefix     = "mfa:recovery:"
	recoveryCodeCount  = 10
)

// commonPasswords is a set of commonly used passwords that must be rejected.
var commonPasswords = map[string]struct{}{
	"password": {}, "123456": {}, "123456789": {}, "12345678": {},
	"12345": {}, "1234567": {}, "1234567890": {}, "qwerty": {},
	"abc123": {}, "password1": {}, "password123": {}, "admin": {},
	"letmein": {}, "welcome": {}, "monkey": {}, "master": {},
	"dragon": {}, "login": {}, "princess": {}, "qwerty123": {},
	"solo": {}, "passw0rd": {}, "starwars": {}, "iloveyou": {},
	"trustno1": {}, "sunshine": {}, "football": {}, "shadow": {},
	"michael": {}, "superman": {}, "access": {}, "hello": {},
	"charlie": {}, "donald": {}, "batman": {}, "qwerty12345": {},
	"password12345": {}, "letmein123": {}, "welcome1": {}, "1q2w3e4r": {},
	"1q2w3e4r5t": {}, "zaq1zaq1": {}, "qazwsx": {}, "1qaz2wsx": {},
	"changeme": {}, "p@ssw0rd": {}, "p@ssword": {}, "passw0rd!": {},
	"clario360": {}, "clario": {}, "administrator": {}, "root": {},
	"toor": {}, "pa$$w0rd": {}, "p@ssw0rd1": {}, "test1234!": {},
	"qwerty123!": {}, "welcome1!": {}, "password1!": {}, "winter2024!": {},
	"summer2024!": {}, "spring2024!": {}, "autumn2024!": {}, "january2024!": {},
	"company123!": {}, "security1!": {}, "admin123!": {}, "user12345!": {},
	"abcdefghijkl": {}, "aaaaaaaaaaaa": {}, "123456789012": {}, "qwertyuiopas": {},
}

type AuthService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	roleRepo    repository.RoleRepository
	tenantRepo  repository.TenantRepository
	jwtMgr      *auth.JWTManager
	redis       *redis.Client
	producer    *events.Producer
	logger      zerolog.Logger
	bcryptCost  int
	refreshTTL  time.Duration
	mfaKey      []byte // 32-byte AES-256 key for MFA secret encryption
}

func NewAuthService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	roleRepo repository.RoleRepository,
	tenantRepo repository.TenantRepository,
	jwtMgr *auth.JWTManager,
	rdb *redis.Client,
	producer *events.Producer,
	logger zerolog.Logger,
	bcryptCost int,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		roleRepo:    roleRepo,
		tenantRepo:  tenantRepo,
		jwtMgr:      jwtMgr,
		redis:       rdb,
		producer:    producer,
		logger:      logger,
		bcryptCost:  bcryptCost,
		refreshTTL:  refreshTTL,
	}
}

// SetMFAEncryptionKey sets the AES-256 key used to encrypt MFA secrets at rest.
func (s *AuthService) SetMFAEncryptionKey(key []byte) {
	s.mfaKey = key
}

func (s *AuthService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.AuthResponse, error) {
	if err := validatePassword(req.Password); err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), model.ErrValidation)
	}

	// Verify tenant exists
	tenant, err := s.tenantRepo.GetByID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant: %w", err)
	}
	if tenant.Status != model.TenantStatusActive && tenant.Status != model.TenantStatusTrial {
		return nil, fmt.Errorf("tenant is not active: %w", model.ErrForbidden)
	}

	// Check if email already exists in tenant
	_, err = s.userRepo.GetByEmail(ctx, req.TenantID, req.Email)
	if err == nil {
		return nil, fmt.Errorf("email %s: %w", req.Email, model.ErrConflict)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &model.User{
		TenantID:     req.TenantID,
		Email:        strings.ToLower(strings.TrimSpace(req.Email)),
		PasswordHash: string(hash),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Status:       model.UserStatusActive,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Seed system roles for this tenant if they don't exist
	if err := s.roleRepo.SeedSystemRoles(ctx, req.TenantID); err != nil {
		s.logger.Error().Err(err).Msg("failed to seed system roles")
	}

	// First user in tenant becomes tenant-admin
	userCount, err := s.userRepo.CountByTenant(ctx, req.TenantID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to count tenant users")
	}

	roleSlug := "viewer"
	if userCount <= 1 {
		roleSlug = "tenant-admin"
	}

	role, err := s.roleRepo.GetBySlug(ctx, req.TenantID, roleSlug)
	if err == nil {
		if err := s.roleRepo.AssignToUser(ctx, user.ID, role.ID, req.TenantID, user.ID); err != nil {
			s.logger.Error().Err(err).Msg("failed to assign role")
		}
		user.Roles = []model.Role{*role}
	}

	s.publishEvent(ctx, "user.registered", req.TenantID, user.ID, nil)

	return s.generateTokens(ctx, user, "", "")
}

func (s *AuthService) Login(ctx context.Context, req *dto.LoginRequest, ip, userAgent string) (any, error) {
	// Hash the key material to avoid storing PII (raw IP + email) in Redis
	lockoutKey := fmt.Sprintf("%s%s", loginLockoutPrefix, sha256Hex(ip+":"+strings.ToLower(req.Email)))

	// Atomic lockout check: INCR the attempt counter first, then check.
	// This prevents race conditions between separate GET and INCR calls.
	attempts, incrErr := s.redis.Incr(ctx, lockoutKey).Result()
	if incrErr == nil {
		if attempts == 1 {
			// First attempt in this window — set the TTL
			s.redis.Expire(ctx, lockoutKey, loginLockoutTTL)
		}
		if attempts > int64(loginLockoutMax) {
			return nil, model.ErrAccountLocked
		}
	}
	// If Redis is unavailable (incrErr != nil), fail-open: allow the request

	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		s.publishEvent(ctx, "user.login.failed", req.TenantID, "", map[string]any{
			"email":         strings.ToLower(strings.TrimSpace(req.Email)),
			"email_hash":    sha256Hex(req.Email),
			"ip_address":    ip,
			"attempt_count": attempts,
			"user_agent":    userAgent,
			"timestamp":     time.Now().UTC(),
			"reason":        "user_not_found",
		})
		return nil, model.ErrUnauthorized
	}

	if user.Status != model.UserStatusActive {
		s.publishEvent(ctx, "user.login.failed", user.TenantID, user.ID, map[string]any{
			"user_id":       user.ID,
			"email":         user.Email,
			"ip_address":    ip,
			"attempt_count": attempts,
			"user_agent":    userAgent,
			"timestamp":     time.Now().UTC(),
			"reason":        "inactive_account",
		})
		return nil, fmt.Errorf("account is %s: %w", user.Status, model.ErrForbidden)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.publishEvent(ctx, "user.login.failed", user.TenantID, user.ID, map[string]any{
			"user_id":       user.ID,
			"email":         user.Email,
			"ip_address":    ip,
			"attempt_count": attempts,
			"user_agent":    userAgent,
			"timestamp":     time.Now().UTC(),
			"reason":        "invalid_password",
		})
		return nil, model.ErrUnauthorized
	}

	// Login successful — clear lockout counter
	s.redis.Del(ctx, lockoutKey)

	// If MFA is enabled, return MFA challenge
	if user.MFAEnabled {
		token, err := generateRandomHex(32)
		if err != nil {
			return nil, fmt.Errorf("generating mfa token: %w", err)
		}
		mfaKey := mfaPendingPrefix + token
		s.redis.Set(ctx, mfaKey, fmt.Sprintf("%s|%s|%s|%s", user.ID, user.TenantID, ip, userAgent), mfaPendingTTL)

		return &dto.MFARequiredResponse{
			MFARequired: true,
			MFAToken:    token,
		}, nil
	}

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)
	s.publishEvent(ctx, "user.login.success", user.TenantID, user.ID, map[string]any{
		"user_id":    user.ID,
		"email":      user.Email,
		"ip_address": ip,
		"user_agent": userAgent,
		"timestamp":  time.Now().UTC(),
	})

	return s.generateTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) VerifyMFA(ctx context.Context, req *dto.VerifyMFARequest) (*dto.AuthResponse, error) {
	mfaKey := mfaPendingPrefix + req.MFAToken
	data, err := s.redis.Get(ctx, mfaKey).Result()
	if err != nil {
		return nil, model.ErrInvalidToken
	}

	parts := strings.SplitN(data, "|", 4)
	if len(parts) != 4 {
		return nil, model.ErrInvalidToken
	}
	userID, _, ip, userAgent := parts[0], parts[1], parts[2], parts[3]

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.MFASecret == nil {
		return nil, model.ErrInvalidMFA
	}

	// Decrypt MFA secret if encrypted
	secret := *user.MFASecret
	if len(s.mfaKey) == 32 {
		decrypted, err := crypto.Decrypt(secret, s.mfaKey)
		if err != nil {
			return nil, fmt.Errorf("decrypting MFA secret: %w", err)
		}
		secret = string(decrypted)
	}

	// Try TOTP code
	valid := totp.Validate(req.Code, secret)

	// If TOTP fails, try recovery codes
	if !valid {
		valid, err = s.tryRecoveryCode(ctx, user.ID, req.Code)
		if err != nil {
			return nil, err
		}
	}

	if !valid {
		return nil, model.ErrInvalidMFA
	}

	// Delete the pending MFA token
	s.redis.Del(ctx, mfaKey)

	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)
	s.publishEvent(ctx, "user.login.success", user.TenantID, user.ID, map[string]any{
		"user_id":    user.ID,
		"email":      user.Email,
		"ip_address": ip,
		"user_agent": userAgent,
		"timestamp":  time.Now().UTC(),
	})

	return s.generateTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) RefreshToken(ctx context.Context, req *dto.RefreshRequest, ip, userAgent string) (*dto.AuthResponse, error) {
	// Validate the refresh token JWT
	userID, err := s.jwtMgr.ValidateRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", model.ErrInvalidToken)
	}

	// Look up session by token hash
	tokenHash := sha256Hex(req.RefreshToken)
	session, err := s.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		// Token not found — possible reuse of an already-rotated token (theft detection).
		// Revoke ALL sessions for this user as a safety measure.
		s.logger.Warn().Str("user_id", userID).Msg("refresh token reuse detected — revoking all sessions for user")
		_ = s.sessionRepo.DeleteByUserID(ctx, userID)
		s.publishEvent(ctx, "user.sessions.revoked", "", userID, map[string]any{"reason": "token_reuse"})
		return nil, fmt.Errorf("session not found: %w", model.ErrInvalidToken)
	}

	if session.UserID != userID {
		// Token/session mismatch — revoke all sessions
		s.logger.Warn().Str("user_id", userID).Msg("refresh token user mismatch — revoking all sessions")
		_ = s.sessionRepo.DeleteByUserID(ctx, userID)
		s.publishEvent(ctx, "user.sessions.revoked", "", userID, map[string]any{"reason": "token_mismatch"})
		return nil, model.ErrInvalidToken
	}

	// Delete old session (rotation — each refresh token is single-use)
	if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
		s.logger.Error().Err(err).Msg("failed to delete old session")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Status != model.UserStatusActive {
		return nil, fmt.Errorf("account is %s: %w", user.Status, model.ErrForbidden)
	}

	return s.generateTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) Logout(ctx context.Context, req *dto.LogoutRequest) error {
	tokenHash := sha256Hex(req.RefreshToken)
	session, err := s.sessionRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil // Silently ignore invalid tokens on logout
	}

	s.publishEvent(ctx, "user.logout", session.TenantID, session.UserID, nil)
	return s.sessionRepo.Delete(ctx, session.ID)
}

func (s *AuthService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error {
	// Generate reset token regardless of whether email exists (prevent enumeration)
	token, err := generateRandomHex(32)
	if err != nil {
		return fmt.Errorf("generating reset token: %w", err)
	}

	tokenHash := sha256Hex(token)
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Look up the user to store their ID (needed for ResetPassword)
	user, err := s.userRepo.GetByEmail(ctx, req.TenantID, email)
	if err != nil {
		// Don't reveal whether user exists; just log and return success
		s.logger.Debug().Str("email", email).Msg("forgot password for unknown email")
		return nil
	}

	// Store in Redis: hash → userID (so ResetPassword can find the user)
	s.redis.Set(ctx, resetTokenPrefix+tokenHash, user.ID, resetTokenTTL)

	// Log the token (dev mode — in production, send via email)
	s.logger.Info().
		Str("email", email).
		Str("reset_token", token).
		Msg("password reset token generated (dev mode)")

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	if err := validatePassword(req.NewPassword); err != nil {
		return fmt.Errorf("%s: %w", err.Error(), model.ErrValidation)
	}

	tokenHash := sha256Hex(req.Token)
	userID, err := s.redis.Get(ctx, resetTokenPrefix+tokenHash).Result()
	if err != nil {
		return model.ErrInvalidToken
	}

	// Delete the token immediately (single use)
	s.redis.Del(ctx, resetTokenPrefix+tokenHash)

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	// Invalidate all sessions for this user
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)

	s.logger.Info().Str("user_id", userID).Msg("password reset completed")
	return nil
}

// ResetPasswordForUser resets password when user ID is known (called from handler after lookup).
func (s *AuthService) ResetPasswordForUser(ctx context.Context, userID, newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return fmt.Errorf("%s: %w", err.Error(), model.ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	// Invalidate all sessions for this user
	return s.sessionRepo.DeleteByUserID(ctx, userID)
}

// IssueTokens exposes the standard token issuance flow for trusted internal
// callers such as the OIDC provider. It preserves session creation and refresh
// token rotation behavior by delegating to the shared token generator.
func (s *AuthService) IssueTokens(ctx context.Context, user *model.User, ip, userAgent string) (*dto.AuthResponse, error) {
	return s.generateTokens(ctx, user, ip, userAgent)
}

func (s *AuthService) generateTokens(ctx context.Context, user *model.User, ip, userAgent string) (*dto.AuthResponse, error) {
	roleSlugs := user.RoleSlugs()

	tokenPair, err := s.jwtMgr.GenerateTokenPair(user.ID, user.TenantID, user.Email, roleSlugs)
	if err != nil {
		return nil, fmt.Errorf("generating tokens: %w", err)
	}

	tokenHash := sha256Hex(tokenPair.RefreshToken)

	var ipPtr, uaPtr *string
	if ip != "" {
		ipPtr = &ip
	}
	if userAgent != "" {
		uaPtr = &userAgent
	}

	session := &model.Session{
		UserID:           user.ID,
		TenantID:         user.TenantID,
		RefreshTokenHash: tokenHash,
		IPAddress:        ipPtr,
		UserAgent:        uaPtr,
		ExpiresAt:        time.Now().Add(s.refreshTTL),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
		TokenType:    "Bearer",
		User:         dto.UserToResponse(user),
	}, nil
}

func (s *AuthService) tryRecoveryCode(ctx context.Context, userID, code string) (bool, error) {
	key := recoveryPrefix + userID
	codes, err := s.redis.SMembers(ctx, key).Result()
	if err != nil {
		return false, nil
	}

	// Recovery codes are bcrypt-hashed
	for _, stored := range codes {
		if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(code)); err == nil {
			// Remove used recovery code
			s.redis.SRem(ctx, key, stored)
			return true, nil
		}
	}
	return false, nil
}

func (s *AuthService) publishEvent(ctx context.Context, eventType, tenantID, userID string, data map[string]any) {
	if s.producer == nil {
		return
	}
	payload := map[string]any{}
	for key, value := range data {
		payload[key] = value
	}
	if tenantID != "" {
		payload["tenant_id"] = tenantID
	}
	if userID != "" {
		payload["user_id"] = userID
	}

	evt, err := events.NewEvent(normalizeIAMEventType(eventType), "iam-service", tenantID, payload)
	if err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to create event")
		return
	}
	if userID != "" {
		evt.UserID = userID
	}
	if err := s.producer.Publish(ctx, "platform.iam.events", evt); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}

// --- Helpers ---

func sha256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func generateRandomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func validatePassword(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters")
	}
	if len(password) > 128 {
		return fmt.Errorf("password must not exceed 128 characters")
	}

	// Check against common passwords (case-insensitive)
	if _, found := commonPasswords[strings.ToLower(password)]; found {
		return fmt.Errorf("password is too common — please choose a different password")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return fmt.Errorf("password must include uppercase, lowercase, digit, and special character")
	}
	return nil
}
