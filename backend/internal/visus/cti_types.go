package visus

type CTIExecutiveDashboardResponse struct {
	Snapshot       CTIExecutiveSnapshot    `json:"snapshot"`
	TopCampaigns   []CTICampaignSummary    `json:"top_campaigns"`
	CriticalBrands []CTIBrandAbuseSummary  `json:"critical_brands"`
	TopSectors     []CTISectorSummary      `json:"top_sectors"`
	RecentEvents   []CTIThreatEventSummary `json:"recent_events"`
}

type CTIExecutiveSnapshot struct {
	TenantID                string   `json:"tenant_id"`
	TotalEvents24h          int64    `json:"total_events_24h"`
	TotalEvents7d           int64    `json:"total_events_7d"`
	TotalEvents30d          int64    `json:"total_events_30d"`
	ActiveCampaignsCount    int      `json:"active_campaigns_count"`
	CriticalCampaignsCount  int      `json:"critical_campaigns_count"`
	TotalIOCs               int64    `json:"total_iocs"`
	BrandAbuseCriticalCount int      `json:"brand_abuse_critical_count"`
	BrandAbuseTotalCount    int      `json:"brand_abuse_total_count"`
	TopTargetedSectorID     *string  `json:"top_targeted_sector_id,omitempty"`
	TopThreatOriginCountry  *string  `json:"top_threat_origin_country,omitempty"`
	MeanTimeToDetectHours   *float64 `json:"mean_time_to_detect_hours,omitempty"`
	MeanTimeToRespondHours  *float64 `json:"mean_time_to_respond_hours,omitempty"`
	RiskScoreOverall        float64  `json:"risk_score_overall"`
	TrendDirection          string   `json:"trend_direction"`
	TrendPercentage         float64  `json:"trend_percentage"`
	ComputedAt              string   `json:"computed_at"`
}

type CTICampaignSummary struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	CampaignCode      string         `json:"campaign_code"`
	Name              string         `json:"name"`
	Description       *string        `json:"description"`
	Status            string         `json:"status"`
	SeverityID        *string        `json:"severity_id"`
	PrimaryActorID    *string        `json:"primary_actor_id"`
	TargetSectors     []string       `json:"target_sectors"`
	TargetRegions     []string       `json:"target_regions"`
	TargetDescription *string        `json:"target_description"`
	MitreTechniqueIDs []string       `json:"mitre_technique_ids"`
	TTPsSummary       *string        `json:"ttps_summary"`
	IOCCount          int            `json:"ioc_count"`
	EventCount        int            `json:"event_count"`
	FirstSeenAt       string         `json:"first_seen_at"`
	LastSeenAt        *string        `json:"last_seen_at"`
	ResolvedAt        *string        `json:"resolved_at"`
	ResolvedBy        *string        `json:"resolved_by"`
	ExternalRefs      map[string]any `json:"external_references,omitempty"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	CreatedBy         *string        `json:"created_by"`
	UpdatedBy         *string        `json:"updated_by"`
	ActorName         string         `json:"actor_name"`
	SeverityCode      string         `json:"severity_code"`
	SeverityLabel     string         `json:"severity_label"`
}

type CTIBrandAbuseSummary struct {
	ID                  string  `json:"id"`
	TenantID            string  `json:"tenant_id"`
	BrandID             string  `json:"brand_id"`
	MaliciousDomain     string  `json:"malicious_domain"`
	AbuseType           string  `json:"abuse_type"`
	RiskLevel           string  `json:"risk_level"`
	RegionID            *string `json:"region_id"`
	DetectionCount      int     `json:"detection_count"`
	SourceID            *string `json:"source_id"`
	WhoisRegistrant     *string `json:"whois_registrant"`
	WhoisCreatedDate    *string `json:"whois_created_date"`
	SSLIssuer           *string `json:"ssl_issuer"`
	HostingIP           *string `json:"hosting_ip"`
	HostingASN          *string `json:"hosting_asn"`
	ScreenshotFileID    *string `json:"screenshot_file_id"`
	TakedownStatus      string  `json:"takedown_status"`
	TakedownRequestedAt *string `json:"takedown_requested_at"`
	TakenDownAt         *string `json:"taken_down_at"`
	FirstDetectedAt     string  `json:"first_detected_at"`
	LastDetectedAt      string  `json:"last_detected_at"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	CreatedBy           *string `json:"created_by"`
	UpdatedBy           *string `json:"updated_by"`
	BrandName           string  `json:"brand_name"`
	RegionLabel         string  `json:"region_label"`
}

type CTISectorSummary struct {
	ID                    string  `json:"id"`
	TenantID              string  `json:"tenant_id"`
	SectorID              string  `json:"sector_id"`
	SeverityCriticalCount int     `json:"severity_critical_count"`
	SeverityHighCount     int     `json:"severity_high_count"`
	SeverityMediumCount   int     `json:"severity_medium_count"`
	SeverityLowCount      int     `json:"severity_low_count"`
	TotalCount            int     `json:"total_count"`
	PeriodStart           string  `json:"period_start"`
	PeriodEnd             string  `json:"period_end"`
	ComputedAt            string  `json:"computed_at"`
	CreatedAt             string  `json:"created_at"`
	UpdatedAt             string  `json:"updated_at"`
	CreatedBy             *string `json:"created_by"`
	UpdatedBy             *string `json:"updated_by"`
	SectorCode            string  `json:"sector_code"`
	SectorLabel           string  `json:"sector_label"`
}

