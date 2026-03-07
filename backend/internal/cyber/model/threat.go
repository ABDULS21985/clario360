package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Threat models a tracked threat campaign or malicious activity cluster.
type Threat struct {
	ID                 uuid.UUID          `json:"id" db:"id"`
	TenantID           uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	Name               string             `json:"name" db:"name"`
	Description        string             `json:"description" db:"description"`
	Type               ThreatType         `json:"type" db:"type"`
	Severity           Severity           `json:"severity" db:"severity"`
	Status             ThreatStatus       `json:"status" db:"status"`
	ThreatActor        *string            `json:"threat_actor,omitempty" db:"threat_actor"`
	Campaign           *string            `json:"campaign,omitempty" db:"campaign"`
	MITRETacticIDs     []string           `json:"mitre_tactic_ids" db:"mitre_tactic_ids"`
	MITRETechniqueIDs  []string           `json:"mitre_technique_ids" db:"mitre_technique_ids"`
	AffectedAssetCount int                `json:"affected_asset_count" db:"affected_asset_count"`
	AlertCount         int                `json:"alert_count" db:"alert_count"`
	FirstSeenAt        time.Time          `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt         time.Time          `json:"last_seen_at" db:"last_seen_at"`
	ContainedAt        *time.Time         `json:"contained_at,omitempty" db:"contained_at"`
	Tags               []string           `json:"tags" db:"tags"`
	Metadata           json.RawMessage    `json:"metadata" db:"metadata"`
	CreatedBy          *uuid.UUID         `json:"created_by,omitempty" db:"created_by"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at"`
	DeletedAt          *time.Time         `json:"-" db:"deleted_at"`
	Indicators         []*ThreatIndicator `json:"indicators,omitempty" db:"-"`
}

// ThreatIndicator models an indicator of compromise.
type ThreatIndicator struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	TenantID    uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	ThreatID    *uuid.UUID      `json:"threat_id,omitempty" db:"threat_id"`
	Type        IndicatorType   `json:"type" db:"type"`
	Value       string          `json:"value" db:"value"`
	Description string          `json:"description" db:"description"`
	Severity    Severity        `json:"severity" db:"severity"`
	Source      string          `json:"source" db:"source"`
	Confidence  float64         `json:"confidence" db:"confidence"`
	Active      bool            `json:"active" db:"active"`
	FirstSeenAt time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt  time.Time       `json:"last_seen_at" db:"last_seen_at"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
	Tags        []string        `json:"tags" db:"tags"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// IndicatorMatch captures a runtime match between a security event and an indicator.
type IndicatorMatch struct {
	Indicator *ThreatIndicator `json:"indicator"`
	Field     string           `json:"field"`
	Value     string           `json:"value"`
}

// ThreatStats holds aggregated threat counts for dashboards.
type ThreatStats struct {
	ByType     []NamedCount `json:"by_type"`
	ByStatus   []NamedCount `json:"by_status"`
	BySeverity []NamedCount `json:"by_severity"`
	Total      int          `json:"total"`
	Active     int          `json:"active"`
}
