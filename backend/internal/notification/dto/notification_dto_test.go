package dto

import (
	"net/http"
	"net/url"
	"testing"
)

func makeRequest(params map[string]string) *http.Request {
	u := &url.URL{Path: "/api/v1/notifications"}
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return &http.Request{URL: u}
}

func TestParseQueryParams_Defaults(t *testing.T) {
	r := makeRequest(map[string]string{})
	qp, err := ParseQueryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qp.Sort != "created_at" {
		t.Errorf("expected default sort created_at, got %s", qp.Sort)
	}
	if qp.Order != "desc" {
		t.Errorf("expected default order desc, got %s", qp.Order)
	}
	if qp.Page != 1 {
		t.Errorf("expected default page 1, got %d", qp.Page)
	}
	if qp.PerPage != 20 {
		t.Errorf("expected default per_page 20, got %d", qp.PerPage)
	}
}

func TestParseQueryParams_InvalidCategory(t *testing.T) {
	r := makeRequest(map[string]string{"category": "unknown"})
	_, err := ParseQueryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestParseQueryParams_ValidCategory(t *testing.T) {
	for _, cat := range []string{"security", "data", "governance", "legal", "system", "workflow"} {
		r := makeRequest(map[string]string{"category": cat})
		qp, err := ParseQueryParams(r)
		if err != nil {
			t.Errorf("expected %q to be valid, got error: %v", cat, err)
		}
		if qp.Category != cat {
			t.Errorf("expected category %q, got %q", cat, qp.Category)
		}
	}
}

func TestParseQueryParams_InvalidPriority(t *testing.T) {
	r := makeRequest(map[string]string{"priority": "extreme"})
	_, err := ParseQueryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestParseQueryParams_InvalidSort(t *testing.T) {
	r := makeRequest(map[string]string{"sort": "DROP TABLE"})
	_, err := ParseQueryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid sort field")
	}
}

func TestParseQueryParams_ValidSortFields(t *testing.T) {
	for _, field := range []string{"created_at", "priority"} {
		r := makeRequest(map[string]string{"sort": field})
		qp, err := ParseQueryParams(r)
		if err != nil {
			t.Errorf("expected sort %q to be valid, got error: %v", field, err)
		}
		if qp.Sort != field {
			t.Errorf("expected sort %q, got %q", field, qp.Sort)
		}
	}
}

func TestParseQueryParams_InvalidOrder(t *testing.T) {
	r := makeRequest(map[string]string{"order": "random"})
	_, err := ParseQueryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid order")
	}
}

func TestParseQueryParams_PerPageClamped(t *testing.T) {
	r := makeRequest(map[string]string{"per_page": "999"})
	qp, err := ParseQueryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qp.PerPage != 100 {
		t.Errorf("expected per_page clamped to 100, got %d", qp.PerPage)
	}
}

func TestParseQueryParams_ReadFilter(t *testing.T) {
	r := makeRequest(map[string]string{"read": "false"})
	qp, err := ParseQueryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qp.Read == nil || *qp.Read != false {
		t.Error("expected read=false")
	}

	r = makeRequest(map[string]string{"read": "true"})
	qp, err = ParseQueryParams(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qp.Read == nil || *qp.Read != true {
		t.Error("expected read=true")
	}
}

func TestParseQueryParams_InvalidReadFilter(t *testing.T) {
	r := makeRequest(map[string]string{"read": "maybe"})
	_, err := ParseQueryParams(r)
	if err == nil {
		t.Fatal("expected error for invalid read filter")
	}
}

func TestParseQueryParams_DateToBeforeDateFrom(t *testing.T) {
	r := makeRequest(map[string]string{
		"date_from": "2026-03-06T00:00:00Z",
		"date_to":   "2026-03-01T00:00:00Z",
	})
	_, err := ParseQueryParams(r)
	if err == nil {
		t.Fatal("expected error when date_to is before date_from")
	}
}

func TestQueryParams_Offset(t *testing.T) {
	qp := &QueryParams{Page: 3, PerPage: 20}
	if qp.Offset() != 40 {
		t.Errorf("expected offset 40, got %d", qp.Offset())
	}
}

func TestNewPagination(t *testing.T) {
	p := NewPagination(2, 20, 55)
	if p.Page != 2 {
		t.Errorf("expected page 2, got %d", p.Page)
	}
	if p.PerPage != 20 {
		t.Errorf("expected per_page 20, got %d", p.PerPage)
	}
	if p.Total != 55 {
		t.Errorf("expected total 55, got %d", p.Total)
	}
	if p.LastPage != 3 {
		t.Errorf("expected last_page 3, got %d", p.LastPage)
	}
}
