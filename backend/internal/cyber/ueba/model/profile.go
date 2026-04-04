package model

import (
	"time"

	"github.com/google/uuid"
)

type EntityType string

const (
	EntityTypeUser           EntityType = "user"
	EntityTypeServiceAccount EntityType = "service_account"
	EntityTypeApplication    EntityType = "application"
	EntityTypeAPIKey         EntityType = "api_key"
)

type ProfileMaturity string

const (
	ProfileMaturityLearning ProfileMaturity = "learning"
	ProfileMaturityBaseline ProfileMaturity = "baseline"
	ProfileMaturityMature   ProfileMaturity = "mature"
)

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type ProfileStatus string

const (
	ProfileStatusActive      ProfileStatus = "active"
	ProfileStatusInactive    ProfileStatus = "inactive"
	ProfileStatusSuppressed  ProfileStatus = "suppressed"
	ProfileStatusWhitelisted ProfileStatus = "whitelisted"
)

type UEBAProfile struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	TenantID         uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EntityType       EntityType      `json:"entity_type" db:"entity_type"`
	EntityID         string          `json:"entity_id" db:"entity_id"`
	EntityName       string          `json:"entity_name,omitempty" db:"entity_name"`
	EntityEmail      string          `json:"entity_email,omitempty" db:"entity_email"`
	Baseline         Baseline        `json:"baseline" db:"baseline"`
	ObservationCount int64           `json:"observation_count" db:"observation_count"`
	ProfileMaturity  ProfileMaturity `json:"profile_maturity" db:"profile_maturity"`
	FirstSeenAt      time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt       time.Time       `json:"last_seen_at" db:"last_seen_at"`
	DaysActive       int             `json:"days_active" db:"days_active"`
	RiskScore        float64         `json:"risk_score" db:"risk_score"`
	RiskLevel        RiskLevel       `json:"risk_level" db:"risk_level"`
	RiskFactors      []RiskFactor    `json:"risk_factors" db:"risk_factors"`
	RiskLastUpdated  *time.Time      `json:"risk_last_updated,omitempty" db:"risk_last_updated"`
	RiskLastDecayed  *time.Time      `json:"risk_last_decayed,omitempty" db:"risk_last_decayed"`
	AlertCount7D     int             `json:"alert_count_7d" db:"alert_count_7d"`
	AlertCount30D    int             `json:"alert_count_30d" db:"alert_count_30d"`
	LastAlertAt      *time.Time      `json:"last_alert_at,omitempty" db:"last_alert_at"`
	Status           ProfileStatus   `json:"status" db:"status"`
	SuppressedUntil  *time.Time      `json:"suppressed_until,omitempty" db:"suppressed_until"`
	SuppressedReason string          `json:"suppressed_reason,omitempty" db:"suppressed_reason"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

type Baseline struct {
	AccessTimes    AccessTimeBaseline    `json:"access_times"`
	DataVolume     DataVolumeBaseline    `json:"data_volume"`
	AccessPatterns AccessPatternBaseline `json:"access_patterns"`
	SourceIPs      []string              `json:"source_ips"`
	SessionStats   SessionStatsBaseline  `json:"session_stats"`
	FailureRate    FailureRateBaseline   `json:"failure_rate"`
	State          BaselineState         `json:"state,omitempty"`
}

type AccessTimeBaseline struct {
	HourlyDistribution [24]float64 `json:"hourly_distribution"`
	DailyDistribution  [7]float64  `json:"daily_distribution"`
	PeakHours          []int       `json:"peak_hours"`
	ActiveHoursCount   int         `json:"active_hours_count"`
}

type DataVolumeBaseline struct {
	DailyBytesMean      float64 `json:"daily_bytes_mean"`
	DailyBytesStddev    float64 `json:"daily_bytes_stddev"`
	DailyBytesM2        float64 `json:"daily_bytes_m2"`
	DailyRowsMean       float64 `json:"daily_rows_mean"`
	DailyRowsStddev     float64 `json:"daily_rows_stddev"`
	DailyRowsM2         float64 `json:"daily_rows_m2"`
	MaxSingleQueryBytes float64 `json:"max_single_query_bytes"`
	MaxSingleQueryRows  float64 `json:"max_single_query_rows"`
}

type FrequencyEntry struct {
	Name         string    `json:"name"`
	Frequency    float64   `json:"frequency"`
	LastAccessed time.Time `json:"last_accessed"`
}

type AccessPatternBaseline struct {
	DatabasesAccessed      []string           `json:"databases_accessed"`
	TablesAccessed         []FrequencyEntry   `json:"tables_accessed"`
	QueryTypes             map[string]float64 `json:"query_types"`
	AvgQueryDurationMS     float64            `json:"avg_query_duration_ms"`
	AvgQueryDurationStddev float64            `json:"avg_query_duration_stddev"`
	AvgQueryDurationM2     float64            `json:"avg_query_duration_m2,omitempty"`
}

type SessionStatsBaseline struct {
	DailySessionCountMean     float64 `json:"daily_session_count_mean"`
	DailySessionCountStddev   float64 `json:"daily_session_count_stddev"`
	DailySessionCountM2       float64 `json:"daily_session_count_m2,omitempty"`
	AvgSessionDurationMinutes float64 `json:"avg_session_duration_minutes"`
	AvgSessionDurationStddev  float64 `json:"avg_session_duration_stddev,omitempty"`
	AvgSessionDurationM2      float64 `json:"avg_session_duration_m2,omitempty"`
}

type FailureRateBaseline struct {
	DailyFailureCountMean   float64 `json:"daily_failure_count_mean"`
	DailyFailureCountStddev float64 `json:"daily_failure_count_stddev"`
	DailyFailureCountM2     float64 `json:"daily_failure_count_m2,omitempty"`
	FailureRatePercent      float64 `json:"failure_rate_percent"`
}

// BaselineState stores bounded daily roll-up state needed to turn event streams
// into daily aggregates without allowing the baseline document to grow.
type BaselineState struct {
	CurrentDay              string  `json:"current_day,omitempty"`
	CurrentDayBytes         float64 `json:"current_day_bytes,omitempty"`
	CurrentDayRows          float64 `json:"current_day_rows,omitempty"`
	CurrentDayFailures      float64 `json:"current_day_failures,omitempty"`
	CurrentDayQueries       float64 `json:"current_day_queries,omitempty"`
	CurrentDaySessions      float64 `json:"current_day_sessions,omitempty"`
	CurrentSessionStartUnix int64   `json:"current_session_start_unix,omitempty"`
	LastActiveDay           string  `json:"last_active_day,omitempty"`
}

type RiskFactor struct {
	AlertID     uuid.UUID `json:"alert_id"`
	AlertType   string    `json:"alert_type"`
	Severity    string    `json:"severity"`
	Confidence  float64   `json:"confidence"`
	Impact      float64   `json:"impact"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	SignalTypes []string  `json:"signal_types"`
	EventCount  int       `json:"event_count"`
}

