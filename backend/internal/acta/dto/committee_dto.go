package dto

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type CreateCommitteeRequest struct {
	Name             string            `json:"name"`
	Type             string            `json:"type"`
	Description      string            `json:"description"`
	ChairUserID      uuid.UUID         `json:"chair_user_id"`
	ViceChairUserID  *uuid.UUID        `json:"vice_chair_user_id"`
	SecretaryUserID  *uuid.UUID        `json:"secretary_user_id"`
	MeetingFrequency string            `json:"meeting_frequency"`
	QuorumPercentage int               `json:"quorum_percentage"`
	QuorumType       string            `json:"quorum_type"`
	QuorumFixedCount *int              `json:"quorum_fixed_count"`
	Charter          *string           `json:"charter"`
	EstablishedDate  *time.Time        `json:"established_date"`
	DissolutionDate  *time.Time        `json:"dissolution_date"`
	Tags             []string          `json:"tags"`
	Metadata         map[string]any    `json:"metadata"`
	ChairName        string            `json:"chair_name"`
	ChairEmail       string            `json:"chair_email"`
	ViceChairName    *string           `json:"vice_chair_name"`
	ViceChairEmail   *string           `json:"vice_chair_email"`
	SecretaryName    *string           `json:"secretary_name"`
	SecretaryEmail   *string           `json:"secretary_email"`
}

func (r *CreateCommitteeRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.MeetingFrequency = strings.TrimSpace(r.MeetingFrequency)
	r.QuorumType = strings.TrimSpace(r.QuorumType)
	r.Type = strings.TrimSpace(r.Type)
}

type UpdateCommitteeRequest struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Description      string         `json:"description"`
	ChairUserID      uuid.UUID      `json:"chair_user_id"`
	ViceChairUserID  *uuid.UUID     `json:"vice_chair_user_id"`
	SecretaryUserID  *uuid.UUID     `json:"secretary_user_id"`
	MeetingFrequency string         `json:"meeting_frequency"`
	QuorumPercentage int            `json:"quorum_percentage"`
	QuorumType       string         `json:"quorum_type"`
	QuorumFixedCount *int           `json:"quorum_fixed_count"`
	Charter          *string        `json:"charter"`
	EstablishedDate  *time.Time     `json:"established_date"`
	DissolutionDate  *time.Time     `json:"dissolution_date"`
	Status           string         `json:"status"`
	Tags             []string       `json:"tags"`
	Metadata         map[string]any `json:"metadata"`
	// Name/email fields accepted from frontend (used for member management).
	ChairName      string  `json:"chair_name"`
	ChairEmail     string  `json:"chair_email"`
	ViceChairName  *string `json:"vice_chair_name"`
	ViceChairEmail *string `json:"vice_chair_email"`
	SecretaryName  *string `json:"secretary_name"`
	SecretaryEmail *string `json:"secretary_email"`
}

func (r *UpdateCommitteeRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.MeetingFrequency = strings.TrimSpace(r.MeetingFrequency)
	r.QuorumType = strings.TrimSpace(r.QuorumType)
	r.Type = strings.TrimSpace(r.Type)
	r.Status = strings.TrimSpace(r.Status)
}

type UpsertCommitteeMemberRequest struct {
	UserID    uuid.UUID `json:"user_id"`
	UserName  string    `json:"user_name"`
	UserEmail string    `json:"user_email"`
	Role      string    `json:"role"`
}

func (r *UpsertCommitteeMemberRequest) Normalize() {
	r.UserName = strings.TrimSpace(r.UserName)
	r.UserEmail = strings.TrimSpace(r.UserEmail)
	r.Role = strings.TrimSpace(r.Role)
}
