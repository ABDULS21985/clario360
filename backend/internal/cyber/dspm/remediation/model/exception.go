package model

import (
	"time"

	"github.com/google/uuid"
)

// ExceptionType enumerates the types of risk exceptions.
type ExceptionType string

const (
	ExceptionPostureFinding      ExceptionType = "posture_finding"
	ExceptionPolicyViolation     ExceptionType = "policy_violation"
	ExceptionOverprivilegedAccess ExceptionType = "overprivileged_access"
	ExceptionExposureRisk        ExceptionType = "exposure_risk"
	ExceptionEncryptionGap       ExceptionType = "encryption_gap"
)

// ValidExceptionTypes returns all valid exception types.
func ValidExceptionTypes() []ExceptionType {
	return []ExceptionType{
		ExceptionPostureFinding, ExceptionPolicyViolation,
		ExceptionOverprivilegedAccess, ExceptionExposureRisk,
		ExceptionEncryptionGap,
	}
}

// IsValid returns true if the exception type is a known value.
func (e ExceptionType) IsValid() bool {
	for _, v := range ValidExceptionTypes() {
		if e == v {
			return true
		}
	}
	return false
}

// ApprovalStatus tracks exception approval workflow.
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
	ApprovalExpired  ApprovalStatus = "expired"
)

// ExceptionStatus tracks the lifecycle of a risk exception.
type ExceptionStatus string

const (
	ExceptionStatusActive     ExceptionStatus = "active"
	ExceptionStatusExpired    ExceptionStatus = "expired"
	ExceptionStatusRevoked    ExceptionStatus = "revoked"
	ExceptionStatusSuperseded ExceptionStatus = "superseded"
)

// RiskException represents an accepted risk with governance controls.
type RiskException struct {
	ID                    uuid.UUID      `json:"id" db:"id"`
	TenantID              uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	ExceptionType         ExceptionType  `json:"exception_type" db:"exception_type"`
	RemediationID         *uuid.UUID     `json:"remediation_id,omitempty" db:"remediation_id"`
	DataAssetID           *uuid.UUID     `json:"data_asset_id,omitempty" db:"data_asset_id"`
	PolicyID              *uuid.UUID     `json:"policy_id,omitempty" db:"policy_id"`
	Justification         string         `json:"justification" db:"justification"`
	BusinessReason        string         `json:"business_reason,omitempty" db:"business_reason"`
	CompensatingControls  string         `json:"compensating_controls,omitempty" db:"compensating_controls"`
	RiskScore             float64        `json:"risk_score" db:"risk_score"`
	RiskLevel             string         `json:"risk_level" db:"risk_level"`
	RequestedBy           uuid.UUID      `json:"requested_by" db:"requested_by"`
	ApprovedBy            *uuid.UUID     `json:"approved_by,omitempty" db:"approved_by"`
	ApprovalStatus        ApprovalStatus `json:"approval_status" db:"approval_status"`
	ApprovedAt            *time.Time     `json:"approved_at,omitempty" db:"approved_at"`
	RejectionReason       string         `json:"rejection_reason,omitempty" db:"rejection_reason"`
	ExpiresAt             time.Time      `json:"expires_at" db:"expires_at"`
	ReviewIntervalDays    int            `json:"review_interval_days" db:"review_interval_days"`
	NextReviewAt          *time.Time     `json:"next_review_at,omitempty" db:"next_review_at"`
	LastReviewedAt        *time.Time     `json:"last_reviewed_at,omitempty" db:"last_reviewed_at"`
	ReviewCount           int            `json:"review_count" db:"review_count"`
	Status                ExceptionStatus `json:"status" db:"status"`
	CreatedAt             time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at" db:"updated_at"`
}
