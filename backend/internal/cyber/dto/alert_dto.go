package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// AlertListParams captures all filters supported by GET /cyber/alerts.
type AlertListParams struct {
	Search           *string    `form:"search"`
	Severities       []string   `form:"severity"`
	Statuses         []string   `form:"status"`
	AssignedTo       *uuid.UUID `form:"assigned_to"`
	Unassigned       *bool      `form:"unassigned"`
	AssetID          *uuid.UUID `form:"asset_id"`
	RuleID           *uuid.UUID `form:"rule_id"`
	MITRETechniqueID *string    `form:"mitre_technique_id"`
	MITRETacticID    *string    `form:"mitre_tactic_id"`
	MinConfidence    *float64   `form:"min_confidence"`
	Tags             []string   `form:"tag"`
	DateFrom         *time.Time `form:"date_from"`
	DateTo           *time.Time `form:"date_to"`
	Sort             string     `form:"sort"`
	Order            string     `form:"order"`
	Page             int        `form:"page"`
	PerPage          int        `form:"per_page"`
}

// SetDefaults applies defaults to list parameters.
func (p *AlertListParams) SetDefaults() {
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
}

// Validate validates filter parameters.
func (p *AlertListParams) Validate() error {
	for _, sev := range p.Severities {
		if !model.Severity(sev).IsValid() {
			return fmt.Errorf("invalid severity: %q", sev)
		}
	}
	for _, status := range p.Statuses {
		if !model.AlertStatus(status).IsValid() {
			return fmt.Errorf("invalid status: %q", status)
		}
	}
	if p.MinConfidence != nil && (*p.MinConfidence < 0 || *p.MinConfidence > 1) {
		return fmt.Errorf("min_confidence must be between 0 and 1")
	}
	switch p.Sort {
	case "", "severity", "confidence_score", "created_at", "event_count", "status":
	default:
		return fmt.Errorf("invalid sort: %q", p.Sort)
	}
	switch p.Order {
	case "", "asc", "desc":
	default:
		return fmt.Errorf("invalid order: %q", p.Order)
	}
	return nil
}

// AlertListResponse is the paginated response for GET /cyber/alerts.
type AlertListResponse struct {
	Data []*model.Alert `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// PaginationMeta is the canonical pagination envelope for cyber list endpoints.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// NewPaginationMeta builds pagination metadata from the current request values.
func NewPaginationMeta(page, perPage, total int) PaginationMeta {
	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationMeta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// AlertStatusUpdateRequest updates the status of an alert.
type AlertStatusUpdateRequest struct {
	Status model.AlertStatus `json:"status" validate:"required"`
	Notes  *string           `json:"notes,omitempty" validate:"omitempty,max=4000"`
	Reason *string           `json:"reason,omitempty" validate:"omitempty,max=1000"`
}

// AlertAssignRequest assigns or reassigns an alert.
type AlertAssignRequest struct {
	AssignedTo uuid.UUID `json:"assigned_to" validate:"required"`
}

// AlertEscalateRequest escalates an alert.
type AlertEscalateRequest struct {
	EscalatedTo uuid.UUID `json:"escalated_to" validate:"required"`
	Reason      string    `json:"reason" validate:"required,min=3,max=1000"`
}

// AlertCommentRequest adds an investigation comment.
type AlertCommentRequest struct {
	Content  string          `json:"content" validate:"required,min=1,max=4000"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// AlertMergeRequest merges related alerts into the target alert.
type AlertMergeRequest struct {
	MergeIDs []uuid.UUID `json:"merge_ids" validate:"required,min=1,max=25,dive,required"`
}

// AlertCountResponse returns a simple count for KPI cards.
type AlertCountResponse struct {
	Count int `json:"count"`
}
