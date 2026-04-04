package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/dto"
	"github.com/clario360/platform/internal/data/model"
)

// sourceService defines the methods used by SourceHandler.
type sourceService interface {
	Create(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateSourceRequest) (*model.DataSource, error)
	List(ctx context.Context, tenantID uuid.UUID, params dto.ListSourcesParams) ([]*model.DataSource, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*model.DataSource, error)
	Update(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateSourceRequest) (*model.DataSource, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	ChangeStatus(ctx context.Context, tenantID, id uuid.UUID, status model.DataSourceStatus) (*model.DataSource, error)
	TestConnection(ctx context.Context, tenantID, id uuid.UUID) (*connector.ConnectionTestResult, error)
	TestConfig(ctx context.Context, tenantID uuid.UUID, req dto.TestSourceConfigRequest) (*connector.ConnectionTestResult, error)
	DiscoverSchema(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error)
	GetSchema(ctx context.Context, tenantID, id uuid.UUID) (*model.DiscoveredSchema, error)
	TriggerSync(ctx context.Context, tenantID, id uuid.UUID, syncType model.SyncType, userID *uuid.UUID) (*model.SyncHistory, error)
	ListSyncHistory(ctx context.Context, tenantID, id uuid.UUID, limit int) ([]*model.SyncHistory, error)
	GetStats(ctx context.Context, tenantID, id uuid.UUID) (*dto.SourceStatsResponse, error)
	AggregateStats(ctx context.Context, tenantID uuid.UUID) (*dto.AggregateSourceStatsResponse, error)
}

// connectorRegistry defines the methods used by SourceHandler for source type metadata.
type connectorRegistry interface {
	ListTypes() []model.DataSourceType
	TypeMetadata(sourceType model.DataSourceType) *connector.ConnectorTypeMetadata
}

// qualityService defines the methods used by QualityHandler.
type qualityService interface {
	CreateRule(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateQualityRuleRequest) (*model.QualityRule, error)
	ListRules(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityRulesParams) ([]*model.QualityRule, int, error)
	GetRule(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityRule, error)
	UpdateRule(ctx context.Context, tenantID, id uuid.UUID, req dto.UpdateQualityRuleRequest) (*model.QualityRule, error)
	DeleteRule(ctx context.Context, tenantID, id uuid.UUID) error
	RunRule(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error)
	ListResults(ctx context.Context, tenantID uuid.UUID, params dto.ListQualityResultsParams) ([]*model.QualityResult, int, error)
	GetResult(ctx context.Context, tenantID, id uuid.UUID) (*model.QualityResult, error)
	Score(ctx context.Context, tenantID uuid.UUID) (*model.QualityScore, error)
	Trend(ctx context.Context, tenantID uuid.UUID, days int) ([]model.QualityTrendPoint, error)
	Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.QualityDashboard, error)
}

// lineageService defines the methods used by LineageHandler.
type lineageService interface {
	FullGraph(ctx context.Context, tenantID uuid.UUID) (*model.LineageGraph, error)
	EntityGraph(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error)
	Upstream(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error)
	Downstream(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID, depth int) (*model.LineageGraph, error)
	Impact(ctx context.Context, tenantID uuid.UUID, entityType model.LineageEntityType, entityID uuid.UUID) (*model.ImpactAnalysis, error)
	Record(ctx context.Context, tenantID uuid.UUID, req dto.RecordLineageEdgeRequest) (*model.LineageEdgeRecord, error)
	DeleteEdge(ctx context.Context, tenantID, edgeID uuid.UUID) error
	Search(ctx context.Context, tenantID uuid.UUID, params dto.SearchLineageParams) ([]model.LineageSearchResult, error)
	Stats(ctx context.Context, tenantID uuid.UUID) (*model.LineageStatsSummary, error)
}
