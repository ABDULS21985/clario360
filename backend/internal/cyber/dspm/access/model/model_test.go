package model

import "testing"

func TestSensitivityWeight(t *testing.T) {
	tests := []struct {
		classification string
		want           float64
	}{
		{"restricted", 10.0},
		{"confidential", 5.0},
		{"internal", 2.0},
		{"public", 1.0},
		{"unknown", 1.0},
		{"", 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.classification, func(t *testing.T) {
			got := SensitivityWeight(tc.classification)
			if got != tc.want {
				t.Errorf("SensitivityWeight(%q) = %v, want %v", tc.classification, got, tc.want)
			}
		})
	}
}

func TestPermissionBreadth(t *testing.T) {
	tests := []struct {
		permType string
		want     float64
	}{
		{"full_control", 5.0},
		{"admin", 4.0},
		{"alter", 3.0},
		{"write", 2.0},
		{"delete", 2.0},
		{"create", 2.0},
		{"execute", 1.5},
		{"read", 1.0},
		{"unknown", 1.0},
		{"", 1.0},
	}
	for _, tc := range tests {
		t.Run(tc.permType, func(t *testing.T) {
			got := PermissionBreadth(tc.permType)
			if got != tc.want {
				t.Errorf("PermissionBreadth(%q) = %v, want %v", tc.permType, got, tc.want)
			}
		})
	}
}

func TestPermissionLevel(t *testing.T) {
	tests := []struct {
		permType string
		want     int
	}{
		{"full_control", 8},
		{"admin", 7},
		{"alter", 6},
		{"delete", 5},
		{"create", 4},
		{"write", 3},
		{"execute", 2},
		{"read", 1},
		{"unknown", 0},
		{"", 0},
	}
	for _, tc := range tests {
		t.Run(tc.permType, func(t *testing.T) {
			got := PermissionLevel(tc.permType)
			if got != tc.want {
				t.Errorf("PermissionLevel(%q) = %d, want %d", tc.permType, got, tc.want)
			}
		})
	}
}

func TestRiskLevel(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{"zero_is_low", 0, "low"},
		{"boundary_25_is_medium", 25, "medium"},
		{"boundary_50_is_high", 50, "high"},
		{"boundary_75_is_critical", 75, "critical"},
		{"max_100_is_critical", 100, "critical"},
		{"low_12", 12, "low"},
		{"medium_37", 37, "medium"},
		{"high_62", 62, "high"},
		{"critical_80", 80, "critical"},
		{"just_below_25_is_low", 24.9, "low"},
		{"just_below_50_is_medium", 49.9, "medium"},
		{"just_below_75_is_high", 74.9, "high"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RiskLevel(tc.score)
			if got != tc.want {
				t.Errorf("RiskLevel(%v) = %q, want %q", tc.score, got, tc.want)
			}
		})
	}
}

func TestPermissionLevel_Ordering(t *testing.T) {
	// Verify that permission levels are strictly ordered.
	ordered := []string{"read", "execute", "write", "create", "delete", "alter", "admin", "full_control"}
	for i := 1; i < len(ordered); i++ {
		prev := PermissionLevel(ordered[i-1])
		curr := PermissionLevel(ordered[i])
		if curr <= prev {
			t.Errorf("PermissionLevel(%q)=%d should be > PermissionLevel(%q)=%d",
				ordered[i], curr, ordered[i-1], prev)
		}
	}
}

func TestClassificationRank(t *testing.T) {
	tests := []struct {
		classification string
		want           int
	}{
		{"restricted", 4},
		{"confidential", 3},
		{"internal", 2},
		{"public", 1},
		{"unknown", 0},
		{"", 0},
	}
	for _, tc := range tests {
		t.Run(tc.classification, func(t *testing.T) {
			got := ClassificationRank(tc.classification)
			if got != tc.want {
				t.Errorf("ClassificationRank(%q) = %d, want %d", tc.classification, got, tc.want)
			}
		})
	}
}
