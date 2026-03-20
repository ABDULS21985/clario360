package dto

import "time"

// ---------- Requests ----------

type RegisterRequest struct {
	TenantID  string `json:"tenant_id" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=12"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name" validate:"required,min=1,max=100"`
}

type LoginRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=12"`
}

type VerifyMFARequest struct {
	MFAToken string `json:"mfa_token" validate:"required"`
	Code     string `json:"code" validate:"required,len=6"`
}

// ---------- Responses ----------

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
	TokenType    string       `json:"token_type"`
	User         UserResponse `json:"user"`
}

type MFARequiredResponse struct {
	MFARequired bool   `json:"mfa_required"`
	MFAToken    string `json:"mfa_token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
