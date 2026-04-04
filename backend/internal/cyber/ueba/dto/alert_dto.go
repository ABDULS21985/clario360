package dto

import (
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type AlertListParams struct {
	Page     int    `json:"page"`
	PerPage  int    `json:"per_page"`
	EntityID string `json:"entity_id,omitempty"`
	Status   string `json:"status,omitempty"`
}

func (p *AlertListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 25
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
}

func (p *AlertListParams) Validate() error {
	p.SetDefaults()
	switch strings.ToLower(strings.TrimSpace(p.Status)) {
	case "", "new", "acknowledged", "investigating", "resolved", "false_positive":
		return nil
	default:
		return fmt.Errorf("invalid alert status")
	}
}

type AlertListResponse struct {
	Data []*model.UEBAAlert `json:"data"`
	Meta PaginationMeta     `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

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

type AlertStatusUpdateRequest struct {
	Status string `json:"status"`
	Notes  string `json:"notes,omitempty"`
}

func (r *AlertStatusUpdateRequest) Validate() error {
	switch strings.ToLower(strings.TrimSpace(r.Status)) {
	case "new", "acknowledged", "investigating", "resolved", "false_positive":
		return nil
	default:
		return fmt.Errorf("invalid status")
	}
}

type FalsePositiveRequest struct {
	Notes string `json:"notes,omitempty"`
}

// BulkAlertStatusRequest allows updating multiple alerts in a single request.
type BulkAlertStatusRequest struct {
	AlertIDs      []string `json:"alert_ids"`
	Status        string   `json:"status"`
	Notes         string   `json:"notes,omitempty"`
	FalsePositive bool     `json:"false_positive,omitempty"`
}

func (r *BulkAlertStatusRequest) Validate() error {
	if len(r.AlertIDs) == 0 {
		return fmt.Errorf("alert_ids is required")
	}
	if len(r.AlertIDs) > 200 {
		return fmt.Errorf("alert_ids must not exceed 200 items")
	}
	if r.FalsePositive {
		return nil
	}
	switch strings.ToLower(strings.TrimSpace(r.Status)) {
	case "acknowledged", "investigating", "resolved":
		return nil
	default:
		return fmt.Errorf("invalid status for bulk update; allowed: acknowledged, investigating, resolved")
	}
}

type BulkAlertStatusResponse struct {
	Updated int      `json:"updated"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors,omitempty"`
}
