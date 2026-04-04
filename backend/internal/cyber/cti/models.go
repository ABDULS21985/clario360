package cti

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Reference tables
// ---------------------------------------------------------------------------

type ThreatSeverityLevel struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Code      string     `json:"code" db:"code"`
	Label     string     `json:"label" db:"label"`
	ColorHex  string     `json:"color_hex" db:"color_hex"`
	SortOrder int        `json:"sort_order" db:"sort_order"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type ThreatCategory struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Code           string     `json:"code" db:"code"`
	Label          string     `json:"label" db:"label"`
	Description    *string    `json:"description,omitempty" db:"description"`
	MitreTacticIDs []string   `json:"mitre_tactic_ids" db:"mitre_tactic_ids"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy      *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type GeographicRegion struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TenantID       uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Code           string     `json:"code" db:"code"`
	Label          string     `json:"label" db:"label"`
	ParentRegionID *uuid.UUID `json:"parent_region_id,omitempty" db:"parent_region_id"`
	Latitude       *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude      *float64   `json:"longitude,omitempty" db:"longitude"`
	ISOCountryCode *string    `json:"iso_country_code,omitempty" db:"iso_country_code"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy      *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type IndustrySector struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	TenantID    uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Code        string     `json:"code" db:"code"`
	Label       string     `json:"label" db:"label"`
	Description *string    `json:"description,omitempty" db:"description"`
	NAICSCode   *string    `json:"naics_code,omitempty" db:"naics_code"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type DataSource struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TenantID         uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name             string     `json:"name" db:"name"`
	SourceType       string     `json:"source_type" db:"source_type"`
	URL              *string    `json:"url,omitempty" db:"url"`
	APIEndpoint      *string    `json:"api_endpoint,omitempty" db:"api_endpoint"`
	APIKeyVaultPath  *string    `json:"api_key_vault_path,omitempty" db:"api_key_vault_path"`
	ReliabilityScore float64    `json:"reliability_score" db:"reliability_score"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	LastPolledAt     *time.Time `json:"last_polled_at,omitempty" db:"last_polled_at"`
	PollIntervalSecs *int       `json:"poll_interval_seconds,omitempty" db:"poll_interval_seconds"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy        *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

// ---------------------------------------------------------------------------
// Core CTI entities
// ---------------------------------------------------------------------------

