package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// CreateExceptionRequest is the input for requesting a risk exception.
type CreateExceptionRequest struct {
	ExceptionType        string     `json:"exception_type"`
	RemediationID        *uuid.UUID `json:"remediation_id,omitempty"`
	DataAssetID          *uuid.UUID `json:"data_asset_id,omitempty"`
	PolicyID             *uuid.UUID `json:"policy_id,omitempty"`
	Justification        string     `json:"justification"`
	BusinessReason       string     `json:"business_reason,omitempty"`
	CompensatingControls string     `json:"compensating_controls,omitempty"`
	RiskScore            float64    `json:"risk_score"`
	RiskLevel            string     `json:"risk_level"`
	ExpiresAt            time.Time  `json:"expires_at"`
	ReviewIntervalDays   int        `json:"review_interval_days"`
}

// Validate validates the create exception request.
func (r *CreateExceptionRequest) Validate() error {
	et := model.ExceptionType(r.ExceptionType)
	if !et.IsValid() {
		return fmt.Errorf("invalid exception_type: %s", r.ExceptionType)
	}
	if r.Justification == "" {
		return fmt.Errorf("justification is required")
	}
	if r.ExpiresAt.Before(time.Now()) {
		return fmt.Errorf("expires_at must be in the future")
	}
	maxExpiry := time.Now().AddDate(0, 0, 365)
	if r.ExpiresAt.After(maxExpiry) {
		return fmt.Errorf("expires_at cannot be more than 365 days in the future")
	}
	switch r.RiskLevel {
	case "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("invalid risk_level: %s", r.RiskLevel)
	}
	if r.RiskScore < 0 || r.RiskScore > 100 {
		return fmt.Errorf("risk_score must be between 0 and 100")
	}
	if r.ReviewIntervalDays <= 0 {
		r.ReviewIntervalDays = 90
	}
	if r.ReviewIntervalDays > 365 {
		return fmt.Errorf("review_interval_days cannot exceed 365")
	}
	return nil
}

// RejectExceptionRequest is the input for rejecting a risk exception.
type RejectExceptionRequest struct {
	Reason string `json:"reason"`
}

// Validate validates the reject request.
func (r *RejectExceptionRequest) Validate() error {
	if r.Reason == "" {
		return fmt.Errorf("rejection reason is required")
	}
	return nil
}

// ExceptionListParams defines filtering and pagination for listing exceptions.
type ExceptionListParams struct {
	Status         []string   `json:"status,omitempty"`
	ApprovalStatus []string   `json:"approval_status,omitempty"`
	ExceptionType  []string   `json:"exception_type,omitempty"`
	AssetID        *uuid.UUID `json:"asset_id,omitempty"`
	Search         string     `json:"search,omitempty"`
	Page           int        `json:"page"`
	PerPage        int        `json:"per_page"`
}

// SetDefaults applies default pagination values.
func (p *ExceptionListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 25
	}
}

// ExceptionListResponse is the paginated response for listing exceptions.
type ExceptionListResponse struct {
	Data  []model.RiskException `json:"data"`
	Total int                   `json:"total"`
}
