package dspm

import (
	"math"
	"testing"
)

func TestCalculateRiskScore_InternetFacing_HighSensitivity(t *testing.T) {
	score, factors := CalculateRiskScore(90, "internet_facing", 10)
	// rawRisk = 90 * 1.8 * (1 - 10/100) = 90 * 1.8 * 0.9 = 145.8 → clamped to 100
	if score != 100 {
		t.Errorf("expected score clamped to 100, got %.2f", score)
	}
	if len(factors) != 3 {
		t.Errorf("expected 3 risk factors, got %d", len(factors))
	}
}

func TestCalculateRiskScore_Private_LowSensitivity(t *testing.T) {
	score, _ := CalculateRiskScore(20, "private", 80)
	// rawRisk = 20 * 1.0 * (1 - 80/100) = 20 * 0.2 = 4.0
	expected := 4.0
	if math.Abs(score-expected) > 0.01 {
		t.Errorf("expected score %.2f, got %.2f", expected, score)
	}
}

func TestCalculateRiskScore_VPNAccessible(t *testing.T) {
	score, _ := CalculateRiskScore(50, "vpn_accessible", 50)
	// rawRisk = 50 * 1.3 * 0.5 = 32.5
	expected := 32.5
	if math.Abs(score-expected) > 0.01 {
		t.Errorf("expected score %.2f, got %.2f", expected, score)
	}
}

func TestCalculateRiskScore_ZeroRisk(t *testing.T) {
	score, _ := CalculateRiskScore(0, "private", 100)
	if score != 0 {
		t.Errorf("expected zero risk, got %.2f", score)
	}
}

func TestCalculateRiskScore_FactorNames(t *testing.T) {
	_, factors := CalculateRiskScore(60, "internet_facing", 40)
	names := map[string]bool{}
	for _, f := range factors {
		names[f.Factor] = true
	}
	for _, expected := range []string{"sensitivity", "exposure", "control_gap"} {
		if !names[expected] {
			t.Errorf("missing factor: %s", expected)
		}
	}
}
