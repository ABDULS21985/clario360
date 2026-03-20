package model

import (
	"time"

	"github.com/google/uuid"
)

type MinutesStatus string

const (
	MinutesStatusDraft             MinutesStatus = "draft"
	MinutesStatusReview            MinutesStatus = "review"
	MinutesStatusRevisionRequested MinutesStatus = "revision_requested"
	MinutesStatusApproved          MinutesStatus = "approved"
	MinutesStatusPublished         MinutesStatus = "published"
)

type MeetingMinutes struct {
	ID                   uuid.UUID         `json:"id"`
	TenantID             uuid.UUID         `json:"tenant_id"`
	MeetingID            uuid.UUID         `json:"meeting_id"`
	Content              string            `json:"content"`
	AISummary            *string           `json:"ai_summary,omitempty"`
	Status               MinutesStatus     `json:"status"`
	SubmittedForReviewAt *time.Time        `json:"submitted_for_review_at,omitempty"`
	SubmittedBy          *uuid.UUID        `json:"submitted_by,omitempty"`
	ReviewedBy           *uuid.UUID        `json:"reviewed_by,omitempty"`
	ReviewNotes          *string           `json:"review_notes,omitempty"`
	ApprovedBy           *uuid.UUID        `json:"approved_by,omitempty"`
	ApprovedAt           *time.Time        `json:"approved_at,omitempty"`
	PublishedAt          *time.Time        `json:"published_at,omitempty"`
	Version              int               `json:"version"`
	PreviousVersionID    *uuid.UUID        `json:"previous_version_id,omitempty"`
	AIActionItems        []ExtractedAction `json:"ai_action_items"`
	AIGenerated          bool              `json:"ai_generated"`
	CreatedBy            uuid.UUID         `json:"created_by"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	MeetingTitle         string            `json:"meeting_title,omitempty"`
}

type ExtractedAction struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AssignedTo  string     `json:"assigned_to"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Priority    string     `json:"priority"`
	Source      string     `json:"source"`
}

type GeneratedMinutes struct {
	Content       string            `json:"content"`
	AISummary     string            `json:"ai_summary"`
	AIActionItems []ExtractedAction `json:"ai_action_items"`
}
