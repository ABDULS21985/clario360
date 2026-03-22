package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/mitre"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

// ---------------------------------------------------------------------------
// mock + helpers
// ---------------------------------------------------------------------------

type mockMitreRuleService struct {
	coverageFn        func(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) ([]dto.MITRECoverageDTO, error)
	techniqueDetailFn func(ctx context.Context, tenantID uuid.UUID, techniqueID string, actor *service.Actor) (*dto.MITRETechniqueDetailDTO, error)
}

func (m *mockMitreRuleService) Coverage(ctx context.Context, tenantID uuid.UUID, actor *service.Actor) ([]dto.MITRECoverageDTO, error) {
	if m.coverageFn != nil {
		return m.coverageFn(ctx, tenantID, actor)
	}
	return []dto.MITRECoverageDTO{}, nil
}

func (m *mockMitreRuleService) TechniqueDetail(ctx context.Context, tenantID uuid.UUID, techniqueID string, actor *service.Actor) (*dto.MITRETechniqueDetailDTO, error) {
	if m.techniqueDetailFn != nil {
		return m.techniqueDetailFn(ctx, tenantID, techniqueID, actor)
	}
	return nil, repository.ErrNotFound
}

func mitreAuthRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, path, nil)
	tenantID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	ctx := r.Context()
	ctx = auth.WithTenantID(ctx, tenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       userID.String(),
		TenantID: tenantID.String(),
		Email:    "admin@example.com",
		Roles:    []string{"security_admin"},
	})
	return r.WithContext(ctx)
}

