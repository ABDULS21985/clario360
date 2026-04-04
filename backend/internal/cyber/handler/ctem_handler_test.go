package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clario360/platform/internal/cyber/dto"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func ctemRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

// ---------------------------------------------------------------------------
// parseCTEMAssessmentListParams
// ---------------------------------------------------------------------------

func TestParseCTEMAssessmentListParams_Defaults(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Status != nil {
		t.Errorf("expected Status nil, got %v", *params.Status)
	}
	if params.Scheduled != nil {
		t.Errorf("expected Scheduled nil, got %v", *params.Scheduled)
	}
	if params.Search != nil {
		t.Errorf("expected Search nil, got %v", *params.Search)
	}
	if params.Tag != nil {
		t.Errorf("expected Tag nil, got %v", *params.Tag)
	}
	if params.Page != 0 {
		t.Errorf("expected Page 0 (pre-default), got %d", params.Page)
	}
	if params.PerPage != 0 {
		t.Errorf("expected PerPage 0 (pre-default), got %d", params.PerPage)
	}
	if params.Sort != "" {
		t.Errorf("expected Sort empty, got %q", params.Sort)
	}
	if params.Order != "" {
		t.Errorf("expected Order empty, got %q", params.Order)
	}
}

func TestParseCTEMAssessmentListParams_StatusFilter(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?status=completed")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Status == nil || *params.Status != "completed" {
		t.Errorf("expected status=completed, got %v", params.Status)
	}
}

func TestParseCTEMAssessmentListParams_ScheduledTrue(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?scheduled=true")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Scheduled == nil || !*params.Scheduled {
		t.Errorf("expected scheduled=true, got %v", params.Scheduled)
	}
}

func TestParseCTEMAssessmentListParams_ScheduledFalse(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?scheduled=false")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Scheduled == nil || *params.Scheduled {
		t.Errorf("expected scheduled=false, got %v", params.Scheduled)
	}
}

func TestParseCTEMAssessmentListParams_InvalidScheduled(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?scheduled=notabool")
	_, err := parseCTEMAssessmentListParams(r)
	if err == nil {
		t.Fatal("expected error for invalid scheduled value, got nil")
	}
}

func TestParseCTEMAssessmentListParams_SearchFilter(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?search=network")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Search == nil || *params.Search != "network" {
		t.Errorf("expected search=network, got %v", params.Search)
	}
}

func TestParseCTEMAssessmentListParams_TagFilter(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?tag=production")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Tag == nil || *params.Tag != "production" {
		t.Errorf("expected tag=production, got %v", params.Tag)
	}
}

func TestParseCTEMAssessmentListParams_Pagination(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?page=3&per_page=50")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 3 {
		t.Errorf("expected page=3, got %d", params.Page)
	}
	if params.PerPage != 50 {
		t.Errorf("expected per_page=50, got %d", params.PerPage)
	}
}

func TestParseCTEMAssessmentListParams_InvalidPage(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?page=abc")
	_, err := parseCTEMAssessmentListParams(r)
	if err == nil {
		t.Fatal("expected error for non-numeric page, got nil")
	}
}

func TestParseCTEMAssessmentListParams_InvalidPerPage(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?per_page=xyz")
	_, err := parseCTEMAssessmentListParams(r)
	if err == nil {
		t.Fatal("expected error for non-numeric per_page, got nil")
	}
}

func TestParseCTEMAssessmentListParams_SortAndOrder(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?sort=exposure_score&order=asc")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Sort != "exposure_score" {
		t.Errorf("expected sort=exposure_score, got %q", params.Sort)
	}
	if params.Order != "asc" {
		t.Errorf("expected order=asc, got %q", params.Order)
	}
}

