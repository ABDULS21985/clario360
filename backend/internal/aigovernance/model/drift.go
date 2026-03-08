package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DriftLevel string

const (
	DriftLevelNone        DriftLevel = "none"
	DriftLevelLow         DriftLevel = "low"
	DriftLevelModerate    DriftLevel = "moderate"
	DriftLevelSignificant DriftLevel = "significant"
)

type DriftAlert struct {
	Type        string `json:"type"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	Recommended string `json:"recommended,omitempty"`
}

type DriftReport struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	TenantID               uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	ModelID                uuid.UUID       `json:"model_id" db:"model_id"`
	ModelVersionID         uuid.UUID       `json:"model_version_id" db:"model_version_id"`
	ModelSlug              string          `json:"model_slug,omitempty"`
	Period                 string          `json:"period" db:"period"`
	PeriodStart            time.Time       `json:"period_start" db:"period_start"`
	PeriodEnd              time.Time       `json:"period_end" db:"period_end"`
	OutputPSI              *float64        `json:"output_psi,omitempty" db:"output_psi"`
	OutputDriftLevel       DriftLevel      `json:"output_drift_level,omitempty" db:"output_drift_level"`
	ConfidencePSI          *float64        `json:"confidence_psi,omitempty" db:"confidence_psi"`
	ConfidenceDriftLevel   DriftLevel      `json:"confidence_drift_level,omitempty" db:"confidence_drift_level"`
	CurrentVolume          int64           `json:"current_volume" db:"current_volume"`
	ReferenceVolume        int64           `json:"reference_volume" db:"reference_volume"`
	VolumeChangePct        *float64        `json:"volume_change_pct,omitempty" db:"volume_change_pct"`
	CurrentP95LatencyMS    *float64        `json:"current_p95_latency_ms,omitempty" db:"current_p95_latency_ms"`
	ReferenceP95LatencyMS  *float64        `json:"reference_p95_latency_ms,omitempty" db:"reference_p95_latency_ms"`
	LatencyChangePct       *float64        `json:"latency_change_pct,omitempty" db:"latency_change_pct"`
	CurrentAccuracy        *float64        `json:"current_accuracy,omitempty" db:"current_accuracy"`
	ReferenceAccuracy      *float64        `json:"reference_accuracy,omitempty" db:"reference_accuracy"`
	AccuracyChange         *float64        `json:"accuracy_change,omitempty" db:"accuracy_change"`
	Alerts                 json.RawMessage `json:"alerts" db:"alerts"`
	AlertCount             int             `json:"alert_count" db:"alert_count"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
}
