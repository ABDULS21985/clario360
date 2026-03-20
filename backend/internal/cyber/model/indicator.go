package model

import "time"

// IndicatorStats holds aggregated IOC metrics for dashboards.
type IndicatorStats struct {
	Total        int          `json:"total"`
	Active       int          `json:"active"`
	ExpiringSoon int          `json:"expiring_soon"`
	BySource     []NamedCount `json:"by_source"`
}

// IndicatorEnrichment contains best-effort enrichment details for an IOC.
type IndicatorEnrichment struct {
	DNS             map[string]interface{} `json:"dns,omitempty"`
	Geolocation     map[string]interface{} `json:"geolocation,omitempty"`
	CVEs            []string               `json:"cves,omitempty"`
	WHOIS           map[string]interface{} `json:"whois,omitempty"`
	ReputationScore *float64               `json:"reputation_score,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// IndicatorDetectionMatch represents a recent alert or security event match for an indicator.
type IndicatorDetectionMatch struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Severity    *Severity `json:"severity,omitempty"`
	Status      *string   `json:"status,omitempty"`
	AssetID     *string   `json:"asset_id,omitempty"`
	AssetName   *string   `json:"asset_name,omitempty"`
	MatchField  string    `json:"match_field,omitempty"`
	MatchValue  string    `json:"match_value,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
