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
	AssetsFound      int             `json:"assets_found" db:"-"` // alias for frontend
	AssetsNew        int             `json:"assets_new" db:"assets_new"`
	AssetsUpdated    int             `json:"assets_updated" db:"assets_updated"`
	ErrorCount       int             `json:"error_count" db:"error_count"`
	Errors           json.RawMessage `json:"errors,omitempty" db:"errors"`
	Target           *string         `json:"target,omitempty" db:"-"` // extracted from config
	Error            *string         `json:"error,omitempty" db:"-"`  // first error string
	StartedAt        time.Time       `json:"started_at" db:"started_at"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs       *int64          `json:"duration_ms,omitempty" db:"duration_ms"`
	CreatedBy        uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
}

// ComputeDerived populates frontend-facing derived fields.
func (s *ScanHistory) ComputeDerived() {
	s.AssetsFound = s.AssetsDiscovered

	// Extract target from config.targets
	if s.Config != nil {
		var cfg ScanConfig
		if json.Unmarshal(s.Config, &cfg) == nil && len(cfg.Targets) > 0 {
			joined := ""
			for i, t := range cfg.Targets {
				if i > 0 {
					joined += ", "
				}
				joined += t
			}
			s.Target = &joined
		}
	}

	// Extract first error string
	if s.Errors != nil && len(s.Errors) > 2 { // not "[]" or "null"
		var errs []string
		if json.Unmarshal(s.Errors, &errs) == nil && len(errs) > 0 {
			s.Error = &errs[0]
		}
	}
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
	ScanID          uuid.UUID      // originating scan ID, set before upsert
	IsNew           bool           // set after upsert (xmax=0 means INSERT)
	AssetID         uuid.UUID      // set after upsert
}
