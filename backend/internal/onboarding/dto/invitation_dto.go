package dto

import (
	"time"

	"github.com/google/uuid"
)

type InvitationInput struct {
	Email    string `json:"email" validate:"required,email,max=255"`
	RoleSlug string `json:"role_slug" validate:"required,min=2,max=100"`
	Message  string `json:"message,omitempty" validate:"omitempty,max=500"`
}

type BatchInviteRequest struct {
	Invitations []InvitationInput `json:"invitations" validate:"required,min=1,max=10,dive"`
}

type AcceptInviteRequest struct {
	Token     string `json:"token" validate:"required,min=20,max=128"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
	Password  string `json:"password" validate:"required,min=12,max=128"`
}

type AcceptInviteResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresAt    string `json:"expires_at"`
	TenantID     string `json:"tenant_id"`
	Message      string `json:"message"`
}

// InvitationListItem is the enriched invitation returned by paginated list endpoints.
type InvitationListItem struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	Email         string     `json:"email"`
	RoleSlug      string     `json:"role_slug"`
	RoleName      string     `json:"role_name"`
	Status        string     `json:"status"`
	InvitedBy     uuid.UUID  `json:"invited_by"`
	InvitedByName string     `json:"invited_by_name"`
	AcceptedAt    *time.Time `json:"accepted_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	Message       *string    `json:"message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// InvitationStatsResponse holds aggregate invitation statistics for a tenant.
type InvitationStatsResponse struct {
	TotalSent      int     `json:"total_sent"`
	Pending        int     `json:"pending"`
	Accepted       int     `json:"accepted"`
	Expired        int     `json:"expired"`
	AcceptanceRate float64 `json:"acceptance_rate"`
}
