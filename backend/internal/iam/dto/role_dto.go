package dto

import (
	"time"

	"github.com/clario360/platform/internal/iam/model"
)

// ---------- Requests ----------

type CreateRoleRequest struct {
	Name        string   `json:"name" validate:"required,min=1,max=100"`
	Slug        string   `json:"slug" validate:"required,min=1,max=100"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions" validate:"required,min=1"`
}

type UpdateRoleRequest struct {
	Name        *string  `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type AssignRoleRequest struct {
	RoleID string `json:"role_id" validate:"required"`
}

// ---------- Responses ----------

type RoleResponse struct {
	ID           string   `json:"id"`
	TenantID     string   `json:"tenant_id"`
	Name         string   `json:"name"`
	Slug         string   `json:"slug"`
	Description  string   `json:"description"`
	IsSystemRole bool     `json:"is_system"`
	Permissions  []string `json:"permissions"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func RoleToResponse(r *model.Role) RoleResponse {
	return RoleResponse{
		ID:           r.ID,
		TenantID:     r.TenantID,
		Name:         r.Name,
		Slug:         r.Slug,
		Description:  r.Description,
		IsSystemRole: r.IsSystemRole,
		Permissions:  r.Permissions,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

func RolesToResponse(roles []model.Role) []RoleResponse {
	resp := make([]RoleResponse, len(roles))
	for i := range roles {
		resp[i] = RoleToResponse(&roles[i])
	}
	return resp
}
