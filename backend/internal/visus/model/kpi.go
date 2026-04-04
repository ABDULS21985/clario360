package model

import (
	"time"

	"github.com/google/uuid"
)

type KPICategory string
type KPISuite string
type KPIUnit string
type KPIDirection string
type KPICalculationType string
type KPISnapshotFrequency string
type KPIStatus string

const (
	KPICategorySecurity   KPICategory = "security"
	KPICategoryData       KPICategory = "data"
	KPICategoryGovernance KPICategory = "governance"
	KPICategoryLegal      KPICategory = "legal"
	KPICategoryOperations KPICategory = "operations"
	KPICategoryGeneral    KPICategory = "general"
)

const (
	KPISuiteCyber    KPISuite = "cyber"
	KPISuiteData     KPISuite = "data"
	KPISuiteActa     KPISuite = "acta"
	KPISuiteLex      KPISuite = "lex"
	KPISuitePlatform KPISuite = "platform"
	KPISuiteCustom   KPISuite = "custom"
)

const (
	KPIUnitCount      KPIUnit = "count"
	KPIUnitPercentage KPIUnit = "percentage"
	KPIUnitHours      KPIUnit = "hours"
	KPIUnitMinutes    KPIUnit = "minutes"
	KPIUnitScore      KPIUnit = "score"
	KPIUnitCurrency   KPIUnit = "currency"
	KPIUnitRatio      KPIUnit = "ratio"
	KPIUnitBytes      KPIUnit = "bytes"
)

const (
	KPIDirectionHigherIsBetter KPIDirection = "higher_is_better"
	KPIDirectionLowerIsBetter  KPIDirection = "lower_is_better"
)

const (
	KPICalcDirect            KPICalculationType = "direct"
	KPICalcDelta             KPICalculationType = "delta"
	KPICalcPercentageChange  KPICalculationType = "percentage_change"
	KPICalcAverageOverPeriod KPICalculationType = "average_over_period"
	KPICalcSumOverPeriod     KPICalculationType = "sum_over_period"
)

const (
	KPIFrequency15m  KPISnapshotFrequency = "every_15m"
	KPIFrequencyHour KPISnapshotFrequency = "hourly"
	KPIFrequency4h   KPISnapshotFrequency = "every_4h"
	KPIFrequencyDay  KPISnapshotFrequency = "daily"
	KPIFrequencyWeek KPISnapshotFrequency = "weekly"
)

const (
	KPIStatusNormal   KPIStatus = "normal"
	KPIStatusWarning  KPIStatus = "warning"
	KPIStatusCritical KPIStatus = "critical"
	KPIStatusUnknown  KPIStatus = "unknown"
)

type KPIDefinition struct {
	ID                uuid.UUID              `json:"id"`
	TenantID          uuid.UUID              `json:"tenant_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Category          KPICategory            `json:"category"`
	Suite             KPISuite               `json:"suite"`
	Icon              *string                `json:"icon,omitempty"`
	QueryEndpoint     string                 `json:"query_endpoint"`
	QueryParams       map[string]any         `json:"query_params"`
	ValuePath         string                 `json:"value_path"`
	Unit              KPIUnit                `json:"unit"`
	FormatPattern     *string                `json:"format_pattern,omitempty"`
	TargetValue       *float64               `json:"target_value,omitempty"`
	WarningThreshold  *float64               `json:"warning_threshold,omitempty"`
	CriticalThreshold *float64               `json:"critical_threshold,omitempty"`
	Direction         KPIDirection           `json:"direction"`
	CalculationType   KPICalculationType     `json:"calculation_type"`
	CalculationWindow *string                `json:"calculation_window,omitempty"`
	SnapshotFrequency KPISnapshotFrequency   `json:"snapshot_frequency"`
	Enabled           bool                   `json:"enabled"`
	IsDefault         bool                   `json:"is_default"`
	LastSnapshotAt    *time.Time             `json:"last_snapshot_at,omitempty"`
	LastValue         *float64               `json:"last_value,omitempty"`
	LastStatus        *KPIStatus             `json:"last_status,omitempty"`
	Tags              []string               `json:"tags"`
	CreatedBy         uuid.UUID              `json:"created_by"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	DeletedAt         *time.Time             `json:"deleted_at,omitempty"`
	LatestSnapshot    *KPISnapshot           `json:"latest_snapshot,omitempty"`
}

type KPISnapshot struct {
	ID             uuid.UUID  `json:"id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	KPIID          uuid.UUID  `json:"kpi_id"`
	Value          float64    `json:"value"`
	PreviousValue  *float64   `json:"previous_value,omitempty"`
	Delta          *float64   `json:"delta,omitempty"`
	DeltaPercent   *float64   `json:"delta_percent,omitempty"`
	Status         KPIStatus  `json:"status"`
	PeriodStart    time.Time  `json:"period_start"`
	PeriodEnd      time.Time  `json:"period_end"`
	FetchSuccess   bool       `json:"fetch_success"`
	FetchLatencyMS *int       `json:"fetch_latency_ms,omitempty"`
	FetchError     *string    `json:"fetch_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type KPIQuery struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
	Limit int        `json:"limit,omitempty"`
}
