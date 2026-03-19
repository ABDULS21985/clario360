package dto

import (
	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// OverprivilegeListResponse is returned for the overprivileged mappings endpoint.
type OverprivilegeListResponse struct {
	Data []model.OverprivilegeResult `json:"data"`
}

// StaleAccessListResponse is returned for the stale mappings endpoint.
type StaleAccessListResponse struct {
	Data []model.StaleAccessResult `json:"data"`
}

// BlastRadiusResponse wraps a blast radius calculation result.
type BlastRadiusResponse struct {
	Data *model.BlastRadius `json:"data"`
}

// RiskRankingResponse wraps the risk-ranked identity list.
type RiskRankingResponse struct {
	Data []model.IdentityProfile `json:"data"`
}

// BlastRadiusRankingResponse wraps the blast-radius-ranked identity list.
type BlastRadiusRankingResponse struct {
	Data []model.IdentityProfile `json:"data"`
}

// EscalationPathResponse wraps escalation path results.
type EscalationPathResponse struct {
	Data []model.EscalationPath `json:"data"`
}

// CrossAssetResponse wraps cross-asset analysis results.
type CrossAssetResponse struct {
	Data []model.CrossAssetResult `json:"data"`
}

// AuditListParams are query parameters for listing audit events.
type AuditListParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

func (p *AuditListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 50
	}
}

// AuditListResponse is returned for audit trail endpoints.
type AuditListResponse struct {
	Data []model.AccessAuditEntry `json:"data"`
	Meta PaginationMeta           `json:"meta"`
}
