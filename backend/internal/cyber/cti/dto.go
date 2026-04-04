package cti

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Request DTOs
// ---------------------------------------------------------------------------

type CreateThreatEventRequest struct {
	EventType         string           `json:"event_type" validate:"required,oneof=indicator_sighting attack_attempt vulnerability_exploit malware_detection anomaly policy_violation"`
	Title             string           `json:"title" validate:"required,max=500"`
	Description       *string          `json:"description,omitempty"`
	SeverityCode      string           `json:"severity_code" validate:"required,oneof=critical high medium low informational"`
	CategoryCode      *string          `json:"category_code,omitempty"`
	SourceName        *string          `json:"source_name,omitempty"`
	SourceReference   *string          `json:"source_reference,omitempty"`
	ConfidenceScore   float64          `json:"confidence_score" validate:"gte=0,lte=1"`
	OriginLatitude    *float64         `json:"origin_latitude,omitempty"`
	OriginLongitude   *float64         `json:"origin_longitude,omitempty"`
	OriginCountryCode *string          `json:"origin_country_code,omitempty"`
	OriginCity        *string          `json:"origin_city,omitempty"`
	TargetSectorCode  *string          `json:"target_sector_code,omitempty"`
	TargetOrgName     *string          `json:"target_org_name,omitempty"`
	TargetCountryCode *string          `json:"target_country_code,omitempty"`
	IOCType           *string          `json:"ioc_type,omitempty"`
	IOCValue          *string          `json:"ioc_value,omitempty"`
	MitreTechniqueIDs []string         `json:"mitre_technique_ids,omitempty"`
	RawPayload        json.RawMessage  `json:"raw_payload,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
	FirstSeenAt       *time.Time       `json:"first_seen_at,omitempty"`
}

type UpdateThreatEventRequest struct {
	Title             *string          `json:"title,omitempty"`
	Description       *string          `json:"description,omitempty"`
	SeverityCode      *string          `json:"severity_code,omitempty"`
	CategoryCode      *string          `json:"category_code,omitempty"`
	ConfidenceScore   *float64         `json:"confidence_score,omitempty"`
	OriginCountryCode *string          `json:"origin_country_code,omitempty"`
	OriginCity        *string          `json:"origin_city,omitempty"`
	TargetSectorCode  *string          `json:"target_sector_code,omitempty"`
	TargetCountryCode *string          `json:"target_country_code,omitempty"`
	IOCType           *string          `json:"ioc_type,omitempty"`
	IOCValue          *string          `json:"ioc_value,omitempty"`
	MitreTechniqueIDs []string         `json:"mitre_technique_ids,omitempty"`
	Tags              []string         `json:"tags,omitempty"`
}

type CreateThreatActorRequest struct {
	Name                string           `json:"name" validate:"required,max=300"`
	Aliases             []string         `json:"aliases,omitempty"`
	ActorType           string           `json:"actor_type" validate:"required,oneof=state_sponsored cybercriminal hacktivist insider unknown"`
	OriginCountryCode   *string          `json:"origin_country_code,omitempty"`
	SophisticationLevel string           `json:"sophistication_level" validate:"required,oneof=advanced intermediate basic"`
	PrimaryMotivation   string           `json:"primary_motivation" validate:"required,oneof=espionage financial_gain disruption ideological unknown"`
	Description         *string          `json:"description,omitempty"`
	MitreGroupID        *string          `json:"mitre_group_id,omitempty"`
	ExternalReferences  json.RawMessage  `json:"external_references,omitempty"`
	RiskScore           float64          `json:"risk_score" validate:"gte=0,lte=100"`
}

type UpdateThreatActorRequest struct {
	Name                *string          `json:"name,omitempty"`
	Aliases             []string         `json:"aliases,omitempty"`
	ActorType           *string          `json:"actor_type,omitempty"`
	OriginCountryCode   *string          `json:"origin_country_code,omitempty"`
	SophisticationLevel *string          `json:"sophistication_level,omitempty"`
	PrimaryMotivation   *string          `json:"primary_motivation,omitempty"`
	Description         *string          `json:"description,omitempty"`
	MitreGroupID        *string          `json:"mitre_group_id,omitempty"`
	RiskScore           *float64         `json:"risk_score,omitempty"`
	IsActive            *bool            `json:"is_active,omitempty"`
}

type CreateCampaignRequest struct {
	CampaignCode      string    `json:"campaign_code" validate:"required,max=50"`
	Name              string    `json:"name" validate:"required,max=300"`
	Description       *string   `json:"description,omitempty"`
	Status            string    `json:"status" validate:"required,oneof=active monitoring dormant resolved archived"`
	SeverityCode      string    `json:"severity_code" validate:"required,oneof=critical high medium low informational"`
	PrimaryActorID    *string   `json:"primary_actor_id,omitempty"`
	TargetSectors     []string  `json:"target_sectors,omitempty"`
	TargetRegions     []string  `json:"target_regions,omitempty"`
	TargetDescription *string   `json:"target_description,omitempty"`
	MitreTechniqueIDs []string  `json:"mitre_technique_ids,omitempty"`
	TTPsSummary       *string   `json:"ttps_summary,omitempty"`
	FirstSeenAt       time.Time `json:"first_seen_at" validate:"required"`
}

type UpdateCampaignRequest struct {
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	SeverityCode      *string  `json:"severity_code,omitempty"`
	PrimaryActorID    *string  `json:"primary_actor_id,omitempty"`
	TargetSectors     []string `json:"target_sectors,omitempty"`
	TargetRegions     []string `json:"target_regions,omitempty"`
	TargetDescription *string  `json:"target_description,omitempty"`
	MitreTechniqueIDs []string `json:"mitre_technique_ids,omitempty"`
	TTPsSummary       *string  `json:"ttps_summary,omitempty"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" validate:"required"`
}