func mitreAuthRequestWithID(method, path, paramName, id string) *http.Request {
	r := mitreAuthRequest(method, path)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(paramName, id)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func mitreNoAuthRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

// ---------------------------------------------------------------------------
// ListTactics
// ---------------------------------------------------------------------------

func TestMITREHandler_ListTactics(t *testing.T) {
	h := NewMITREHandler(&mockMitreRuleService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/mitre/tactics", nil)
	h.ListTactics(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]dto.MITRETacticDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	tactics := resp["data"]
	allTactics := mitre.AllTactics()
	if len(tactics) != len(allTactics) {
		t.Fatalf("expected %d tactics, got %d", len(allTactics), len(tactics))
	}
	// Verify first tactic has expected fields
	if tactics[0].ID == "" || tactics[0].Name == "" || tactics[0].ShortName == "" {
		t.Error("tactic fields should be populated")
	}
}

// ---------------------------------------------------------------------------
// ListTechniques
// ---------------------------------------------------------------------------

func TestMITREHandler_ListTechniques_All(t *testing.T) {
	h := NewMITREHandler(&mockMitreRuleService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/mitre/techniques", nil)
	h.ListTechniques(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]dto.MITRETechniqueDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	techniques := resp["data"]
	allTechniques := mitre.AllTechniques()
	if len(techniques) != len(allTechniques) {
		t.Fatalf("expected %d techniques, got %d", len(allTechniques), len(techniques))
	}
	// Verify fields
	for _, tech := range techniques {
		if tech.ID == "" || tech.Name == "" {
			t.Error("technique fields should be populated")
		}
		if len(tech.TacticIDs) == 0 {
			t.Errorf("technique %s should have at least one tactic_id", tech.ID)
		}
	}
}

func TestMITREHandler_ListTechniques_FilteredByTactic(t *testing.T) {
	h := NewMITREHandler(&mockMitreRuleService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/mitre/techniques?tactic_id=TA0002", nil)
	h.ListTechniques(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]dto.MITRETechniqueDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	techniques := resp["data"]
	if len(techniques) == 0 {
		t.Fatal("expected at least one technique for TA0002 (Execution)")
	}
	for _, tech := range techniques {
		found := false
		for _, tid := range tech.TacticIDs {
			if tid == "TA0002" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("technique %s should belong to TA0002", tech.ID)
		}
	}
}

func TestMITREHandler_ListTechniques_CommaSeparatedTactics(t *testing.T) {
	h := NewMITREHandler(&mockMitreRuleService{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/cyber/mitre/techniques?tactic_id=TA0002,TA0040", nil)
	h.ListTechniques(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string][]dto.MITRETechniqueDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp["data"]) == 0 {
		t.Fatal("expected techniques for TA0002 and TA0040")
	}
}

// ---------------------------------------------------------------------------
// GetTechnique
// ---------------------------------------------------------------------------

func TestMITREHandler_GetTechnique_NoAuth(t *testing.T) {
	h := NewMITREHandler(&mockMitreRuleService{})
	w := httptest.NewRecorder()
	r := mitreNoAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/techniques/T1059")
	h.GetTechnique(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestMITREHandler_GetTechnique_Success(t *testing.T) {
	now := time.Now().UTC()
	ruleID := uuid.New()
	detail := &dto.MITRETechniqueDetailDTO{
		ID:                "T1059",
		Name:              "Command and Scripting Interpreter",
		Description:       "Adversaries may abuse command and script interpreters.",
		TacticIDs:         []string{"TA0002"},
		Platforms:         []string{"Windows", "Linux", "macOS"},
		DataSources:       []string{"Process Creation"},
		CoverageState:     "covered",
		RuleCount:         1,
		AlertCount:        5,
		ThreatCount:       2,
		ActiveThreatCount: 1,
		HighFPRuleCount:   0,
		LastAlertAt:       &now,
		LinkedRules: []dto.MITRERuleReferenceDTO{
			{
				ID:                 ruleID,
				Name:               "PowerShell Suspicious Activity",
				RuleType:           "sigma",
				Severity:           "high",
				Enabled:            true,
				TriggerCount:       10,
				TruePositiveCount:  8,
				FalsePositiveCount: 2,
				LastTriggeredAt:    &now,
			},
		},
		LinkedThreats: []dto.MITREThreatReferenceDTO{},
		RecentAlerts:  []dto.MITREAlertReferenceDTO{},
	}

	mock := &mockMitreRuleService{
		techniqueDetailFn: func(_ context.Context, _ uuid.UUID, techniqueID string, _ *service.Actor) (*dto.MITRETechniqueDetailDTO, error) {
			if techniqueID == "T1059" {
				return detail, nil
			}
			return nil, repository.ErrNotFound
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequestWithID(http.MethodGet, "/api/v1/cyber/mitre/techniques/T1059", "id", "T1059")
	h.GetTechnique(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]*dto.MITRETechniqueDetailDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := resp["data"]
	if got.ID != "T1059" {
		t.Errorf("expected id T1059, got %s", got.ID)
	}
	if got.CoverageState != "covered" {
		t.Errorf("expected coverage_state covered, got %s", got.CoverageState)
	}
	if len(got.LinkedRules) != 1 {
		t.Errorf("expected 1 linked rule, got %d", len(got.LinkedRules))
	}
	if got.LinkedRules[0].ID != ruleID {
		t.Errorf("expected rule id %s, got %s", ruleID, got.LinkedRules[0].ID)
	}
}

func TestMITREHandler_GetTechnique_NotFound(t *testing.T) {
	mock := &mockMitreRuleService{
		techniqueDetailFn: func(_ context.Context, _ uuid.UUID, _ string, _ *service.Actor) (*dto.MITRETechniqueDetailDTO, error) {
			return nil, repository.ErrNotFound
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequestWithID(http.MethodGet, "/api/v1/cyber/mitre/techniques/T9999", "id", "T9999")
	h.GetTechnique(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestMITREHandler_GetTechnique_ServiceError(t *testing.T) {
	mock := &mockMitreRuleService{
		techniqueDetailFn: func(_ context.Context, _ uuid.UUID, _ string, _ *service.Actor) (*dto.MITRETechniqueDetailDTO, error) {
			return nil, fmt.Errorf("database error")
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequestWithID(http.MethodGet, "/api/v1/cyber/mitre/techniques/T1059", "id", "T1059")
	h.GetTechnique(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for generic error, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Coverage
// ---------------------------------------------------------------------------

func TestMITREHandler_Coverage_NoAuth(t *testing.T) {
	h := NewMITREHandler(&mockMitreRuleService{})
	w := httptest.NewRecorder()
	r := mitreNoAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/coverage")
	h.Coverage(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestMITREHandler_Coverage_Success(t *testing.T) {
	now := time.Now().UTC()
	coverageItems := []dto.MITRECoverageDTO{
		{
			TechniqueID:       "T1059",
			TechniqueName:     "Command and Scripting Interpreter",
			TacticIDs:         []string{"TA0002"},
			HasDetection:      true,
			RuleCount:         2,
			RuleNames:         []string{"Rule A", "Rule B"},
			CoverageState:     "covered",
			HighFPRuleCount:   0,
			AlertCount:        3,
			ThreatCount:       1,
			ActiveThreatCount: 0,
			LastAlertAt:       &now,
			Description:       "Command interpreter",
			Platforms:         []string{"Windows"},
		},
		{
			TechniqueID:       "T1566",
			TechniqueName:     "Phishing",
			TacticIDs:         []string{"TA0001"},
			HasDetection:      false,
			RuleCount:         0,
			RuleNames:         []string{},
			CoverageState:     "gap",
			HighFPRuleCount:   0,
			AlertCount:        0,
			ThreatCount:       1,
			ActiveThreatCount: 1,
			Description:       "Phishing attempts",
			Platforms:         []string{"PRE"},
		},
		{
			TechniqueID:       "T1110",
			TechniqueName:     "Brute Force",
			TacticIDs:         []string{"TA0006"},
			HasDetection:      true,
			RuleCount:         1,
			RuleNames:         []string{"Brute Force Rule"},
			CoverageState:     "noisy",
			HighFPRuleCount:   1,
			AlertCount:        10,
			ThreatCount:       0,
			ActiveThreatCount: 0,
			Description:       "Brute force attacks",
			Platforms:         []string{"Windows", "Linux"},
		},
	}

	mock := &mockMitreRuleService{
		coverageFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor) ([]dto.MITRECoverageDTO, error) {
			return coverageItems, nil
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/coverage")
	h.Coverage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]dto.MITRECoverageResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := resp["data"]

	// Verify aggregation
	if got.TotalTechniques != 3 {
		t.Errorf("expected total_techniques=3, got %d", got.TotalTechniques)
	}
	if got.CoveredTechniques != 2 {
		t.Errorf("expected covered_techniques=2, got %d", got.CoveredTechniques)
	}
	if got.ActiveTechniques != 2 {
		t.Errorf("expected active_techniques=2 (T1059 has 3 alerts, T1110 has 10), got %d", got.ActiveTechniques)
	}
	if got.PassiveTechniques != 0 {
		t.Errorf("expected passive_techniques=0, got %d", got.PassiveTechniques)
	}
	if got.CriticalGapCount != 1 {
		t.Errorf("expected critical_gap_count=1 (T1566), got %d", got.CriticalGapCount)
	}

	expectedPercent := float64(2) / float64(3) * 100
	if got.CoveragePercent < expectedPercent-0.01 || got.CoveragePercent > expectedPercent+0.01 {
		t.Errorf("expected coverage_percent=%.2f, got %.2f", expectedPercent, got.CoveragePercent)
	}

	// Verify tactics are populated (14 tactics)
	if len(got.Tactics) != len(mitre.AllTactics()) {
		t.Errorf("expected %d tactics, got %d", len(mitre.AllTactics()), len(got.Tactics))
	}

	// Verify technique list
	if len(got.Techniques) != 3 {
		t.Errorf("expected 3 techniques, got %d", len(got.Techniques))
	}
}

func TestMITREHandler_Coverage_Empty(t *testing.T) {
	mock := &mockMitreRuleService{
		coverageFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor) ([]dto.MITRECoverageDTO, error) {
			return []dto.MITRECoverageDTO{}, nil
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/coverage")
	h.Coverage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]dto.MITRECoverageResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := resp["data"]
	if got.TotalTechniques != 0 {
		t.Errorf("expected total_techniques=0, got %d", got.TotalTechniques)
	}
	if got.CoveragePercent != 0 {
		t.Errorf("expected coverage_percent=0, got %f", got.CoveragePercent)
	}
}

func TestMITREHandler_Coverage_ServiceError(t *testing.T) {
	mock := &mockMitreRuleService{
		coverageFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor) ([]dto.MITRECoverageDTO, error) {
			return nil, fmt.Errorf("database down")
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/coverage")
	h.Coverage(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Coverage aggregation: multi-tactic technique
// ---------------------------------------------------------------------------

func TestMITREHandler_Coverage_MultiTacticTechnique(t *testing.T) {
	// T1078 "Valid Accounts" belongs to multiple tactics
	coverageItems := []dto.MITRECoverageDTO{
		{
			TechniqueID:   "T1078",
			TechniqueName: "Valid Accounts",
			TacticIDs:     []string{"TA0001", "TA0003", "TA0004", "TA0011"},
			HasDetection:  true,
			RuleCount:     1,
			RuleNames:     []string{"Suspicious Login"},
			CoverageState: "covered",
			Description:   "Valid accounts abused",
			Platforms:     []string{"Windows", "Cloud"},
		},
	}

	mock := &mockMitreRuleService{
		coverageFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor) ([]dto.MITRECoverageDTO, error) {
			return coverageItems, nil
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/coverage")
	h.Coverage(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]dto.MITRECoverageResponseDTO
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := resp["data"]

	// T1078 should count as 1 technique total, 1 covered
	if got.TotalTechniques != 1 {
		t.Errorf("expected total_techniques=1, got %d", got.TotalTechniques)
	}
	if got.CoveredTechniques != 1 {
		t.Errorf("expected covered_techniques=1, got %d", got.CoveredTechniques)
	}

	// But multiple tactics should each count this technique
	tacticMap := map[string]dto.MITRETacticCoverageDTO{}
	for _, tc := range got.Tactics {
		tacticMap[tc.ID] = tc
	}

	for _, tid := range []string{"TA0001", "TA0003", "TA0004", "TA0011"} {
		tc, ok := tacticMap[tid]
		if !ok {
			t.Errorf("tactic %s not found in response", tid)
			continue
		}
		if tc.TechniqueCount != 1 {
			t.Errorf("tactic %s: expected technique_count=1, got %d", tid, tc.TechniqueCount)
		}
		if tc.CoveredCount != 1 {
			t.Errorf("tactic %s: expected covered_count=1, got %d", tid, tc.CoveredCount)
		}
	}
}

// ---------------------------------------------------------------------------
// Contract verification: JSON field names match frontend expectations
// ---------------------------------------------------------------------------

func TestMITREHandler_Coverage_JSONFieldNames(t *testing.T) {
	mock := &mockMitreRuleService{
		coverageFn: func(_ context.Context, _ uuid.UUID, _ *service.Actor) ([]dto.MITRECoverageDTO, error) {
			return []dto.MITRECoverageDTO{
				{
					TechniqueID:       "T1059",
					TechniqueName:     "PowerShell",
					TacticIDs:         []string{"TA0002"},
					HasDetection:      true,
					RuleCount:         1,
					RuleNames:         []string{"TestRule"},
					CoverageState:     "covered",
					HighFPRuleCount:   0,
					AlertCount:        5,
					ThreatCount:       2,
					ActiveThreatCount: 1,
					Description:       "test",
					Platforms:         []string{"Windows"},
				},
			}, nil
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequest(http.MethodGet, "/api/v1/cyber/mitre/coverage")
	h.Coverage(w, r)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	// Verify all top-level fields the frontend expects
	expectedFields := []string{
		"tactics", "techniques", "total_techniques", "covered_techniques",
		"coverage_percent", "active_techniques", "passive_techniques", "critical_gap_count",
	}
	for _, field := range expectedFields {
		if _, ok := data[field]; !ok {
			t.Errorf("missing expected field %q in coverage response", field)
		}
	}

	// Verify technique-level fields
	var techniques []map[string]json.RawMessage
	if err := json.Unmarshal(data["techniques"], &techniques); err != nil {
		t.Fatalf("unmarshal techniques: %v", err)
	}
	if len(techniques) == 0 {
		t.Fatal("expected at least one technique")
	}
	techFields := []string{
		"technique_id", "technique_name", "tactic_ids", "has_detection",
		"rule_count", "rule_names", "coverage_state", "high_fp_rule_count",
		"alert_count", "threat_count", "active_threat_count",
		"description", "platforms",
	}
	for _, field := range techFields {
		if _, ok := techniques[0][field]; !ok {
			t.Errorf("missing expected technique field %q", field)
		}
	}
}

func TestMITREHandler_TechniqueDetail_JSONFieldNames(t *testing.T) {
	now := time.Now().UTC()
	alertID := uuid.New()
	ruleID := uuid.New()
	threatID := uuid.New()

	mock := &mockMitreRuleService{
		techniqueDetailFn: func(_ context.Context, _ uuid.UUID, _ string, _ *service.Actor) (*dto.MITRETechniqueDetailDTO, error) {
			return &dto.MITRETechniqueDetailDTO{
				ID:                "T1059",
				Name:              "Command and Scripting Interpreter",
				Description:       "desc",
				TacticIDs:         []string{"TA0002"},
				Platforms:         []string{"Windows"},
				DataSources:       []string{"Process Creation"},
				CoverageState:     "covered",
				RuleCount:         1,
				AlertCount:        2,
				ThreatCount:       1,
				ActiveThreatCount: 1,
				HighFPRuleCount:   0,
				LastAlertAt:       &now,
				LinkedRules: []dto.MITRERuleReferenceDTO{
					{ID: ruleID, Name: "Rule1", RuleType: "sigma", Severity: "high", Enabled: true, TriggerCount: 5, TruePositiveCount: 4, FalsePositiveCount: 1, LastTriggeredAt: &now},
				},
				LinkedThreats: []dto.MITREThreatReferenceDTO{
					{ID: threatID, Name: "APT29", Type: "apt", Severity: "critical", Status: "active", LastSeenAt: now},
				},
				RecentAlerts: []dto.MITREAlertReferenceDTO{
					{ID: alertID, Title: "PowerShell Alert", Severity: "high", Status: "new", ConfidenceScore: 0.85, CreatedAt: now},
				},
			}, nil
		},
	}
	h := NewMITREHandler(mock)
	w := httptest.NewRecorder()
	r := mitreAuthRequestWithID(http.MethodGet, "/api/v1/cyber/mitre/techniques/T1059", "id", "T1059")
	h.GetTechnique(w, r)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var data map[string]json.RawMessage
	if err := json.Unmarshal(raw["data"], &data); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}

	// Verify all fields the frontend MITRETechniqueDetail type expects
	expectedFields := []string{
		"id", "name", "description", "tactic_ids", "platforms", "data_sources",
		"coverage_state", "rule_count", "alert_count", "threat_count",
		"active_threat_count", "high_fp_rule_count", "last_alert_at",
		"linked_rules", "linked_threats", "recent_alerts",
	}
	for _, field := range expectedFields {
		if _, ok := data[field]; !ok {
			t.Errorf("missing expected field %q in technique detail response", field)
		}
	}

	// Verify linked_rules fields
	var rules []map[string]json.RawMessage
	if err := json.Unmarshal(data["linked_rules"], &rules); err != nil {
		t.Fatalf("unmarshal linked_rules: %v", err)
	}
	ruleFields := []string{"id", "name", "rule_type", "severity", "enabled", "trigger_count", "true_positive_count", "false_positive_count", "last_triggered_at"}
	for _, field := range ruleFields {
		if _, ok := rules[0][field]; !ok {
			t.Errorf("missing expected rule field %q", field)
		}
	}

	// Verify linked_threats fields
	var threats []map[string]json.RawMessage
	if err := json.Unmarshal(data["linked_threats"], &threats); err != nil {
		t.Fatalf("unmarshal linked_threats: %v", err)
	}
	threatFields := []string{"id", "name", "type", "severity", "status", "last_seen_at"}
	for _, field := range threatFields {
		if _, ok := threats[0][field]; !ok {
			t.Errorf("missing expected threat field %q", field)
		}
	}

	// Verify recent_alerts fields
	var alerts []map[string]json.RawMessage
	if err := json.Unmarshal(data["recent_alerts"], &alerts); err != nil {
		t.Fatalf("unmarshal recent_alerts: %v", err)
	}
	alertFields := []string{"id", "title", "severity", "status", "confidence_score", "created_at"}
	for _, field := range alertFields {
		if _, ok := alerts[0][field]; !ok {
			t.Errorf("missing expected alert field %q", field)
		}
	}
}
