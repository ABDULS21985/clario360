package ctem

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestHasProtectionFromTags(t *testing.T) {
	asset := &model.Asset{
		ID:   uuid.New(),
		Tags: []string{"waf"},
	}
	if !hasProtection(asset, "waf") {
		t.Fatal("expected waf protection from tags")
	}
}

func TestHasProtectionFromMetadata(t *testing.T) {
	metadata, err := json.Marshal(map[string]any{
		"vlan_segment": "prod-secure",
	})
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	asset := &model.Asset{
		ID:       uuid.New(),
		Metadata: metadata,
	}
	if !hasProtection(asset, "segmented") {
		t.Fatal("expected segmented protection from metadata")
	}
}

func TestStringOrEmptyCVSS(t *testing.T) {
	vector := "CVSS:3.1/AV:N/PR:L"
	evidence, err := json.Marshal(map[string]any{
		"cvss_vector": vector,
	})
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	finding := &model.CTEMFinding{Evidence: evidence}
	if got := stringOrEmptyCVSS(finding); got != vector {
		t.Fatalf("expected CVSS vector %q, got %q", vector, got)
	}
}

func TestCountPriorityGroups(t *testing.T) {
	findings := []*model.CTEMFinding{
		{PriorityGroup: 1},
		{PriorityGroup: 2},
		{PriorityGroup: 2},
		{PriorityGroup: 4},
	}
	if got := countPriorityGroups(findings, 1, 2); got != 3 {
		t.Fatalf("expected 3 findings in groups 1 and 2, got %d", got)
	}
}
