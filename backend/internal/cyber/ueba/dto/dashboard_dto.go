package dto

import "time"

type DashboardKPIs struct {
	ActiveProfiles   int     `json:"active_profiles"`
	HighRiskEntities int     `json:"high_risk_entities"`
	Alerts7D         int     `json:"alerts_7d"`
	AverageRiskScore float64 `json:"average_risk_score"`
}

type ChartDatum struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type TrendDatum struct {
	Bucket    time.Time `json:"bucket"`
	AlertType string    `json:"alert_type"`
	Count     int       `json:"count"`
}

type RiskRankingItem struct {
	EntityID        string    `json:"entity_id"`
	EntityName      string    `json:"entity_name"`
	EntityType      string    `json:"entity_type"`
	RiskScore       float64   `json:"risk_score"`
	RiskLevel       string    `json:"risk_level"`
	AlertCount7D    int       `json:"alert_count_7d"`
	AlertCount30D   int       `json:"alert_count_30d"`
	ProfileMaturity string    `json:"profile_maturity"`
	LastSeenAt      time.Time `json:"last_seen_at"`
	Status          string    `json:"status"`
}

type DashboardResponse struct {
	KPIs                  DashboardKPIs     `json:"kpis"`
	RiskRanking           []RiskRankingItem `json:"risk_ranking"`
	AlertTypeDistribution []ChartDatum      `json:"alert_type_distribution"`
	AlertTrend            []TrendDatum      `json:"alert_trend"`
	Profiles              []RiskRankingItem `json:"profiles"`
}

type UEBAConfigDTO struct {
	CycleInterval       string  `json:"cycle_interval"`
	MaxEventsPerCycle   int     `json:"max_events_per_cycle"`
	MaxProcessingTime   string  `json:"max_processing_time"`
	EMAAlpha            float64 `json:"ema_alpha"`
	MinMaturityForAlert string  `json:"min_maturity_for_alert"`
	CorrelationWindow   string  `json:"correlation_window"`
	RiskDecayRatePerDay float64 `json:"risk_decay_rate_per_day"`
	BatchSize           int     `json:"batch_size"`

	UnusualTimeMatureHighProb   float64 `json:"unusual_time_mature_high_prob"`
	UnusualTimeMatureMediumProb float64 `json:"unusual_time_mature_medium_prob"`
	UnusualTimeBaseHighProb     float64 `json:"unusual_time_base_high_prob"`
	UnusualTimeBaseMediumProb   float64 `json:"unusual_time_base_medium_prob"`

	UnusualVolumeMediumZ   float64 `json:"unusual_volume_medium_z"`
	UnusualVolumeHighZ     float64 `json:"unusual_volume_high_z"`
	UnusualVolumeCriticalZ float64 `json:"unusual_volume_critical_z"`
	UnusualVolumeStddevMin float64 `json:"unusual_volume_stddev_min"`

	FailureSpikeMediumZ   float64 `json:"failure_spike_medium_z"`
	FailureSpikeHighZ     float64 `json:"failure_spike_high_z"`
	FailureSpikeCriticalZ float64 `json:"failure_spike_critical_z"`
	FailureStddevMin      float64 `json:"failure_stddev_min"`
	FailureCriticalCount  float64 `json:"failure_critical_count"`

	BulkRowsMediumMultiplier float64 `json:"bulk_rows_medium_multiplier"`
	BulkRowsHighMultiplier   float64 `json:"bulk_rows_high_multiplier"`
	DDLUnusualThreshold      float64 `json:"ddl_unusual_threshold"`
}
