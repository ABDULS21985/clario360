package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestHeatmapToResponse_FullTransform(t *testing.T) {
	hm := &model.RiskHeatmap{
		Rows: []model.HeatmapRow{
			{
				AssetType:  "server",
				AssetCount: 10,
				Cells: map[string]model.HeatmapCell{
					"critical": {VulnCount: 5, AffectedAssets: 3},
					"high":     {VulnCount: 12, AffectedAssets: 7},
					"medium":   {VulnCount: 8, AffectedAssets: 4},
					"low":      {VulnCount: 2, AffectedAssets: 1},
				},
				TotalVulns: 27,
			},
			{
				AssetType:  "endpoint",
				AssetCount: 20,
				Cells: map[string]model.HeatmapCell{
					"critical": {VulnCount: 0, AffectedAssets: 0},
					"high":     {VulnCount: 3, AffectedAssets: 2},
				},
				TotalVulns: 3,
			},
		},
		MaxValue: 12,
	}

	resp := HeatmapToResponse(hm)

	// Verify asset_types are correct and in order
	if len(resp.AssetTypes) != 2 {
		t.Fatalf("expected 2 asset types, got %d", len(resp.AssetTypes))
	}
	if resp.AssetTypes[0] != "server" || resp.AssetTypes[1] != "endpoint" {
		t.Errorf("asset_types order = %v, want [server, endpoint]", resp.AssetTypes)
	}

	// 2 asset types × 4 severities = 8 cells
	if len(resp.Cells) != 8 {
		t.Fatalf("expected 8 cells, got %d", len(resp.Cells))
	}

	// Verify total_vulnerabilities = 5+12+8+2+0+3+0+0 = 30
	if resp.TotalVulnerabilities != 30 {
		t.Errorf("total_vulnerabilities = %d, want 30", resp.TotalVulnerabilities)
	}

	// Verify generated_at is a valid RFC3339 timestamp near now
	ts, err := time.Parse(time.RFC3339, resp.GeneratedAt)
	if err != nil {
		t.Fatalf("generated_at is not valid RFC3339: %v", err)
	}
	if time.Since(ts) > 5*time.Second {
		t.Errorf("generated_at is too old: %s", resp.GeneratedAt)
	}

	// Verify specific cells
	findCell := func(assetType, severity string) *RiskHeatmapCellResponse {
		for i := range resp.Cells {
			if resp.Cells[i].AssetType == assetType && resp.Cells[i].Severity == severity {
				return &resp.Cells[i]
			}
		}
		return nil
	}

	c := findCell("server", "critical")
	if c == nil {
		t.Fatal("missing cell: server/critical")
	}
	if c.Count != 5 {
		t.Errorf("server/critical count = %d, want 5", c.Count)
	}
	if c.AffectedAssetCount != 3 {
		t.Errorf("server/critical affected_asset_count = %d, want 3", c.AffectedAssetCount)
	}
	if c.TotalAssetsOfType != 10 {
		t.Errorf("server/critical total_assets_of_type = %d, want 10", c.TotalAssetsOfType)
	}

	c = findCell("endpoint", "high")
	if c == nil {
		t.Fatal("missing cell: endpoint/high")
	}
	if c.Count != 3 {
		t.Errorf("endpoint/high count = %d, want 3", c.Count)
	}

	// Missing severities should produce zero-count cells
	c = findCell("endpoint", "low")
	if c == nil {
		t.Fatal("missing cell: endpoint/low")
	}
	if c.Count != 0 {
		t.Errorf("endpoint/low count = %d, want 0", c.Count)
	}
	if c.TotalAssetsOfType != 20 {
		t.Errorf("endpoint/low total_assets_of_type = %d, want 20", c.TotalAssetsOfType)
	}
}

func TestHeatmapToResponse_EmptyHeatmap(t *testing.T) {
	hm := &model.RiskHeatmap{
		Rows:     []model.HeatmapRow{},
		MaxValue: 0,
	}

	resp := HeatmapToResponse(hm)

	if len(resp.Cells) != 0 {
		t.Errorf("expected 0 cells, got %d", len(resp.Cells))
	}
	if len(resp.AssetTypes) != 0 {
		t.Errorf("expected 0 asset_types, got %d", len(resp.AssetTypes))
	}
	if resp.TotalVulnerabilities != 0 {
		t.Errorf("total_vulnerabilities = %d, want 0", resp.TotalVulnerabilities)
	}
}

func TestHeatmapToResponse_SeverityOrder(t *testing.T) {
	hm := &model.RiskHeatmap{
		Rows: []model.HeatmapRow{
			{
				AssetType:  "database",
				AssetCount: 5,
				Cells: map[string]model.HeatmapCell{
					"low":      {VulnCount: 1, AffectedAssets: 1},
					"critical": {VulnCount: 4, AffectedAssets: 2},
				},
				TotalVulns: 5,
			},
		},
		MaxValue: 4,
	}

	resp := HeatmapToResponse(hm)

	// Cells for "database" should be in severity order: critical, high, medium, low
	expectedOrder := []string{"critical", "high", "medium", "low"}
	for i, expected := range expectedOrder {
		if resp.Cells[i].Severity != expected {
			t.Errorf("cell[%d].severity = %q, want %q", i, resp.Cells[i].Severity, expected)
		}
	}
}

func TestHeatmapToResponse_JSONContract(t *testing.T) {
	hm := &model.RiskHeatmap{
		Rows: []model.HeatmapRow{
			{
				AssetType:  "server",
				AssetCount: 10,
				Cells: map[string]model.HeatmapCell{
					"critical": {VulnCount: 5, AffectedAssets: 3},
				},
				TotalVulns: 5,
			},
		},
		MaxValue: 5,
	}

	resp := HeatmapToResponse(hm)
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify JSON field names match the frontend contract
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	requiredKeys := []string{"cells", "asset_types", "total_vulnerabilities", "generated_at"}
	for _, key := range requiredKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing required JSON key: %q", key)
		}
	}

	// Verify cell JSON field names
	var cells []map[string]json.RawMessage
	if err := json.Unmarshal(raw["cells"], &cells); err != nil {
		t.Fatalf("failed to unmarshal cells: %v", err)
	}
	if len(cells) == 0 {
		t.Fatal("expected at least one cell")
	}
	cellKeys := []string{"asset_type", "severity", "count", "affected_asset_count", "total_assets_of_type"}
	for _, key := range cellKeys {
		if _, ok := cells[0][key]; !ok {
			t.Errorf("missing required cell JSON key: %q", key)
		}
	}
}

func TestRiskTrendParams_SetDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{"zero defaults to 90", 0, 90},
		{"negative clamped to 1", -5, 1},
		{"over max clamped to 365", 999, 365},
		{"valid stays unchanged", 30, 30},
		{"boundary 1", 1, 1},
		{"boundary 365", 365, 365},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &RiskTrendParams{Days: tt.in}
			p.SetDefaults()
			if p.Days != tt.want {
				t.Errorf("Days = %d, want %d", p.Days, tt.want)
			}
		})
	}
}