func TestParseCTEMAssessmentListParams_AllFilters(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments?status=scoping&scheduled=true&search=quarterly&tag=critical&page=2&per_page=10&sort=name&order=desc")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Status == nil || *params.Status != "scoping" {
		t.Errorf("expected status=scoping, got %v", params.Status)
	}
	if params.Scheduled == nil || !*params.Scheduled {
		t.Errorf("expected scheduled=true, got %v", params.Scheduled)
	}
	if params.Search == nil || *params.Search != "quarterly" {
		t.Errorf("expected search=quarterly, got %v", params.Search)
	}
	if params.Tag == nil || *params.Tag != "critical" {
		t.Errorf("expected tag=critical, got %v", params.Tag)
	}
	if params.Page != 2 {
		t.Errorf("expected page=2, got %d", params.Page)
	}
	if params.PerPage != 10 {
		t.Errorf("expected per_page=10, got %d", params.PerPage)
	}
	if params.Sort != "name" {
		t.Errorf("expected sort=name, got %q", params.Sort)
	}
	if params.Order != "desc" {
		t.Errorf("expected order=desc, got %q", params.Order)
	}
}

// ---------------------------------------------------------------------------
// parseCTEMFindingListParams
// ---------------------------------------------------------------------------

func TestParseCTEMFindingListParams_Defaults(t *testing.T) {
	r := ctemRequest("GET", "/api/v1/cyber/ctem/assessments/abc/findings")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Severity != nil {
		t.Errorf("expected Severity nil, got %v", *params.Severity)
	}
	if params.Type != nil {
		t.Errorf("expected Type nil, got %v", *params.Type)
	}
	if params.Status != nil {
		t.Errorf("expected Status nil, got %v", *params.Status)
	}
	if params.PriorityGroup != nil {
		t.Errorf("expected PriorityGroup nil, got %v", *params.PriorityGroup)
	}
	if params.Search != nil {
		t.Errorf("expected Search nil, got %v", *params.Search)
	}
	if params.Page != 0 {
		t.Errorf("expected Page 0 (pre-default), got %d", params.Page)
	}
	if params.PerPage != 0 {
		t.Errorf("expected PerPage 0 (pre-default), got %d", params.PerPage)
	}
	if params.Sort != "" {
		t.Errorf("expected Sort empty, got %q", params.Sort)
	}
	if params.Order != "" {
		t.Errorf("expected Order empty, got %q", params.Order)
	}
}

func TestParseCTEMFindingListParams_SeverityFilter(t *testing.T) {
	r := ctemRequest("GET", "/findings?severity=critical")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Severity == nil || *params.Severity != "critical" {
		t.Errorf("expected severity=critical, got %v", params.Severity)
	}
}

func TestParseCTEMFindingListParams_TypeFilter(t *testing.T) {
	r := ctemRequest("GET", "/findings?type=vulnerability")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Type == nil || *params.Type != "vulnerability" {
		t.Errorf("expected type=vulnerability, got %v", params.Type)
	}
}

func TestParseCTEMFindingListParams_StatusFilter(t *testing.T) {
	r := ctemRequest("GET", "/findings?status=open")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Status == nil || *params.Status != "open" {
		t.Errorf("expected status=open, got %v", params.Status)
	}
}

func TestParseCTEMFindingListParams_PriorityGroupFilter(t *testing.T) {
	r := ctemRequest("GET", "/findings?priority_group=2")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.PriorityGroup == nil || *params.PriorityGroup != 2 {
		t.Errorf("expected priority_group=2, got %v", params.PriorityGroup)
	}
}

func TestParseCTEMFindingListParams_InvalidPriorityGroup(t *testing.T) {
	r := ctemRequest("GET", "/findings?priority_group=notanumber")
	_, err := parseCTEMFindingListParams(r)
	if err == nil {
		t.Fatal("expected error for non-numeric priority_group, got nil")
	}
}

func TestParseCTEMFindingListParams_SearchFilter(t *testing.T) {
	r := ctemRequest("GET", "/findings?search=CVE-2024")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Search == nil || *params.Search != "CVE-2024" {
		t.Errorf("expected search=CVE-2024, got %v", params.Search)
	}
}

