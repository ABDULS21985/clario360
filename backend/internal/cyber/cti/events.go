package cti

import (
	"encoding/json"
	"time"
)

// ---------------------------------------------------------------------------
// CTI Kafka Topics
// ---------------------------------------------------------------------------

const (
	TopicCTIThreatEvents        = "cyber.cti.threat-events"
	TopicCTICampaigns           = "cyber.cti.campaigns"
	TopicCTIBrandAbuse          = "cyber.cti.brand-abuse"
	TopicCTIFeedIngestion       = "cyber.cti.feed-ingestion"
	TopicCTIAggregationTriggers = "cyber.cti.aggregation-triggers"
	TopicCTIAlerts              = "cyber.cti.alerts"
	TopicCTIDLQ                 = "cyber.cti.dlq"
)

// AllCTITopics returns all CTI topic names for admin/setup tooling.
func AllCTITopics() []string {
	return []string{
		TopicCTIThreatEvents,
		TopicCTICampaigns,
		TopicCTIBrandAbuse,
		TopicCTIFeedIngestion,
		TopicCTIAggregationTriggers,
		TopicCTIAlerts,
		TopicCTIDLQ,
	}
}

// ---------------------------------------------------------------------------
// CloudEvents `type` field values
// These are passed to events.NewEvent which normalizes them to
// "com.clario360.cyber.cti.threat-event.created" etc.
// ---------------------------------------------------------------------------

const (
	// Threat events
	EventThreatEventCreated       = "cyber.cti.threat-event.created"
	EventThreatEventUpdated       = "cyber.cti.threat-event.updated"
	EventThreatEventResolved      = "cyber.cti.threat-event.resolved"
	EventThreatEventFalsePositive = "cyber.cti.threat-event.false-positive"
	EventThreatEventDeleted       = "cyber.cti.threat-event.deleted"

	// Campaigns
	EventCampaignCreated       = "cyber.cti.campaign.created"
	EventCampaignUpdated       = "cyber.cti.campaign.updated"
	EventCampaignStatusChanged = "cyber.cti.campaign.status-changed"
	EventCampaignIOCAdded      = "cyber.cti.campaign.ioc-added"
	EventCampaignEventLinked   = "cyber.cti.campaign.event-linked"

	// Brand abuse
	EventBrandAbuseDetected        = "cyber.cti.brand-abuse.detected"
	EventBrandAbuseTakedownChanged = "cyber.cti.brand-abuse.takedown-changed"
	EventBrandAbuseUpdated         = "cyber.cti.brand-abuse.updated"

	// Feed ingestion
	EventFeedRawIngested = "cyber.cti.feed.raw-ingested"
	EventFeedNormalized  = "cyber.cti.feed.normalized"

	// Aggregation
	EventAggregationTriggered = "cyber.cti.aggregation.triggered"

	// Alerts
	EventCriticalThreatAlert = "cyber.cti.alert.critical-threat"
	EventCampaignEscalation  = "cyber.cti.alert.campaign-escalation"
	EventBrandAbuseUrgent    = "cyber.cti.alert.brand-abuse-urgent"
)

// ---------------------------------------------------------------------------
// Event payload structs
// ---------------------------------------------------------------------------

// ThreatEventPayload is published when a threat event is created/updated/resolved.
type ThreatEventPayload struct {
	EventID         string    `json:"event_id"`
	TenantID        string    `json:"tenant_id"`
	EventType       string    `json:"event_type"`
	Title           string    `json:"title"`
	SeverityCode    string    `json:"severity_code"`
	CategoryCode    string    `json:"category_code,omitempty"`
	ConfidenceScore float64   `json:"confidence_score"`
	OriginCountry   string    `json:"origin_country,omitempty"`
	OriginCity      string    `json:"origin_city,omitempty"`
	TargetSector    string    `json:"target_sector,omitempty"`
	IOCType         string    `json:"ioc_type,omitempty"`
	IOCValue        string    `json:"ioc_value,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// CampaignPayload is published for campaign lifecycle events.
type CampaignPayload struct {
	CampaignID   string `json:"campaign_id"`
	TenantID     string `json:"tenant_id"`
	CampaignCode string `json:"campaign_code"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	SeverityCode string `json:"severity_code"`
	ActorName    string `json:"actor_name,omitempty"`
	IOCCount     int    `json:"ioc_count"`
	EventCount   int    `json:"event_count"`
}

// BrandAbusePayload is published for brand abuse incident events.
type BrandAbusePayload struct {
	IncidentID      string `json:"incident_id"`
	TenantID        string `json:"tenant_id"`
	BrandName       string `json:"brand_name"`
	MaliciousDomain string `json:"malicious_domain"`
	AbuseType       string `json:"abuse_type"`
	RiskLevel       string `json:"risk_level"`
	TakedownStatus  string `json:"takedown_status"`
}

// FeedIngestionPayload wraps raw data from an external feed for the ingestion pipeline.
type FeedIngestionPayload struct {
	SourceID   string          `json:"source_id"`
	SourceName string          `json:"source_name"`
	SourceType string          `json:"source_type"`
	TenantID   string          `json:"tenant_id"`
	RawData    json.RawMessage `json:"raw_data"`
	ReceivedAt time.Time       `json:"received_at"`
}

// AlertPayload is published for high-priority CTI alerts destined for the notification service.
type AlertPayload struct {
	AlertType    string `json:"alert_type"`
	TenantID     string `json:"tenant_id"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SeverityCode string `json:"severity_code"`
	SourceEntity string `json:"source_entity"`
	SourceID     string `json:"source_id"`
	ActionURL    string `json:"action_url,omitempty"`
}
