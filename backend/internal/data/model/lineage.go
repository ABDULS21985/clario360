package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LineageEntityType string

const (
	LineageEntityDataSource     LineageEntityType = "data_source"
	LineageEntityDataModel      LineageEntityType = "data_model"
	LineageEntityPipeline       LineageEntityType = "pipeline"
	LineageEntityQualityRule    LineageEntityType = "quality_rule"
	LineageEntitySuiteConsumer  LineageEntityType = "suite_consumer"
	LineageEntityReport         LineageEntityType = "report"
	LineageEntityAnalyticsQuery LineageEntityType = "analytics_query"
	LineageEntityExternal       LineageEntityType = "external"
)

func (t LineageEntityType) IsValid() bool {
	switch t {
	case LineageEntityDataSource, LineageEntityDataModel, LineageEntityPipeline, LineageEntityQualityRule,
		LineageEntitySuiteConsumer, LineageEntityReport, LineageEntityAnalyticsQuery, LineageEntityExternal:
		return true
	default:
		return false
	}
}

type LineageRelationship string

const (
	LineageRelationshipFeeds          LineageRelationship = "feeds"
	LineageRelationshipDerivedFrom    LineageRelationship = "derived_from"
	LineageRelationshipTransformsInto LineageRelationship = "transforms_into"
	LineageRelationshipConsumedBy     LineageRelationship = "consumed_by"
	LineageRelationshipValidatedBy    LineageRelationship = "validated_by"
	LineageRelationshipReportedIn     LineageRelationship = "reported_in"
	LineageRelationshipQueriedBy      LineageRelationship = "queried_by"
	LineageRelationshipDependsOn      LineageRelationship = "depends_on"
)

func (r LineageRelationship) IsValid() bool {
	switch r {
	case LineageRelationshipFeeds, LineageRelationshipDerivedFrom, LineageRelationshipTransformsInto,
		LineageRelationshipConsumedBy, LineageRelationshipValidatedBy, LineageRelationshipReportedIn,
		LineageRelationshipQueriedBy, LineageRelationshipDependsOn:
		return true
	default:
		return false
	}
}

type LineageRecordedBy string

const (
	LineageRecordedBySystem   LineageRecordedBy = "system"
	LineageRecordedByPipeline LineageRecordedBy = "pipeline"
	LineageRecordedByQuery    LineageRecordedBy = "query"
	LineageRecordedByManual   LineageRecordedBy = "manual"
	LineageRecordedByEvent    LineageRecordedBy = "event"
)

type LineageEdgeRecord struct {
	ID                 uuid.UUID           `json:"id"`
	TenantID           uuid.UUID           `json:"tenant_id"`
	SourceType         LineageEntityType   `json:"source_type"`
	SourceID           uuid.UUID           `json:"source_id"`
	SourceName         string              `json:"source_name"`
	TargetType         LineageEntityType   `json:"target_type"`
	TargetID           uuid.UUID           `json:"target_id"`
	TargetName         string              `json:"target_name"`
	Relationship       LineageRelationship `json:"relationship"`
	TransformationDesc *string             `json:"transformation_desc,omitempty"`
	TransformationType *string             `json:"transformation_type,omitempty"`
	ColumnsAffected    []string            `json:"columns_affected"`
	PipelineID         *uuid.UUID          `json:"pipeline_id,omitempty"`
	PipelineRunID      *uuid.UUID          `json:"pipeline_run_id,omitempty"`
	RecordedBy         LineageRecordedBy   `json:"recorded_by"`
	Active             bool                `json:"active"`
	FirstSeenAt        time.Time           `json:"first_seen_at"`
	LastSeenAt         time.Time           `json:"last_seen_at"`
	Metadata           json.RawMessage     `json:"metadata"`
	CreatedAt          time.Time           `json:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at"`
}

type LineageNode struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	EntityID   uuid.UUID      `json:"entity_id"`
	Name       string         `json:"name"`
	Status     string         `json:"status,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Depth      int            `json:"depth"`
	InDegree   int            `json:"in_degree"`
	OutDegree  int            `json:"out_degree"`
	IsCritical bool           `json:"is_critical"`
}

type LineageEdge struct {
	ID              uuid.UUID  `json:"id"`
	Source          string     `json:"source"`
	Target          string     `json:"target"`
	Relationship    string     `json:"relationship"`
	TransformDesc   string     `json:"transformation,omitempty"`
	ColumnsAffected []string   `json:"columns_affected,omitempty"`
	PipelineID      *uuid.UUID `json:"pipeline_id,omitempty"`
	Active          bool       `json:"active"`
	LastSeenAt      time.Time  `json:"last_seen_at"`
}

type GraphStats struct {
	NodeCount     int            `json:"node_count"`
	EdgeCount     int            `json:"edge_count"`
	MaxDepth      int            `json:"max_depth"`
	SourceCount   int            `json:"source_count"`
	ConsumerCount int            `json:"consumer_count"`
	NodesByType   map[string]int `json:"nodes_by_type"`
}

type LineageGraph struct {
	Nodes []LineageNode `json:"nodes"`
	Edges []LineageEdge `json:"edges"`
	Stats GraphStats    `json:"stats"`
}

type ImpactedEntity struct {
	Node               LineageNode `json:"node"`
	HopDistance        int         `json:"hop_distance"`
	PathDescription    string      `json:"path_description"`
	DataClassification string      `json:"data_classification,omitempty"`
}

type AffectedSuite struct {
	SuiteName  string `json:"suite_name"`
	Capability string `json:"capability"`
	Impact     string `json:"impact"`
	Severity   string `json:"severity"`
}

type ImpactAnalysis struct {
	Entity             LineageNode       `json:"entity"`
	DirectlyAffected   []ImpactedEntity  `json:"directly_affected"`
	IndirectlyAffected []ImpactedEntity  `json:"indirectly_affected"`
	AffectedSuites     []AffectedSuite   `json:"affected_suites"`
	TotalAffected      int               `json:"total_affected"`
	Severity           string            `json:"severity"`
	Summary            string            `json:"summary"`
}

type LineageStatsSummary struct {
	NodeCount            int            `json:"node_count"`
	EdgeCount            int            `json:"edge_count"`
	MaxDepth             int            `json:"max_depth"`
	SourceCount          int            `json:"source_count"`
	ConsumerCount        int            `json:"consumer_count"`
	NodesByType          map[string]int `json:"nodes_by_type"`
	CriticalPathNodes    int            `json:"critical_path_nodes"`
	LastUpdatedAtUnixSec int64          `json:"last_updated_at_unix_sec"`
}
