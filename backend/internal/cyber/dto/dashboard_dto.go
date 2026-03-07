package dto

import "github.com/clario360/platform/internal/cyber/model"

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

type DashboardTrendsResponse struct {
	Days        int                 `json:"days"`
	AlertTrend  []model.DailyMetric `json:"alert_trend"`
	VulnTrend   []model.DailyMetric `json:"vulnerability_trend"`
	ThreatTrend []model.DailyMetric `json:"threat_trend"`
}
