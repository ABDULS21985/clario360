package dto

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// CreateRemediationRequest is the request body for creating a remediation action.
type CreateRemediationRequest struct {
	AlertID            *uuid.UUID              `json:"alert_id"`
	VulnerabilityID    *uuid.UUID              `json:"vulnerability_id"`
	AssessmentID       *uuid.UUID              `json:"assessment_id"`
	CTEMFindingID      *uuid.UUID              `json:"ctem_finding_id"`
	RemediationGroupID *uuid.UUID              `json:"remediation_group_id"`
	Type               string                  `json:"type"`
	Severity           string                  `json:"severity"`
	Title              string                  `json:"title"`
	Description        string                  `json:"description"`
	Plan               model.RemediationPlan   `json:"plan"`
	AffectedAssetIDs   []uuid.UUID             `json:"affected_asset_ids"`
	ExecutionMode      string                  `json:"execution_mode"`
	RequiresApprovalFrom string               `json:"requires_approval_from"`
	Tags               []string                `json:"tags"`
	Metadata           map[string]interface{}  `json:"metadata"`
}

func (r *CreateRemediationRequest) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("type is required")
	}
	validTypes := map[string]bool{
		"patch": true, "config_change": true, "block_ip": true, "isolate_asset": true,
		"firewall_rule": true, "access_revoke": true, "certificate_renew": true, "custom": true,
	}
	if !validTypes[r.Type] {
		return fmt.Errorf("invalid type: %s", r.Type)
	}
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(r.Plan.Steps) == 0 {
		return fmt.Errorf("plan must have at least one step")
	}
	validSeverities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
	if r.Severity != "" && !validSeverities[r.Severity] {
		return fmt.Errorf("invalid severity: %s", r.Severity)
	}
	return nil
}

// UpdateRemediationRequest allows updating a draft or revision_requested remediation.
type UpdateRemediationRequest struct {
	Title              *string                `json:"title"`
	Description        *string                `json:"description"`
	Plan               *model.RemediationPlan `json:"plan"`
	AffectedAssetIDs   []uuid.UUID            `json:"affected_asset_ids"`
	Severity           *string                `json:"severity"`
	Tags               []string               `json:"tags"`
	Metadata           map[string]interface{} `json:"metadata"`
}

func (r *UpdateRemediationRequest) Validate() error {
	if r.Plan != nil && len(r.Plan.Steps) == 0 {
		return fmt.Errorf("plan must have at least one step")
	}
	if r.Severity != nil {
		validSeverities := map[string]bool{"critical": true, "high": true, "medium": true, "low": true}
		if !validSeverities[*r.Severity] {
			return fmt.Errorf("invalid severity: %s", *r.Severity)
		}
	}
	return nil
}

// ApproveRemediationRequest carries approval notes.
type ApproveRemediationRequest struct {
	Notes string `json:"notes"`
}

// RejectRemediationRequest carries the rejection reason.
type RejectRemediationRequest struct {
	Reason string `json:"reason"`
}

func (r *RejectRemediationRequest) Validate() error {
	if r.Reason == "" {
		return fmt.Errorf("rejection reason is required")
	}
	return nil
}

// RequestRevisionRequest carries the revision guidance.
type RequestRevisionRequest struct {
	Notes string `json:"notes"`
}

func (r *RequestRevisionRequest) Validate() error {
	if r.Notes == "" {
		return fmt.Errorf("revision notes are required")
	}
	return nil
}

// RollbackRequest carries the rollback reason and approver.
type RollbackRequest struct {
	Reason string `json:"reason"`
}

func (r *RollbackRequest) Validate() error {
	if r.Reason == "" {
		return fmt.Errorf("rollback reason is required")
	}
	return nil
}

// ExecuteRemediationRequest optionally confirms manual/custom execution.
type ExecuteRemediationRequest struct {
	ManualConfirmation *string `json:"manual_confirmation,omitempty"`
}

// VerifyRemediationRequest optionally confirms manual/custom verification.
type VerifyRemediationRequest struct {
	ManualConfirmation *string `json:"manual_confirmation,omitempty"`
}

// RemediationListParams are query parameters for listing remediations.
type RemediationListParams struct {
	Statuses   []string   `json:"statuses"`
	Types      []string   `json:"types"`
	Severities []string   `json:"severities"`
	AssetID    *uuid.UUID `json:"asset_id"`
	AlertID    *uuid.UUID `json:"alert_id"`
	VulnID     *uuid.UUID `json:"vulnerability_id"`
	Search     *string    `json:"search"`
	Tags       []string   `json:"tags"`
	Sort       string     `json:"sort"`
	Order      string     `json:"order"`
	Page       int        `json:"page"`
	PerPage    int        `json:"per_page"`
}

func (p *RemediationListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 50
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

func (p *RemediationListParams) Validate() error {
	validSorts := map[string]bool{
		"created_at": true,
		"updated_at": true,
		"status":     true,
		"severity":   true,
		"type":       true,
		"title":      true,
	}
	if p.Sort != "" && !validSorts[p.Sort] {
		return fmt.Errorf("invalid sort field: %s", p.Sort)
	}
	if p.Order != "" && p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order: %s", p.Order)
	}
	if p.Page < 1 {
		return fmt.Errorf("page must be at least 1")
	}
	if p.PerPage < 1 || p.PerPage > 200 {
		return fmt.Errorf("per_page must be between 1 and 200")
	}
	return nil
}

type RemediationListResponse struct {
	Data       []*model.RemediationAction `json:"data"`
	Total      int                        `json:"total"`
	Page       int                        `json:"page"`
	PerPage    int                        `json:"per_page"`
	TotalPages int                        `json:"total_pages"`
}

// ConfirmManualExecutionRequest is for custom strategy manual confirmation.
type ConfirmManualExecutionRequest struct {
	Confirmation string `json:"confirmation"` // must equal "I have manually performed the remediation steps."
}

func (r *ConfirmManualExecutionRequest) Validate() error {
	const required = "I have manually performed the remediation steps."
	if r.Confirmation != required {
		return fmt.Errorf("confirmation must equal: %q", required)
	}
	return nil
}
