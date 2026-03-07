package dto

type RiskTrendParams struct {
	Days int `form:"days"`
}

func (p *RiskTrendParams) SetDefaults() {
	if p.Days == 0 {
		p.Days = 90
	}
	if p.Days < 1 {
		p.Days = 1
	}
	if p.Days > 365 {
		p.Days = 365
	}
}
