package tools

import (
	"testing"

	"github.com/google/uuid"
)

func TestInferDashboardWidgets(t *testing.T) {
	t.Parallel()

	kpiID := uuid.New()
	kpis := map[string]uuid.UUID{"Security Risk Score": kpiID}

	tests := []struct {
		name        string
		description string
		wantCount   int
		wantTitle   string
	}{
		{"alerts", "alerts", 2, "Alert Timeline"},
		{"risk", "risk", 2, "Risk Trend"},
		{"mixed", "alerts and risk and compliance", 5, "Compliance By Committee"},
		{"mitre", "MITRE coverage", 1, "MITRE Heatmap"},
		{"default", "general overview", 5, "Open Alerts By Severity"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			items := inferDashboardWidgets(tc.description, kpis)
			if len(items) != tc.wantCount {
				t.Fatalf("len = %d, want %d", len(items), tc.wantCount)
			}
			found := false
			for _, item := range items {
				if item.Title == tc.wantTitle {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("did not find widget %q", tc.wantTitle)
			}
		})
	}
}
