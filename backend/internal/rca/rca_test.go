package rca

import (
	"testing"
	"time"
)

// --- CausalChain ---

func TestBuildFromTimeline_Security_OrdersByKillChain(t *testing.T) {
	builder := NewChainBuilder()
	events := []TimelineEvent{
		{ID: "3", Timestamp: time.Now(), Source: "alert", Type: "cyber_alert", Summary: "Exfil detected", MITREPhase: "exfiltration"},
		{ID: "1", Timestamp: time.Now().Add(-2 * time.Hour), Source: "alert", Type: "cyber_alert", Summary: "Phishing email", MITREPhase: "initial-access"},
		{ID: "2", Timestamp: time.Now().Add(-1 * time.Hour), Source: "alert", Type: "cyber_alert", Summary: "Malware executed", MITREPhase: "execution"},
	}

	chain := builder.BuildFromTimeline(events, AnalysisTypeSecurity)
	if len(chain) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(chain))
	}

	// Should be ordered by kill chain: initial-access < execution < exfiltration
	expectedPhases := []string{"initial-access", "execution", "exfiltration"}
	for i, expected := range expectedPhases {
		if chain[i].MITREPhase != expected {
			t.Errorf("step[%d].MITREPhase = %q, want %q", i, chain[i].MITREPhase, expected)
		}
	}

	// First step should be root cause
	if !chain[0].IsRootCause {
		t.Error("first step should be marked as root cause")
	}
}

func TestBuildFromTimeline_Pipeline_OrdersByTimestamp(t *testing.T) {
	builder := NewChainBuilder()
	now := time.Now()
	events := []TimelineEvent{
		{ID: "3", Timestamp: now, Source: "pipeline", Type: "pipeline_run", Summary: "Run C"},
		{ID: "1", Timestamp: now.Add(-2 * time.Hour), Source: "pipeline", Type: "pipeline_run", Summary: "Run A"},
		{ID: "2", Timestamp: now.Add(-1 * time.Hour), Source: "pipeline", Type: "failed", Summary: "Run B failed"},
	}

	chain := builder.BuildFromTimeline(events, AnalysisTypePipeline)
	if len(chain) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(chain))
	}

	// Should be ordered by timestamp
	for i := 1; i < len(chain); i++ {
		if chain[i].Timestamp.Before(chain[i-1].Timestamp) {
			t.Errorf("step[%d] timestamp before step[%d]", i, i-1)
		}
	}

	// The first "failed" event should be root cause
	found := false
	for _, step := range chain {
		if step.IsRootCause && step.EventType == "failed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("failed event should be marked as root cause")
	}
}

func TestBuildFromTimeline_Empty(t *testing.T) {
	builder := NewChainBuilder()
	chain := builder.BuildFromTimeline(nil, AnalysisTypeSecurity)
	if chain != nil {
		t.Errorf("expected nil for empty events, got %d steps", len(chain))
	}
}

func TestBuildFromTimeline_Quality_SameAsPipeline(t *testing.T) {
	builder := NewChainBuilder()
	now := time.Now()
	events := []TimelineEvent{
		{ID: "1", Timestamp: now, Source: "quality", Type: "quality_rule_failure", Summary: "Rule failed"},
	}

	chain := builder.BuildFromTimeline(events, AnalysisTypeQuality)
	if len(chain) != 1 {
		t.Fatalf("expected 1 step, got %d", len(chain))
	}
	if !chain[0].IsRootCause {
		t.Error("single event should be root cause")
	}
}

func TestBuildFromTimeline_Security_UnphasedEventsAtEnd(t *testing.T) {
	builder := NewChainBuilder()
	now := time.Now()
	events := []TimelineEvent{
		{ID: "1", Timestamp: now, Source: "alert", Type: "cyber_alert", Summary: "Phishing", MITREPhase: "initial-access"},
		{ID: "2", Timestamp: now.Add(1 * time.Minute), Source: "audit", Type: "login", Summary: "Audit event"},
	}

	chain := builder.BuildFromTimeline(events, AnalysisTypeSecurity)
	if len(chain) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(chain))
	}

	// Phased event first, unphased at end
	if chain[0].MITREPhase != "initial-access" {
		t.Errorf("first step should be phased, got phase=%q", chain[0].MITREPhase)
	}
	if chain[1].MITREPhase != "" {
		t.Errorf("second step should be unphased, got phase=%q", chain[1].MITREPhase)
	}
}

