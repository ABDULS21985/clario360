package dto

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// AccessMappingListParams are query parameters for listing access mappings.
type AccessMappingListParams struct {
	IdentityType       *string    `json:"identity_type"`
	IdentityID         *string    `json:"identity_id"`
	DataAssetID        *uuid.UUID `json:"data_asset_id"`
	PermissionType     *string    `json:"permission_type"`
	DataClassification *string    `json:"data_classification"`
	Status             *string    `json:"status"`
	IsStale            *bool      `json:"is_stale"`
	Search             *string    `json:"search"`
	Sort               string     `json:"sort"`
	Order              string     `json:"order"`
	Page               int        `json:"page"`
	PerPage            int        `json:"per_page"`
}

func (p *AccessMappingListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 50
	}
	if p.Sort == "" {
		p.Sort = "access_risk_score"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

func (p *AccessMappingListParams) Validate() error {
	validSorts := map[string]bool{
		"access_risk_score":  true,
		"last_used_at":       true,
		"sensitivity_weight": true,
		"created_at":         true,
		"updated_at":         true,
	}
	if p.Sort != "" && !validSorts[p.Sort] {
		return fmt.Errorf("invalid sort field: %s", p.Sort)
	}
	if p.Order != "" && p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order: %s", p.Order)
	}
	return nil
}

// IdentityListParams are query parameters for listing identity profiles.
type IdentityListParams struct {
	IdentityType *string `json:"identity_type"`
	Status       *string `json:"status"`
	MinRiskScore *float64 `json:"min_risk_score"`
	Search       *string `json:"search"`
	Sort         string  `json:"sort"`
	Order        string  `json:"order"`
	Page         int     `json:"page"`
	PerPage      int     `json:"per_page"`
}

func (p *IdentityListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 50
	}
	if p.Sort == "" {
		p.Sort = "access_risk_score"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

func (p *IdentityListParams) Validate() error {
	validSorts := map[string]bool{
		"access_risk_score":  true,
		"blast_radius_score": true,
		"overprivileged_count": true,
		"stale_permission_count": true,
		"last_activity_at":   true,
		"created_at":         true,
	}
	if p.Sort != "" && !validSorts[p.Sort] {
		return fmt.Errorf("invalid sort field: %s", p.Sort)
	}
	if p.Order != "" && p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order: %s", p.Order)
	}
	return nil
}

// IdentityListResponse is the paginated response for identity profiles.
type IdentityListResponse struct {
	Data       []*model.IdentityProfile `json:"data"`
	Meta       PaginationMeta           `json:"meta"`
}

// AccessMappingListResponse is the paginated response for access mappings.
type AccessMappingListResponse struct {
	Data       []*model.AccessMapping `json:"data"`
	Meta       PaginationMeta         `json:"meta"`
}

// PaginationMeta holds pagination info.
type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
