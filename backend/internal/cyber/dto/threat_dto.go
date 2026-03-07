package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// ThreatListParams captures threat list filters.
type ThreatListParams struct {
	Search     *string  `form:"search"`
	Types      []string `form:"type"`
	Statuses   []string `form:"status"`
	Severities []string `form:"severity"`
	Page       int      `form:"page"`
	PerPage    int      `form:"per_page"`
}

// SetDefaults applies default paging.
func (p *ThreatListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
}

// Validate validates threat list filters.
func (p *ThreatListParams) Validate() error {
	for _, v := range p.Types {
		if !model.ThreatType(v).IsValid() {
			return fmt.Errorf("invalid threat type: %q", v)
		}
	}
	for _, v := range p.Statuses {
		if !model.ThreatStatus(v).IsValid() {
			return fmt.Errorf("invalid threat status: %q", v)
		}
	}
	for _, v := range p.Severities {
		if !model.Severity(v).IsValid() || model.Severity(v) == model.SeverityInfo {
			return fmt.Errorf("invalid threat severity: %q", v)
		}
	}
	return nil
}

// ThreatListResponse returns paginated threat data.
type ThreatListResponse struct {
	Data       []*model.Threat `json:"data"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"per_page"`
	TotalPages int             `json:"total_pages"`
}

// ThreatStatusUpdateRequest updates a threat status.
type ThreatStatusUpdateRequest struct {
	Status model.ThreatStatus `json:"status" validate:"required"`
}

// ThreatIndicatorRequest creates or adds an indicator to a threat.
type ThreatIndicatorRequest struct {
	Type        model.IndicatorType `json:"type" validate:"required"`
	Value       string              `json:"value" validate:"required,min=1,max=2048"`
	Description string              `json:"description,omitempty" validate:"omitempty,max=2000"`
	Severity    model.Severity      `json:"severity" validate:"required"`
	Source      string              `json:"source" validate:"required,oneof=manual stix_feed osint internal vendor"`
	Confidence  float64             `json:"confidence" validate:"required,gte=0,lte=1"`
	ExpiresAt   *time.Time          `json:"expires_at,omitempty"`
	Tags        []string            `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	Metadata    json.RawMessage     `json:"metadata,omitempty"`
}

// IndicatorCheckRequest checks arbitrary indicator values against stored IOCs.
type IndicatorCheckRequest struct {
	Values []string `json:"values" validate:"required,min=1,max=500,dive,min=1,max=2048"`
}

// IndicatorCheckResult returns indicator matches for arbitrary values.
type IndicatorCheckResult struct {
	Value      string                   `json:"value"`
	Indicators []*model.ThreatIndicator `json:"indicators"`
}

// IndicatorBulkImportRequest imports STIX/TAXII indicator bundles.
type IndicatorBulkImportRequest struct {
	Payload json.RawMessage `json:"payload" validate:"required"`
	Source  string          `json:"source,omitempty" validate:"omitempty,oneof=stix_feed vendor internal osint manual"`
}

// IndicatorListParams captures filters for GET /cyber/indicators.
type IndicatorListParams struct {
	Type     *string    `form:"type"`
	ThreatID *uuid.UUID `form:"threat_id"`
	Active   *bool      `form:"active"`
	Search   *string    `form:"search"`
	Page     int        `form:"page"`
	PerPage  int        `form:"per_page"`
}

// SetDefaults applies default paging.
func (p *IndicatorListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
}

// Validate validates indicator filters.
func (p *IndicatorListParams) Validate() error {
	if p.Type != nil && !model.IndicatorType(*p.Type).IsValid() {
		return fmt.Errorf("invalid indicator type: %q", *p.Type)
	}
	return nil
}

// IndicatorListResponse returns paginated IOC data.
type IndicatorListResponse struct {
	Data       []*model.ThreatIndicator `json:"data"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"per_page"`
	TotalPages int                      `json:"total_pages"`
}