type ThreatEvent struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	TenantID          uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	EventType         string          `json:"event_type" db:"event_type"`
	Title             string          `json:"title" db:"title"`
	Description       *string         `json:"description,omitempty" db:"description"`
	SeverityID        *uuid.UUID      `json:"severity_id,omitempty" db:"severity_id"`
	CategoryID        *uuid.UUID      `json:"category_id,omitempty" db:"category_id"`
	SourceID          *uuid.UUID      `json:"source_id,omitempty" db:"source_id"`
	SourceReference   *string         `json:"source_reference,omitempty" db:"source_reference"`
	ConfidenceScore   float64         `json:"confidence_score" db:"confidence_score"`
	OriginLatitude    *float64        `json:"origin_latitude,omitempty" db:"origin_latitude"`
	OriginLongitude   *float64        `json:"origin_longitude,omitempty" db:"origin_longitude"`
	OriginCountryCode *string         `json:"origin_country_code,omitempty" db:"origin_country_code"`
	OriginCity        *string         `json:"origin_city,omitempty" db:"origin_city"`
	OriginRegionID    *uuid.UUID      `json:"origin_region_id,omitempty" db:"origin_region_id"`
	TargetSectorID    *uuid.UUID      `json:"target_sector_id,omitempty" db:"target_sector_id"`
	TargetOrgName     *string         `json:"target_org_name,omitempty" db:"target_org_name"`
	TargetCountryCode *string         `json:"target_country_code,omitempty" db:"target_country_code"`
	IOCType           *string         `json:"ioc_type,omitempty" db:"ioc_type"`
	IOCValue          *string         `json:"ioc_value,omitempty" db:"ioc_value"`
	MitreTechniqueIDs []string        `json:"mitre_technique_ids" db:"mitre_technique_ids"`
	RawPayload        json.RawMessage `json:"raw_payload,omitempty" db:"raw_payload"`
	IsFalsePositive   bool            `json:"is_false_positive" db:"is_false_positive"`
	ResolvedAt        *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy        *uuid.UUID      `json:"resolved_by,omitempty" db:"resolved_by"`
	FirstSeenAt       time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt        time.Time       `json:"last_seen_at" db:"last_seen_at"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy         *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy         *uuid.UUID      `json:"updated_by,omitempty" db:"updated_by"`
	DeletedAt         *time.Time      `json:"-" db:"deleted_at"`
}

type ThreatEventTag struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TenantID  uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	EventID   uuid.UUID  `json:"event_id" db:"event_id"`
	Tag       string     `json:"tag" db:"tag"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type ThreatActor struct {
	ID                  uuid.UUID       `json:"id" db:"id"`
	TenantID            uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	Name                string          `json:"name" db:"name"`
	Aliases             []string        `json:"aliases" db:"aliases"`
	ActorType           string          `json:"actor_type" db:"actor_type"`
	OriginCountryCode   *string         `json:"origin_country_code,omitempty" db:"origin_country_code"`
	OriginRegionID      *uuid.UUID      `json:"origin_region_id,omitempty" db:"origin_region_id"`
	SophisticationLevel string          `json:"sophistication_level" db:"sophistication_level"`
	PrimaryMotivation   string          `json:"primary_motivation" db:"primary_motivation"`
	Description         *string         `json:"description,omitempty" db:"description"`
	FirstObservedAt     *time.Time      `json:"first_observed_at,omitempty" db:"first_observed_at"`
	LastActivityAt      *time.Time      `json:"last_activity_at,omitempty" db:"last_activity_at"`
	MitreGroupID        *string         `json:"mitre_group_id,omitempty" db:"mitre_group_id"`
	ExternalReferences  json.RawMessage `json:"external_references,omitempty" db:"external_references"`
	IsActive            bool            `json:"is_active" db:"is_active"`
	RiskScore           float64         `json:"risk_score" db:"risk_score"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy           *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy           *uuid.UUID      `json:"updated_by,omitempty" db:"updated_by"`
	DeletedAt           *time.Time      `json:"-" db:"deleted_at"`
}

type Campaign struct {
	ID                uuid.UUID       `json:"id" db:"id"`
	TenantID          uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CampaignCode      string          `json:"campaign_code" db:"campaign_code"`
	Name              string          `json:"name" db:"name"`
	Description       *string         `json:"description,omitempty" db:"description"`
	Status            string          `json:"status" db:"status"`
	SeverityID        *uuid.UUID      `json:"severity_id,omitempty" db:"severity_id"`
	PrimaryActorID    *uuid.UUID      `json:"primary_actor_id,omitempty" db:"primary_actor_id"`
	TargetSectors     []uuid.UUID     `json:"target_sectors" db:"target_sectors"`
	TargetRegions     []uuid.UUID     `json:"target_regions" db:"target_regions"`
	TargetDescription *string         `json:"target_description,omitempty" db:"target_description"`
	MitreTechniqueIDs []string        `json:"mitre_technique_ids" db:"mitre_technique_ids"`
	TTPsSummary       *string         `json:"ttps_summary,omitempty" db:"ttps_summary"`
	IOCCount          int             `json:"ioc_count" db:"ioc_count"`
	EventCount        int             `json:"event_count" db:"event_count"`
	FirstSeenAt       time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt        *time.Time      `json:"last_seen_at,omitempty" db:"last_seen_at"`
	ResolvedAt        *time.Time      `json:"resolved_at,omitempty" db:"resolved_at"`
	ResolvedBy        *uuid.UUID      `json:"resolved_by,omitempty" db:"resolved_by"`
	ExternalRefs      json.RawMessage `json:"external_references,omitempty" db:"external_references"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
	CreatedBy         *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy         *uuid.UUID      `json:"updated_by,omitempty" db:"updated_by"`
	DeletedAt         *time.Time      `json:"-" db:"deleted_at"`
}

