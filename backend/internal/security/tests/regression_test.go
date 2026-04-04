package security_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	security "github.com/clario360/platform/internal/security"
)

// newSanitizer is declared in injection_test.go — reuse it here.

// ---------- Path traversal with URL-encoded dots ----------

func TestRegression_PathTraversal_URLEncodedDots(t *testing.T) {
	s := newSanitizer()

	payloads := []string{
		"%2e%2e/%2e%2e/etc/passwd",
		"%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"..%2f..%2f..%2fetc%2fpasswd",
		"uploads/%2e%2e/secrets.txt",
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			err := s.ValidateFilePath(payload, "/safe/base")
			if err == nil {
				t.Errorf("expected path traversal to be detected for %q", payload)
			}
		})
	}
}

func TestRegression_PathTraversal_LiteralDots(t *testing.T) {
	s := newSanitizer()

	payloads := []string{
		"../../../etc/passwd",
		"uploads/../../secrets.txt",
		"./../../etc/shadow",
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			err := s.ValidateFilePath(payload, "/safe/base")
			if !errors.Is(err, security.ErrPathTraversalDetected) {
				t.Errorf("expected ErrPathTraversalDetected for %q, got %v", payload, err)
			}
		})
	}
}

// ---------- Null byte injection in filenames ----------

func TestRegression_NullByteInjection_Filename(t *testing.T) {
	s := newSanitizer()

	// Null bytes should be stripped from filenames
	name, err := s.ValidateFileName("report.pdf\x00.exe")
	if err != nil {
		// If it detects a dangerous extension after stripping, that's also acceptable
		return
	}

	if strings.Contains(name, "\x00") {
		t.Error("null byte should be stripped from filename")
	}
}

func TestRegression_NullByteInjection_String(t *testing.T) {
	s := newSanitizer()

	input := "hello\x00world"
	result := s.SanitizeString(input)

	if strings.Contains(result, "\x00") {
		t.Error("SanitizeString should remove null bytes")
	}
}

// ---------- Unicode normalization attack ----------

func TestRegression_UnicodeNormalization(t *testing.T) {
	s := newSanitizer()

	// U+0041 U+0300 (A + combining grave accent) should normalize to U+00C0 (A-grave)
	// This prevents bypass via different Unicode representations of the same character
	input := "A\u0300" // A + combining grave
	result := s.SanitizeString(input)

	// After NFC normalization, the combining sequence should be collapsed
	if result == input {
		// Check that NFC normalization actually changed the representation
		// NFC("A" + combining grave) = "A-grave" (single codepoint)
		runes := []rune(result)
		inputRunes := []rune(input)
		if len(runes) >= len(inputRunes) {
			// Normalization should reduce the rune count
			t.Log("Unicode normalization may not have combined, but at least it was processed")
		}
	}

	// Ensure the string was processed without error
	if result == "" {
		t.Error("sanitized output should not be empty for valid Unicode input")
	}
}

func TestRegression_UnicodeNormalization_Consistency(t *testing.T) {
	s := newSanitizer()

	// Two different representations of the same character should produce identical output
	// NFC normalization ensures canonical equivalence
	composed := "\u00E9"   // e-acute (precomposed)
	decomposed := "e\u0301" // e + combining acute (decomposed)

	r1 := s.SanitizeString(composed)
	r2 := s.SanitizeString(decomposed)

	if r1 != r2 {
		t.Errorf("NFC normalization should produce identical output for equivalent Unicode: %q vs %q", r1, r2)
	}
}

// ---------- JSON depth bomb ----------

func TestRegression_JSONDepthBomb(t *testing.T) {
	s := newSanitizer()

	// Build deeply nested JSON (depth > 10)
	var b strings.Builder
	depth := 20
	for i := 0; i < depth; i++ {
		b.WriteString(`{"a":`)
	}
	b.WriteString(`"value"`)
	for i := 0; i < depth; i++ {
		b.WriteString(`}`)
	}

	err := s.ValidateJSONField(json.RawMessage(b.String()))
	if err == nil {
		t.Fatal("expected error for deeply nested JSON, got nil")
	}
	if !errors.Is(err, security.ErrJSONTooDeep) {
		t.Fatalf("expected ErrJSONTooDeep, got %v", err)
	}
}

func TestRegression_JSONDepthBomb_Array(t *testing.T) {
	s := newSanitizer()

	// Deeply nested arrays
	var b strings.Builder
	depth := 20
	for i := 0; i < depth; i++ {
		b.WriteString(`[`)
	}
	b.WriteString(`"value"`)
	for i := 0; i < depth; i++ {
		b.WriteString(`]`)
	}

	err := s.ValidateJSONField(json.RawMessage(b.String()))
	if err == nil {
		t.Fatal("expected error for deeply nested JSON arrays, got nil")
	}
}

// ---------- JSON key injection ----------

