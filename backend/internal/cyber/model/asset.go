package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AssetType enumerates the categories of IT/OT assets.
type AssetType string

const (
	AssetTypeServer        AssetType = "server"
	AssetTypeEndpoint      AssetType = "endpoint"
	AssetTypeNetworkDevice AssetType = "network_device"
	AssetTypeCloudResource AssetType = "cloud_resource"
	AssetTypeIoTDevice     AssetType = "iot_device"
	AssetTypeApplication   AssetType = "application"
	AssetTypeDatabase      AssetType = "database"
	AssetTypeContainer     AssetType = "container"
)

// ValidAssetTypes is the authoritative list of valid AssetType values.
var ValidAssetTypes = []AssetType{
	AssetTypeServer, AssetTypeEndpoint, AssetTypeNetworkDevice,
	AssetTypeCloudResource, AssetTypeIoTDevice, AssetTypeApplication,
	AssetTypeDatabase, AssetTypeContainer,
}

// IsValid reports whether t is a recognized AssetType.
func (t AssetType) IsValid() bool {
	for _, v := range ValidAssetTypes {
		if v == t {
			return true
		}
	}
	return false
}

// Criticality represents the business criticality level of an asset.
type Criticality string

const (
	CriticalityCritical Criticality = "critical"
	CriticalityHigh     Criticality = "high"
	CriticalityMedium   Criticality = "medium"
	CriticalityLow      Criticality = "low"
)

// ValidCriticalities is the authoritative list of valid Criticality values.
var ValidCriticalities = []Criticality{
	CriticalityCritical, CriticalityHigh, CriticalityMedium, CriticalityLow,
}

// IsValid reports whether c is a recognized Criticality.
func (c Criticality) IsValid() bool {
	for _, v := range ValidCriticalities {
		if v == c {
			return true
		}
	}
	return false
}

// Order returns a numeric rank for sorting (higher = more critical).
func (c Criticality) Order() int {
	switch c {
	case CriticalityCritical:
		return 4
	case CriticalityHigh:
		return 3
	case CriticalityMedium:
		return 2
	case CriticalityLow:
		return 1
	default:
		return 0
	}
}

// AssetStatus represents the lifecycle state of an asset.
type AssetStatus string

const (
	AssetStatusActive         AssetStatus = "active"
	AssetStatusInactive       AssetStatus = "inactive"
	AssetStatusDecommissioned AssetStatus = "decommissioned"
	AssetStatusUnknown        AssetStatus = "unknown"
)

// ValidAssetStatuses is the authoritative list of valid AssetStatus values.
var ValidAssetStatuses = []AssetStatus{
	AssetStatusActive, AssetStatusInactive, AssetStatusDecommissioned, AssetStatusUnknown,
}

// IsValid reports whether s is a recognized AssetStatus.
func (s AssetStatus) IsValid() bool {
	for _, v := range ValidAssetStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// ValidDiscoverySources is the authoritative list of valid discovery_source values.
var ValidDiscoverySources = []string{
	"manual", "network_scan", "cloud_scan", "agent", "import",
}

// Asset represents an IT/OT asset in the inventory.
// Fields annotated with db:"..." match the PostgreSQL column names.
// Computed fields (from JOINs) are not stored in the assets table.
type Asset struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TenantID        uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name            string          `json:"name" db:"name"`
	Type            AssetType       `json:"type" db:"type"`
	IPAddress       *string         `json:"ip_address,omitempty" db:"ip_address"`
	Hostname        *string         `json:"hostname,omitempty" db:"hostname"`
	MACAddress      *string         `json:"mac_address,omitempty" db:"mac_address"`
	OS              *string         `json:"os,omitempty" db:"os"`
	OSVersion       *string         `json:"os_version,omitempty" db:"os_version"`
	Owner           *string         `json:"owner,omitempty" db:"owner"`
	Department      *string         `json:"department,omitempty" db:"department"`
	Location        *string         `json:"location,omitempty" db:"location"`
	Criticality     Criticality     `json:"criticality" db:"criticality"`
	Status          AssetStatus     `json:"status" db:"status"`
	DiscoveredAt    time.Time       `json:"discovered_at" db:"discovered_at"`
	LastSeenAt      time.Time       `json:"last_seen_at" db:"last_seen_at"`
	DiscoverySource string          `json:"discovery_source" db:"discovery_source"`
	Metadata        json.RawMessage `json:"metadata" db:"metadata"`
	Tags            []string        `json:"tags" db:"tags"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt       *time.Time      `json:"-" db:"deleted_at"`

	// Computed fields (populated via LEFT JOINs or sub-queries in list/get endpoints)
	OpenVulnerabilityCount  int     `json:"open_vulnerability_count" db:"open_vulnerability_count"`
	HighestVulnSeverity     *string `json:"highest_vulnerability_severity,omitempty" db:"highest_vulnerability_severity"`
	RelationshipCount       int     `json:"relationship_count" db:"relationship_count"`
}

// AssetCountByName is used in aggregated responses such as top departments and OS families.
type AssetCountByName struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
