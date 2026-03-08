package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
	pkgvalidator "github.com/clario360/platform/pkg/validator"
)

// CreateAssetRequest is the body for POST /api/v1/cyber/assets.
type CreateAssetRequest struct {
	Name        string            `json:"name" validate:"required,min=2,max=255"`
	Type        model.AssetType   `json:"type" validate:"required,asset_type"`
	IPAddress   *string           `json:"ip_address,omitempty" validate:"omitempty,ip"`
	Hostname    *string           `json:"hostname,omitempty" validate:"omitempty,min=1,max=255"`
	MACAddress  *string           `json:"mac_address,omitempty" validate:"omitempty,mac"`
	OS          *string           `json:"os,omitempty" validate:"omitempty,max=100"`
	OSVersion   *string           `json:"os_version,omitempty" validate:"omitempty,max=100"`
	Owner       *string           `json:"owner,omitempty" validate:"omitempty,max=255"`
	Department  *string           `json:"department,omitempty" validate:"omitempty,max=255"`
	Location    *string           `json:"location,omitempty" validate:"omitempty,max=255"`
	Criticality model.Criticality `json:"criticality" validate:"required,criticality"`
	Metadata    json.RawMessage   `json:"metadata,omitempty"`
	Tags        []string          `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
}

// UpdateAssetRequest is the body for PUT /api/v1/cyber/assets/:id.
// All fields are optional — only non-nil fields are applied.
type UpdateAssetRequest struct {
	Name        *string            `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Type        *model.AssetType   `json:"type,omitempty" validate:"omitempty,asset_type"`
	IPAddress   *string            `json:"ip_address,omitempty" validate:"omitempty,ip"`
	Hostname    *string            `json:"hostname,omitempty" validate:"omitempty,min=1,max=255"`
	MACAddress  *string            `json:"mac_address,omitempty" validate:"omitempty,mac"`
	OS          *string            `json:"os,omitempty" validate:"omitempty,max=100"`
	OSVersion   *string            `json:"os_version,omitempty" validate:"omitempty,max=100"`
	Owner       *string            `json:"owner,omitempty" validate:"omitempty,max=255"`
	Department  *string            `json:"department,omitempty" validate:"omitempty,max=255"`
	Location    *string            `json:"location,omitempty" validate:"omitempty,max=255"`
	Criticality *model.Criticality `json:"criticality,omitempty" validate:"omitempty,criticality"`
	Status      *model.AssetStatus `json:"status,omitempty" validate:"omitempty,asset_status"`
	Metadata    json.RawMessage    `json:"metadata,omitempty"`
	Tags        *[]string          `json:"tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
}

// TagPatchRequest is the body for PATCH /api/v1/cyber/assets/:id/tags.
type TagPatchRequest struct {
	Add    []string `json:"add,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
	Remove []string `json:"remove,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
}

// AssetListParams holds all query parameters for GET /api/v1/cyber/assets.
type AssetListParams struct {
	Search                *string    `form:"search" validate:"omitempty,max=200"`
	Types                 []string   `form:"type"`
	Criticalities         []string   `form:"criticality"`
	Statuses              []string   `form:"status"`
	OS                    *string    `form:"os"`
	Department            *string    `form:"department"`
	Owner                 *string    `form:"owner"`
	Location              *string    `form:"location"`
	Tags                  []string   `form:"tag"`
	DiscoverySource       *string    `form:"discovery_source"`
	DiscoveredAfter       *time.Time `form:"discovered_after"`
	DiscoveredBefore      *time.Time `form:"discovered_before"`
	LastSeenAfter         *time.Time `form:"last_seen_after"`
	HasVulnerabilities    *bool      `form:"has_vulnerabilities"`
	VulnerabilitySeverity *string    `form:"vulnerability_severity"`
	MinVulnCount          *int       `form:"min_vuln_count"`
	Sort                  string     `form:"sort" validate:"omitempty,oneof=name type criticality status discovered_at last_seen_at vulnerability_count created_at"`
	Order                 string     `form:"order" validate:"omitempty,oneof=asc desc"`
	Page                  int        `form:"page" validate:"omitempty,min=1"`
	PerPage               int        `form:"per_page" validate:"omitempty,min=1,max=200"`
}

// SetDefaults applies default values to params that were not provided.
func (p *AssetListParams) SetDefaults() {
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 25
	}
	if p.PerPage > 200 {
		p.PerPage = 200
	}
}

// Validate checks enum values and other business rules not covered by struct tags.
func (p *AssetListParams) Validate() error {
	allowedSorts := map[string]struct{}{
		"name": {}, "type": {}, "criticality": {}, "status": {},
		"discovered_at": {}, "last_seen_at": {}, "vulnerability_count": {}, "created_at": {},
	}
	if p.Sort != "" {
		if _, ok := allowedSorts[p.Sort]; !ok {
			return fmt.Errorf("invalid sort: %q", p.Sort)
		}
	}
	if p.Order != "" && p.Order != "asc" && p.Order != "desc" {
		return fmt.Errorf("invalid order: %q", p.Order)
	}
	if p.Search != nil {
		trimmed := strings.TrimSpace(*p.Search)
		if trimmed == "" {
			p.Search = nil
		} else {
			*p.Search = trimmed
		}
	}
	for _, t := range p.Types {
		if !model.AssetType(t).IsValid() {
			return fmt.Errorf("invalid asset type: %q", t)
		}
	}
	for _, c := range p.Criticalities {
		if !model.Criticality(c).IsValid() {
			return fmt.Errorf("invalid criticality: %q", c)
		}
	}
	for _, s := range p.Statuses {
		if !model.AssetStatus(s).IsValid() {
			return fmt.Errorf("invalid status: %q", s)
		}
	}
	if p.DiscoverySource != nil {
		valid := false
		for _, source := range model.ValidDiscoverySources {
			if *p.DiscoverySource == source {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid discovery_source: %q", *p.DiscoverySource)
		}
	}
	if len(p.Tags) > 20 {
		return fmt.Errorf("tag filter supports at most 20 values")
	}
	for _, tag := range p.Tags {
		if len(tag) == 0 || len(tag) > 50 {
			return fmt.Errorf("invalid tag length for %q", tag)
		}
		if err := pkgvalidator.ValidateVar(tag, "alphanumdash"); err != nil {
			return fmt.Errorf("invalid tag value: %q", tag)
		}
	}
	if p.MinVulnCount != nil && (*p.MinVulnCount < 0 || *p.MinVulnCount > 10000) {
		return fmt.Errorf("min_vuln_count must be in [0, 10000]")
	}
	return nil
}

// AssetListResponse is the paginated response for the asset list endpoint.
type AssetListResponse struct {
	Data []*model.Asset `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// CreateVulnerabilityRequest is the body for POST /api/v1/cyber/assets/:id/vulnerabilities.
type CreateVulnerabilityRequest struct {
	CVEID       *string  `json:"cve_id,omitempty" validate:"omitempty,max=20"`
	Title       string   `json:"title" validate:"required,min=2,max=500"`
	Description string   `json:"description,omitempty" validate:"omitempty,max=5000"`
	Severity    string   `json:"severity" validate:"required,oneof=critical high medium low info"`
	CVSSScore   *float64 `json:"cvss_score,omitempty" validate:"omitempty,min=0,max=10"`
	CVSSVector  *string  `json:"cvss_vector,omitempty" validate:"omitempty,max=100"`
	Source      string   `json:"source" validate:"required,oneof=cve_enrichment manual scan_tool penetration_test"`
	Remediation *string  `json:"remediation,omitempty" validate:"omitempty,max=5000"`
	Proof       *string  `json:"proof,omitempty" validate:"omitempty,max=5000"`
}

// UpdateVulnerabilityRequest is the body for PUT /api/v1/cyber/assets/:id/vulnerabilities/:vid.
type UpdateVulnerabilityRequest struct {
	Status      *string `json:"status,omitempty" validate:"omitempty,oneof=open in_progress mitigated resolved accepted false_positive"`
	Remediation *string `json:"remediation,omitempty" validate:"omitempty,max=5000"`
}

// VulnerabilityListParams holds query params for vulnerability list endpoints.
type VulnerabilityListParams struct {
	Status   *string `form:"status"`
	Severity *string `form:"severity"`
	Page     int     `form:"page"`
	PerPage  int     `form:"per_page"`
}

// SetDefaults applies defaults to VulnerabilityListParams.
func (p *VulnerabilityListParams) SetDefaults() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PerPage == 0 {
		p.PerPage = 25
	}
}

// CreateRelationshipRequest is the body for POST /api/v1/cyber/assets/:id/relationships.
type CreateRelationshipRequest struct {
	TargetAssetID    string                 `json:"target_asset_id" validate:"required,uuid"`
	RelationshipType model.RelationshipType `json:"relationship_type" validate:"required,relationship_type"`
	Metadata         json.RawMessage        `json:"metadata,omitempty"`
}
