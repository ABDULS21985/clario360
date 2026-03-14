package dto

import "github.com/google/uuid"

// EnhancedClassificationResponse wraps classification results.
type EnhancedClassificationResponse struct {
	Classifications []ClassificationResult `json:"classifications"`
	TotalAssets     int                    `json:"total_assets"`
	NeedingReview   int                    `json:"needing_review"`
	AvgConfidence   float64                `json:"avg_confidence"`
}

// ClassificationResult is a single asset classification output.
type ClassificationResult struct {
	AssetID          uuid.UUID `json:"asset_id"`
	AssetName        string    `json:"asset_name"`
	Classification   string    `json:"classification"`
	PIITypes         []string  `json:"pii_types"`
	Confidence       float64   `json:"confidence"`
	NeedsHumanReview bool      `json:"needs_human_review"`
	DetectedBy       string    `json:"detected_by"`
	Explanation      string    `json:"explanation"`
}

// CreateCustomRuleRequest defines a tenant-defined classification rule.
type CreateCustomRuleRequest struct {
	Name           string   `json:"name" validate:"required"`
	ColumnPatterns []string `json:"column_patterns" validate:"required,min=1"`
	ValuePattern   string   `json:"value_pattern,omitempty"`
	Classification string   `json:"classification" validate:"required"`
	PIIType        string   `json:"pii_type,omitempty"`
}

// ClassificationHistoryParams controls classification history queries.
type ClassificationHistoryParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// SetDefaults applies default values to classification history params.
func (p *ClassificationHistoryParams) SetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 25
	}
}