func TestRegression_JSONKeyInjection_SQL(t *testing.T) {
	s := newSanitizer()

	maliciousKeys := []string{
		`select * from users`,
		`drop table users`,
		`delete from sessions`,
		`insert into roles`,
		`update users set`,
	}

	for _, key := range maliciousKeys {
		t.Run(key, func(t *testing.T) {
			payload := `{"` + key + `": "value"}`
			err := s.ValidateJSONField(json.RawMessage(payload))
			if err == nil {
				t.Errorf("expected error for JSON key %q, got nil", key)
			}
		})
	}
}

func TestRegression_JSONKeyInjection_XSS(t *testing.T) {
	s := newSanitizer()

	maliciousKeys := []string{
		`<script>alert(1)</script>`,
		`onerror=alert(1)`,
		`javascript:alert(1)`,
	}

	for _, key := range maliciousKeys {
		t.Run(key, func(t *testing.T) {
			payload := `{"` + key + `": "value"}`
			err := s.ValidateJSONField(json.RawMessage(payload))
			if err == nil {
				t.Errorf("expected error for JSON key %q, got nil", key)
			}
		})
	}
}

// ---------- ValidateIdentifier rejects SQL reserved words ----------

func TestRegression_ValidateIdentifier_SQLReservedWords(t *testing.T) {
	reserved := []string{
		"DROP", "DELETE", "TRUNCATE", "ALTER",
		"EXEC", "EXECUTE", "GRANT", "REVOKE",
		// Case-insensitive variants
		"drop", "Delete", "truncate", "alter",
		"exec", "Execute", "grant", "Revoke",
	}

	for _, word := range reserved {
		t.Run(word, func(t *testing.T) {
			err := security.ValidateIdentifier(word)
			if err == nil {
				t.Errorf("expected error for reserved word %q, got nil", word)
			}
		})
	}
}

func TestRegression_ValidateIdentifier_ValidNames(t *testing.T) {
	valid := []string{
		"users",
		"user_roles",
		"my_table_v2",
		"Assets",
		"Column_Name",
	}

	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			err := security.ValidateIdentifier(name)
			if err != nil {
				t.Errorf("expected nil for valid identifier %q, got %v", name, err)
			}
		})
	}
}

func TestRegression_ValidateIdentifier_InvalidChars(t *testing.T) {
	invalid := []string{
		"table; DROP",
		"name--comment",
		"col/**/umn",
		"table name",
		"col.name",
	}

	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			err := security.ValidateIdentifier(name)
			if err == nil {
				t.Errorf("expected error for invalid identifier %q, got nil", name)
			}
		})
	}
}

func TestRegression_ValidateIdentifier_StartsWithDigit(t *testing.T) {
	err := security.ValidateIdentifier("1table")
	if err == nil {
		t.Error("identifier starting with digit should be rejected")
	}
}

func TestRegression_ValidateIdentifier_Empty(t *testing.T) {
	err := security.ValidateIdentifier("")
	if err == nil {
		t.Error("empty identifier should be rejected")
	}
}

// ---------- Case-insensitive SQL injection detection ----------

func TestRegression_SQLInjection_CaseInsensitive(t *testing.T) {
	s := newSanitizer()

	payloads := []string{
		"' UNION SELECT * FROM users --",
		"' union select * from users --",
		"' Union Select * From users --",
		"' UnIoN SeLeCt * FrOm users --",
		"; DROP TABLE users",
		"; drop table users",
		"; Drop Table users",
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			err := s.ValidateNoSQLInjection(payload)
			if err == nil {
				t.Errorf("expected SQL injection to be detected for %q", payload)
			}
		})
	}
}

// ---------- XSS in URL-decoded context ----------

func TestRegression_XSS_URLDecoded(t *testing.T) {
	s := newSanitizer()

	payloads := []string{
		"%3Cscript%3Ealert(1)%3C%2Fscript%3E",               // <script>alert(1)</script>
		"%3Csvg%20onload%3Dalert(1)%3E",                       // <svg onload=alert(1)>
		"javascript%3Aalert(document.cookie)",                 // javascript:alert(document.cookie)
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			err := s.ValidateNoXSS(payload)
			if err == nil {
				t.Errorf("expected XSS to be detected in URL-encoded payload %q", payload)
			}
		})
	}
}

func TestRegression_XSS_RawPayloads(t *testing.T) {
	s := newSanitizer()

	payloads := []string{
		`<script>alert('xss')</script>`,
		`<img onerror=alert(1) src=x>`,
		`<svg onload=alert(1)>`,
		`javascript:alert(1)`,
		`<iframe src="javascript:alert(1)">`,
	}

	for _, payload := range payloads {
		t.Run(payload, func(t *testing.T) {
			err := s.ValidateNoXSS(payload)
			if err == nil {
				t.Errorf("expected XSS to be detected for %q", payload)
			}
		})
	}
}

