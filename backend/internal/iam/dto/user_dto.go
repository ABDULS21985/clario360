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

// AdminCreateUserRequest is used by admins to create users within their tenant.
// Unlike the public Register endpoint, this allows setting initial status,
// role assignments, and optionally sending a welcome email.
type AdminCreateUserRequest struct {
	Email            string   `json:"email" validate:"required,email"`
	Password         string   `json:"password" validate:"required,min=12"`
	FirstName        string   `json:"first_name" validate:"required,min=1,max=100"`
	LastName         string   `json:"last_name" validate:"required,min=1,max=100"`
	Status           string   `json:"status,omitempty" validate:"omitempty,oneof=active inactive suspended"`
	RoleIDs          []string `json:"role_ids,omitempty"`
	SendWelcomeEmail bool     `json:"send_welcome_email,omitempty"`
}

// ---------- Responses ----------

type UserResponse struct {
	ID          string         `json:"id"`
	TenantID    string         `json:"tenant_id"`
	Email       string         `json:"email"`
	FirstName   string         `json:"first_name"`
	LastName    string         `json:"last_name"`
	FullName    string         `json:"full_name"`
	AvatarURL   *string        `json:"avatar_url,omitempty"`
	Status      string         `json:"status"`
	MFAEnabled  bool           `json:"mfa_enabled"`
	LastLoginAt *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Roles       []RoleResponse `json:"roles"`
}

type MFASetupResponse struct {
	Secret        string   `json:"secret"`
	OTPURL        string   `json:"otp_url"`
	RecoveryCodes []string `json:"recovery_codes"`
}

type PaginatedResponse struct {
	Data any            `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type SessionResponse struct {
	ID           string    `json:"id"`
	UserAgent    string    `json:"user_agent"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	IsCurrent    bool      `json:"is_current"`
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
	resp.Roles = make([]RoleResponse, len(u.Roles))
	for i, r := range u.Roles {
		resp.Roles[i] = RoleToResponse(&r)
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

func SessionsToResponse(sessions []model.Session) []SessionResponse {
	currentID := ""
	if len(sessions) > 0 {
		currentID = sessions[0].ID
	}

	resp := make([]SessionResponse, 0, len(sessions))
	for _, session := range sessions {
		userAgent := ""
		if session.UserAgent != nil {
			userAgent = *session.UserAgent
		}
		ipAddress := ""
		if session.IPAddress != nil {
			ipAddress = *session.IPAddress
		}
		lastActive := session.LastActiveAt
		if lastActive.IsZero() {
			lastActive = session.CreatedAt
		}
		resp = append(resp, SessionResponse{
			ID:           session.ID,
			UserAgent:    userAgent,
			IPAddress:    ipAddress,
			CreatedAt:    session.CreatedAt,
			LastActiveAt: lastActive,
			IsCurrent:    session.ID == currentID,
		})
	}
	return resp
}
