package model

import (
	"time"

	"github.com/google/uuid"
)

type CommitteeType string

const (
	CommitteeTypeBoard        CommitteeType = "board"
	CommitteeTypeAudit        CommitteeType = "audit"
	CommitteeTypeRisk         CommitteeType = "risk"
	CommitteeTypeCompensation CommitteeType = "compensation"
	CommitteeTypeNomination   CommitteeType = "nomination"
	CommitteeTypeExecutive    CommitteeType = "executive"
	CommitteeTypeGovernance   CommitteeType = "governance"
	CommitteeTypeAdHoc        CommitteeType = "ad_hoc"
)

type MeetingFrequency string

const (
	MeetingFrequencyWeekly     MeetingFrequency = "weekly"
	MeetingFrequencyBiWeekly   MeetingFrequency = "bi_weekly"
	MeetingFrequencyMonthly    MeetingFrequency = "monthly"
	MeetingFrequencyQuarterly  MeetingFrequency = "quarterly"
	MeetingFrequencySemiAnnual MeetingFrequency = "semi_annual"
	MeetingFrequencyAnnual     MeetingFrequency = "annual"
	MeetingFrequencyAdHoc      MeetingFrequency = "ad_hoc"
)

type QuorumType string

const (
	QuorumTypePercentage QuorumType = "percentage"
	QuorumTypeFixedCount QuorumType = "fixed_count"
)

type CommitteeStatus string

const (
	CommitteeStatusActive    CommitteeStatus = "active"
	CommitteeStatusInactive  CommitteeStatus = "inactive"
	CommitteeStatusDissolved CommitteeStatus = "dissolved"
)

type CommitteeMemberRole string

const (
	CommitteeMemberRoleChair      CommitteeMemberRole = "chair"
	CommitteeMemberRoleViceChair  CommitteeMemberRole = "vice_chair"
	CommitteeMemberRoleSecretary  CommitteeMemberRole = "secretary"
	CommitteeMemberRoleMember     CommitteeMemberRole = "member"
	CommitteeMemberRoleObserver   CommitteeMemberRole = "observer"
)

type Committee struct {
	ID               uuid.UUID        `json:"id"`
	TenantID         uuid.UUID        `json:"tenant_id"`
	Name             string           `json:"name"`
	Type             CommitteeType    `json:"type"`
	Description      string           `json:"description"`
	ChairUserID      uuid.UUID        `json:"chair_user_id"`
	ViceChairUserID  *uuid.UUID       `json:"vice_chair_user_id,omitempty"`
	SecretaryUserID  *uuid.UUID       `json:"secretary_user_id,omitempty"`
	MeetingFrequency MeetingFrequency `json:"meeting_frequency"`
	QuorumPercentage int              `json:"quorum_percentage"`
	QuorumType       QuorumType       `json:"quorum_type"`
	QuorumFixedCount *int             `json:"quorum_fixed_count,omitempty"`
	Charter          *string          `json:"charter,omitempty"`
	EstablishedDate  *time.Time       `json:"established_date,omitempty"`
	DissolutionDate  *time.Time       `json:"dissolution_date,omitempty"`
	Status           CommitteeStatus  `json:"status"`
	Tags             []string         `json:"tags"`
	Metadata         map[string]any   `json:"metadata"`
	CreatedBy        uuid.UUID        `json:"created_by"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	DeletedAt        *time.Time       `json:"deleted_at,omitempty"`
	Members          []CommitteeMember `json:"members,omitempty"`
	Stats            *CommitteeStats  `json:"stats,omitempty"`
}

type CommitteeMember struct {
	ID          uuid.UUID           `json:"id"`
	TenantID    uuid.UUID           `json:"tenant_id"`
	CommitteeID uuid.UUID           `json:"committee_id"`
	UserID      uuid.UUID           `json:"user_id"`
	UserName    string              `json:"user_name"`
	UserEmail   string              `json:"user_email"`
	Role        CommitteeMemberRole `json:"role"`
	JoinedAt    time.Time           `json:"joined_at"`
	LeftAt      *time.Time          `json:"left_at,omitempty"`
	Active      bool                `json:"active"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type CommitteeStats struct {
	ActiveMembers          int `json:"active_members"`
	UpcomingMeetings       int `json:"upcoming_meetings"`
	CompletedMeetings      int `json:"completed_meetings"`
	OpenActionItems        int `json:"open_action_items"`
	OverdueActionItems     int `json:"overdue_action_items"`
	PendingMinutesApproval int `json:"pending_minutes_approval"`
}
