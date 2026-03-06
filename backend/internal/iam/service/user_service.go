package service

import (
	"context"
	"fmt"

	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

type UserService struct {
	userRepo    repository.UserRepository
	roleRepo    repository.RoleRepository
	sessionRepo repository.SessionRepository
	redis       *redis.Client
	producer    *events.Producer
	logger      zerolog.Logger
	bcryptCost  int
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

	s.publishEvent(ctx, "user.updated", user.TenantID, user.ID)

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

	s.publishEvent(ctx, "user.deleted", user.TenantID, user.ID)
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

	s.publishEvent(ctx, "user.updated", user.TenantID, user.ID)
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
	if err := s.userRepo.UpdateMFA(ctx, userID, true, &secret); err != nil {
		return nil, err
	}

	// Generate recovery codes
	codes := make([]string, recoveryCodeCount)
	recoveryKey := recoveryPrefix + userID
	s.redis.Del(ctx, recoveryKey)

	for i := 0; i < recoveryCodeCount; i++ {
		code, err := generateRandomHex(4)
		if err != nil {
			return nil, fmt.Errorf("generating recovery code: %w", err)
		}
		codes[i] = code
		s.redis.SAdd(ctx, recoveryKey, sha256Hex(code))
	}
	// Recovery codes persist indefinitely
	s.redis.Persist(ctx, recoveryKey)

	s.publishEvent(ctx, "user.mfa.enabled", user.TenantID, user.ID)

	return &dto.MFASetupResponse{
		Secret:        secret,
		OTPURL:        key.URL(),
		RecoveryCodes: codes,
	}, nil
}

func (s *UserService) DisableMFA(ctx context.Context, userID string, req *dto.DisableMFARequest) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.MFAEnabled || user.MFASecret == nil {
		return fmt.Errorf("MFA is not enabled: %w", model.ErrValidation)
	}

	if !totp.Validate(req.Code, *user.MFASecret) {
		return model.ErrInvalidMFA
	}

	if err := s.userRepo.UpdateMFA(ctx, userID, false, nil); err != nil {
		return err
	}

	// Remove recovery codes
	s.redis.Del(ctx, recoveryPrefix+userID)

	s.publishEvent(ctx, "user.mfa.disabled", user.TenantID, user.ID)
	return nil
}

func (s *UserService) publishEvent(ctx context.Context, eventType, tenantID, userID string) {
	if s.producer == nil {
		return
	}
	evt, err := events.NewEvent(eventType, "iam-service", tenantID, nil)
	if err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to create event")
		return
	}
	evt.UserID = userID
	if err := s.producer.Publish(ctx, "platform.iam.events", evt); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}
