package model

import (
	"time"

	"github.com/google/uuid"
)

type AlertCategory string
type AlertSeverity string
type AlertStatus string

const (
	AlertCategoryRisk        AlertCategory = "risk"
	AlertCategoryCompliance  AlertCategory = "compliance"
	AlertCategoryDataQuality AlertCategory = "data_quality"
	AlertCategoryGovernance  AlertCategory = "governance"
	AlertCategoryLegal       AlertCategory = "legal"
	AlertCategoryOperational AlertCategory = "operational"
	AlertCategoryFinancial   AlertCategory = "financial"
	AlertCategoryStrategic   AlertCategory = "strategic"
)

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityHigh     AlertSeverity = "high"
	AlertSeverityMedium   AlertSeverity = "medium"
	AlertSeverityLow      AlertSeverity = "low"
	AlertSeverityInfo     AlertSeverity = "info"
)

const (
	AlertStatusNew          AlertStatus = "new"
	AlertStatusViewed       AlertStatus = "viewed"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusActioned     AlertStatus = "actioned"
	AlertStatusDismissed    AlertStatus = "dismissed"
	AlertStatusEscalated    AlertStatus = "escalated"
)

type ExecutiveAlert struct {
	ID                uuid.UUID     `json:"id"`
	TenantID          uuid.UUID     `json:"tenant_id"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Category          AlertCategory `json:"category"`
	Severity          AlertSeverity `json:"severity"`
	SourceSuite       string        `json:"source_suite"`
	SourceType        string        `json:"source_type"`
	SourceEntityID    *uuid.UUID    `json:"source_entity_id,omitempty"`
	SourceEventType   *string       `json:"source_event_type,omitempty"`
	Status            AlertStatus   `json:"status"`
	ViewedAt          *time.Time    `json:"viewed_at,omitempty"`
	ViewedBy          *uuid.UUID    `json:"viewed_by,omitempty"`
	ActionedAt        *time.Time    `json:"actioned_at,omitempty"`
	ActionedBy        *uuid.UUID    `json:"actioned_by,omitempty"`
	ActionNotes       *string       `json:"action_notes,omitempty"`
	DismissedAt       *time.Time    `json:"dismissed_at,omitempty"`
	DismissedBy       *uuid.UUID    `json:"dismissed_by,omitempty"`
	DismissReason     *string       `json:"dismiss_reason,omitempty"`
	DedupKey          *string       `json:"dedup_key,omitempty"`
	OccurrenceCount   int           `json:"occurrence_count"`
	FirstSeenAt       time.Time     `json:"first_seen_at"`
	LastSeenAt        time.Time     `json:"last_seen_at"`
	LinkedKPIID       *uuid.UUID    `json:"linked_kpi_id,omitempty"`
	LinkedDashboardID *uuid.UUID    `json:"linked_dashboard_id,omitempty"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type AlertStats struct {
	ByCategory map[string]int `json:"by_category"`
	BySeverity map[string]int `json:"by_severity"`
	ByStatus   map[string]int `json:"by_status"`
	Total      int            `json:"total"`
}
