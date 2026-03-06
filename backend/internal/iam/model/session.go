package model

import "time"

type Session struct {
	ID               string
	UserID           string
	TenantID         string
	RefreshTokenHash string
	IPAddress        *string
	UserAgent        *string
	ExpiresAt        time.Time
	CreatedAt        time.Time
}
