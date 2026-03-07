package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScanType indicates how a scan was initiated.
type ScanType string

const (
	ScanTypeNetwork ScanType = "network"
	ScanTypeCloud   ScanType = "cloud"
	ScanTypeAgent   ScanType = "agent"
	ScanTypeImport  ScanType = "import"
)

// ScanStatus reflects the current state of a scan job.
type ScanStatus string

const (
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusCompleted ScanStatus = "completed"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusCancelled ScanStatus = "cancelled"
)

// ScanConfig is stored as JSONB in scan_history.config.
type ScanConfig struct {
	Targets []string       `json:"targets"` // CIDR ranges for network scans; account IDs for cloud
	Ports   []int          `json:"ports,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// ScanHistory is a record of a completed or in-progress discovery scan.
type ScanHistory struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	TenantID         uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	ScanType         ScanType        `json:"scan_type" db:"scan_type"`
	Config           json.RawMessage `json:"config" db:"config"`
	Status           ScanStatus      `json:"status" db:"status"`
	AssetsDiscovered int             `json:"assets_discovered" db:"assets_discovered"`
	AssetsNew        int             `json:"assets_new" db:"assets_new"`
	AssetsUpdated    int             `json:"assets_updated" db:"assets_updated"`
	ErrorCount       int             `json:"error_count" db:"error_count"`
	Errors           json.RawMessage `json:"errors,omitempty" db:"errors"`
	StartedAt        time.Time       `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs       *int64          `json:"duration_ms,omitempty" db:"duration_ms"`
	CreatedBy        uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// ScanResult is the in-memory result returned after a scan completes.
type ScanResult struct {
	ScanID           uuid.UUID  `json:"scan_id"`
	Status           ScanStatus `json:"status"`
	AssetsDiscovered int        `json:"assets_discovered"`
	AssetsNew        int        `json:"assets_new"`
	AssetsUpdated    int        `json:"assets_updated"`
	DurationMs       int64      `json:"duration_ms"`
	Errors           []string   `json:"errors,omitempty"`
}

// DiscoveredAsset carries the raw output of a network probe or agent report before DB upsert.
type DiscoveredAsset struct {
	IPAddress       string
	Hostname        *string
	OS              *string
	OSVersion       *string
	MACAddress      *string // populated by agent collector
	AssetType       AssetType
	OpenPorts       []int
	Banners         map[int]string // port → banner
	ExtraMetadata   map[string]any // additional provider/agent-specific fields merged into metadata JSONB
	DiscoverySource string         // overrides scanner default ("network_scan","cloud_scan","agent","import")
	IsNew           bool           // set after upsert (xmax=0 means INSERT)
	AssetID         uuid.UUID      // set after upsert
}
