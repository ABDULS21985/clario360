package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/iam/repository"
)

type APIKeyService struct {
	keyRepo  repository.APIKeyRepository
	producer *events.Producer
	logger   zerolog.Logger
}

func NewAPIKeyService(
	keyRepo repository.APIKeyRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *APIKeyService {
	return &APIKeyService{
		keyRepo:  keyRepo,
		producer: producer,
		logger:   logger,
	}
}

// CreateAPIKeyRequest for the service layer.
type CreateAPIKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	ExpiresIn   *int     `json:"expires_in_days,omitempty"` // days
}

// CreateAPIKeyResponse includes the raw key (only shown once).
type CreateAPIKeyResponse struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Key       string     `json:"key"` // Full key, only returned on creation
	KeyPrefix string     `json:"key_prefix"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// APIKeyResponse for list operations (no raw key).
type APIKeyListItem struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

func (s *APIKeyService) Create(ctx context.Context, tenantID string, req *CreateAPIKeyRequest, createdBy string) (*CreateAPIKeyResponse, error) {
	// Generate 32-byte random key
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generating api key: %w", err)
	}

	// Create key with prefix: clario_ + 6-char identifier + _ + hex key
	prefix := hex.EncodeToString(rawBytes[:3]) // 6 hex chars
	rawKey := fmt.Sprintf("clario_%s_%s", prefix, hex.EncodeToString(rawBytes))

	keyHash := sha256Hex(rawKey)

	permsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return nil, fmt.Errorf("marshaling permissions: %w", err)
	}

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * 24 * time.Hour)
		expiresAt = &t
	}

	apiKey := &model.APIKey{
		TenantID:    tenantID,
		Name:        req.Name,
		KeyHash:     keyHash,
		KeyPrefix:   "clario_" + prefix,
		Permissions: permsJSON,
		ExpiresAt:   expiresAt,
		CreatedBy:   &createdBy,
	}

	if err := s.keyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("storing api key: %w", err)
	}

	s.publishEvent(ctx, "apikey.created", tenantID, createdBy)

	return &CreateAPIKeyResponse{
		ID:        apiKey.ID,
		Name:      apiKey.Name,
		Key:       rawKey,
		KeyPrefix: apiKey.KeyPrefix,
		ExpiresAt: expiresAt,
		CreatedAt: apiKey.CreatedAt,
	}, nil
}

func (s *APIKeyService) List(ctx context.Context, tenantID string) ([]APIKeyListItem, error) {
	keys, err := s.keyRepo.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	items := make([]APIKeyListItem, len(keys))
	for i, k := range keys {
		items[i] = APIKeyListItem{
			ID:         k.ID,
			Name:       k.Name,
			KeyPrefix:  k.KeyPrefix,
			LastUsedAt: k.LastUsedAt,
			ExpiresAt:  k.ExpiresAt,
			CreatedAt:  k.CreatedAt,
			RevokedAt:  k.RevokedAt,
		}
	}
	return items, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, keyID, tenantID, userID string) error {
	if err := s.keyRepo.Revoke(ctx, keyID); err != nil {
		return err
	}
	s.publishEvent(ctx, "apikey.revoked", tenantID, userID)
	return nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*model.APIKey, error) {
	keyHash := sha256Hex(rawKey)
	key, err := s.keyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		return nil, err
	}

	if key.IsRevoked() {
		return nil, fmt.Errorf("api key is revoked: %w", model.ErrUnauthorized)
	}
	if key.IsExpired() {
		return nil, fmt.Errorf("api key is expired: %w", model.ErrUnauthorized)
	}

	// Update last used timestamp asynchronously
	go func() {
		_ = s.keyRepo.UpdateLastUsed(context.Background(), key.ID)
	}()

	return key, nil
}

func (s *APIKeyService) publishEvent(ctx context.Context, eventType, tenantID, userID string) {
	if s.producer == nil {
		return
	}
	evt, err := events.NewEvent(eventType, "iam-service", tenantID, nil)
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
