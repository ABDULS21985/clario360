package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateMeetingRequest struct {
	CommitteeID     uuid.UUID      `json:"committee_id"`
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	ScheduledAt     time.Time      `json:"scheduled_at"`
	ScheduledEndAt  *time.Time     `json:"scheduled_end_at"`
	DurationMinutes int            `json:"duration_minutes"`
	Location        *string        `json:"location"`
	LocationType    string         `json:"location_type"`
	VirtualLink     *string        `json:"virtual_link"`
	VirtualPlatform *string        `json:"virtual_platform"`
	Tags            []string       `json:"tags"`
	Metadata        map[string]any `json:"metadata"`
}

func (r *CreateMeetingRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.LocationType = strings.TrimSpace(r.LocationType)
}

type UpdateMeetingRequest struct {
	Title           string         `json:"title"`
	Description     string         `json:"description"`
	ScheduledAt     time.Time      `json:"scheduled_at"`
	ScheduledEndAt  *time.Time     `json:"scheduled_end_at"`
	DurationMinutes int            `json:"duration_minutes"`
	Location        *string        `json:"location"`
	LocationType    string         `json:"location_type"`
	VirtualLink     *string        `json:"virtual_link"`
	VirtualPlatform *string        `json:"virtual_platform"`
	Tags            []string       `json:"tags"`
	Metadata        map[string]any `json:"metadata"`
}

func (r *UpdateMeetingRequest) Normalize() {
	r.Title = strings.TrimSpace(r.Title)
	r.Description = strings.TrimSpace(r.Description)
	r.LocationType = strings.TrimSpace(r.LocationType)
}

type CancelMeetingRequest struct {
	Reason string `json:"reason"`
}

type PostponeMeetingRequest struct {
	NewScheduledAt    time.Time  `json:"new_scheduled_at"`
	NewScheduledEndAt *time.Time `json:"new_scheduled_end_at"`
	Reason            string     `json:"reason"`
}

type AttendanceRequest struct {
	UserID            uuid.UUID  `json:"user_id"`
	Status            string     `json:"status"`
	Notes             *string    `json:"notes"`
	ProxyUserID       *uuid.UUID `json:"proxy_user_id"`
	ProxyUserName     *string    `json:"proxy_user_name"`
	ProxyAuthorizedBy *uuid.UUID `json:"proxy_authorized_by"`
}

type BulkAttendanceRequest struct {
	Attendance []AttendanceRequest `json:"attendance"`
}

type AttachmentRequest struct {
	FileID      uuid.UUID  `json:"file_id"`
	Name        string     `json:"name"`
	ContentType *string    `json:"content_type"`
	UploadedBy  *uuid.UUID `json:"uploaded_by"`
}
