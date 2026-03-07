package model

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SecurityEvent is the normalized event representation evaluated by the detection engine.
type SecurityEvent struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Timestamp     time.Time       `json:"timestamp" db:"timestamp"`
	Source        string          `json:"source" db:"source"`
	Type          string          `json:"type" db:"type"`
	Severity      Severity        `json:"severity" db:"severity"`
	SourceIP      *string         `json:"source_ip,omitempty" db:"source_ip"`
	DestIP        *string         `json:"dest_ip,omitempty" db:"dest_ip"`
	DestPort      *int            `json:"dest_port,omitempty" db:"dest_port"`
	Protocol      *string         `json:"protocol,omitempty" db:"protocol"`
	Username      *string         `json:"username,omitempty" db:"username"`
	Process       *string         `json:"process,omitempty" db:"process"`
	ParentProcess *string         `json:"parent_process,omitempty" db:"parent_process"`
	CommandLine   *string         `json:"command_line,omitempty" db:"command_line"`
	FilePath      *string         `json:"file_path,omitempty" db:"file_path"`
	FileHash      *string         `json:"file_hash,omitempty" db:"file_hash"`
	AssetID       *uuid.UUID      `json:"asset_id,omitempty" db:"asset_id"`
	RawEvent      json.RawMessage `json:"raw_event" db:"raw_event"`
	MatchedRules  []uuid.UUID     `json:"matched_rules" db:"matched_rules"`
	ProcessedAt   time.Time       `json:"processed_at" db:"processed_at"`
}

// RawMap decodes the raw event payload into a map for flexible field resolution.
func (e *SecurityEvent) RawMap() map[string]interface{} {
	if len(e.RawEvent) == 0 {
		return map[string]interface{}{}
	}
	var out map[string]interface{}
	if err := json.Unmarshal(e.RawEvent, &out); err != nil {
		return map[string]interface{}{}
	}
	return out
}

// SetRawMap replaces RawEvent with the provided map.
func (e *SecurityEvent) SetRawMap(payload map[string]interface{}) error {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	e.RawEvent = encoded
	return nil
}

// GroupKey returns the best-effort grouping key used by timeframe and dedup logic.
func (e SecurityEvent) GroupKey() string {
	if e.AssetID != nil {
		return e.AssetID.String()
	}
	if e.SourceIP != nil && *e.SourceIP != "" {
		return *e.SourceIP
	}
	if e.Username != nil && *e.Username != "" {
		return *e.Username
	}
	return "global"
}

// StringField normalizes a string field value for comparisons.
func StringField(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}