func TestParseCTEMFindingListParams_Pagination(t *testing.T) {
	r := ctemRequest("GET", "/findings?page=5&per_page=100")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 5 {
		t.Errorf("expected page=5, got %d", params.Page)
	}
	if params.PerPage != 100 {
		t.Errorf("expected per_page=100, got %d", params.PerPage)
	}
}

func TestParseCTEMFindingListParams_InvalidPage(t *testing.T) {
	r := ctemRequest("GET", "/findings?page=abc")
	_, err := parseCTEMFindingListParams(r)
	if err == nil {
		t.Fatal("expected error for non-numeric page, got nil")
	}
}

func TestParseCTEMFindingListParams_InvalidPerPage(t *testing.T) {
	r := ctemRequest("GET", "/findings?per_page=xyz")
	_, err := parseCTEMFindingListParams(r)
	if err == nil {
		t.Fatal("expected error for non-numeric per_page, got nil")
	}
}

func TestParseCTEMFindingListParams_SortAndOrder(t *testing.T) {
	r := ctemRequest("GET", "/findings?sort=priority_score&order=desc")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Sort != "priority_score" {
		t.Errorf("expected sort=priority_score, got %q", params.Sort)
	}
	if params.Order != "desc" {
		t.Errorf("expected order=desc, got %q", params.Order)
	}
}

func TestParseCTEMFindingListParams_AllFilters(t *testing.T) {
	r := ctemRequest("GET", "/findings?severity=high&type=misconfiguration&status=in_remediation&priority_group=1&search=firewall&page=2&per_page=20&sort=severity&order=asc")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Severity == nil || *params.Severity != "high" {
		t.Errorf("expected severity=high, got %v", params.Severity)
	}
	if params.Type == nil || *params.Type != "misconfiguration" {
		t.Errorf("expected type=misconfiguration, got %v", params.Type)
	}
	if params.Status == nil || *params.Status != "in_remediation" {
		t.Errorf("expected status=in_remediation, got %v", params.Status)
	}
	if params.PriorityGroup == nil || *params.PriorityGroup != 1 {
		t.Errorf("expected priority_group=1, got %v", params.PriorityGroup)
	}
	if params.Search == nil || *params.Search != "firewall" {
		t.Errorf("expected search=firewall, got %v", params.Search)
	}
	if params.Page != 2 {
		t.Errorf("expected page=2, got %d", params.Page)
	}
	if params.PerPage != 20 {
		t.Errorf("expected per_page=20, got %d", params.PerPage)
	}
	if params.Sort != "severity" {
		t.Errorf("expected sort=severity, got %q", params.Sort)
	}
	if params.Order != "asc" {
		t.Errorf("expected order=asc, got %q", params.Order)
	}
}

// ---------------------------------------------------------------------------
// CTEMAssessmentListParams.SetDefaults
// ---------------------------------------------------------------------------

func TestCTEMAssessmentListParams_SetDefaults_AllZero(t *testing.T) {
	p := &dto.CTEMAssessmentListParams{}
	p.SetDefaults()

	if p.Page != 1 {
		t.Errorf("expected default Page=1, got %d", p.Page)
	}
	if p.PerPage != 25 {
		t.Errorf("expected default PerPage=25, got %d", p.PerPage)
	}
	if p.Sort != "created_at" {
		t.Errorf("expected default Sort=created_at, got %q", p.Sort)
	}
	if p.Order != "desc" {
		t.Errorf("expected default Order=desc, got %q", p.Order)
	}
}

func TestCTEMAssessmentListParams_SetDefaults_PreservesExisting(t *testing.T) {
	p := &dto.CTEMAssessmentListParams{
		Page:    3,
		PerPage: 50,
		Sort:    "name",
		Order:   "asc",
	}
	p.SetDefaults()

	if p.Page != 3 {
		t.Errorf("expected Page=3 preserved, got %d", p.Page)
	}
	if p.PerPage != 50 {
		t.Errorf("expected PerPage=50 preserved, got %d", p.PerPage)
	}
	if p.Sort != "name" {
		t.Errorf("expected Sort=name preserved, got %q", p.Sort)
	}
	if p.Order != "asc" {
		t.Errorf("expected Order=asc preserved, got %q", p.Order)
	}
}

