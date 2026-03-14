package dto

import (
	"encoding/json"
	"fmt"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// CreatePolicyRequest is the input for creating a new data policy.
type CreatePolicyRequest struct {
	Name                 string          `json:"name"`
	Description          string          `json:"description,omitempty"`
	Category             string          `json:"category"`
	Rule                 json.RawMessage `json:"rule"`
	Enforcement          string          `json:"enforcement"`
	AutoPlaybookID       string          `json:"auto_playbook_id,omitempty"`
	Severity             string          `json:"severity"`
	ScopeClassification  []string        `json:"scope_classification,omitempty"`
	ScopeAssetTypes      []string        `json:"scope_asset_types,omitempty"`
	ComplianceFrameworks []string        `json:"compliance_frameworks,omitempty"`
}

// Validate validates the create policy request.
func (r *CreatePolicyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	cat := model.PolicyCategory(r.Category)
	if !cat.IsValid() {
		return fmt.Errorf("invalid category: %s", r.Category)
	}
	if len(r.Rule) == 0 {
		return fmt.Errorf("rule is required")
	}
	enf := model.PolicyEnforcement(r.Enforcement)
	if !enf.IsValid() {
		return fmt.Errorf("invalid enforcement: %s", r.Enforcement)
	}
	switch r.Severity {
	case "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("invalid severity: %s", r.Severity)
	}
	if r.Enforcement == string(model.EnforcementAutoRemediate) && r.AutoPlaybookID == "" {
		return fmt.Errorf("auto_playbook_id is required when enforcement is auto_remediate")
	}
	return nil
}

// UpdatePolicyRequest is the input for updating a data policy.
type UpdatePolicyRequest struct {
	Name                 *string          `json:"name,omitempty"`
	Description          *string          `json:"description,omitempty"`
	Rule                 json.RawMessage  `json:"rule,omitempty"`
	Enforcement          *string          `json:"enforcement,omitempty"`
	AutoPlaybookID       *string          `json:"auto_playbook_id,omitempty"`
	Severity             *string          `json:"severity,omitempty"`
	ScopeClassification  []string         `json:"scope_classification,omitempty"`
	ScopeAssetTypes      []string         `json:"scope_asset_types,omitempty"`
	Enabled              *bool            `json:"enabled,omitempty"`
	ComplianceFrameworks []string         `json:"compliance_frameworks,omitempty"`
}

// Validate validates the update policy request.
func (r *UpdatePolicyRequest) Validate() error {
	if r.Enforcement != nil {
		enf := model.PolicyEnforcement(*r.Enforcement)
		if !enf.IsValid() {
			return fmt.Errorf("invalid enforcement: %s", *r.Enforcement)
		}
	}
	if r.Severity != nil {
		switch *r.Severity {
		case "low", "medium", "high", "critical":
		default:
			return fmt.Errorf("invalid severity: %s", *r.Severity)
		}
	}
	return nil
}

// PolicyListParams defines filtering and pagination for listing policies.
type PolicyListParams struct {
	Category    string `json:"category,omitempty"`
	Enforcement string `json:"enforcement,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Search      string `json:"search,omitempty"`
	Page        int    `json:"page"`
	PerPage     int    `json:"per_page"`
}

// SetDefaults applies default pagination values.
func (p *PolicyListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 25
	}
}

// PolicyListResponse is the paginated response for listing policies.
type PolicyListResponse struct {
	Data  []model.DataPolicy `json:"data"`
	Total int                `json:"total"`
}

// PolicyViolationListResponse wraps a list of policy violations.
type PolicyViolationListResponse struct {
	Data  []model.PolicyViolation `json:"data"`
	Total int                     `json:"total"`
}

// PolicyImpactResponse wraps a policy dry-run impact analysis.
type PolicyImpactResponse struct {
	Data model.PolicyImpact `json:"data"`
}
