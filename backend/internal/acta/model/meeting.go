package model

import (
	"time"

	"github.com/google/uuid"
)

type MeetingStatus string

const (
	MeetingStatusDraft      MeetingStatus = "draft"
	MeetingStatusScheduled  MeetingStatus = "scheduled"
	MeetingStatusInProgress MeetingStatus = "in_progress"
	MeetingStatusCompleted  MeetingStatus = "completed"
	MeetingStatusCancelled  MeetingStatus = "cancelled"
	MeetingStatusPostponed  MeetingStatus = "postponed"
)

type LocationType string

const (
	LocationTypePhysical LocationType = "physical"
	LocationTypeVirtual  LocationType = "virtual"
	LocationTypeHybrid   LocationType = "hybrid"
)

type AttendanceStatus string

const (
	AttendanceStatusInvited   AttendanceStatus = "invited"
	AttendanceStatusConfirmed AttendanceStatus = "confirmed"
	AttendanceStatusDeclined  AttendanceStatus = "declined"
	AttendanceStatusPresent   AttendanceStatus = "present"
	AttendanceStatusAbsent    AttendanceStatus = "absent"
	AttendanceStatusProxy     AttendanceStatus = "proxy"
	AttendanceStatusExcused   AttendanceStatus = "excused"
)

type Meeting struct {
	ID                 uuid.UUID      `json:"id"`
	TenantID           uuid.UUID      `json:"tenant_id"`
	CommitteeID        uuid.UUID      `json:"committee_id"`
	CommitteeName      string         `json:"committee_name"`
	Title              string         `json:"title"`
	Description        string         `json:"description"`
	MeetingNumber      *int           `json:"meeting_number,omitempty"`
	ScheduledAt        time.Time      `json:"scheduled_at"`
	ScheduledEndAt     *time.Time     `json:"scheduled_end_at,omitempty"`
	ActualStartAt      *time.Time     `json:"actual_start_at,omitempty"`
	ActualEndAt        *time.Time     `json:"actual_end_at,omitempty"`
	DurationMinutes    int            `json:"duration_minutes"`
	Location           *string        `json:"location,omitempty"`
	LocationType       LocationType   `json:"location_type"`
	VirtualLink        *string        `json:"virtual_link,omitempty"`
	VirtualPlatform    *string        `json:"virtual_platform,omitempty"`
	Status             MeetingStatus  `json:"status"`
	CancellationReason *string        `json:"cancellation_reason,omitempty"`
	QuorumRequired     int            `json:"quorum_required"`
	AttendeeCount      int            `json:"attendee_count"`
	PresentCount       int            `json:"present_count"`
	QuorumMet          *bool          `json:"quorum_met,omitempty"`
	AgendaItemCount    int            `json:"agenda_item_count"`
	ActionItemCount    int            `json:"action_item_count"`
	HasMinutes         bool           `json:"has_minutes"`
	MinutesStatus      *string        `json:"minutes_status,omitempty"`
	WorkflowInstanceID *uuid.UUID     `json:"workflow_instance_id,omitempty"`
	Tags               []string       `json:"tags"`
	Metadata           map[string]any `json:"metadata"`
	CreatedBy          uuid.UUID      `json:"created_by"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          *time.Time     `json:"deleted_at,omitempty"`
	Agenda             []AgendaItem   `json:"agenda,omitempty"`
	Attendance         []Attendee     `json:"attendance,omitempty"`
	LatestMinutes      *MeetingMinutes `json:"latest_minutes,omitempty"`
	Attachments        []MeetingAttachment `json:"attachments,omitempty"`
}

type Attendee struct {
	ID                uuid.UUID            `json:"id"`
	TenantID          uuid.UUID            `json:"tenant_id"`
	MeetingID         uuid.UUID            `json:"meeting_id"`
	UserID            uuid.UUID            `json:"user_id"`
	UserName          string               `json:"user_name"`
	UserEmail         string               `json:"user_email"`
	MemberRole        CommitteeMemberRole  `json:"member_role"`
	Status            AttendanceStatus     `json:"status"`
	ConfirmedAt       *time.Time           `json:"confirmed_at,omitempty"`
	CheckedInAt       *time.Time           `json:"checked_in_at,omitempty"`
	CheckedOutAt      *time.Time           `json:"checked_out_at,omitempty"`
	ProxyUserID       *uuid.UUID           `json:"proxy_user_id,omitempty"`
	ProxyUserName     *string              `json:"proxy_user_name,omitempty"`
	ProxyAuthorizedBy *uuid.UUID           `json:"proxy_authorized_by,omitempty"`
	Notes             *string              `json:"notes,omitempty"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
}

type MeetingAttachment struct {
	FileID      uuid.UUID  `json:"file_id"`
	Name        string     `json:"name"`
	ContentType *string    `json:"content_type,omitempty"`
	UploadedBy  *uuid.UUID `json:"uploaded_by,omitempty"`
	UploadedAt  time.Time  `json:"uploaded_at"`
}

type MeetingFilters struct {
	CommitteeID *uuid.UUID
	Statuses    []MeetingStatus
	DateFrom    *time.Time
	DateTo      *time.Time
	Search      string
	Page        int
	PerPage     int
}

type CalendarDay struct {
	Date     time.Time        `json:"date"`
	Meetings []MeetingSummary `json:"meetings"`
}

type MeetingSummary struct {
	ID             uuid.UUID     `json:"id"`
	CommitteeID    uuid.UUID     `json:"committee_id"`
	CommitteeName  string        `json:"committee_name"`
	Title          string        `json:"title"`
	Status         MeetingStatus `json:"status"`
	ScheduledAt    time.Time     `json:"scheduled_at"`
	DurationMinutes int          `json:"duration_minutes"`
	Location       *string       `json:"location,omitempty"`
	QuorumMet      *bool         `json:"quorum_met,omitempty"`
}
