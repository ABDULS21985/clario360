package dto

import "github.com/google/uuid"

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
