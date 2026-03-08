package model

import "time"

type SyncDirection string

const (
	SyncDirectionOutbound      SyncDirection = "outbound"
	SyncDirectionInbound       SyncDirection = "inbound"
	SyncDirectionBidirectional SyncDirection = "bidirectional"
)

type ExternalTicketLink struct {
	ID                string        `json:"id" db:"id"`
	TenantID          string        `json:"tenant_id" db:"tenant_id"`
	IntegrationID     string        `json:"integration_id" db:"integration_id"`
	EntityType        string        `json:"entity_type" db:"entity_type"`
	EntityID          string        `json:"entity_id" db:"entity_id"`
	ExternalSystem    string        `json:"external_system" db:"external_system"`
	ExternalID        string        `json:"external_id" db:"external_id"`
	ExternalKey       string        `json:"external_key" db:"external_key"`
	ExternalURL       string        `json:"external_url" db:"external_url"`
	ExternalStatus    *string       `json:"external_status,omitempty" db:"external_status"`
	ExternalPriority  *string       `json:"external_priority,omitempty" db:"external_priority"`
	SyncDirection     SyncDirection `json:"sync_direction" db:"sync_direction"`
	LastSyncedAt      *time.Time    `json:"last_synced_at,omitempty" db:"last_synced_at"`
	LastSyncDirection *string       `json:"last_sync_direction,omitempty" db:"last_sync_direction"`
	SyncError         *string       `json:"sync_error,omitempty" db:"sync_error"`
	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`
}
