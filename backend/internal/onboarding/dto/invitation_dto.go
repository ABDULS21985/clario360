package dto

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