// --- CorrelateEvents ---

func TestCorrelateEvents_GroupsBySharedAttributes(t *testing.T) {
	events := []TimelineEvent{
		{ID: "1", SourceIP: "10.0.0.1", UserID: "user1"},
		{ID: "2", SourceIP: "10.0.0.1", AssetID: "asset1"},
		{ID: "3", UserID: "user1"},
	}

	groups := CorrelateEvents(events)

	if len(groups["ip:10.0.0.1"]) != 2 {
		t.Errorf("expected 2 events for IP 10.0.0.1, got %d", len(groups["ip:10.0.0.1"]))
	}
	if len(groups["user:user1"]) != 2 {
		t.Errorf("expected 2 events for user1, got %d", len(groups["user:user1"]))
	}
	if len(groups["asset:asset1"]) != 1 {
		t.Errorf("expected 1 event for asset1, got %d", len(groups["asset:asset1"]))
	}
}

func TestCorrelateEvents_Empty(t *testing.T) {
	groups := CorrelateEvents(nil)
	if len(groups) != 0 {
		t.Errorf("expected empty groups, got %d", len(groups))
	}
}

// --- buildEvidence ---

func TestBuildEvidence_AllFields(t *testing.T) {
	evt := TimelineEvent{
		SourceIP:    "10.0.0.1",
		UserID:      "user1",
		AssetID:     "asset1",
		MITRETechID: "T1078",
	}

	evidence := buildEvidence(evt)
	if len(evidence) != 4 {
		t.Fatalf("expected 4 evidence items, got %d", len(evidence))
	}
}

func TestBuildEvidence_Empty(t *testing.T) {
	evt := TimelineEvent{}
	evidence := buildEvidence(evt)
	if len(evidence) != 0 {
		t.Errorf("expected 0 evidence items for empty event, got %d", len(evidence))
	}
}

// --- ClassifySecurityRootCause ---

func TestClassifySecurityRootCause_ExposedService(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "initial-access", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "exposed_service" {
		t.Errorf("expected 'exposed_service', got %q", got)
	}
}

func TestClassifySecurityRootCause_CredentialCompromise(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "credential-access", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "credential_compromise" {
		t.Errorf("expected 'credential_compromise', got %q", got)
	}
}

func TestClassifySecurityRootCause_InsiderThreat(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "exfiltration", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "insider_threat" {
		t.Errorf("expected 'insider_threat', got %q", got)
	}
}

func TestClassifySecurityRootCause_LateralMovement(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "lateral-movement", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "lateral_movement" {
		t.Errorf("expected 'lateral_movement', got %q", got)
	}
}

func TestClassifySecurityRootCause_UnpatchedVulnerability(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "execution", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "unpatched_vulnerability" {
		t.Errorf("expected 'unpatched_vulnerability', got %q", got)
	}
}

