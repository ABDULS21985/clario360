package dto

type DashboardTrendParams struct {
	Days int `form:"days"`
}

func (p *DashboardTrendParams) SetDefaults() {
	if p.Days == 0 {
		p.Days = 30
	}
	if p.Days < 1 {
		p.Days = 1
	}
	if p.Days > 365 {
		p.Days = 365
	}
}
