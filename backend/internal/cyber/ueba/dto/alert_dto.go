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
	Data       []*model.UEBAAlert `json:"data"`
	Total      int                `json:"total"`
	Page       int                `json:"page"`
	PerPage    int                `json:"per_page"`
	TotalPages int                `json:"total_pages"`
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
