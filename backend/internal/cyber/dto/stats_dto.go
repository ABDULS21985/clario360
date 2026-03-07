package dto

import "github.com/clario360/platform/internal/cyber/model"

// AssetStats is the aggregated stats response for GET /api/v1/cyber/assets/stats.
type AssetStats struct {
	TotalAssets          int                      `json:"total_assets"`
	ActiveAssets         int                      `json:"active_assets"`
	ByType               map[string]int           `json:"by_type"`
	ByCriticality        map[string]int           `json:"by_criticality"`
	ByStatus             map[string]int           `json:"by_status"`
	ByOS                 map[string]int           `json:"by_os"`
	ByDepartment         map[string]int           `json:"by_department"`
	ByDiscoverySource    map[string]int           `json:"by_discovery_source"`
	TopDepartments       []model.AssetCountByName `json:"top_departments"`
	TopOS                []model.AssetCountByName `json:"top_os"`
	TotalVulnerabilities int                      `json:"total_vulnerabilities"`
	OpenVulnerabilities  int                      `json:"open_vulnerabilities"`
	VulnsBySeverity      map[string]int           `json:"vulns_by_severity"`
	AssetsWithCritical   int                      `json:"assets_with_critical_vulns"`
	LastScanAt           *string                  `json:"last_scan_at,omitempty"`
}

// AssetCountResponse is the response for GET /api/v1/cyber/assets/count.
type AssetCountResponse struct {
	Count int `json:"count"`
}