type CTIThreatEventSummary struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	EventType         string         `json:"event_type"`
	Title             string         `json:"title"`
	Description       *string        `json:"description"`
	SeverityID        *string        `json:"severity_id"`
	CategoryID        *string        `json:"category_id"`
	SourceID          *string        `json:"source_id"`
	SourceReference   *string        `json:"source_reference"`
	ConfidenceScore   float64        `json:"confidence_score"`
	OriginLatitude    *float64       `json:"origin_latitude"`
	OriginLongitude   *float64       `json:"origin_longitude"`
	OriginCountryCode *string        `json:"origin_country_code"`
	OriginCity        *string        `json:"origin_city"`
	OriginRegionID    *string        `json:"origin_region_id"`
	TargetSectorID    *string        `json:"target_sector_id"`
	TargetOrgName     *string        `json:"target_org_name"`
	TargetCountryCode *string        `json:"target_country_code"`
	IOCType           *string        `json:"ioc_type"`
	IOCValue          *string        `json:"ioc_value"`
	MitreTechniqueIDs []string       `json:"mitre_technique_ids"`
	RawPayload        map[string]any `json:"raw_payload,omitempty"`
	IsFalsePositive   bool           `json:"is_false_positive"`
	ResolvedAt        *string        `json:"resolved_at"`
	ResolvedBy        *string        `json:"resolved_by"`
	FirstSeenAt       string         `json:"first_seen_at"`
	LastSeenAt        string         `json:"last_seen_at"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
	CreatedBy         *string        `json:"created_by"`
	UpdatedBy         *string        `json:"updated_by"`
	SeverityCode      string         `json:"severity_code"`
	SeverityLabel     string         `json:"severity_label"`
	CategoryCode      string         `json:"category_code"`
	CategoryLabel     string         `json:"category_label"`
	SourceName        string         `json:"source_name"`
	SectorLabel       string         `json:"sector_label"`
	Tags              []string       `json:"tags"`
}

type CTIGlobalThreatMapResponse struct {
	Hotspots    []CTIGeoThreatHotspot `json:"hotspots"`
	TotalEvents int64                 `json:"total_events"`
	Period      string                `json:"period"`
}

type CTIGeoThreatHotspot struct {
	ID                    string   `json:"id"`
	TenantID              string   `json:"tenant_id"`
	CountryCode           string   `json:"country_code"`
	City                  string   `json:"city"`
	Latitude              *float64 `json:"latitude"`
	Longitude             *float64 `json:"longitude"`
	RegionID              *string  `json:"region_id"`
	SeverityCriticalCount int      `json:"severity_critical_count"`
	SeverityHighCount     int      `json:"severity_high_count"`
	SeverityMediumCount   int      `json:"severity_medium_count"`
	SeverityLowCount      int      `json:"severity_low_count"`
	TotalCount            int      `json:"total_count"`
	TopCategoryID         *string  `json:"top_category_id"`
	TopThreatType         *string  `json:"top_threat_type"`
	PeriodStart           string   `json:"period_start"`
	PeriodEnd             string   `json:"period_end"`
	ComputedAt            string   `json:"computed_at"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
	CreatedBy             *string  `json:"created_by"`
	UpdatedBy             *string  `json:"updated_by"`
}

type CTISectorThreatResponse struct {
	Sectors []CTISectorSummary `json:"sectors"`
	Period  string             `json:"period"`
}

type CTIPaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type CTICampaignListResponse struct {
	Data []CTICampaignSummary `json:"data"`
	Meta CTIPaginationMeta    `json:"meta"`
}

type CTIBrandAbuseListResponse struct {
	Data []CTIBrandAbuseSummary `json:"data"`
	Meta CTIPaginationMeta      `json:"meta"`
}

type CTIActorListResponse struct {
	Data []CTIActorSummary `json:"data"`
	Meta CTIPaginationMeta `json:"meta"`
}

type CTIActorSummary struct {
	ID                  string         `json:"id"`
	TenantID            string         `json:"tenant_id"`
	Name                string         `json:"name"`
	Aliases             []string       `json:"aliases"`
	ActorType           string         `json:"actor_type"`
	OriginCountryCode   *string        `json:"origin_country_code"`
	OriginRegionID      *string        `json:"origin_region_id"`
	SophisticationLevel string         `json:"sophistication_level"`
	PrimaryMotivation   string         `json:"primary_motivation"`
	Description         *string        `json:"description"`
	FirstObservedAt     *string        `json:"first_observed_at"`
	LastActivityAt      *string        `json:"last_activity_at"`
	MitreGroupID        *string        `json:"mitre_group_id"`
	ExternalReferences  map[string]any `json:"external_references,omitempty"`
	IsActive            bool           `json:"is_active"`
	RiskScore           float64        `json:"risk_score"`
	CreatedAt           string         `json:"created_at"`
	UpdatedAt           string         `json:"updated_at"`
	CreatedBy           *string        `json:"created_by"`
	UpdatedBy           *string        `json:"updated_by"`
}

type CTIRiskScoreResponse struct {
	RiskScore       float64 `json:"risk_score"`
	TrendDirection  string  `json:"trend_direction"`
	TrendPercentage float64 `json:"trend_percentage"`
	TotalEvents24h  int64   `json:"total_events_24h"`
	MTTDHours       float64 `json:"mttd_hours"`
	MTTRHours       float64 `json:"mttr_hours"`
	ComputedAt      string  `json:"computed_at"`
}
