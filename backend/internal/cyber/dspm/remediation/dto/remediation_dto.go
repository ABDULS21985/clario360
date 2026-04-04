package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// CreateRemediationRequest is the input for creating a new remediation item.
type CreateRemediationRequest struct {
	FindingType   string     `json:"finding_type"`
	FindingID     *uuid.UUID `json:"finding_id,omitempty"`
	DataAssetID   *uuid.UUID `json:"data_asset_id,omitempty"`
	DataAssetName string     `json:"data_asset_name,omitempty"`
	IdentityID    string     `json:"identity_id,omitempty"`
	PlaybookID    string     `json:"playbook_id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Severity      string     `json:"severity"`
	AssignedTo    *uuid.UUID `json:"assigned_to,omitempty"`
	AssignedTeam  string     `json:"assigned_team,omitempty"`
	ComplianceTags []string  `json:"compliance_tags,omitempty"`
}

// Validate validates the create remediation request.
func (r *CreateRemediationRequest) Validate() error {
	if r.Title == "" {
		return fmt.Errorf("title is required")
	}
	if r.Description == "" {
		return fmt.Errorf("description is required")
	}
	if r.PlaybookID == "" {
		return fmt.Errorf("playbook_id is required")
	}
	ft := model.FindingType(r.FindingType)
	if !ft.IsValid() {
		return fmt.Errorf("invalid finding_type: %s", r.FindingType)
	}
	switch r.Severity {
	case "low", "medium", "high", "critical":
	default:
		return fmt.Errorf("invalid severity: %s", r.Severity)
	}
	return nil
}

// AssignRemediationRequest is the input for assigning a remediation.
type AssignRemediationRequest struct {
	AssignedTo   *uuid.UUID `json:"assigned_to,omitempty"`
	AssignedTeam string     `json:"assigned_team,omitempty"`
}

// Validate validates the assign request.
func (r *AssignRemediationRequest) Validate() error {
	if r.AssignedTo == nil && r.AssignedTeam == "" {
		return fmt.Errorf("either assigned_to or assigned_team is required")
	}
	return nil
}

// RemediationListParams defines filtering and pagination for listing remediations.
type RemediationListParams struct {
	Status      []string   `json:"status,omitempty"`
	Severity    []string   `json:"severity,omitempty"`
	FindingType []string   `json:"finding_type,omitempty"`
	AssignedTo  *uuid.UUID `json:"assigned_to,omitempty"`
	AssetID     *uuid.UUID `json:"asset_id,omitempty"`
	SLABreached *bool      `json:"sla_breached,omitempty"`
	Search      string     `json:"search,omitempty"`
	Sort        string     `json:"sort,omitempty"`
	Order       string     `json:"order,omitempty"`
	Page        int        `json:"page"`
	PerPage     int        `json:"per_page"`
}

// SetDefaults applies default pagination values.
func (p *RemediationListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 25
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

// RemediationListResponse is the paginated response for listing remediations.
type RemediationListResponse struct {
	Data  []model.Remediation `json:"data"`
	Total int                 `json:"total"`
	Page  int                 `json:"page"`
	Limit int                 `json:"limit"`
}

// RemediationDetailResponse wraps a single remediation with its history.
type RemediationDetailResponse struct {
	Remediation model.Remediation        `json:"remediation"`
	History     []model.RemediationHistory `json:"history,omitempty"`
}

// RemediationStatsResponse wraps remediation statistics.
type RemediationStatsResponse struct {
	Data model.RemediationStats `json:"data"`
}

// RemediationDashboardResponse wraps the full remediation dashboard.
type RemediationDashboardResponse struct {
	Data model.RemediationDashboard `json:"data"`
}

// HistoryListParams defines pagination for listing remediation history entries.
type HistoryListParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// SetDefaults applies default values to history list params.
func (p *HistoryListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 50
	}
}

// HistoryListResponse wraps a list of remediation history entries.
type HistoryListResponse struct {
	Data  []model.RemediationHistory `json:"data"`
	Total int                        `json:"total"`
}

// DryRunResponse wraps a dry-run result.
type DryRunResponse struct {
	Data model.DryRunResult `json:"data"`
}

// SLADueAt calculates the SLA deadline from creation time and severity.
func SLADueAt(createdAt time.Time, severity string) time.Time {
	cfg := model.DefaultSLAConfig()
	hours := cfg.SLAHoursForSeverity(severity)
	return createdAt.Add(time.Duration(hours) * time.Hour)
}
