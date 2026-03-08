package dto

import (
	"fmt"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type ProfileListParams struct {
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
	Status  model.ProfileStatus `json:"status"`
}

func (p *ProfileListParams) SetDefaults() {
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

func (p *ProfileListParams) Validate() error {
	p.SetDefaults()
	switch p.Status {
	case "", model.ProfileStatusActive, model.ProfileStatusInactive, model.ProfileStatusSuppressed, model.ProfileStatusWhitelisted:
		return nil
	default:
		return fmt.Errorf("invalid profile status")
	}
}

type ProfileListResponse struct {
	Data       []*model.UEBAProfile `json:"data"`
	Total      int                  `json:"total"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"per_page"`
	TotalPages int                  `json:"total_pages"`
}

type HeatmapResponse struct {
	EntityID string     `json:"entity_id"`
	Days     int        `json:"days"`
	Matrix   [7][24]int `json:"matrix"`
}

type TimelineResponse struct {
	Data       []*model.DataAccessEvent `json:"data"`
	Total      int                      `json:"total"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"per_page"`
	TotalPages int                      `json:"total_pages"`
}

type ProfileStatusUpdateRequest struct {
	EntityType      model.EntityType    `json:"entity_type"`
	Status          model.ProfileStatus `json:"status"`
	SuppressedUntil *time.Time          `json:"suppressed_until,omitempty"`
	Reason          string              `json:"reason,omitempty"`
}

func (r *ProfileStatusUpdateRequest) Validate() error {
	switch r.EntityType {
	case model.EntityTypeUser, model.EntityTypeServiceAccount, model.EntityTypeApplication, model.EntityTypeAPIKey:
	default:
		return fmt.Errorf("entity_type is required")
	}
	switch r.Status {
	case model.ProfileStatusActive, model.ProfileStatusInactive, model.ProfileStatusSuppressed, model.ProfileStatusWhitelisted:
	default:
		return fmt.Errorf("invalid status")
	}
	if r.Status == model.ProfileStatusSuppressed && r.SuppressedUntil == nil {
		return fmt.Errorf("suppressed_until is required when status=suppressed")
	}
	return nil
}

type RiskHistoryPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Score     float64   `json:"score"`
	AlertID   string    `json:"alert_id,omitempty"`
	Severity  string    `json:"severity,omitempty"`
	AlertType string    `json:"alert_type,omitempty"`
}

type ProfileDetailResponse struct {
	Profile            *model.UEBAProfile `json:"profile"`
	BaselineComparison map[string]any     `json:"baseline_comparison"`
	RiskHistory        []RiskHistoryPoint `json:"risk_history"`
}

func NormalizeStatus(value string) model.ProfileStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(model.ProfileStatusActive):
		return model.ProfileStatusActive
	case string(model.ProfileStatusInactive):
		return model.ProfileStatusInactive
	case string(model.ProfileStatusSuppressed):
		return model.ProfileStatusSuppressed
	case string(model.ProfileStatusWhitelisted):
		return model.ProfileStatusWhitelisted
	default:
		return ""
	}
}
