package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePageParams_Defaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	page, perPage := parsePageParams(req, 50)

	if page != 1 {
		t.Errorf("expected default page=1, got %d", page)
	}
	if perPage != 50 {
		t.Errorf("expected default perPage=50, got %d", perPage)
	}
}

func TestParsePageParams_Custom(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=3&per_page=25", nil)
	page, perPage := parsePageParams(req, 50)

	if page != 3 {
		t.Errorf("expected page=3, got %d", page)
	}
	if perPage != 25 {
		t.Errorf("expected perPage=25, got %d", perPage)
	}
}

func TestParsePageParams_Invalid(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=abc&per_page=xyz", nil)
	page, perPage := parsePageParams(req, 50)

	if page != 1 {
		t.Errorf("expected default page=1 for invalid input, got %d", page)
	}
	if perPage != 50 {
		t.Errorf("expected default perPage=50 for invalid input, got %d", perPage)
	}
}

func TestParsePageParams_ZeroValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=0&per_page=0", nil)
	page, perPage := parsePageParams(req, 50)

	// Zero is not > 0, so defaults should be used.
	if page != 1 {
		t.Errorf("expected default page=1 for zero, got %d", page)
	}
	if perPage != 50 {
		t.Errorf("expected default perPage=50 for zero, got %d", perPage)
	}
}

func TestParsePageParams_NegativeValues(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test?page=-1&per_page=-10", nil)
	page, perPage := parsePageParams(req, 50)

	if page != 1 {
		t.Errorf("expected default page=1 for negative, got %d", page)
	}
	if perPage != 50 {
		t.Errorf("expected default perPage=50 for negative, got %d", perPage)
	}
}

func TestParsePageParams_PartialParams(t *testing.T) {
	// Only page is provided.
	req := httptest.NewRequest(http.MethodGet, "/test?page=5", nil)
	page, perPage := parsePageParams(req, 50)

	if page != 5 {
		t.Errorf("expected page=5, got %d", page)
	}
	if perPage != 50 {
		t.Errorf("expected default perPage=50, got %d", perPage)
	}

	// Only per_page is provided.
	req = httptest.NewRequest(http.MethodGet, "/test?per_page=10", nil)
	page, perPage = parsePageParams(req, 50)

	if page != 1 {
		t.Errorf("expected default page=1, got %d", page)
	}
	if perPage != 10 {
		t.Errorf("expected perPage=10, got %d", perPage)
	}
}

func TestStringPtr_Empty(t *testing.T) {
	result := stringPtr("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", *result)
	}
}

func TestStringPtr_Value(t *testing.T) {
	result := stringPtr("hello")
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != "hello" {
		t.Errorf("expected 'hello', got %q", *result)
	}
}

func TestStringPtr_Whitespace(t *testing.T) {
	result := stringPtr(" ")
	if result == nil {
		t.Fatal("expected non-nil pointer for whitespace string")
	}
	if *result != " " {
		t.Errorf("expected ' ', got %q", *result)
	}
}

func TestBoolPtr_Empty(t *testing.T) {
	result := boolPtr("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", *result)
	}
}

func TestBoolPtr_True(t *testing.T) {
	result := boolPtr("true")
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != true {
		t.Errorf("expected true, got %v", *result)
	}
}

func TestBoolPtr_False(t *testing.T) {
	result := boolPtr("false")
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != false {
		t.Errorf("expected false, got %v", *result)
	}
}

func TestBoolPtr_Invalid(t *testing.T) {
	result := boolPtr("notabool")
	if result != nil {
		t.Errorf("expected nil for invalid bool string, got %v", *result)
	}
}

func TestBoolPtr_NumericTrue(t *testing.T) {
	result := boolPtr("1")
	if result == nil {
		t.Fatal("expected non-nil pointer for '1'")
	}
	if *result != true {
		t.Errorf("expected true for '1', got %v", *result)
	}
}

func TestFloatPtr_Empty(t *testing.T) {
	result := floatPtr("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", *result)
	}
}

func TestFloatPtr_Value(t *testing.T) {
	result := floatPtr("42.5")
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != 42.5 {
		t.Errorf("expected 42.5, got %v", *result)
	}
}

func TestFloatPtr_Integer(t *testing.T) {
	result := floatPtr("100")
	if result == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *result != 100.0 {
		t.Errorf("expected 100.0, got %v", *result)
	}
}

func TestFloatPtr_Invalid(t *testing.T) {
	result := floatPtr("notanumber")
	if result != nil {
		t.Errorf("expected nil for invalid float string, got %v", *result)
	}
}

func TestFloatPtr_Zero(t *testing.T) {
	result := floatPtr("0")
	if result == nil {
		t.Fatal("expected non-nil pointer for '0'")
	}
	if *result != 0.0 {
		t.Errorf("expected 0.0, got %v", *result)
	}
}

func TestFloatPtr_Negative(t *testing.T) {
	result := floatPtr("-3.14")
	if result == nil {
		t.Fatal("expected non-nil pointer for negative float")
	}
	if *result != -3.14 {
		t.Errorf("expected -3.14, got %v", *result)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"key": "value"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty body")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got %q", ct)
	}
}

func TestWriteError_InternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusInternalServerError, "CUSTOM_CODE", "custom message")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	// For 500+, code and message should be overwritten.
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty body")
	}
}

func TestParseUUID_Valid(t *testing.T) {
	w := httptest.NewRecorder()
	id, ok := parseUUID(w, "550e8400-e29b-41d4-a716-446655440000")
	if !ok {
		t.Fatal("expected parseUUID to succeed for valid UUID")
	}
	if id.String() != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("expected parsed UUID, got %v", id)
	}
}

func TestParseUUID_Invalid(t *testing.T) {
	w := httptest.NewRecorder()
	_, ok := parseUUID(w, "not-a-uuid")
	if ok {
		t.Error("expected parseUUID to fail for invalid UUID")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
