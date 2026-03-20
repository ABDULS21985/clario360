package model

import (
	"time"

	"github.com/google/uuid"
)

type ComplianceCheckType string

const (
	ComplianceCheckMeetingFrequency   ComplianceCheckType = "meeting_frequency"
	ComplianceCheckQuorumCompliance   ComplianceCheckType = "quorum_compliance"
	ComplianceCheckMinutesCompletion  ComplianceCheckType = "minutes_completion"
	ComplianceCheckActionTracking     ComplianceCheckType = "action_item_tracking"
	ComplianceCheckAttendanceRate     ComplianceCheckType = "attendance_rate"
	ComplianceCheckCharterReview      ComplianceCheckType = "charter_review"
	ComplianceCheckDocumentRetention  ComplianceCheckType = "document_retention"
	ComplianceCheckConflictOfInterest ComplianceCheckType = "conflict_of_interest"
)

type ComplianceStatus string

const (
	ComplianceStatusCompliant    ComplianceStatus = "compliant"
	ComplianceStatusNonCompliant ComplianceStatus = "non_compliant"
	ComplianceStatusWarning      ComplianceStatus = "warning"
	ComplianceStatusNotApplicable ComplianceStatus = "not_applicable"
)

type ComplianceSeverity string

const (
	ComplianceSeverityCritical ComplianceSeverity = "critical"
	ComplianceSeverityHigh     ComplianceSeverity = "high"
	ComplianceSeverityMedium   ComplianceSeverity = "medium"
	ComplianceSeverityLow      ComplianceSeverity = "low"
)

type ComplianceCheck struct {
	ID             uuid.UUID          `json:"id"`
	TenantID       uuid.UUID          `json:"tenant_id"`
	CommitteeID    *uuid.UUID         `json:"committee_id,omitempty"`
	CheckType      ComplianceCheckType `json:"check_type"`
	CheckName      string             `json:"check_name"`
	Status         ComplianceStatus   `json:"status"`
	Severity       ComplianceSeverity `json:"severity"`
	Description    string             `json:"description"`
	Finding        *string            `json:"finding,omitempty"`
	Recommendation *string            `json:"recommendation,omitempty"`
	Evidence       map[string]any     `json:"evidence"`
	PeriodStart    time.Time          `json:"period_start"`
	PeriodEnd      time.Time          `json:"period_end"`
	CheckedAt      time.Time          `json:"checked_at"`
	CheckedBy      string             `json:"checked_by"`
	CreatedAt      time.Time          `json:"created_at"`
}

type ComplianceFilters struct {
	CommitteeID *uuid.UUID
	CheckType   *ComplianceCheckType
	Statuses    []ComplianceStatus
	DateFrom    *time.Time
	DateTo      *time.Time
	Page        int
	PerPage     int
}

type ComplianceReport struct {
	TenantID            uuid.UUID                 `json:"tenant_id"`
	Results             []ComplianceCheck         `json:"results"`
	ByStatus            map[string]int            `json:"by_status"`
	ByCheckType         map[string]int            `json:"by_check_type"`
	ByCommittee         []CommitteeCompliance     `json:"by_committee"`
	Score               float64                   `json:"score"`
	NonCompliantCount   int                       `json:"non_compliant_count"`
	WarningCount        int                       `json:"warning_count"`
	GeneratedAt         time.Time                 `json:"generated_at"`
}

type CommitteeCompliance struct {
	CommitteeID   uuid.UUID `json:"committee_id"`
	CommitteeName string    `json:"committee_name"`
	Score         float64   `json:"score"`
	Warnings      int       `json:"warnings"`
	NonCompliant  int       `json:"non_compliant"`
}
