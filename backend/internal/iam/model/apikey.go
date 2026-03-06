package model

import (
	"encoding/json"
	"time"
)

type APIKey struct {
	ID          string
	TenantID    string
	Name        string
	KeyHash     string
	KeyPrefix   string
	Permissions json.RawMessage
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	CreatedBy   *string
	RevokedAt   *time.Time
}

func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

func (k *APIKey) IsExpired() bool {
	if k.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*k.ExpiresAt)
}
