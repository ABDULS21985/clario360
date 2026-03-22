package dto

import (
	"github.com/clario360/platform/internal/cyber/model"
)

// ScanTriggerRequest is the body for POST /api/v1/cyber/assets/scan.
type ScanTriggerRequest struct {
	ScanType string         `json:"scan_type" validate:"required,oneof=network cloud agent"`
	Targets  []string       `json:"targets" validate:"required,min=1,max=100,dive,min=1,max=200"`
	Ports    []int          `json:"ports,omitempty" validate:"omitempty,max=1000,dive,min=1,max=65535"`
	Options  map[string]any `json:"options,omitempty"`
}

// ScanTriggerResponse is returned immediately after accepting a scan request.
type ScanTriggerResponse struct {
	ScanID  string           `json:"scan_id"`
	Status  model.ScanStatus `json:"status"`
	Message string           `json:"message"`
}

// ScanListParams holds query params for GET /api/v1/cyber/assets/scans.
type ScanListParams struct {
	ScanType *string `form:"scan_type"`
	Status   *string `form:"status"`
	Sort     string  `form:"sort"`
	Order    string  `form:"order"`
	Page     int     `form:"page"`
	PerPage  int     `form:"per_page"`
}

// validScanSorts is the set of columns allowed for scan list ordering.
var validScanSorts = map[string]string{
	"scan_type":  "scan_type",
	"status":     "status",
	"started_at": "started_at",
	"created_at": "created_at",
}

// SetDefaults applies defaults to ScanListParams.
func (p *ScanListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
}

// SortColumn returns the SQL column name for the requested sort, falling
// back to "created_at" if the requested value is not in the allow-list.
func (p *ScanListParams) SortColumn() string {
	if col, ok := validScanSorts[p.Sort]; ok {
		return col
	}
	return "created_at"
}

// SortDirection returns "ASC" or "DESC".
func (p *ScanListParams) SortDirection() string {
	if p.Order == "asc" {
		return "ASC"
	}
	return "DESC"
}

// ScanListResponse is the paginated scan history response.
type ScanListResponse struct {
	Data    []*model.ScanHistory `json:"data"`
	Total   int                  `json:"total"`
	Page    int                  `json:"page"`
	PerPage int                  `json:"per_page"`
}
