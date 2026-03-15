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

// CreateAPIKeyRequest matches the frontend contract.
type CreateAPIKeyRequest struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt *string  `json:"expires_at"` // ISO 8601 string or null
}

// CreateAPIKeyResponse matches the frontend CreateApiKeyResponse type.
type CreateAPIKeyResponse struct {
	Key    APIKeyListItem `json:"key"`
	Secret string         `json:"secret"`
}

// APIKeyListItem matches the frontend ApiKey type.
type APIKeyListItem struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Prefix     string          `json:"prefix"`
	Scopes     json.RawMessage `json:"scopes"`
	Status     string          `json:"status"`
	LastUsedAt *time.Time      `json:"last_used_at"`
	ExpiresAt  *time.Time      `json:"expires_at"`
	CreatedAt  time.Time       `json:"created_at"`
	CreatedBy  *string         `json:"created_by"`
}

func apiKeyStatus(k *model.APIKey) string {
	if k.RevokedAt != nil {
		return "revoked"
	}
	if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
		return "expired"
	}
	return "active"
}

func toAPIKeyListItem(k model.APIKey) APIKeyListItem {
	scopes := k.Permissions
	if scopes == nil {
		scopes = json.RawMessage("[]")
	}
	return APIKeyListItem{
		ID:         k.ID,
		Name:       k.Name,
		Prefix:     k.KeyPrefix,
		Scopes:     scopes,
		Status:     apiKeyStatus(&k),
		LastUsedAt: k.LastUsedAt,
		ExpiresAt:  k.ExpiresAt,
		CreatedAt:  k.CreatedAt,
		CreatedBy:  k.CreatedBy,
	}
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

	permsJSON, err := json.Marshal(req.Scopes)
	if err != nil {
		return nil, fmt.Errorf("marshaling scopes: %w", err)
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("invalid expires_at format: %w", model.ErrValidation)
		}
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
		Key:    toAPIKeyListItem(*apiKey),
		Secret: rawKey,
	}, nil
}

func (s *APIKeyService) ListPaginated(ctx context.Context, tenantID string, page, perPage int, search, status string) ([]APIKeyListItem, int, error) {
	keys, total, err := s.keyRepo.ListPaginated(ctx, tenantID, page, perPage, search, status)
	if err != nil {
		return nil, 0, err
	}

	items := make([]APIKeyListItem, len(keys))
	for i, k := range keys {
		items[i] = toAPIKeyListItem(k)
	}
	return items, total, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, keyID, tenantID, userID string) error {
	if err := s.keyRepo.Revoke(ctx, keyID, tenantID); err != nil {
		return err
	}
	s.publishEvent(ctx, "apikey.revoked", tenantID, userID)
	return nil
}

func (s *APIKeyService) Rotate(ctx context.Context, keyID, tenantID, userID string) (*CreateAPIKeyResponse, error) {
	key, err := s.keyRepo.GetByID(ctx, keyID, tenantID)
	if err != nil {
		return nil, err
	}

	if key.IsRevoked() {
		return nil, fmt.Errorf("cannot rotate revoked key: %w", model.ErrValidation)
	}

	// Generate new key
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return nil, fmt.Errorf("generating api key: %w", err)
	}

	prefix := hex.EncodeToString(rawBytes[:3])
	rawKey := fmt.Sprintf("clario_%s_%s", prefix, hex.EncodeToString(rawBytes))
	keyHash := sha256Hex(rawKey)
	newPrefix := "clario_" + prefix

	if err := s.keyRepo.RotateKey(ctx, keyID, tenantID, keyHash, newPrefix); err != nil {
		return nil, err
	}

	key.KeyPrefix = newPrefix
	key.KeyHash = keyHash

	s.publishEvent(ctx, "apikey.rotated", tenantID, userID)

	return &CreateAPIKeyResponse{
		Key:    toAPIKeyListItem(*key),
		Secret: rawKey,
	}, nil
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
	payload := map[string]any{}
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
