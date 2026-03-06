package dto

import (
	"time"

	"github.com/clario360/platform/internal/iam/model"
)

// ---------- Requests ----------

type UpdateUserRequest struct {
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=active inactive suspended"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=12"`
}

type DisableMFARequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

// ---------- Responses ----------

type UserResponse struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Email     string         `json:"email"`
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	FullName  string         `json:"full_name"`
	AvatarURL *string        `json:"avatar_url,omitempty"`
	Status    string         `json:"status"`
	MFAEnabled bool          `json:"mfa_enabled"`
	LastLoginAt *time.Time   `json:"last_login_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Roles     []RoleResponse `json:"roles,omitempty"`
}

type MFASetupResponse struct {
	Secret        string   `json:"secret"`
	OTPURL        string   `json:"otp_url"`
	RecoveryCodes []string `json:"recovery_codes"`
}

type PaginatedResponse struct {
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page     int `json:"page"`
	PerPage  int `json:"per_page"`
	Total    int `json:"total"`
	LastPage int `json:"last_page"`
}

func UserToResponse(u *model.User) UserResponse {
	resp := UserResponse{
		ID:          u.ID,
		TenantID:    u.TenantID,
		Email:       u.Email,
		FirstName:   u.FirstName,
		LastName:    u.LastName,
		FullName:    u.FullName(),
		AvatarURL:   u.AvatarURL,
		Status:      string(u.Status),
		MFAEnabled:  u.MFAEnabled,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
	if len(u.Roles) > 0 {
		resp.Roles = make([]RoleResponse, len(u.Roles))
		for i, r := range u.Roles {
			resp.Roles[i] = RoleToResponse(&r)
		}
	}
	return resp
}

func UsersToResponse(users []model.User) []UserResponse {
	resp := make([]UserResponse, len(users))
	for i := range users {
		resp[i] = UserToResponse(&users[i])
	}
	return resp
}
