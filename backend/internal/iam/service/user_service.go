package service

import (
	"context"
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
	"github.com/clario360/platform/pkg/crypto"
)

type UserService struct {
	userRepo    repository.UserRepository
	roleRepo    repository.RoleRepository
	sessionRepo repository.SessionRepository
	redis       *redis.Client
	producer    *events.Producer
	logger      zerolog.Logger
	bcryptCost  int
	mfaKey      []byte // 32-byte AES-256 key for MFA secret encryption
}

func NewUserService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	sessionRepo repository.SessionRepository,
	rdb *redis.Client,
	producer *events.Producer,
	logger zerolog.Logger,
	bcryptCost int,
) *UserService {
	return &UserService{
		userRepo:    userRepo,
		roleRepo:    roleRepo,
		sessionRepo: sessionRepo,
		redis:       rdb,
		producer:    producer,
		logger:      logger,
		bcryptCost:  bcryptCost,
	}
}

// SetMFAEncryptionKey sets the AES-256 key used to encrypt MFA secrets at rest.
func (s *UserService) SetMFAEncryptionKey(key []byte) {
	s.mfaKey = key
}

func (s *UserService) List(ctx context.Context, tenantID string, page, perPage int, search, status string) ([]dto.UserResponse, int, error) {
	filter := repository.UserFilter{
		Page:    page,
		PerPage: perPage,
	}
	if search != "" {
		filter.Search = &search
	}
	if status != "" {
		filter.Status = &status
	}

	users, total, err := s.userRepo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}

	return dto.UsersToResponse(users), total, nil
}

func (s *UserService) GetByID(ctx context.Context, userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := dto.UserToResponse(user)
	return &resp, nil
}

func (s *UserService) GetByEmail(ctx context.Context, tenantID, email string) (*dto.UserResponse, error) {
	var (
		user *model.User
		err  error
	)

	if tenantID != "" {
		user, err = s.userRepo.GetByEmail(ctx, tenantID, email)
	} else {
		user, err = s.userRepo.GetByEmailGlobal(ctx, email)
	}
	if err != nil {
		return nil, err
	}

	resp := dto.UserToResponse(user)
	return &resp, nil
}

func (s *UserService) Update(ctx context.Context, userID string, req *dto.UpdateUserRequest, updatedBy string) (*dto.UserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	user.UpdatedBy = &updatedBy

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	s.publishEvent(ctx, "user.updated", user.TenantID, user.ID, nil)

	updated, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := dto.UserToResponse(updated)
	return &resp, nil
}

func (s *UserService) Delete(ctx context.Context, userID, deletedBy string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.userRepo.SoftDelete(ctx, userID, deletedBy); err != nil {
		return err
	}

	// Invalidate all sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)

	s.publishEvent(ctx, "user.deleted", user.TenantID, user.ID, nil)
	return nil
}

func (s *UserService) UpdateStatus(ctx context.Context, userID string, req *dto.UpdateStatusRequest, updatedBy string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	status := model.UserStatus(req.Status)
	if err := s.userRepo.UpdateStatus(ctx, userID, status, updatedBy); err != nil {
		return err
	}

	// If suspended, invalidate sessions
	if status == model.UserStatusSuspended {
		_ = s.sessionRepo.DeleteByUserID(ctx, userID)
	}

	s.publishEvent(ctx, "user.updated", user.TenantID, user.ID, nil)
	return nil
}

func (s *UserService) ChangePassword(ctx context.Context, userID string, req *dto.ChangePasswordRequest) error {
	if err := validatePassword(req.NewPassword); err != nil {
		return fmt.Errorf("%s: %w", err.Error(), model.ErrValidation)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect: %w", model.ErrUnauthorized)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return err
	}

	// Invalidate all other sessions
	_ = s.sessionRepo.DeleteByUserID(ctx, userID)

	return nil
}

func (s *UserService) ListSessions(ctx context.Context, userID string) ([]dto.SessionResponse, error) {
	sessions, err := s.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.SessionsToResponse(sessions), nil
}

func (s *UserService) DeleteSession(ctx context.Context, userID, sessionID string) error {
	sessions, err := s.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if len(sessions) > 0 && sessions[0].ID == sessionID {
		return fmt.Errorf("cannot revoke current session: %w", model.ErrForbidden)
	}
	for _, session := range sessions {
		if session.ID == sessionID {
			return s.sessionRepo.Delete(ctx, sessionID)
		}
	}
	return model.ErrNotFound
}

func (s *UserService) DeleteSessions(ctx context.Context, userID string, excludeCurrent bool) error {
	if !excludeCurrent {
		return s.sessionRepo.DeleteByUserID(ctx, userID)
	}

	sessions, err := s.sessionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	for idx, session := range sessions {
		if idx == 0 {
			continue
		}
		if err := s.sessionRepo.Delete(ctx, session.ID); err != nil {
			return err
		}
	}
	return nil
}

