package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ID is a UUID string used for all entity identifiers.
type ID = string

// Metadata holds common metadata for all entities.
type Metadata struct {
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	CreatedBy ID         `json:"created_by" db:"created_by"`
	UpdatedBy ID         `json:"updated_by" db:"updated_by"`
}

// TenantScoped is embedded by entities that belong to a tenant.
type TenantScoped struct {
	TenantID ID `json:"tenant_id" db:"tenant_id"`
}

// SoftDelete provides soft deletion support.
type SoftDelete struct {
	DeletedAt pgtype.Timestamptz `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted returns true if the entity has been soft-deleted.
func (s SoftDelete) IsDeleted() bool {
	return s.DeletedAt.Valid
}

// Tenant represents a tenant in the multi-tenant system.
type Tenant struct {
	ID          ID        `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	Settings    JSONMap   `json:"settings" db:"settings"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// User represents an authenticated user.
type User struct {
	ID        ID        `json:"id" db:"id"`
	TenantID  ID        `json:"tenant_id" db:"tenant_id"`
	Email     string    `json:"email" db:"email"`
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	Roles     []string  `json:"roles" db:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

func (u User) FullName() string {
	return u.FirstName + " " + u.LastName
}

// Role represents a role in the RBAC system.
type Role struct {
	ID          ID       `json:"id" db:"id"`
	TenantID    ID       `json:"tenant_id" db:"tenant_id"`
	Name        string   `json:"name" db:"name"`
	Description string   `json:"description" db:"description"`
	Permissions []string `json:"permissions" db:"-"`
	IsSystem    bool     `json:"is_system" db:"is_system"`
}

// JSONMap is a map[string]any that can be stored as JSONB in PostgreSQL.
type JSONMap map[string]any
