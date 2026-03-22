package dto

import (
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

type RiskTrendParams struct {
	Days int `form:"days"`
}

func (p *RiskTrendParams) SetDefaults() {
	if p.Days == 0 {
		p.Days = 90
	}
	if p.Days < 1 {
		p.Days = 1
	}
	if p.Days > 365 {
		p.Days = 365
	}
}

// ─── Risk Heatmap Response ────────────────────────────────────────────────────

// RiskHeatmapCellResponse is the flat cell format expected by the frontend.
type RiskHeatmapCellResponse struct {
	AssetType         string `json:"asset_type"`
	Severity          string `json:"severity"`
	Count             int    `json:"count"`
	AffectedAssetCount int   `json:"affected_asset_count"`
	TotalAssetsOfType int    `json:"total_assets_of_type"`
}

// RiskHeatmapResponse is the API response format expected by the frontend.
type RiskHeatmapResponse struct {
	Cells                []RiskHeatmapCellResponse `json:"cells"`
	AssetTypes           []string                  `json:"asset_types"`
	TotalVulnerabilities int                       `json:"total_vulnerabilities"`
	GeneratedAt          string                    `json:"generated_at"`
}

// HeatmapToResponse transforms the internal RiskHeatmap model into the flat
// response structure that the frontend expects.
func HeatmapToResponse(hm *model.RiskHeatmap) *RiskHeatmapResponse {
	severities := []string{"critical", "high", "medium", "low"}
	resp := &RiskHeatmapResponse{
		Cells:       make([]RiskHeatmapCellResponse, 0),
		AssetTypes:  make([]string, 0, len(hm.Rows)),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	totalVulns := 0
	for _, row := range hm.Rows {
		resp.AssetTypes = append(resp.AssetTypes, row.AssetType)
		for _, sev := range severities {
			cell := row.Cells[sev]
			resp.Cells = append(resp.Cells, RiskHeatmapCellResponse{
				AssetType:          row.AssetType,
				Severity:           sev,
				Count:              cell.VulnCount,
				AffectedAssetCount: cell.AffectedAssets,
				TotalAssetsOfType:  row.AssetCount,
			})
			totalVulns += cell.VulnCount
		}
	}
	resp.TotalVulnerabilities = totalVulns
	return resp
}
