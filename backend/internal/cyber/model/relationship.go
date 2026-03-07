package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RelationshipType enumerates valid directed relationship kinds between assets.
type RelationshipType string

const (
	RelationshipHosts        RelationshipType = "hosts"
	RelationshipRunsOn       RelationshipType = "runs_on"
	RelationshipConnectsTo   RelationshipType = "connects_to"
	RelationshipDependsOn    RelationshipType = "depends_on"
	RelationshipManagedBy    RelationshipType = "managed_by"
	RelationshipBacksUp      RelationshipType = "backs_up"
	RelationshipLoadBalances RelationshipType = "load_balances"
)

// ValidRelationshipTypes is the authoritative list of relationship types.
var ValidRelationshipTypes = []RelationshipType{
	RelationshipHosts, RelationshipRunsOn, RelationshipConnectsTo,
	RelationshipDependsOn, RelationshipManagedBy, RelationshipBacksUp,
	RelationshipLoadBalances,
}

// IsValid reports whether r is a recognized RelationshipType.
func (r RelationshipType) IsValid() bool {
	for _, v := range ValidRelationshipTypes {
		if v == r {
			return true
		}
	}
	return false
}

// AssetRelationship represents a directed edge in the asset dependency graph.
type AssetRelationship struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	TenantID         uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	SourceAssetID    uuid.UUID        `json:"source_asset_id" db:"source_asset_id"`
	TargetAssetID    uuid.UUID        `json:"target_asset_id" db:"target_asset_id"`
	RelationshipType RelationshipType `json:"relationship_type" db:"relationship_type"`
	Metadata         json.RawMessage  `json:"metadata" db:"metadata"`
	CreatedBy        *uuid.UUID       `json:"created_by,omitempty" db:"created_by"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`

	// Populated via JOIN when fetching asset relationships
	SourceAssetName *string `json:"source_asset_name,omitempty" db:"source_asset_name"`
	TargetAssetName *string `json:"target_asset_name,omitempty" db:"target_asset_name"`
}
