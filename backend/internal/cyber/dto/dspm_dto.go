package dto

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

// DSPMAssetListParams are query parameters for listing DSPM data assets.
type DSPMAssetListParams struct {
	Classification *string    `json:"classification"`
	ContainsPII    *bool      `json:"contains_pii"`
	MinRiskScore   *float64   `json:"min_risk_score"`
	NetworkExposure *string   `json:"network_exposure"`
	AssetID        *uuid.UUID `json:"asset_id"`
	Search         *string    `json:"search"`
	Sort           string     `json:"sort"`
	Order          string     `json:"order"`
	Page           int        `json:"page"`
	PerPage        int        `json:"per_page"`
}

func (p *DSPMAssetListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 200 {
		p.PerPage = 50
	}
	if p.Sort == "" {
		p.Sort = "risk_score"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

func (p *DSPMAssetListParams) Validate() error {
	validSorts := map[string]bool{
		"risk_score":           true,
		"posture_score":        true,
		"data_classification":  true,
		"sensitivity_score":    true,
		"created_at":           true,
		"updated_at":           true,
	}
	if p.Sort != "" && !validSorts[p.Sort] {
		return fmt.Errorf("invalid sort field: %s", p.Sort)
	}
	if p.Order != "" && p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order: %s", p.Order)
	}
	return nil
}

// DSPMScanListParams are query parameters for listing scans.
type DSPMScanListParams struct {
	Status  *string `json:"status"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
}

func (p *DSPMScanListParams) SetDefaults() {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 || p.PerPage > 100 {
		p.PerPage = 20
	}
}

func (p *DSPMScanListParams) Validate() error {
	if p.Page < 1 {
		return fmt.Errorf("page must be at least 1")
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		return fmt.Errorf("per_page must be between 1 and 100")
	}
	if p.Status != nil {
		switch *p.Status {
		case "running", "completed", "failed":
		default:
			return fmt.Errorf("invalid status: %s", *p.Status)
		}
	}
	return nil
}

type DSPMAssetListResponse struct {
	Data       []*model.DSPMDataAsset `json:"data"`
	Meta       PaginationMeta         `json:"meta"`
	Total      int                    `json:"-"`
	Page       int                    `json:"-"`
	PerPage    int                    `json:"-"`
	TotalPages int                    `json:"-"`
}

type DSPMScanListResponse struct {
	Data       []*model.DSPMScan `json:"data"`
	Meta       PaginationMeta    `json:"meta"`
	Total      int               `json:"-"`
	Page       int               `json:"-"`
	PerPage    int               `json:"-"`
	TotalPages int               `json:"-"`
}

type DSPMScanTriggerResponse struct {
	Scan *model.DSPMScan `json:"scan"`
}