type CampaignEvent struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	TenantID   uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	CampaignID uuid.UUID  `json:"campaign_id" db:"campaign_id"`
	EventID    uuid.UUID  `json:"event_id" db:"event_id"`
	LinkedAt   time.Time  `json:"linked_at" db:"linked_at"`
	LinkedBy   *uuid.UUID `json:"linked_by,omitempty" db:"linked_by"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy  *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy  *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type CampaignIOC struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	CampaignID      uuid.UUID  `json:"campaign_id" db:"campaign_id"`
	IOCType         string     `json:"ioc_type" db:"ioc_type"`
	IOCValue        string     `json:"ioc_value" db:"ioc_value"`
	ConfidenceScore float64    `json:"confidence_score" db:"confidence_score"`
	FirstSeenAt     time.Time  `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt      time.Time  `json:"last_seen_at" db:"last_seen_at"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	SourceID        *uuid.UUID `json:"source_id,omitempty" db:"source_id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy       *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy       *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

// ---------------------------------------------------------------------------
// Brand abuse
// ---------------------------------------------------------------------------

type MonitoredBrand struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	TenantID      uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	BrandName     string     `json:"brand_name" db:"brand_name"`
	DomainPattern *string    `json:"domain_pattern,omitempty" db:"domain_pattern"`
	LogoFileID    *uuid.UUID `json:"logo_file_id,omitempty" db:"logo_file_id"`
	Keywords      []string   `json:"keywords" db:"keywords"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy     *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy     *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type BrandAbuseIncident struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	TenantID            uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	BrandID             uuid.UUID  `json:"brand_id" db:"brand_id"`
	MaliciousDomain     string     `json:"malicious_domain" db:"malicious_domain"`
	AbuseType           string     `json:"abuse_type" db:"abuse_type"`
	RiskLevel           string     `json:"risk_level" db:"risk_level"`
	RegionID            *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	DetectionCount      int        `json:"detection_count" db:"detection_count"`
	SourceID            *uuid.UUID `json:"source_id,omitempty" db:"source_id"`
	WhoisRegistrant     *string    `json:"whois_registrant,omitempty" db:"whois_registrant"`
	WhoisCreatedDate    *string    `json:"whois_created_date,omitempty" db:"whois_created_date"`
	SSLIssuer           *string    `json:"ssl_issuer,omitempty" db:"ssl_issuer"`
	HostingIP           *string    `json:"hosting_ip,omitempty" db:"hosting_ip"`
	HostingASN          *string    `json:"hosting_asn,omitempty" db:"hosting_asn"`
	ScreenshotFileID    *uuid.UUID `json:"screenshot_file_id,omitempty" db:"screenshot_file_id"`
	TakedownStatus      string     `json:"takedown_status" db:"takedown_status"`
	TakedownRequestedAt *time.Time `json:"takedown_requested_at,omitempty" db:"takedown_requested_at"`
	TakenDownAt         *time.Time `json:"taken_down_at,omitempty" db:"taken_down_at"`
	FirstDetectedAt     time.Time  `json:"first_detected_at" db:"first_detected_at"`
	LastDetectedAt      time.Time  `json:"last_detected_at" db:"last_detected_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy           *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy           *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	DeletedAt           *time.Time `json:"-" db:"deleted_at"`
}

// ---------------------------------------------------------------------------
// Aggregation / dashboard
// ---------------------------------------------------------------------------