// ---------------------------------------------------------------------------
// CTEMAssessmentListParams.Validate
// ---------------------------------------------------------------------------

func TestCTEMAssessmentListParams_Validate_Valid(t *testing.T) {
	statuses := []string{
		"created", "scoping", "discovery", "prioritizing",
		"validating", "mobilizing", "completed", "failed", "cancelled",
	}
	for _, s := range statuses {
		t.Run("status_"+s, func(t *testing.T) {
			status := s
			p := &dto.CTEMAssessmentListParams{
				Status:  &status,
				Page:    1,
				PerPage: 25,
			}
			if err := p.Validate(); err != nil {
				t.Errorf("expected no error for valid status %q, got: %v", s, err)
			}
		})
	}
}

func TestCTEMAssessmentListParams_Validate_InvalidStatus(t *testing.T) {
	invalid := "nonexistent"
	p := &dto.CTEMAssessmentListParams{
		Status:  &invalid,
		Page:    1,
		PerPage: 25,
	}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestCTEMAssessmentListParams_Validate_NoStatus(t *testing.T) {
	p := &dto.CTEMAssessmentListParams{
		Page:    1,
		PerPage: 25,
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("expected no error when status is nil, got: %v", err)
	}
}

func TestCTEMAssessmentListParams_Validate_PerPageBounds(t *testing.T) {
	cases := []struct {
		name    string
		perPage int
		wantErr bool
	}{
		{"zero", 0, true},
		{"one", 1, false},
		{"mid", 100, false},
		{"max", 200, false},
		{"over_max", 201, true},
		{"negative", -1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &dto.CTEMAssessmentListParams{
				Page:    1,
				PerPage: tc.perPage,
			}
			err := p.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for per_page=%d, got nil", tc.perPage)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error for per_page=%d, got: %v", tc.perPage, err)
			}
		})
	}
}

