package model

import (
	"time"

	"github.com/google/uuid"
)

type AlertType string

const (
	AlertTypePossibleDataExfiltration     AlertType = "possible_data_exfiltration"
	AlertTypePossibleCredentialCompromise AlertType = "possible_credential_compromise"
	AlertTypePossibleInsiderThreat        AlertType = "possible_insider_threat"
	AlertTypePossibleLateralMovement      AlertType = "possible_lateral_movement"
	AlertTypePossiblePrivilegeAbuse       AlertType = "possible_privilege_abuse"
	AlertTypeUnusualActivity              AlertType = "unusual_activity"
	AlertTypeDataReconnaissance           AlertType = "data_reconnaissance"
	AlertTypePolicyViolation              AlertType = "policy_violation"
)

type UEBAAlert struct {
	ID                     uuid.UUID       `json:"id" db:"id"`
	TenantID               uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CyberAlertID           *uuid.UUID      `json:"cyber_alert_id,omitempty" db:"cyber_alert_id"`
	EntityType             EntityType      `json:"entity_type" db:"entity_type"`
	EntityID               string          `json:"entity_id" db:"entity_id"`
	EntityName             string          `json:"entity_name,omitempty" db:"entity_name"`
	AlertType              AlertType       `json:"alert_type" db:"alert_type"`
	Severity               string          `json:"severity" db:"severity"`
	Confidence             float64         `json:"confidence" db:"confidence"`
	RiskScoreBefore        float64         `json:"risk_score_before" db:"risk_score_before"`
	RiskScoreAfter         float64         `json:"risk_score_after" db:"risk_score_after"`
	RiskScoreDelta         float64         `json:"risk_score_delta" db:"risk_score_delta"`
	Title                  string          `json:"title" db:"title"`
	Description            string          `json:"description" db:"description"`
	TriggeringSignals      []AnomalySignal `json:"triggering_signals" db:"triggering_signals"`
	TriggeringEventIDs     []uuid.UUID     `json:"triggering_event_ids" db:"triggering_event_ids"`
	BaselineComparison     map[string]any  `json:"baseline_comparison" db:"baseline_comparison"`
	CorrelatedSignalCount  int             `json:"correlated_signal_count" db:"correlated_signal_count"`
	CorrelationWindowStart time.Time       `json:"correlation_window_start" db:"correlation_window_start"`
	CorrelationWindowEnd   time.Time       `json:"correlation_window_end" db:"correlation_window_end"`
	MITRETechniqueIDs      []string        `json:"mitre_technique_ids,omitempty" db:"mitre_technique_ids"`
	MITRETactic            string          `json:"mitre_tactic,omitempty" db:"mitre_tactic"`
	Status                 string          `json:"status" db:"status"`
	ResolvedAt             *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy             *uuid.UUID      `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolutionNotes        string          `json:"resolution_notes,omitempty" db:"resolution_notes"`
	CreatedAt              time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at" db:"updated_at"`
}
