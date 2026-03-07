package dto

import (
	"strings"

	"github.com/clario360/platform/internal/lex/model"
)

type CreateComplianceRuleRequest struct {
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	RuleType      model.ComplianceRuleType `json:"rule_type"`
	Severity      model.ComplianceSeverity `json:"severity"`
	Config        map[string]any           `json:"config"`
	ContractTypes []string                 `json:"contract_types"`
	Enabled       bool                     `json:"enabled"`
}

type UpdateComplianceRuleRequest struct {
	Name          *string                   `json:"name,omitempty"`
	Description   *string                   `json:"description,omitempty"`
	RuleType      *model.ComplianceRuleType `json:"rule_type,omitempty"`
	Severity      *model.ComplianceSeverity `json:"severity,omitempty"`
	Config        map[string]any            `json:"config,omitempty"`
	ContractTypes []string                  `json:"contract_types,omitempty"`
	Enabled       *bool                     `json:"enabled,omitempty"`
}

type RunComplianceRequest struct {
	ContractIDs []string `json:"contract_ids,omitempty"`
}

type UpdateAlertStatusRequest struct {
	Status          model.ComplianceAlertStatus `json:"status"`
	ResolutionNotes string                      `json:"resolution_notes"`
}

func (r *CreateComplianceRuleRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.Config == nil {
		r.Config = map[string]any{}
	}
}

func (r *UpdateAlertStatusRequest) Normalize() {
	r.ResolutionNotes = strings.TrimSpace(r.ResolutionNotes)
}
