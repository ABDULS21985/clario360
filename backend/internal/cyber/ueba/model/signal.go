package model

import (
	"time"

	"github.com/google/uuid"
)

type SignalType string

const (
	SignalTypeUnusualTime         SignalType = "unusual_time"
	SignalTypeUnusualVolume       SignalType = "unusual_volume"
	SignalTypeNewTableAccess      SignalType = "new_table_access"
	SignalTypeNewSourceIP         SignalType = "new_source_ip"
	SignalTypeFailedAccessSpike   SignalType = "failed_access_spike"
	SignalTypeBulkDataAccess      SignalType = "bulk_data_access"
	SignalTypePrivilegeEscalation SignalType = "privilege_escalation"
)

type AnomalySignal struct {
	SignalType       SignalType `json:"signal_type"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	Severity         string     `json:"severity"`
	Confidence       float64    `json:"confidence"`
	DeviationZ       float64    `json:"deviation_z"`
	ExpectedValue    string     `json:"expected_value"`
	ActualValue      string     `json:"actual_value"`
	EventID          uuid.UUID  `json:"event_id"`
	MITRETechnique   string     `json:"mitre_technique"`
	MITRETactic      string     `json:"mitre_tactic,omitempty"`
	TableSensitivity string     `json:"table_sensitivity,omitempty"`
	EntityID         string     `json:"entity_id,omitempty"`
	EventTimestamp   time.Time  `json:"event_timestamp,omitempty"`
}
