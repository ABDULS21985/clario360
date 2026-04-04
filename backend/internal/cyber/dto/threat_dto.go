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
	Sort       string   `form:"sort"`
	Order      string   `form:"order"`
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
	Data []*model.Threat `json:"data"`
	Meta PaginationMeta  `json:"meta"`
}

// ThreatStatusUpdateRequest updates a threat status.
type ThreatStatusUpdateRequest struct {
	Status model.ThreatStatus `json:"status" validate:"required"`
}

// CreateThreatIndicatorRequest captures an IOC created alongside a threat.
type CreateThreatIndicatorRequest struct {
	Type        model.IndicatorType `json:"type" validate:"required"`
	Value       string              `json:"value" validate:"required,min=1,max=2048"`
	Description string              `json:"description,omitempty" validate:"omitempty,max=2000"`
	Severity    model.Severity      `json:"severity" validate:"required"`
	Confidence  float64             `json:"confidence" validate:"gte=0,lte=1"`
	Source      string              `json:"source,omitempty" validate:"omitempty,oneof=manual stix_feed osint internal vendor"`
	Tags        []string            `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
}

// CreateThreatRequest creates a new threat record with optional initial IOCs.
type CreateThreatRequest struct {
	Name              string                         `json:"name" validate:"required,min=1,max=255"`
	Type              model.ThreatType               `json:"type" validate:"required"`
	Severity          model.Severity                 `json:"severity" validate:"required"`
	Description       string                         `json:"description,omitempty" validate:"omitempty,max=5000"`
	ThreatActor       string                         `json:"threat_actor,omitempty" validate:"omitempty,max=255"`
	Campaign          string                         `json:"campaign,omitempty" validate:"omitempty,max=255"`
	MITRETacticIDs    []string                       `json:"mitre_tactic_ids,omitempty" validate:"omitempty,max=14,dive,min=1,max=20"`
	MITRETechniqueIDs []string                       `json:"mitre_technique_ids,omitempty" validate:"omitempty,max=100,dive,min=1,max=20"`
	Tags              []string                       `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	Indicators        []CreateThreatIndicatorRequest `json:"indicators,omitempty" validate:"omitempty,max=100,dive"`
}

// UpdateThreatRequest updates the editable fields for a threat.
type UpdateThreatRequest struct {
	Name              string           `json:"name" validate:"required,min=1,max=255"`
	Type              model.ThreatType `json:"type" validate:"required"`
	Severity          model.Severity   `json:"severity" validate:"required"`
	Description       string           `json:"description,omitempty" validate:"omitempty,max=5000"`
	ThreatActor       string           `json:"threat_actor,omitempty" validate:"omitempty,max=255"`
	Campaign          string           `json:"campaign,omitempty" validate:"omitempty,max=255"`
	MITRETacticIDs    []string         `json:"mitre_tactic_ids,omitempty" validate:"omitempty,max=14,dive,min=1,max=20"`
	MITRETechniqueIDs []string         `json:"mitre_technique_ids,omitempty" validate:"omitempty,max=100,dive,min=1,max=20"`
	Tags              []string         `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
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

// StandaloneIndicatorRequest creates or updates an indicator outside a threat detail page.
type StandaloneIndicatorRequest struct {
	Type        model.IndicatorType `json:"type" validate:"required"`
	Value       string              `json:"value" validate:"required,min=1,max=2048"`
	Description string              `json:"description,omitempty" validate:"omitempty,max=2000"`
	Severity    model.Severity      `json:"severity" validate:"required"`
	Source      string              `json:"source" validate:"required,oneof=manual stix_feed osint internal vendor"`
	Confidence  float64             `json:"confidence" validate:"required,gte=0,lte=1"`
	ThreatID    *uuid.UUID          `json:"threat_id,omitempty"`
	ExpiresAt   *time.Time          `json:"expires_at,omitempty"`
	Tags        []string            `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	Metadata    json.RawMessage     `json:"metadata,omitempty"`
}

