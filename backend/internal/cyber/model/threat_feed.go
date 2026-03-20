package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ThreatFeedType string

const (
	ThreatFeedTypeSTIX   ThreatFeedType = "stix"
	ThreatFeedTypeTAXII  ThreatFeedType = "taxii"
	ThreatFeedTypeMISP   ThreatFeedType = "misp"
	ThreatFeedTypeCSVURL ThreatFeedType = "csv_url"
	ThreatFeedTypeManual ThreatFeedType = "manual"
)

func (t ThreatFeedType) IsValid() bool {
	switch t {
	case ThreatFeedTypeSTIX, ThreatFeedTypeTAXII, ThreatFeedTypeMISP, ThreatFeedTypeCSVURL, ThreatFeedTypeManual:
		return true
	default:
		return false
	}
}

type ThreatFeedAuthType string

const (
	ThreatFeedAuthNone        ThreatFeedAuthType = "none"
	ThreatFeedAuthAPIKey      ThreatFeedAuthType = "api_key"
	ThreatFeedAuthBasic       ThreatFeedAuthType = "basic"
	ThreatFeedAuthCertificate ThreatFeedAuthType = "certificate"
)

func (t ThreatFeedAuthType) IsValid() bool {
	switch t {
	case ThreatFeedAuthNone, ThreatFeedAuthAPIKey, ThreatFeedAuthBasic, ThreatFeedAuthCertificate:
		return true
	default:
		return false
	}
}

type ThreatFeedInterval string

const (
	ThreatFeedIntervalHourly  ThreatFeedInterval = "hourly"
	ThreatFeedIntervalEvery6H ThreatFeedInterval = "every_6h"
	ThreatFeedIntervalDaily   ThreatFeedInterval = "daily"
	ThreatFeedIntervalWeekly  ThreatFeedInterval = "weekly"
	ThreatFeedIntervalManual  ThreatFeedInterval = "manual"
)

func (t ThreatFeedInterval) IsValid() bool {
	switch t {
	case ThreatFeedIntervalHourly, ThreatFeedIntervalEvery6H, ThreatFeedIntervalDaily, ThreatFeedIntervalWeekly, ThreatFeedIntervalManual:
		return true
	default:
		return false
	}
}

type ThreatFeedStatus string

const (
	ThreatFeedStatusActive ThreatFeedStatus = "active"
	ThreatFeedStatusPaused ThreatFeedStatus = "paused"
	ThreatFeedStatusError  ThreatFeedStatus = "error"
)

func (t ThreatFeedStatus) IsValid() bool {
	switch t {
	case ThreatFeedStatusActive, ThreatFeedStatusPaused, ThreatFeedStatusError:
		return true
	default:
		return false
	}
}

// ThreatFeedConfig stores the configuration required to ingest an external threat feed.
type ThreatFeedConfig struct {
	ID                uuid.UUID          `json:"id" db:"id"`
	TenantID          uuid.UUID          `json:"tenant_id" db:"tenant_id"`
	Name              string             `json:"name" db:"name"`
	Type              ThreatFeedType     `json:"type" db:"type"`
	URL               *string            `json:"url,omitempty" db:"url"`
	AuthType          ThreatFeedAuthType `json:"auth_type" db:"auth_type"`
	AuthConfig        json.RawMessage    `json:"auth_config" db:"auth_config"`
	SyncInterval      ThreatFeedInterval `json:"sync_interval" db:"sync_interval"`
	DefaultSeverity   Severity           `json:"default_severity" db:"default_severity"`
	DefaultConfidence float64            `json:"default_confidence" db:"default_confidence"`
	DefaultTags       []string           `json:"default_tags" db:"default_tags"`
	IndicatorTypes    []string           `json:"indicator_types" db:"indicator_types"`
	Enabled           bool               `json:"enabled" db:"enabled"`
	Status            ThreatFeedStatus   `json:"status" db:"status"`
	LastSyncAt        *time.Time         `json:"last_sync_at,omitempty" db:"last_sync_at"`
	LastSyncStatus    *string            `json:"last_sync_status,omitempty" db:"last_sync_status"`
	LastError         *string            `json:"last_error,omitempty" db:"last_error"`
	NextSyncAt        *time.Time         `json:"next_sync_at,omitempty" db:"-"`
	CreatedBy         *uuid.UUID         `json:"created_by,omitempty" db:"created_by"`
	CreatedAt         time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" db:"updated_at"`
}

// ThreatFeedSyncHistory captures the outcome of one feed sync execution.
type ThreatFeedSyncHistory struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	FeedID             uuid.UUID       `json:"feed_id" db:"feed_id"`
	Status             string          `json:"status" db:"status"`
	IndicatorsParsed   int             `json:"indicators_parsed" db:"indicators_parsed"`
	IndicatorsImported int             `json:"indicators_imported" db:"indicators_imported"`
	IndicatorsSkipped  int             `json:"indicators_skipped" db:"indicators_skipped"`
	IndicatorsFailed   int             `json:"indicators_failed" db:"indicators_failed"`
	DurationMs         int             `json:"duration_ms" db:"duration_ms"`
	ErrorMessage       *string         `json:"error_message,omitempty" db:"error_message"`
	Metadata           json.RawMessage `json:"metadata" db:"metadata"`
	StartedAt          time.Time       `json:"started_at" db:"started_at"`
	CompletedAt        *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
}
