package dto

// AIUsageListParams controls AI usage queries.
type AIUsageListParams struct {
	UsageType *string `json:"usage_type,omitempty"`
	RiskLevel *string `json:"risk_level,omitempty"`
	ModelSlug *string `json:"model_slug,omitempty"`
	PIIOnly   *bool   `json:"pii_only,omitempty"`
	Status    *string `json:"status,omitempty"`
	Sort      string  `json:"sort"`
	Order     string  `json:"order"`
	Page      int     `json:"page"`
	PerPage   int     `json:"per_page"`
}

// SetDefaults applies default values to AI usage list params.
func (p *AIUsageListParams) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 25
	}
	if p.Sort == "" {
		p.Sort = "ai_risk_score"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}
