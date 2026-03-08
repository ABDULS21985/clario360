package model

import (
	"time"

	"github.com/google/uuid"
)

type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusExpired   InvitationStatus = "expired"
	InvitationStatusCancelled InvitationStatus = "cancelled"
	InvitationStatusRevoked   InvitationStatus = "revoked"
)

type Invitation struct {
	ID            uuid.UUID        `json:"id"`
	TenantID      uuid.UUID        `json:"tenant_id"`
	Email         string           `json:"email"`
	RoleSlug      string           `json:"role_slug"`
	TokenHash     string           `json:"-"`
	TokenPrefix   string           `json:"token_prefix"`
	Status        InvitationStatus `json:"status"`
	InvitedBy     uuid.UUID        `json:"invited_by"`
	InvitedByName string           `json:"invited_by_name"`
	AcceptedAt    *time.Time       `json:"accepted_at,omitempty"`
	AcceptedBy    *uuid.UUID       `json:"accepted_by,omitempty"`
	ExpiresAt     time.Time        `json:"expires_at"`
	Message       *string          `json:"message,omitempty"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type InvitationDetails struct {
	InvitationID     uuid.UUID `json:"invitation_id"`
	TenantID         uuid.UUID `json:"tenant_id"`
	Email            string    `json:"email"`
	RoleSlug         string    `json:"role_slug"`
	RoleName         string    `json:"role_name"`
	OrganizationName string    `json:"organization_name"`
	InviterName      string    `json:"inviter_name"`
	ExpiresAt        time.Time `json:"expires_at"`
	Message          *string   `json:"message,omitempty"`
}
