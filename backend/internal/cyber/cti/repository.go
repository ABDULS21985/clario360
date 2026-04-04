package cti

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Repository defines the data access interface for the CTI module.
type Repository interface {
	// Reference data
	ListSeverityLevels(ctx context.Context, tenantID uuid.UUID) ([]ThreatSeverityLevel, error)
	ListCategories(ctx context.Context, tenantID uuid.UUID) ([]ThreatCategory, error)
	ListRegions(ctx context.Context, tenantID uuid.UUID, parentID *uuid.UUID) ([]GeographicRegion, error)
	ListSectors(ctx context.Context, tenantID uuid.UUID) ([]IndustrySector, error)
	ListDataSources(ctx context.Context, tenantID uuid.UUID) ([]DataSource, error)
	GetSeverityByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ThreatSeverityLevel, error)
	GetCategoryByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ThreatCategory, error)
	GetSectorByCode(ctx context.Context, tenantID uuid.UUID, code string) (*IndustrySector, error)
	GetSourceByName(ctx context.Context, tenantID uuid.UUID, name string) (*DataSource, error)

	// Threat events
	CreateThreatEvent(ctx context.Context, tenantID uuid.UUID, event *ThreatEvent) error
	GetThreatEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*ThreatEventDetail, error)
	ListThreatEvents(ctx context.Context, tenantID uuid.UUID, f ThreatEventFilters) ([]ThreatEventDetail, int, error)
	UpdateThreatEvent(ctx context.Context, tenantID, eventID uuid.UUID, updates map[string]interface{}) error
	DeleteThreatEvent(ctx context.Context, tenantID, eventID, userID uuid.UUID) error
	MarkFalsePositive(ctx context.Context, tenantID, eventID, userID uuid.UUID) error
	ResolveThreatEvent(ctx context.Context, tenantID, eventID, userID uuid.UUID) error

	// Event tags
	AddEventTags(ctx context.Context, tenantID, eventID uuid.UUID, tags []string) error
	RemoveEventTag(ctx context.Context, tenantID, eventID uuid.UUID, tag string) error
	GetEventTags(ctx context.Context, tenantID, eventID uuid.UUID) ([]string, error)

	// Threat actors
	CreateThreatActor(ctx context.Context, tenantID uuid.UUID, actor *ThreatActor) error
	GetThreatActor(ctx context.Context, tenantID, actorID uuid.UUID) (*ThreatActor, error)
	ListThreatActors(ctx context.Context, tenantID uuid.UUID, f ThreatActorFilters) ([]ThreatActor, int, error)
	UpdateThreatActor(ctx context.Context, tenantID, actorID uuid.UUID, updates map[string]interface{}) error
	DeleteThreatActor(ctx context.Context, tenantID, actorID, userID uuid.UUID) error

	// Campaigns
	CreateCampaign(ctx context.Context, tenantID uuid.UUID, c *Campaign) error
	GetCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) (*CampaignDetail, error)
	ListCampaigns(ctx context.Context, tenantID uuid.UUID, f CampaignFilters) ([]CampaignDetail, int, error)
	UpdateCampaign(ctx context.Context, tenantID, campaignID uuid.UUID, updates map[string]interface{}) error
	DeleteCampaign(ctx context.Context, tenantID, campaignID, userID uuid.UUID) error
	UpdateCampaignStatus(ctx context.Context, tenantID, campaignID uuid.UUID, status string, userID uuid.UUID) error

	// Campaign events
	LinkEventToCampaign(ctx context.Context, tenantID, campaignID, eventID uuid.UUID, userID *uuid.UUID) error
	UnlinkEventFromCampaign(ctx context.Context, tenantID, campaignID, eventID uuid.UUID) error
	ListCampaignEvents(ctx context.Context, tenantID, campaignID uuid.UUID, p ListParams) ([]ThreatEventDetail, int, error)

	// Campaign IOCs
	CreateCampaignIOC(ctx context.Context, tenantID uuid.UUID, ioc *CampaignIOC) error
	ListCampaignIOCs(ctx context.Context, tenantID, campaignID uuid.UUID, p ListParams) ([]CampaignIOC, int, error)
	DeleteCampaignIOC(ctx context.Context, tenantID, iocID uuid.UUID) error

	// Brand abuse
	CreateMonitoredBrand(ctx context.Context, tenantID uuid.UUID, brand *MonitoredBrand) error
	ListMonitoredBrands(ctx context.Context, tenantID uuid.UUID) ([]MonitoredBrand, error)
	UpdateMonitoredBrand(ctx context.Context, tenantID, brandID uuid.UUID, updates map[string]interface{}) error
	DeleteMonitoredBrand(ctx context.Context, tenantID, brandID uuid.UUID) error
	CreateBrandAbuseIncident(ctx context.Context, tenantID uuid.UUID, inc *BrandAbuseIncident) error
	GetBrandAbuseIncident(ctx context.Context, tenantID, incidentID uuid.UUID) (*BrandAbuseDetail, error)
	ListBrandAbuseIncidents(ctx context.Context, tenantID uuid.UUID, f BrandAbuseFilters) ([]BrandAbuseDetail, int, error)
	UpdateBrandAbuseIncident(ctx context.Context, tenantID, incidentID uuid.UUID, updates map[string]interface{}) error
	UpdateTakedownStatus(ctx context.Context, tenantID, incidentID uuid.UUID, status string, userID uuid.UUID) error

	// Dashboard
	GetGeoThreatMap(ctx context.Context, tenantID uuid.UUID, period string) ([]GeoThreatSummary, error)
	GetSectorThreatSummary(ctx context.Context, tenantID uuid.UUID, period string) ([]SectorThreatSummary, error)
	GetExecutiveSnapshot(ctx context.Context, tenantID uuid.UUID) (*ExecutiveSnapshot, error)

	// Idempotent ingestion
	FindThreatEventBySourceRef(ctx context.Context, tenantID, sourceID uuid.UUID, sourceRef string) (*ThreatEvent, error)
	UpdateThreatEventLastSeen(ctx context.Context, tenantID, eventID uuid.UUID) error
	FindMatchingCampaignIOCs(ctx context.Context, tenantID uuid.UUID, iocType, iocValue string) ([]CampaignIOC, error)
	ListPollingTenants(ctx context.Context) ([]uuid.UUID, error)
	UpdateDataSourceLastPolled(ctx context.Context, tenantID, sourceID uuid.UUID, polledAt time.Time) error

	// Aggregation refresh
	RefreshGeoThreatSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) error
	RefreshSectorThreatSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) error
	RefreshExecutiveSnapshot(ctx context.Context, tenantID uuid.UUID) error
}