type GeoThreatSummary struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	TenantID              uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	CountryCode           string     `json:"country_code" db:"country_code"`
	City                  string     `json:"city" db:"city"`
	Latitude              *float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude             *float64   `json:"longitude,omitempty" db:"longitude"`
	RegionID              *uuid.UUID `json:"region_id,omitempty" db:"region_id"`
	SeverityCriticalCount int        `json:"severity_critical_count" db:"severity_critical_count"`
	SeverityHighCount     int        `json:"severity_high_count" db:"severity_high_count"`
	SeverityMediumCount   int        `json:"severity_medium_count" db:"severity_medium_count"`
	SeverityLowCount      int        `json:"severity_low_count" db:"severity_low_count"`
	TotalCount            int        `json:"total_count" db:"total_count"`
	TopCategoryID         *uuid.UUID `json:"top_category_id,omitempty" db:"top_category_id"`
	TopThreatType         *string    `json:"top_threat_type,omitempty" db:"top_threat_type"`
	PeriodStart           time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd             time.Time  `json:"period_end" db:"period_end"`
	ComputedAt            time.Time  `json:"computed_at" db:"computed_at"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy             *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy             *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type SectorThreatSummary struct {
	ID                    uuid.UUID  `json:"id" db:"id"`
	TenantID              uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	SectorID              uuid.UUID  `json:"sector_id" db:"sector_id"`
	SeverityCriticalCount int        `json:"severity_critical_count" db:"severity_critical_count"`
	SeverityHighCount     int        `json:"severity_high_count" db:"severity_high_count"`
	SeverityMediumCount   int        `json:"severity_medium_count" db:"severity_medium_count"`
	SeverityLowCount      int        `json:"severity_low_count" db:"severity_low_count"`
	TotalCount            int        `json:"total_count" db:"total_count"`
	PeriodStart           time.Time  `json:"period_start" db:"period_start"`
	PeriodEnd             time.Time  `json:"period_end" db:"period_end"`
	ComputedAt            time.Time  `json:"computed_at" db:"computed_at"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy             *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy             *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
	// Enriched fields (from JOIN)
	SectorCode  string `json:"sector_code,omitempty" db:"sector_code"`
	SectorLabel string `json:"sector_label,omitempty" db:"sector_label"`
}

type ExecutiveSnapshot struct {
	ID                      uuid.UUID  `json:"id" db:"id"`
	TenantID                uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	TotalEvents24h          int        `json:"total_events_24h" db:"total_events_24h"`
	TotalEvents7d           int        `json:"total_events_7d" db:"total_events_7d"`
	TotalEvents30d          int        `json:"total_events_30d" db:"total_events_30d"`
	ActiveCampaignsCount    int        `json:"active_campaigns_count" db:"active_campaigns_count"`
	CriticalCampaignsCount  int        `json:"critical_campaigns_count" db:"critical_campaigns_count"`
	TotalIOCs               int        `json:"total_iocs" db:"total_iocs"`
	BrandAbuseCriticalCount int        `json:"brand_abuse_critical_count" db:"brand_abuse_critical_count"`
	BrandAbuseTotalCount    int        `json:"brand_abuse_total_count" db:"brand_abuse_total_count"`
	TopTargetedSectorID     *uuid.UUID `json:"top_targeted_sector_id,omitempty" db:"top_targeted_sector_id"`
	TopThreatOriginCountry  *string    `json:"top_threat_origin_country,omitempty" db:"top_threat_origin_country"`
	MeanTimeToDetectHours   *float64   `json:"mean_time_to_detect_hours,omitempty" db:"mean_time_to_detect_hours"`
	MeanTimeToRespondHours  *float64   `json:"mean_time_to_respond_hours,omitempty" db:"mean_time_to_respond_hours"`
	RiskScoreOverall        float64    `json:"risk_score_overall" db:"risk_score_overall"`
	TrendDirection          string     `json:"trend_direction" db:"trend_direction"`
	TrendPercentage         float64    `json:"trend_percentage" db:"trend_percentage"`
	ComputedAt              time.Time  `json:"computed_at" db:"computed_at"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy               *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	UpdatedBy               *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

// ---------------------------------------------------------------------------
// Enriched / joined models for API responses
// ---------------------------------------------------------------------------

type ThreatEventDetail struct {
	ThreatEvent
	SeverityCode  string   `json:"severity_code" db:"severity_code"`
	SeverityLabel string   `json:"severity_label" db:"severity_label"`
	CategoryCode  string   `json:"category_code" db:"category_code"`
	CategoryLabel string   `json:"category_label" db:"category_label"`
	SourceName    string   `json:"source_name" db:"source_name"`
	SectorLabel   string   `json:"sector_label" db:"sector_label"`
	Tags          []string `json:"tags,omitempty"`
}

type CampaignDetail struct {
	Campaign
	ActorName     string `json:"actor_name" db:"actor_name"`
	SeverityCode  string `json:"severity_code" db:"severity_code"`
	SeverityLabel string `json:"severity_label" db:"severity_label"`
}

type BrandAbuseDetail struct {
	BrandAbuseIncident
	BrandName   string `json:"brand_name" db:"brand_name"`
	RegionLabel string `json:"region_label" db:"region_label"`
}