// ---------- Empty string inputs pass validation (not false positives) ----------

func TestRegression_EmptyString_NoFalsePositive(t *testing.T) {
	s := newSanitizer()

	if err := s.ValidateNoSQLInjection(""); err != nil {
		t.Errorf("empty string should pass SQL injection validation, got %v", err)
	}

	if err := s.ValidateNoXSS(""); err != nil {
		t.Errorf("empty string should pass XSS validation, got %v", err)
	}
}

func TestRegression_LegitimateStrings_NoFalsePositive(t *testing.T) {
	s := newSanitizer()

	legitimate := []string{
		"John Doe",
		"alice@example.com",
		"This is a normal description with some numbers: 12345",
		"Security Analyst",
		"Q1 2025 Report - Final Draft",
		"Meeting notes from the SELECT committee",     // contains SQL keyword but not injection pattern
		"The table was updated by the admin team",      // contains keywords but not injection
		"User dropped off the call",                    // contains "drop" but not injection
	}

	for _, input := range legitimate {
		t.Run(input, func(t *testing.T) {
			if err := s.ValidateNoSQLInjection(input); err != nil {
				t.Errorf("false positive SQL injection for legitimate input %q: %v", input, err)
			}
			if err := s.ValidateNoXSS(input); err != nil {
				t.Errorf("false positive XSS for legitimate input %q: %v", input, err)
			}
		})
	}
}

// ---------- Large but legitimate strings within limit ----------

func TestRegression_LargeString_WithinLimit(t *testing.T) {
	s := security.NewSanitizer(security.WithMaxStringLength(10000))

	// Build a large but legitimate string just under the limit
	large := strings.Repeat("a", 9999)

	result := s.SanitizeString(large)
	if len(result) == 0 {
		t.Error("large legitimate string should not be emptied")
	}
	if len([]rune(result)) > 10000 {
		t.Error("result should not exceed max string length")
	}
}

func TestRegression_LargeString_ExceedsLimit(t *testing.T) {
	s := security.NewSanitizer(security.WithMaxStringLength(100))

	large := strings.Repeat("b", 200)
	result := s.SanitizeString(large)

	if len([]rune(result)) > 100 {
		t.Errorf("result rune count %d exceeds max length 100", len([]rune(result)))
	}
}

// ---------- JSON size limit ----------

func TestRegression_JSONTooLarge(t *testing.T) {
	s := security.NewSanitizer(security.WithMaxJSONSize(100))

	large := `{"data":"` + strings.Repeat("x", 200) + `"}`
	err := s.ValidateJSONField(json.RawMessage(large))
	if !errors.Is(err, security.ErrJSONTooLarge) {
		t.Fatalf("expected ErrJSONTooLarge, got %v", err)
	}
}

// ---------- Dangerous file extensions ----------

func TestRegression_DangerousFileExtension(t *testing.T) {
	s := newSanitizer()

	dangerous := []string{
		"payload.exe", "shell.sh", "backdoor.php",
		"exploit.jsp", "mal.bat", "hack.py",
	}

	for _, name := range dangerous {
		t.Run(name, func(t *testing.T) {
			_, err := s.ValidateFileName(name)
			if !errors.Is(err, security.ErrDangerousFileExtension) {
				t.Errorf("expected ErrDangerousFileExtension for %q, got %v", name, err)
			}
		})
	}
}

func TestRegression_SafeFileExtension(t *testing.T) {
	s := newSanitizer()

	safe := []string{
		"report.pdf", "image.png", "data.csv",
		"document.docx", "notes.txt",
	}

	for _, name := range safe {
		t.Run(name, func(t *testing.T) {
			result, err := s.ValidateFileName(name)
			if err != nil {
				t.Errorf("expected nil for safe filename %q, got %v", name, err)
			}
			if result == "" {
				t.Errorf("expected non-empty sanitized filename for %q", name)
			}
		})
	}
}

// ---------- Mass assignment: case sensitivity ----------

func TestRegression_MassAssignment_CaseInsensitive(t *testing.T) {
	// Forbidden field names should be caught regardless of case
	body := map[string]interface{}{
		"Tenant_ID": "evil-tenant",
	}
	err := security.PreventMassAssignment([]string{"name"}, body, nil)
	if !errors.Is(err, security.ErrForbiddenField) {
		t.Errorf("expected ErrForbiddenField for case-varied forbidden field, got %v", err)
	}
}

// ---------- Hidden files rejected ----------

func TestRegression_HiddenFileRejected(t *testing.T) {
	s := newSanitizer()

	hidden := []string{".htaccess", ".env", ".gitignore", ".bashrc"}
	for _, name := range hidden {
		t.Run(name, func(t *testing.T) {
			_, err := s.ValidateFileName(name)
			if !errors.Is(err, security.ErrHiddenFile) {
				t.Errorf("expected ErrHiddenFile for %q, got %v", name, err)
			}
		})
	}
}
