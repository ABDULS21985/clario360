package dto

import (
	"encoding/json"
	"fmt"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// CreatePolicyRequest is the body for creating a new access policy.
type CreatePolicyRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	PolicyType  string          `json:"policy_type"`
	RuleConfig  json.RawMessage `json:"rule_config"`
	Enforcement string          `json:"enforcement"`
	Severity    string          `json:"severity"`
	Enabled     bool            `json:"enabled"`
}

func (r *CreatePolicyRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	validTypes := map[string]bool{
		"max_idle_days":           true,
		"classification_restrict": true,
		"separation_of_duties":    true,
		"time_bound_access":       true,
		"blast_radius_limit":      true,
		"periodic_review":         true,
	}
	if !validTypes[r.PolicyType] {
		return fmt.Errorf("invalid policy_type: %s", r.PolicyType)
	}
	if len(r.RuleConfig) == 0 {
		return fmt.Errorf("rule_config is required")
	}
	validEnforcements := map[string]bool{
		"alert":          true,
		"block":          true,
		"auto_remediate": true,
	}
	if r.Enforcement == "" {
		r.Enforcement = "alert"
	}
	if !validEnforcements[r.Enforcement] {
		return fmt.Errorf("invalid enforcement: %s", r.Enforcement)
	}
	validSeverities := map[string]bool{
		"low": true, "medium": true, "high": true, "critical": true,
	}
	if r.Severity == "" {
		r.Severity = "medium"
	}
	if !validSeverities[r.Severity] {
		return fmt.Errorf("invalid severity: %s", r.Severity)
	}
	return nil
}

// UpdatePolicyRequest is the body for updating an access policy.
type UpdatePolicyRequest struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	RuleConfig  *json.RawMessage `json:"rule_config,omitempty"`
	Enforcement *string          `json:"enforcement,omitempty"`
	Severity    *string          `json:"severity,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"`
}

func (r *UpdatePolicyRequest) Validate() error {
	if r.Enforcement != nil {
		validEnforcements := map[string]bool{
			"alert": true, "block": true, "auto_remediate": true,
		}
		if !validEnforcements[*r.Enforcement] {
			return fmt.Errorf("invalid enforcement: %s", *r.Enforcement)
		}
	}
	if r.Severity != nil {
		validSeverities := map[string]bool{
			"low": true, "medium": true, "high": true, "critical": true,
		}
		if !validSeverities[*r.Severity] {
			return fmt.Errorf("invalid severity: %s", *r.Severity)
		}
	}
	return nil
}

// PolicyListResponse wraps a list of access policies.
type PolicyListResponse struct {
	Data []model.AccessPolicy `json:"data"`
}

// PolicyViolationListResponse wraps a list of policy violations.
type PolicyViolationListResponse struct {
	Data  []model.PolicyViolation `json:"data"`
	Total int                     `json:"total"`
}

// RecommendationListResponse wraps a list of recommendations.
type RecommendationListResponse struct {
	Data []model.Recommendation `json:"data"`
}