func (p *UEBAProfile) EnsureDefaults() {
	if p.Status == "" {
		p.Status = ProfileStatusActive
	}
	if p.ProfileMaturity == "" {
		p.ProfileMaturity = ProfileMaturityLearning
	}
	if p.RiskLevel == "" {
		p.RiskLevel = RiskLevelLow
	}
	if p.Baseline.AccessPatterns.QueryTypes == nil {
		p.Baseline.AccessPatterns.QueryTypes = map[string]float64{
			"select": 0,
			"insert": 0,
			"update": 0,
			"delete": 0,
			"ddl":    0,
		}
	}
	if p.Baseline.AccessPatterns.DatabasesAccessed == nil {
		p.Baseline.AccessPatterns.DatabasesAccessed = []string{}
	}
	if p.Baseline.AccessPatterns.TablesAccessed == nil {
		p.Baseline.AccessPatterns.TablesAccessed = []FrequencyEntry{}
	}
	if p.Baseline.AccessTimes.PeakHours == nil {
		p.Baseline.AccessTimes.PeakHours = []int{}
	}
	if p.Baseline.SourceIPs == nil {
		p.Baseline.SourceIPs = []string{}
	}
	if p.RiskFactors == nil {
		p.RiskFactors = []RiskFactor{}
	}
}
