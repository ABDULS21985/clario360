package ctem

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestBuildRemediationGroupPatch(t *testing.T) {
	now := time.Now().UTC()
	assessment := &model.CTEMAssessment{ID: uuid.New(), TenantID: uuid.New()}
	finding := &model.CTEMFinding{
		ID:            uuid.New(),
		Type:          model.CTEMFindingTypeVulnerability,
		CVEIDs:        []string{"CVE-2024-3094"},
		PriorityScore: 88,
		PriorityGroup: 1,
	}

	signature, group := buildRemediationGroup(assessment, finding, nil, now)

	if signature != "patch:CVE-2024-3094" {
		t.Fatalf("expected patch signature, got %s", signature)
	}
	if group.Type != model.CTEMRemediationPatch {
		t.Fatalf("expected patch remediation type, got %s", group.Type)
	}
}

func TestBuildRemediationGroupAttackPath(t *testing.T) {
	now := time.Now().UTC()
	assessment := &model.CTEMAssessment{ID: uuid.New(), TenantID: uuid.New()}
	finding := &model.CTEMFinding{
		ID:            uuid.New(),
		Type:          model.CTEMFindingTypeAttackPath,
		PriorityScore: 75,
		PriorityGroup: 2,
	}

	_, group := buildRemediationGroup(assessment, finding, nil, now)

	if group.Type != model.CTEMRemediationArchitecture {
		t.Fatalf("expected architecture remediation type, got %s", group.Type)
	}
}

func TestRemediationEffortEstimation(t *testing.T) {
	if effort, days := remediationEffortForGroup(model.CTEMRemediationConfiguration, 3); effort != model.CTEMRemediationEffortLow || days != 1 {
		t.Fatalf("expected low effort/1 day, got %s/%d", effort, days)
	}
	if effort, days := remediationEffortForGroup(model.CTEMRemediationConfiguration, 15); effort != model.CTEMRemediationEffortMedium || days != 3 {
		t.Fatalf("expected medium effort/3 days, got %s/%d", effort, days)
	}
	if effort, days := remediationEffortForGroup(model.CTEMRemediationArchitecture, 50); effort != model.CTEMRemediationEffortHigh || days != 21 {
		t.Fatalf("expected architecture effort high/21 days, got %s/%d", effort, days)
	}
}

func TestRemediationTimeline(t *testing.T) {
	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	target := remediationTargetDate(now, 1)
	if target == nil {
		t.Fatal("expected target date for priority group 1")
	}
	if got := target.Sub(now).Hours(); got < 71 || got > 73 {
		t.Fatalf("expected target date about 72h later, got %.2f hours", got)
	}
}

func TestProjectedScoreReduction(t *testing.T) {
	findings := []*model.CTEMFinding{
		{PriorityScore: 85},
		{PriorityScore: 65},
		{PriorityScore: 50},
	}
	if got := projectedScoreReduction(findings); got != 2 {
		t.Fatalf("expected projected score reduction 2.00, got %.2f", got)
	}
}

func TestRemediationConfigurationTitle(t *testing.T) {
	evidence, err := json.Marshal(map[string]any{"port": 3389})
	if err != nil {
		t.Fatalf("marshal evidence: %v", err)
	}
	title := remediationConfigurationTitle(&model.CTEMFinding{Evidence: evidence})
	if title != "Close exposed management port 3389" {
		t.Fatalf("unexpected title: %s", title)
	}
}
