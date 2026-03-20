package model

import (
	"time"

	"github.com/google/uuid"
)

// LineageEdgeType classifies the mechanism by which data flows.
type LineageEdgeType string

const (
	EdgeTypeETLPipeline  LineageEdgeType = "etl_pipeline"
	EdgeTypeReplication  LineageEdgeType = "replication"
	EdgeTypeAPITransfer  LineageEdgeType = "api_transfer"
	EdgeTypeManualCopy   LineageEdgeType = "manual_copy"
	EdgeTypeQueryDerived LineageEdgeType = "query_derived"
	EdgeTypeStream       LineageEdgeType = "stream"
	EdgeTypeExport       LineageEdgeType = "export"
	EdgeTypeInferred     LineageEdgeType = "inferred"
)

// LineageEdgeStatus tracks the operational state of a lineage edge.
type LineageEdgeStatus string

const (
	EdgeStatusActive     LineageEdgeStatus = "active"
	EdgeStatusInactive   LineageEdgeStatus = "inactive"
	EdgeStatusBroken     LineageEdgeStatus = "broken"
	EdgeStatusDeprecated LineageEdgeStatus = "deprecated"
)

// LineageEdge is a single data flow connection between two assets.
type LineageEdge struct {
	ID                    uuid.UUID              `json:"id"`
	TenantID              uuid.UUID              `json:"tenant_id"`
	SourceAssetID         uuid.UUID              `json:"source_asset_id"`
	SourceAssetName       string                 `json:"source_asset_name"`
	SourceTable           string                 `json:"source_table,omitempty"`
	TargetAssetID         uuid.UUID              `json:"target_asset_id"`
	TargetAssetName       string                 `json:"target_asset_name"`
	TargetTable           string                 `json:"target_table,omitempty"`
	EdgeType              LineageEdgeType        `json:"edge_type"`
	Transformation        string                 `json:"transformation,omitempty"`
	PipelineID            string                 `json:"pipeline_id,omitempty"`
	PipelineName          string                 `json:"pipeline_name,omitempty"`
	SourceClassification  string                 `json:"source_classification,omitempty"`
	TargetClassification  string                 `json:"target_classification,omitempty"`
	ClassificationChanged bool                   `json:"classification_changed"`
	PIITypesTransferred   []string               `json:"pii_types_transferred"`
	Confidence            float64                `json:"confidence"`
	Evidence              map[string]interface{} `json:"evidence"`
	Status                LineageEdgeStatus      `json:"status"`
	LastTransferAt        *time.Time             `json:"last_transfer_at,omitempty"`
	TransferCount30d      int                    `json:"transfer_count_30d"`
	CreatedAt             time.Time              `json:"created_at"`
	UpdatedAt             time.Time              `json:"updated_at"`
}

// LineageNode represents an asset in the lineage graph.
type LineageNode struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	AssetType      string    `json:"asset_type"`
	Classification string    `json:"classification"`
	ContainsPII    bool      `json:"contains_pii"`
	PIITypes       []string  `json:"pii_types"`
	RiskScore      float64   `json:"risk_score"`
	PostureScore   float64   `json:"posture_score"`
	Depth          int       `json:"depth"`
}

// LineageGraph is the complete lineage graph for visualization.
type LineageGraph struct {
	Nodes         []LineageNode `json:"nodes"`
	Edges         []LineageEdge `json:"edges"`
	TotalNodes    int           `json:"total_nodes"`
	TotalEdges    int           `json:"total_edges"`
	PIIFlowCount  int           `json:"pii_flow_count"`
	InferredCount int           `json:"inferred_count"`
}

// ImpactResult is the downstream impact analysis for an asset.
type ImpactResult struct {
	AssetID             uuid.UUID     `json:"asset_id"`
	AssetName           string        `json:"asset_name"`
	DownstreamAssets    int           `json:"downstream_assets"`
	DownstreamPIIAssets int           `json:"downstream_pii_assets"`
	MaxDepth            int           `json:"max_depth"`
	AffectedNodes       []LineageNode `json:"affected_nodes"`
	AffectedEdges       []LineageEdge `json:"affected_edges"`
	RiskAmplification   float64       `json:"risk_amplification"`
}

// SQLLineageExtraction is the result of parsing SQL for lineage.
type SQLLineageExtraction struct {
	SourceTables   []string `json:"source_tables"`
	TargetTable    string   `json:"target_table"`
	Transformation string   `json:"transformation"`
	StatementType  string   `json:"statement_type"`
}

// InferredLineageEvidence records why a lineage edge was inferred.
type InferredLineageEvidence struct {
	SchemaSimilarity    float64  `json:"schema_similarity"`
	TemporalCorrelation float64  `json:"temporal_correlation"`
	ColumnOverlap       []string `json:"column_overlap"`
	OverlapRatio        float64  `json:"overlap_ratio"`
}
