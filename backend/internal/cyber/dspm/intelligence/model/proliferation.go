package model

import (
	"time"

	"github.com/google/uuid"
)

// ProliferationStatus tracks whether sensitive data spread is authorized.
type ProliferationStatus string

const (
	ProliferationAuthorized   ProliferationStatus = "authorized"
	ProliferationUnauthorized ProliferationStatus = "unauthorized"
	ProliferationUnderReview  ProliferationStatus = "under_review"
)

// DataProliferation tracks how sensitive data has spread across systems.
type DataProliferation struct {
	AssetID            uuid.UUID     `json:"asset_id"`
	AssetName          string        `json:"asset_name"`
	Classification     string        `json:"classification"`
	PIITypes           []string      `json:"pii_types"`
	TotalCopies        int           `json:"total_copies"`
	AuthorizedCopies   int           `json:"authorized_copies"`
	UnauthorizedCopies int           `json:"unauthorized_copies"`
	SpreadEvents       []SpreadEvent `json:"spread_events"`
	FirstDetectedAt    time.Time     `json:"first_detected_at"`
	LastDetectedAt     time.Time     `json:"last_detected_at"`
}

// SpreadEvent records a single proliferation occurrence.
type SpreadEvent struct {
	ID              uuid.UUID           `json:"id"`
	SourceAssetID   uuid.UUID           `json:"source_asset_id"`
	SourceAssetName string              `json:"source_asset_name"`
	TargetAssetID   uuid.UUID           `json:"target_asset_id"`
	TargetAssetName string              `json:"target_asset_name"`
	EdgeType        string              `json:"edge_type"`
	Status          ProliferationStatus `json:"status"`
	DetectedAt      time.Time           `json:"detected_at"`
	Similarity      float64             `json:"similarity"`
}

// ProliferationOverview summarizes data spread across the tenant.
type ProliferationOverview struct {
	TotalSensitiveAssets    int                   `json:"total_sensitive_assets"`
	AssetsWithCopies        int                   `json:"assets_with_copies"`
	TotalUnauthorizedCopies int                   `json:"total_unauthorized_copies"`
	SpreadTrend             []SpreadTrendPoint    `json:"spread_trend"`
	TopProliferators        []DataProliferation   `json:"top_proliferators"`
}

// SpreadTrendPoint is a point in the proliferation trend line.
type SpreadTrendPoint struct {
	Date            string `json:"date"`
	TotalCopies     int    `json:"total_copies"`
	NewCopies       int    `json:"new_copies"`
	UnauthorizedNew int    `json:"unauthorized_new"`
}

// ClassificationDrift tracks classification changes over time for an asset.
type ClassificationDrift struct {
	AssetID     uuid.UUID    `json:"asset_id"`
	AssetName   string       `json:"asset_name"`
	DriftEvents []DriftEvent `json:"drift_events"`
}

// DriftEvent is a single classification drift occurrence.
type DriftEvent struct {
	OldClassification string    `json:"old_classification"`
	NewClassification string    `json:"new_classification"`
	ChangeType        string    `json:"change_type"`
	DetectedAt        time.Time `json:"detected_at"`
	Confidence        float64   `json:"confidence"`
}
