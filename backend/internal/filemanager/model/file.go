package model

import (
	"encoding/json"
	"time"
)

// FileRecord represents a stored file's metadata.
type FileRecord struct {
	ID                  string          `json:"id" db:"id"`
	TenantID            string          `json:"tenant_id" db:"tenant_id"`
	Bucket              string          `json:"bucket" db:"bucket"`
	StorageKey          string          `json:"-" db:"storage_key"` // never expose in API
	OriginalName        string          `json:"original_name" db:"original_name"`
	SanitizedName       string          `json:"sanitized_name" db:"sanitized_name"`
	ContentType         string          `json:"content_type" db:"content_type"`
	DetectedContentType string          `json:"detected_content_type,omitempty" db:"detected_content_type"`
	SizeBytes           int64           `json:"size_bytes" db:"size_bytes"`
	ChecksumSHA256      string          `json:"checksum_sha256" db:"checksum_sha256"`
	Encrypted           bool            `json:"encrypted" db:"encrypted"`
	EncryptionMetadata  json.RawMessage `json:"-" db:"encryption_metadata"`
	VirusScanStatus     string          `json:"virus_scan_status" db:"virus_scan_status"`
	VirusScanResult     *string         `json:"virus_scan_result,omitempty" db:"virus_scan_result"`
	VirusScannedAt      *time.Time      `json:"virus_scanned_at,omitempty" db:"virus_scanned_at"`
	UploadedBy          string          `json:"uploaded_by" db:"uploaded_by"`
	Suite               string          `json:"suite" db:"suite"`
	EntityType          *string         `json:"entity_type,omitempty" db:"entity_type"`
	EntityID            *string         `json:"entity_id,omitempty" db:"entity_id"`
	Tags                []string        `json:"tags" db:"tags"`
	VersionID           *string         `json:"version_id,omitempty" db:"version_id"`
	VersionNumber       int             `json:"version_number" db:"version_number"`
	IsPublic            bool            `json:"is_public" db:"is_public"`
	LifecyclePolicy     string          `json:"lifecycle_policy" db:"lifecycle_policy"`
	ExpiresAt           *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt           *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
}

// ScanStatusPending indicates the file hasn't been scanned yet.
const (
	ScanStatusPending  = "pending"
	ScanStatusScanning = "scanning"
	ScanStatusClean    = "clean"
	ScanStatusInfected = "infected"
	ScanStatusError    = "error"
	ScanStatusSkipped  = "skipped"
)

// LifecyclePolicy constants.
const (
	LifecycleStandard       = "standard"
	LifecycleTemporary      = "temporary"
	LifecycleArchive        = "archive"
	LifecycleAuditRetention = "audit_retention"
)

// Suite constants.
const (
	SuiteCyber    = "cyber"
	SuiteData     = "data"
	SuiteActa     = "acta"
	SuiteLex      = "lex"
	SuiteVisus    = "visus"
	SuitePlatform = "platform"
	SuiteModels   = "models"
)

// ValidSuites is the set of valid suite values.
var ValidSuites = map[string]bool{
	SuiteCyber: true, SuiteData: true, SuiteActa: true, SuiteLex: true,
	SuiteVisus: true, SuitePlatform: true, SuiteModels: true,
}

// FileAccessLog records each access to a file.
type FileAccessLog struct {
	ID        string    `json:"id" db:"id"`
	FileID    string    `json:"file_id" db:"file_id"`
	TenantID  string    `json:"tenant_id" db:"tenant_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Action    string    `json:"action" db:"action"`
	IPAddress string    `json:"ip_address" db:"ip_address"`
	UserAgent string    `json:"user_agent" db:"user_agent"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// QuarantineLog records files moved to quarantine.
type QuarantineLog struct {
	ID               string     `json:"id" db:"id"`
	FileID           string     `json:"file_id" db:"file_id"`
	OriginalBucket   string     `json:"original_bucket" db:"original_bucket"`
	OriginalKey      string     `json:"original_key" db:"original_key"`
	QuarantineBucket string     `json:"quarantine_bucket" db:"quarantine_bucket"`
	QuarantineKey    string     `json:"quarantine_key" db:"quarantine_key"`
	VirusName        string     `json:"virus_name" db:"virus_name"`
	ScannedAt        time.Time  `json:"scanned_at" db:"scanned_at"`
	QuarantinedAt    time.Time  `json:"quarantined_at" db:"quarantined_at"`
	Resolved         bool       `json:"resolved" db:"resolved"`
	ResolvedBy       *string    `json:"resolved_by,omitempty" db:"resolved_by"`
	ResolvedAt       *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolutionAction *string    `json:"resolution_action,omitempty" db:"resolution_action"`
}
