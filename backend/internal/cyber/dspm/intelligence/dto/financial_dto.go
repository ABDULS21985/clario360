package dto

// FinancialImpactListParams controls financial impact queries.
type FinancialImpactListParams struct {
	MinBreachCost *float64 `json:"min_breach_cost,omitempty"`
	Regulation    *string  `json:"regulation,omitempty"`
	Sort          string   `json:"sort"`
	Order         string   `json:"order"`
	Page          int      `json:"page"`
	PerPage       int      `json:"per_page"`
}

// SetDefaults applies default values to financial impact list params.
func (p *FinancialImpactListParams) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 25
	}
	if p.Sort == "" {
		p.Sort = "annual_expected_loss"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}
