package model

import (
	"time"

	"github.com/google/uuid"
)

type ActionItemPriority string

const (
	ActionItemPriorityCritical ActionItemPriority = "critical"
	ActionItemPriorityHigh     ActionItemPriority = "high"
	ActionItemPriorityMedium   ActionItemPriority = "medium"
	ActionItemPriorityLow      ActionItemPriority = "low"
)

type ActionItemStatus string

const (
	ActionItemStatusPending    ActionItemStatus = "pending"
	ActionItemStatusInProgress ActionItemStatus = "in_progress"
	ActionItemStatusCompleted  ActionItemStatus = "completed"
	ActionItemStatusOverdue    ActionItemStatus = "overdue"
	ActionItemStatusCancelled  ActionItemStatus = "cancelled"
	ActionItemStatusDeferred   ActionItemStatus = "deferred"
)

type ActionItem struct {
	ID                 uuid.UUID          `json:"id"`
	TenantID           uuid.UUID          `json:"tenant_id"`
	MeetingID          uuid.UUID          `json:"meeting_id"`
	AgendaItemID       *uuid.UUID         `json:"agenda_item_id,omitempty"`
	CommitteeID        uuid.UUID          `json:"committee_id"`
	Title              string             `json:"title"`
	Description        string             `json:"description"`
	Priority           ActionItemPriority `json:"priority"`
	AssignedTo         uuid.UUID          `json:"assigned_to"`
	AssigneeName       string             `json:"assignee_name"`
	AssignedBy         uuid.UUID          `json:"assigned_by"`
	DueDate            time.Time          `json:"due_date"`
	OriginalDueDate    time.Time          `json:"original_due_date"`
	ExtendedCount      int                `json:"extended_count"`
	ExtensionReason    *string            `json:"extension_reason,omitempty"`
	Status             ActionItemStatus   `json:"status"`
	CompletedAt        *time.Time         `json:"completed_at,omitempty"`
	CompletionNotes    *string            `json:"completion_notes,omitempty"`
	CompletionEvidence []uuid.UUID        `json:"completion_evidence"`
	FollowUpMeetingID  *uuid.UUID         `json:"follow_up_meeting_id,omitempty"`
	ReviewedAt         *time.Time         `json:"reviewed_at,omitempty"`
	Tags               []string           `json:"tags"`
	Metadata           map[string]any     `json:"metadata"`
	CreatedBy          uuid.UUID          `json:"created_by"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	MeetingTitle       string             `json:"meeting_title,omitempty"`
}

type ActionItemFilters struct {
	CommitteeID *uuid.UUID
	MeetingID   *uuid.UUID
	AssigneeID  *uuid.UUID
	Statuses    []ActionItemStatus
	OverdueOnly bool
	Page        int
	PerPage     int
}

type ActionItemStats struct {
	ByStatus   map[string]int `json:"by_status"`
	ByPriority map[string]int `json:"by_priority"`
	Open       int            `json:"open"`
	Overdue    int            `json:"overdue"`
	Completed  int            `json:"completed"`
}

type ActionItemSummary struct {
	ID           uuid.UUID          `json:"id"`
	Title        string             `json:"title"`
	CommitteeID  uuid.UUID          `json:"committee_id"`
	CommitteeName string            `json:"committee_name"`
	AssigneeName string             `json:"assignee_name"`
	DueDate      time.Time          `json:"due_date"`
	Priority     ActionItemPriority `json:"priority"`
	Status       ActionItemStatus   `json:"status"`
}
