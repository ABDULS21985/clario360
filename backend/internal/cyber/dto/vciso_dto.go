package dto

import "fmt"

// VCISOBriefingParams are query parameters for generating an executive briefing.
type VCISOBriefingParams struct {
	PeriodDays int `json:"period_days"`
}

func (p *VCISOBriefingParams) SetDefaults() {
	if p.PeriodDays <= 0 {
		p.PeriodDays = 30
	}
	if p.PeriodDays > 365 {
		p.PeriodDays = 365
	}
}

// VCISOReportRequest is the request body for generating an on-demand report.
type VCISOReportRequest struct {
	Type       string `json:"type"`
	PeriodDays int    `json:"period_days"`
}

func (r *VCISOReportRequest) Validate() error {
	validTypes := map[string]bool{
		"executive": true, "technical": true, "compliance": true, "custom": true,
	}
	if !validTypes[r.Type] {
		return fmt.Errorf("invalid type: must be executive, technical, compliance, or custom")
	}
	if r.PeriodDays <= 0 {
		r.PeriodDays = 30
	}
	if r.PeriodDays > 365 {
		return fmt.Errorf("period_days must be <= 365")
	}
	return nil
}

// VCISOBriefingHistoryParams are query parameters for listing briefing history.
type VCISOBriefingHistoryParams struct {
	Type    *string `json:"type"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
}

func (p *VCISOBriefingHistoryParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 100 {
		p.PerPage = 20
	}
}
