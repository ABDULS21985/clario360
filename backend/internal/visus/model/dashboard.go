package model

import (
	"time"

	"github.com/google/uuid"
)

type DashboardVisibility string

const (
	DashboardVisibilityPrivate      DashboardVisibility = "private"
	DashboardVisibilityTeam         DashboardVisibility = "team"
	DashboardVisibilityOrganization DashboardVisibility = "organization"
	DashboardVisibilityPublic       DashboardVisibility = "public"
)

type Dashboard struct {
	ID          uuid.UUID           `json:"id"`
	TenantID    uuid.UUID           `json:"tenant_id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	GridColumns int                 `json:"grid_columns"`
	Visibility  DashboardVisibility `json:"visibility"`
	SharedWith  []uuid.UUID         `json:"shared_with"`
	IsDefault   bool                `json:"is_default"`
	IsSystem    bool                `json:"is_system"`
	Tags        []string            `json:"tags"`
	Metadata    map[string]any      `json:"metadata"`
	CreatedBy   uuid.UUID           `json:"created_by"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
	DeletedAt   *time.Time          `json:"deleted_at,omitempty"`
	Widgets     []Widget            `json:"widgets,omitempty"`
	WidgetCount int                 `json:"widget_count,omitempty"`
}
