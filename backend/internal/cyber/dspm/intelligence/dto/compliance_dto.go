package dto

// CompliancePostureParams controls compliance posture queries.
type CompliancePostureParams struct {
	Framework *string `json:"framework,omitempty"`
}

// ComplianceGapParams controls gap analysis queries.
type ComplianceGapParams struct {
	Framework *string `json:"framework,omitempty"`
	Severity  *string `json:"severity,omitempty"`
	Page      int     `json:"page"`
	PerPage   int     `json:"per_page"`
}

// SetDefaults applies default values to compliance gap params.
func (p *ComplianceGapParams) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 25
	}
}