// ThreatIndicatorStatusUpdateRequest toggles IOC activity without editing the value.
type ThreatIndicatorStatusUpdateRequest struct {
	Active bool `json:"active"`
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
	Payload      json.RawMessage `json:"payload" validate:"required"`
	Source       string          `json:"source,omitempty" validate:"omitempty,oneof=stix_feed vendor internal osint manual"`
	ConflictMode string          `json:"conflict_mode,omitempty" validate:"omitempty,oneof=skip update fail"`
}

// IndicatorBatchRequest creates multiple standalone indicators in one request.
type IndicatorBatchRequest struct {
	Indicators   []StandaloneIndicatorRequest `json:"indicators" validate:"required,min=1,max=1000,dive"`
	ConflictMode string                       `json:"conflict_mode,omitempty" validate:"omitempty,oneof=skip update fail"`
}

// IndicatorBatchResponse summarises the outcome of a batch create.
type IndicatorBatchResponse struct {
	Imported int                    `json:"imported"`
	Skipped  int                    `json:"skipped"`
	Failed   int                    `json:"failed"`
	Errors   []IndicatorBatchError  `json:"errors"`
}

// IndicatorBatchError describes one failed item in a batch import.
type IndicatorBatchError struct {
	Index   int    `json:"index"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// IndicatorListParams captures filters for GET /cyber/indicators.
type IndicatorListParams struct {
	Types         []string   `form:"type"`
	Sources       []string   `form:"source"`
	Severities    []string   `form:"severity"`
	ThreatID      *uuid.UUID `form:"threat_id"`
	Active        *bool      `form:"active"`
	Linked        *bool      `form:"linked"`
	Search        *string    `form:"search"`
	MinConfidence *float64   `form:"min_confidence"`
	MaxConfidence *float64   `form:"max_confidence"`
	Sort          string     `form:"sort"`
	Order         string     `form:"order"`
	Page          int        `form:"page"`
	PerPage       int        `form:"per_page"`
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
	for _, value := range p.Types {
		if !model.IndicatorType(value).IsValid() {
			return fmt.Errorf("invalid indicator type: %q", value)
		}
	}
	for _, value := range p.Sources {
		switch value {
		case "manual", "stix_feed", "osint", "internal", "vendor":
		default:
			return fmt.Errorf("invalid indicator source: %q", value)
		}
	}
	for _, value := range p.Severities {
		if !model.Severity(value).IsValid() || model.Severity(value) == model.SeverityInfo {
			return fmt.Errorf("invalid indicator severity: %q", value)
		}
	}
	if p.MinConfidence != nil && (*p.MinConfidence < 0 || *p.MinConfidence > 1) {
		return fmt.Errorf("invalid min_confidence")
	}
	if p.MaxConfidence != nil && (*p.MaxConfidence < 0 || *p.MaxConfidence > 1) {
		return fmt.Errorf("invalid max_confidence")
	}
	return nil
}

// IndicatorListResponse returns paginated IOC data.
type IndicatorListResponse struct {
	Data []*model.ThreatIndicator `json:"data"`
	Meta PaginationMeta           `json:"meta"`
}

// Normalize ensures all slice fields are non-nil so they serialize as [] not null.
func (r *IndicatorListResponse) Normalize() {
	if r.Data == nil {
		r.Data = []*model.ThreatIndicator{}
	}
	for _, item := range r.Data {
		item.Normalize()
	}
}

// IndicatorStatsResponse returns dashboard metrics for standalone IOC management.
type IndicatorStatsResponse struct {
	Data *model.IndicatorStats `json:"data"`
}

// ThreatTrendPoint returns daily trend counts for dashboard visualizations.
type ThreatTrendPoint struct {
	Date      time.Time `json:"date"`
	Total     int       `json:"total"`
	Active    int       `json:"active"`
	Contained int       `json:"contained"`
}

// ThreatTimelineEntry represents a user-facing threat activity event.
type ThreatTimelineEntry struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Variant     string    `json:"variant,omitempty"`
}