// EnableMFA generates a TOTP secret and recovery codes but does NOT enable MFA yet.
// The user must call VerifyMFASetup with a valid code to confirm their authenticator is configured.
func (s *UserService) EnableMFA(ctx context.Context, userID string) (*dto.MFASetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.MFAEnabled {
		return nil, fmt.Errorf("MFA is already enabled: %w", model.ErrConflict)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Clario360",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("generating TOTP key: %w", err)
	}

	secret := key.Secret()

	// Encrypt MFA secret before storing
	storedSecret := secret
	if len(s.mfaKey) == 32 {
		encrypted, err := crypto.Encrypt([]byte(secret), s.mfaKey)
		if err != nil {
			return nil, fmt.Errorf("encrypting MFA secret: %w", err)
		}
		storedSecret = encrypted
	}

	// Store the secret but do NOT enable MFA yet (two-step flow)
	if err := s.userRepo.UpdateMFA(ctx, userID, false, &storedSecret); err != nil {
		return nil, err
	}

	// Generate recovery codes (bcrypt hashed)
	codes := make([]string, recoveryCodeCount)
	recoveryKey := recoveryPrefix + userID
	s.redis.Del(ctx, recoveryKey)

	for i := 0; i < recoveryCodeCount; i++ {
		code, err := generateRandomHex(4)
		if err != nil {
			return nil, fmt.Errorf("generating recovery code: %w", err)
		}
		codes[i] = code
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.MinCost)
		if err != nil {
			return nil, fmt.Errorf("hashing recovery code: %w", err)
		}
		s.redis.SAdd(ctx, recoveryKey, string(hash))
	}
	s.redis.Persist(ctx, recoveryKey)

	return &dto.MFASetupResponse{
		Secret:        secret,
		OTPURL:        key.URL(),
		RecoveryCodes: codes,
	}, nil
}

// VerifyMFASetup confirms the user's authenticator is correctly configured by validating a TOTP code.
// Only after this succeeds is MFA actually enabled on the account.
func (s *UserService) VerifyMFASetup(ctx context.Context, userID string, code string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.MFAEnabled {
		return fmt.Errorf("MFA is already enabled: %w", model.ErrConflict)
	}

	if user.MFASecret == nil {
		return fmt.Errorf("MFA setup not initiated — call enable first: %w", model.ErrValidation)
	}

	// Decrypt the stored secret
	secret := *user.MFASecret
	if len(s.mfaKey) == 32 {
		decrypted, err := crypto.Decrypt(secret, s.mfaKey)
		if err != nil {
			return fmt.Errorf("decrypting MFA secret: %w", err)
		}
		secret = string(decrypted)
	}

	if !totp.Validate(code, secret) {
		return fmt.Errorf("invalid code — please re-scan QR code and try again: %w", model.ErrInvalidMFA)
	}

	// Code is valid — enable MFA
	storedSecret := *user.MFASecret // keep the encrypted version
	if err := s.userRepo.UpdateMFA(ctx, userID, true, &storedSecret); err != nil {
		return err
	}

	s.publishEvent(ctx, "user.mfa.enabled", user.TenantID, user.ID, map[string]any{
		"user_id":   user.ID,
		"email":     user.Email,
		"timestamp": time.Now().UTC(),
	})
	return nil
}

func (s *UserService) DisableMFA(ctx context.Context, userID string, req *dto.DisableMFARequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.MFAEnabled || user.MFASecret == nil {
		return fmt.Errorf("MFA is not enabled: %w", model.ErrValidation)
	}

	// Decrypt secret for TOTP validation
	secret := *user.MFASecret
	if len(s.mfaKey) == 32 {
		decrypted, err := crypto.Decrypt(secret, s.mfaKey)
		if err != nil {
			return fmt.Errorf("decrypting MFA secret: %w", err)
		}
		secret = string(decrypted)
	}

	if !totp.Validate(req.Code, secret) {
		return model.ErrInvalidMFA
	}

	if err := s.userRepo.UpdateMFA(ctx, userID, false, nil); err != nil {
		return err
	}

	// Remove recovery codes
	s.redis.Del(ctx, recoveryPrefix+userID)

	s.publishEvent(ctx, "user.mfa.disabled", user.TenantID, user.ID, map[string]any{
		"user_id":     user.ID,
		"email":       user.Email,
		"disabled_by": user.ID,
		"reason":      "user_requested",
		"timestamp":   time.Now().UTC(),
	})
	return nil
}

// decryptMFASecret decrypts a stored MFA secret for TOTP validation.
func (s *UserService) decryptMFASecret(stored string) (string, error) {
	if len(s.mfaKey) == 32 {
		decrypted, err := crypto.Decrypt(stored, s.mfaKey)
		if err != nil {
			return "", fmt.Errorf("decrypting MFA secret: %w", err)
		}
		return string(decrypted), nil
	}
	return stored, nil
}

func (s *UserService) publishEvent(ctx context.Context, eventType, tenantID, userID string, data map[string]any) {
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
	evt.UserID = userID
	if err := s.producer.Publish(ctx, "platform.iam.events", evt); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}