type CreateMonitoredBrandRequest struct {
	BrandName     string   `json:"brand_name" validate:"required,max=300"`
	DomainPattern *string  `json:"domain_pattern,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
}

type UpdateMonitoredBrandRequest struct {
	BrandName     *string  `json:"brand_name,omitempty"`
	DomainPattern *string  `json:"domain_pattern,omitempty"`
	Keywords      []string `json:"keywords,omitempty"`
	IsActive      *bool    `json:"is_active,omitempty"`
}

type CreateBrandAbuseIncidentRequest struct {
	BrandID         string  `json:"brand_id" validate:"required"`
	MaliciousDomain string  `json:"malicious_domain" validate:"required,max=500"`
	AbuseType       string  `json:"abuse_type" validate:"required"`
	RiskLevel       string  `json:"risk_level" validate:"required,oneof=critical high medium low"`
	SourceName      *string `json:"source_name,omitempty"`
	WhoisRegistrant *string `json:"whois_registrant,omitempty"`
	SSLIssuer       *string `json:"ssl_issuer,omitempty"`
	HostingIP       *string `json:"hosting_ip,omitempty"`
	HostingASN      *string `json:"hosting_asn,omitempty"`
}

type UpdateTakedownStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=detected reported takedown_requested taken_down monitoring false_positive"`
}

type CreateCampaignIOCRequest struct {
	IOCType         string  `json:"ioc_type" validate:"required"`
	IOCValue        string  `json:"ioc_value" validate:"required"`
	ConfidenceScore float64 `json:"confidence_score" validate:"gte=0,lte=1"`
	SourceName      *string `json:"source_name,omitempty"`
}

type AddTagsRequest struct {
	Tags []string `json:"tags" validate:"required,min=1,max=20"`
}

// ---------------------------------------------------------------------------
// Response DTOs
// ---------------------------------------------------------------------------

type ListResponse[T any] struct {
	Data []T            `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type GlobalThreatMapResponse struct {
	Hotspots    []GeoThreatSummary `json:"hotspots"`
	TotalEvents int64              `json:"total_events"`
	Period      string             `json:"period"`
}

type SectorThreatResponse struct {
	Sectors []SectorThreatSummary `json:"sectors"`
	Period  string                `json:"period"`
}

type ExecutiveDashboardResponse struct {
	Snapshot       ExecutiveSnapshot       `json:"snapshot"`
	TopCampaigns   []CampaignDetail        `json:"top_campaigns"`
	CriticalBrands []BrandAbuseDetail      `json:"critical_brands"`
	TopSectors     []SectorThreatSummary   `json:"top_sectors"`
	RecentEvents   []ThreatEventDetail     `json:"recent_events"`
}

// ---------------------------------------------------------------------------
// Helpers for uuid arrays
// ---------------------------------------------------------------------------

func ParseUUIDs(strs []string) ([]uuid.UUID, error) {
	if len(strs) == 0 {
		return nil, nil
	}
	out := make([]uuid.UUID, 0, len(strs))
	for _, s := range strs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
