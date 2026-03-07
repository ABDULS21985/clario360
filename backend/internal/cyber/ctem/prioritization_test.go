package ctem

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestCalculateBusinessImpactCriticalAsset(t *testing.T) {
	department := "finance"
	asset := &model.Asset{
		ID:          uuid.New(),
		Name:        "db-prod-01",
		Criticality: model.CriticalityCritical,
		Department:  &department,
		Tags:        []string{"pci"},
	}

	score, factors := CalculateBusinessImpact(asset, 5)

	if score != 95 {
		t.Fatalf("expected business impact 95, got %.2f", score)
	}
	if len(factors) != 4 {
		t.Fatalf("expected 4 factors, got %d", len(factors))
	}
	if factors[0].Factor != "asset_criticality" || factors[0].Value != 40 {
		t.Fatalf("expected first factor to be criticality=40, got %+v", factors[0])
	}
}

func TestCalculateBusinessImpactBlastRadiusCap(t *testing.T) {
	asset := &model.Asset{
		ID:          uuid.New(),
		Name:        "api-prod-01",
		Criticality: model.CriticalityHigh,
	}

	score, factors := CalculateBusinessImpact(asset, 10)

	if score != 50 {
		t.Fatalf("expected capped business impact 50, got %.2f", score)
	}
	found := false
	for _, factor := range factors {
		if factor.Factor == "blast_radius" {
			found = true
			if factor.Value != 20 {
				t.Fatalf("expected blast radius to cap at 20, got %.2f", factor.Value)
			}
		}
	}
	if !found {
		t.Fatal("expected blast radius factor to be present")
	}
}

func TestCalculateExploitabilityHighCVSS(t *testing.T) {
	vector := "CVSS:3.1/AV:N/AC:L/PR:N/UI:N"
	evidence, err := json.Marshal(map[string]any{
		"cvss_score":  9.8,
		"cvss_vector": vector,
	})
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	metadata, err := json.Marshal(map[string]any{
		"public_exploit_available": true,
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	score, factors := CalculateExploitability(&model.CTEMFinding{
		Type:             model.CTEMFindingTypeVulnerability,
		Severity:         "critical",
		Evidence:         evidence,
		Metadata:         metadata,
		ValidationStatus: model.CTEMValidationPending,
	}, &model.Asset{Tags: []string{"internet-facing"}}, true, true, true)

	if score != 100 {
		t.Fatalf("expected exploitability to clamp at 100, got %.2f", score)
	}
	if len(factors) < 5 {
		t.Fatalf("expected multiple exploitability factors, got %d", len(factors))
	}
}

func TestCalculateExploitabilityAttackPath(t *testing.T) {
	attackPath, err := json.Marshal([]map[string]any{
		{"asset_id": uuid.New().String()},
		{"asset_id": uuid.New().String(), "vuln_severity": "critical"},
	})
	if err != nil {
		t.Fatalf("marshal attack path: %v", err)
	}

	score, factors := CalculateExploitability(&model.CTEMFinding{
		Type:       model.CTEMFindingTypeAttackPath,
		Severity:   "critical",
		AttackPath: attackPath,
	}, &model.Asset{Tags: []string{"public"}}, false, false, false)

	if score <= 0 {
		t.Fatalf("expected positive attack path exploitability, got %.2f", score)
	}
	if factors[0].Factor != "attack_path_base" {
		t.Fatalf("expected attack path base factor, got %+v", factors[0])
	}
}

func TestPriorityScoreComputation(t *testing.T) {
	score := CalculatePriorityScore(80, 90)
	if score != 84 {
		t.Fatalf("expected priority score 84, got %.2f", score)
	}
}

func TestPriorityGroupThresholds(t *testing.T) {
	cases := []struct {
		score float64
		want  int
	}{
		{85, 1},
		{60, 2},
		{55, 3},
		{20, 4},
	}

	for _, tc := range cases {
		if got := PriorityGroupForScore(tc.score); got != tc.want {
			t.Fatalf("score %.2f: expected group %d, got %d", tc.score, tc.want, got)
		}
	}
}