func TestClassifySecurityRootCause_ByTechniqueID(t *testing.T) {
	chain := []CausalStep{
		{MITRETechID: "T1078.001", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "credential_compromise" {
		t.Errorf("expected 'credential_compromise' for T1078, got %q", got)
	}
}

func TestClassifySecurityRootCause_ByDescription(t *testing.T) {
	chain := []CausalStep{
		{Description: "Brute force login attempt detected", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "credential_compromise" {
		t.Errorf("expected 'credential_compromise' for brute force, got %q", got)
	}
}

func TestClassifySecurityRootCause_Unknown(t *testing.T) {
	chain := []CausalStep{
		{Description: "Something happened", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "unknown" {
		t.Errorf("expected 'unknown', got %q", got)
	}
}

func TestClassifySecurityRootCause_EmptyChain(t *testing.T) {
	if got := ClassifySecurityRootCause(nil); got != "unknown" {
		t.Errorf("expected 'unknown' for empty chain, got %q", got)
	}
}

func TestClassifySecurityRootCause_UsesRootCauseStep(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "execution", IsRootCause: false},
		{MITREPhase: "credential-access", IsRootCause: true},
	}
	if got := ClassifySecurityRootCause(chain); got != "credential_compromise" {
		t.Errorf("should use the IsRootCause step, got %q", got)
	}
}

func TestClassifySecurityRootCause_FallsBackToFirstStep(t *testing.T) {
	chain := []CausalStep{
		{MITREPhase: "initial-access"},
		{MITREPhase: "execution"},
	}
	// Neither is marked IsRootCause, should fall back to first
	if got := ClassifySecurityRootCause(chain); got != "exposed_service" {
		t.Errorf("should fall back to first step, got %q", got)
	}
}

// --- Recommender ---

func TestRecommender_ForSecurityAlert_ExposedService(t *testing.T) {
	r := NewRecommender()
	recs := r.ForSecurityAlert("exposed_service", nil)
	if len(recs) < 3 {
		t.Errorf("expected at least 3 recommendations, got %d", len(recs))
	}
	if recs[0].Category != "immediate" {
		t.Errorf("first recommendation should be 'immediate', got %q", recs[0].Category)
	}
}

func TestRecommender_ForSecurityAlert_CredentialCompromise(t *testing.T) {
	r := NewRecommender()
	recs := r.ForSecurityAlert("credential_compromise", nil)
	if len(recs) < 3 {
		t.Errorf("expected at least 3 recommendations, got %d", len(recs))
	}
}

func TestRecommender_ForSecurityAlert_Unknown(t *testing.T) {
	r := NewRecommender()
	recs := r.ForSecurityAlert("unknown", nil)
	if len(recs) == 0 {
		t.Error("should return default recommendations for unknown type")
	}
}

func TestRecommender_ForPipelineFailure_AllTypes(t *testing.T) {
	r := NewRecommender()
	types := []string{"upstream_failure", "schema_drift", "connection_timeout", "resource_exhaustion", "credential_expiry", "quality_gate", "unknown"}

	for _, rcType := range types {
		recs := r.ForPipelineFailure(rcType)
		if len(recs) == 0 {
			t.Errorf("ForPipelineFailure(%q) returned 0 recommendations", rcType)
		}
		// Verify priority ordering
		for i, rec := range recs {
			if rec.Priority != i+1 {
				t.Errorf("ForPipelineFailure(%q): recommendation[%d].Priority = %d, want %d", rcType, i, rec.Priority, i+1)
			}
		}
	}
}

func TestRecommender_ForQualityIssue_AllTypes(t *testing.T) {
	r := NewRecommender()
	types := []string{"upstream_quality", "schema_change", "unknown"}

	for _, rcType := range types {
		recs := r.ForQualityIssue(rcType)
		if len(recs) == 0 {
			t.Errorf("ForQualityIssue(%q) returned 0 recommendations", rcType)
		}
		for _, rec := range recs {
			if rec.RootCauseType == "" {
				t.Errorf("ForQualityIssue(%q): recommendation has empty RootCauseType", rcType)
			}
		}
	}
}

func TestRecommender_ForSecurityAlert_PrioritiesAscending(t *testing.T) {
	r := NewRecommender()
	recs := r.ForSecurityAlert("lateral_movement", nil)
	for i := 1; i < len(recs); i++ {
		if recs[i].Priority <= recs[i-1].Priority {
			t.Errorf("recommendations not in ascending priority order at index %d", i)
		}
	}
}

func TestRecommender_ForSecurityAlert_HasImmediateAndLongTerm(t *testing.T) {
	r := NewRecommender()
	types := []string{"exposed_service", "credential_compromise", "insider_threat", "lateral_movement", "unpatched_vulnerability"}

	for _, rcType := range types {
		recs := r.ForSecurityAlert(rcType, nil)
		hasImmediate := false
		hasLongTerm := false
		for _, rec := range recs {
			if rec.Category == "immediate" {
				hasImmediate = true
			}
			if rec.Category == "long_term" {
				hasLongTerm = true
			}
		}
		if !hasImmediate {
			t.Errorf("ForSecurityAlert(%q): no 'immediate' recommendation", rcType)
		}
		if !hasLongTerm {
			t.Errorf("ForSecurityAlert(%q): no 'long_term' recommendation", rcType)
		}
	}
}

// --- Impact Assessment (assessBusinessImpact) ---

func TestAssessBusinessImpact_Critical(t *testing.T) {
	direct := []AffectedAsset{{Criticality: "critical"}}
	dataRisk := []DataRisk{{Classification: "restricted"}}
	if got := assessBusinessImpact(direct, nil, dataRisk); got != "critical" {
		t.Errorf("expected 'critical', got %q", got)
	}
}

func TestAssessBusinessImpact_High_CriticalAssetOnly(t *testing.T) {
	direct := []AffectedAsset{{Criticality: "critical"}}
	if got := assessBusinessImpact(direct, nil, nil); got != "high" {
		t.Errorf("expected 'high', got %q", got)
	}
}

func TestAssessBusinessImpact_High_RestrictedDataOnly(t *testing.T) {
	dataRisk := []DataRisk{{Classification: "restricted"}}
	if got := assessBusinessImpact(nil, nil, dataRisk); got != "high" {
		t.Errorf("expected 'high', got %q", got)
	}
}

func TestAssessBusinessImpact_Medium_ManyAssets(t *testing.T) {
	direct := []AffectedAsset{{}, {}, {}, {}}
	if got := assessBusinessImpact(direct, nil, nil); got != "medium" {
		t.Errorf("expected 'medium' for >3 assets, got %q", got)
	}
}

func TestAssessBusinessImpact_Medium_AnyDataRisk(t *testing.T) {
	dataRisk := []DataRisk{{Classification: "confidential"}}
	if got := assessBusinessImpact(nil, nil, dataRisk); got != "medium" {
		t.Errorf("expected 'medium' for non-restricted data risk, got %q", got)
	}
}

func TestAssessBusinessImpact_Low(t *testing.T) {
	direct := []AffectedAsset{{Criticality: "low"}}
	if got := assessBusinessImpact(direct, nil, nil); got != "low" {
		t.Errorf("expected 'low', got %q", got)
	}
}

// --- buildImpactSummary ---

func TestBuildImpactSummary(t *testing.T) {
	s := buildImpactSummary(5, 100, 2, "high")
	if s == "" {
		t.Error("summary should not be empty")
	}
	// Should contain the numbers and impact level
	for _, expected := range []string{"5", "100", "2", "high"} {
		if !contains(s, expected) {
			t.Errorf("summary should contain %q, got %q", expected, s)
		}
	}
}

// --- Helpers ---

func TestPtrToString_Nil(t *testing.T) {
	if got := ptrToString(nil); got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestPtrToString_Value(t *testing.T) {
	s := "hello"
	if got := ptrToString(&s); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestItoa(t *testing.T) {
	if got := itoa(42); got != "42" {
		t.Errorf("expected '42', got %q", got)
	}
}

// --- killChainOrder coverage ---

func TestKillChainOrder_AllPhasesPresent(t *testing.T) {
	expected := []string{
		"reconnaissance", "resource-development", "initial-access",
		"execution", "persistence", "privilege-escalation",
		"defense-evasion", "credential-access", "discovery",
		"lateral-movement", "collection", "command-and-control",
		"exfiltration", "impact",
	}

	for _, phase := range expected {
		if _, ok := killChainOrder[phase]; !ok {
			t.Errorf("killChainOrder missing phase %q", phase)
		}
	}

	if len(killChainOrder) != len(expected) {
		t.Errorf("killChainOrder has %d entries, expected %d", len(killChainOrder), len(expected))
	}
}

func TestKillChainOrder_Monotonic(t *testing.T) {
	phases := []string{
		"reconnaissance", "resource-development", "initial-access",
		"execution", "persistence", "privilege-escalation",
		"defense-evasion", "credential-access", "discovery",
		"lateral-movement", "collection", "command-and-control",
		"exfiltration", "impact",
	}

	for i := 1; i < len(phases); i++ {
		if killChainOrder[phases[i]] <= killChainOrder[phases[i-1]] {
			t.Errorf("phase %q (order %d) should be after %q (order %d)",
				phases[i], killChainOrder[phases[i]], phases[i-1], killChainOrder[phases[i-1]])
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
