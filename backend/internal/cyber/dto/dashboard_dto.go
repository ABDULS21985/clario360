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

// DashboardMetricsResponse is the aggregated metrics strip for the main dashboard.
type DashboardMetricsResponse struct {
	MTTRMinutes      *float64 `json:"mttr_minutes"`
	MTTAMinutes      *float64 `json:"mtta_minutes"`
	SLACompliancePct *float64 `json:"sla_compliance_pct"`
	ActiveIncidents  *int     `json:"active_incidents"`
	ActiveUsersToday *int     `json:"active_users_today"`
	PendingReviews   *int     `json:"pending_reviews"`
}