func TestCTEMAssessmentListParams_Validate_PageBounds(t *testing.T) {
	cases := []struct {
		name    string
		page    int
		wantErr bool
	}{
		{"zero", 0, true},
		{"one", 1, false},
		{"large", 999, false},
		{"negative", -1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &dto.CTEMAssessmentListParams{
				Page:    tc.page,
				PerPage: 25,
			}
			err := p.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for page=%d, got nil", tc.page)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error for page=%d, got: %v", tc.page, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CTEMFindingsListParams.SetDefaults
// ---------------------------------------------------------------------------

func TestCTEMFindingsListParams_SetDefaults_AllZero(t *testing.T) {
	p := &dto.CTEMFindingsListParams{}
	p.SetDefaults()

	if p.Page != 1 {
		t.Errorf("expected default Page=1, got %d", p.Page)
	}
	if p.PerPage != 25 {
		t.Errorf("expected default PerPage=25, got %d", p.PerPage)
	}
	if p.Sort != "priority_score" {
		t.Errorf("expected default Sort=priority_score, got %q", p.Sort)
	}
	if p.Order != "desc" {
		t.Errorf("expected default Order=desc, got %q", p.Order)
	}
}

func TestCTEMFindingsListParams_SetDefaults_PreservesExisting(t *testing.T) {
	p := &dto.CTEMFindingsListParams{
		Page:    4,
		PerPage: 10,
		Sort:    "severity",
		Order:   "asc",
	}
	p.SetDefaults()

	if p.Page != 4 {
		t.Errorf("expected Page=4 preserved, got %d", p.Page)
	}
	if p.PerPage != 10 {
		t.Errorf("expected PerPage=10 preserved, got %d", p.PerPage)
	}
	if p.Sort != "severity" {
		t.Errorf("expected Sort=severity preserved, got %q", p.Sort)
	}
	if p.Order != "asc" {
		t.Errorf("expected Order=asc preserved, got %q", p.Order)
	}
}

// ---------------------------------------------------------------------------
// CTEMFindingsListParams.Validate
// ---------------------------------------------------------------------------

func TestCTEMFindingsListParams_Validate_ValidSeverities(t *testing.T) {
	severities := []string{"critical", "high", "medium", "low", "info"}
	for _, s := range severities {
		t.Run("severity_"+s, func(t *testing.T) {
			sev := s
			p := &dto.CTEMFindingsListParams{Severity: &sev}
			if err := p.Validate(); err != nil {
				t.Errorf("expected no error for severity %q, got: %v", s, err)
			}
		})
	}
}

func TestCTEMFindingsListParams_Validate_InvalidSeverity(t *testing.T) {
	invalid := "extreme"
	p := &dto.CTEMFindingsListParams{Severity: &invalid}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for invalid severity, got nil")
	}
}

func TestCTEMFindingsListParams_Validate_ValidTypes(t *testing.T) {
	types := []string{
		"vulnerability", "misconfiguration", "attack_path", "exposure",
		"weak_credential", "missing_patch", "expired_certificate", "insecure_protocol",
	}
	for _, tp := range types {
		t.Run("type_"+tp, func(t *testing.T) {
			ty := tp
			p := &dto.CTEMFindingsListParams{Type: &ty}
			if err := p.Validate(); err != nil {
				t.Errorf("expected no error for type %q, got: %v", tp, err)
			}
		})
	}
}

func TestCTEMFindingsListParams_Validate_InvalidType(t *testing.T) {
	invalid := "unknown_type"
	p := &dto.CTEMFindingsListParams{Type: &invalid}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestCTEMFindingsListParams_Validate_ValidStatuses(t *testing.T) {
	statuses := []string{
		"open", "in_remediation", "remediated",
		"accepted_risk", "false_positive", "deferred",
	}
	for _, s := range statuses {
		t.Run("status_"+s, func(t *testing.T) {
			st := s
			p := &dto.CTEMFindingsListParams{Status: &st}
			if err := p.Validate(); err != nil {
				t.Errorf("expected no error for status %q, got: %v", s, err)
			}
		})
	}
}

func TestCTEMFindingsListParams_Validate_InvalidStatus(t *testing.T) {
	invalid := "bogus_status"
	p := &dto.CTEMFindingsListParams{Status: &invalid}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
}

func TestCTEMFindingsListParams_Validate_PriorityGroupBounds(t *testing.T) {
	cases := []struct {
		name    string
		pg      int
		wantErr bool
	}{
		{"zero", 0, true},
		{"one", 1, false},
		{"two", 2, false},
		{"three", 3, false},
		{"four", 4, false},
		{"five", 5, true},
		{"negative", -1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pg := tc.pg
			p := &dto.CTEMFindingsListParams{PriorityGroup: &pg}
			err := p.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("expected error for priority_group=%d, got nil", tc.pg)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error for priority_group=%d, got: %v", tc.pg, err)
			}
		})
	}
}

func TestCTEMFindingsListParams_Validate_NilPriorityGroup(t *testing.T) {
	p := &dto.CTEMFindingsListParams{PriorityGroup: nil}
	if err := p.Validate(); err != nil {
		t.Fatalf("expected no error when priority_group is nil, got: %v", err)
	}
}

func TestCTEMFindingsListParams_Validate_MultipleInvalid(t *testing.T) {
	// When multiple fields are invalid, Validate returns error on the first one.
	invalidSev := "extreme"
	invalidType := "bad_type"
	p := &dto.CTEMFindingsListParams{
		Severity: &invalidSev,
		Type:     &invalidType,
	}
	err := p.Validate()
	if err == nil {
		t.Fatal("expected error when severity is invalid, got nil")
	}
}

func TestCTEMFindingsListParams_Validate_AllNil(t *testing.T) {
	p := &dto.CTEMFindingsListParams{}
	if err := p.Validate(); err != nil {
		t.Fatalf("expected no error when all optional fields are nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Edge cases for parse functions
// ---------------------------------------------------------------------------

func TestParseCTEMAssessmentListParams_EmptyStringValues(t *testing.T) {
	// Empty query values (e.g., ?status=) produce an empty string from q.Get(),
	// but Go's URL parser treats ?status= as status="" which is non-empty.
	r := ctemRequest("GET", "/assessments?status=&search=&tag=")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty string values are still set as pointers since the query key is present
	// and q.Get returns "" which passes the v != "" check as false.
	// So they should remain nil.
	if params.Status != nil {
		t.Errorf("expected Status nil for empty value, got %v", *params.Status)
	}
	if params.Search != nil {
		t.Errorf("expected Search nil for empty value, got %v", *params.Search)
	}
	if params.Tag != nil {
		t.Errorf("expected Tag nil for empty value, got %v", *params.Tag)
	}
}

func TestParseCTEMFindingListParams_EmptyStringValues(t *testing.T) {
	r := ctemRequest("GET", "/findings?severity=&type=&status=&search=")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Severity != nil {
		t.Errorf("expected Severity nil for empty value, got %v", *params.Severity)
	}
	if params.Type != nil {
		t.Errorf("expected Type nil for empty value, got %v", *params.Type)
	}
	if params.Status != nil {
		t.Errorf("expected Status nil for empty value, got %v", *params.Status)
	}
	if params.Search != nil {
		t.Errorf("expected Search nil for empty value, got %v", *params.Search)
	}
}

func TestParseCTEMAssessmentListParams_PageOnly(t *testing.T) {
	r := ctemRequest("GET", "/assessments?page=2")
	params, err := parseCTEMAssessmentListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Page != 2 {
		t.Errorf("expected page=2, got %d", params.Page)
	}
	// PerPage should be 0 (pre-default)
	if params.PerPage != 0 {
		t.Errorf("expected per_page=0, got %d", params.PerPage)
	}
}

func TestParseCTEMFindingListParams_PerPageOnly(t *testing.T) {
	r := ctemRequest("GET", "/findings?per_page=50")
	params, err := parseCTEMFindingListParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.PerPage != 50 {
		t.Errorf("expected per_page=50, got %d", params.PerPage)
	}
	if params.Page != 0 {
		t.Errorf("expected page=0 (pre-default), got %d", params.Page)
	}
}

func TestParseCTEMAssessmentListParams_ScheduledNumericValues(t *testing.T) {
	// strconv.ParseBool also accepts "1" and "0"
	cases := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"0", false},
		{"TRUE", true},
		{"FALSE", false},
		{"True", true},
		{"False", false},
	}
	for _, tc := range cases {
		t.Run("scheduled_"+tc.val, func(t *testing.T) {
			r := ctemRequest("GET", "/assessments?scheduled="+tc.val)
			params, err := parseCTEMAssessmentListParams(r)
			if err != nil {
				t.Fatalf("unexpected error for scheduled=%s: %v", tc.val, err)
			}
			if params.Scheduled == nil {
				t.Fatalf("expected Scheduled non-nil for %s", tc.val)
			}
			if *params.Scheduled != tc.want {
				t.Errorf("expected scheduled=%v for input %q, got %v", tc.want, tc.val, *params.Scheduled)
			}
		})
	}
}

func TestParseCTEMFindingListParams_PriorityGroupBoundaryValues(t *testing.T) {
	// priority_group is parsed as int, validation happens in Validate() not parse
	cases := []struct {
		val  string
		want int
	}{
		{"0", 0},
		{"1", 1},
		{"4", 4},
		{"99", 99},
		{"-1", -1},
	}
	for _, tc := range cases {
		t.Run("priority_group_"+tc.val, func(t *testing.T) {
			r := ctemRequest("GET", "/findings?priority_group="+tc.val)
			params, err := parseCTEMFindingListParams(r)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if params.PriorityGroup == nil {
				t.Fatal("expected PriorityGroup non-nil")
			}
			if *params.PriorityGroup != tc.want {
				t.Errorf("expected priority_group=%d, got %d", tc.want, *params.PriorityGroup)
			}
		})
	}
}
