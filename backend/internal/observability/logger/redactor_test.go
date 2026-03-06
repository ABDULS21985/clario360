package logger

import (
	"testing"
)

func TestRedactionHook_PasswordField(t *testing.T) {
	hook := NewRedactionHook(DefaultRedactedFields)
	got := hook.Redact("password", "secret123")
	if got != "[REDACTED]" {
		t.Errorf("Redact(\"password\", \"secret123\") = %q, want %q", got, "[REDACTED]")
	}
}

func TestRedactionHook_CaseInsensitive(t *testing.T) {
	hook := NewRedactionHook(DefaultRedactedFields)

	cases := []string{"Password", "PASSWORD", "passWord", "pAsSWoRd"}
	for _, fieldName := range cases {
		got := hook.Redact(fieldName, "myvalue")
		if got != "[REDACTED]" {
			t.Errorf("Redact(%q, \"myvalue\") = %q, want %q", fieldName, got, "[REDACTED]")
		}
	}
}

func TestRedactionHook_NonSensitiveField(t *testing.T) {
	hook := NewRedactionHook(DefaultRedactedFields)
	got := hook.Redact("username", "john")
	if got != "john" {
		t.Errorf("Redact(\"username\", \"john\") = %q, want %q", got, "john")
	}
}

func TestRedactionHook_AllSensitiveFields(t *testing.T) {
	hook := NewRedactionHook(DefaultRedactedFields)

	for _, field := range DefaultRedactedFields {
		got := hook.Redact(field, "sensitive_value")
		if got != "[REDACTED]" {
			t.Errorf("Redact(%q, ...) = %q, want %q", field, got, "[REDACTED]")
		}
	}
}

func TestRedactionHook_IsSensitive(t *testing.T) {
	hook := NewRedactionHook(DefaultRedactedFields)

	// All default fields should be sensitive.
	for _, field := range DefaultRedactedFields {
		if !hook.IsSensitive(field) {
			t.Errorf("IsSensitive(%q) = false, want true", field)
		}
	}

	// Case-insensitive check.
	if !hook.IsSensitive("TOKEN") {
		t.Error("IsSensitive(\"TOKEN\") = false, want true")
	}
	if !hook.IsSensitive("Api_Key") {
		t.Error("IsSensitive(\"Api_Key\") = false, want true")
	}

	// Non-sensitive fields.
	nonSensitive := []string{"username", "email", "host", "port", "method"}
	for _, field := range nonSensitive {
		if hook.IsSensitive(field) {
			t.Errorf("IsSensitive(%q) = true, want false", field)
		}
	}
}

func TestSanitizeMap_NestedRedaction(t *testing.T) {
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"secret": "my-deep-secret",
				"name":   "visible",
			},
		},
	}

	sanitized := SanitizeMap(data, DefaultRedactedFields)

	level1, ok := sanitized["level1"].(map[string]interface{})
	if !ok {
		t.Fatal("level1 is not a map[string]interface{}")
	}
	level2, ok := level1["level2"].(map[string]interface{})
	if !ok {
		t.Fatal("level2 is not a map[string]interface{}")
	}

	if level2["secret"] != "[REDACTED]" {
		t.Errorf("nested secret = %q, want %q", level2["secret"], "[REDACTED]")
	}
	if level2["name"] != "visible" {
		t.Errorf("nested name = %q, want %q", level2["name"], "visible")
	}
}

func TestSanitizeMap_NoMutation(t *testing.T) {
	original := map[string]interface{}{
		"password": "super-secret",
		"username": "admin",
		"nested": map[string]interface{}{
			"token": "abc123",
			"host":  "localhost",
		},
	}

	// Preserve original values for comparison.
	origPassword := original["password"]
	origNested := original["nested"].(map[string]interface{})
	origToken := origNested["token"]

	_ = SanitizeMap(original, DefaultRedactedFields)

	// Original map must remain unchanged.
	if original["password"] != origPassword {
		t.Errorf("original password was mutated: got %q, want %q", original["password"], origPassword)
	}
	if original["username"] != "admin" {
		t.Errorf("original username was mutated: got %q, want %q", original["username"], "admin")
	}

	nested := original["nested"].(map[string]interface{})
	if nested["token"] != origToken {
		t.Errorf("original nested token was mutated: got %q, want %q", nested["token"], origToken)
	}
	if nested["host"] != "localhost" {
		t.Errorf("original nested host was mutated: got %q, want %q", nested["host"], "localhost")
	}
}

func TestSanitizeMap_SliceOfMaps(t *testing.T) {
	data := map[string]interface{}{
		"users": []interface{}{
			map[string]interface{}{
				"name":     "alice",
				"password": "pass1",
			},
			map[string]interface{}{
				"name":     "bob",
				"password": "pass2",
			},
		},
	}

	sanitized := SanitizeMap(data, DefaultRedactedFields)

	users, ok := sanitized["users"].([]interface{})
	if !ok {
		t.Fatal("users is not a []interface{}")
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}

	for i, u := range users {
		user, ok := u.(map[string]interface{})
		if !ok {
			t.Fatalf("users[%d] is not a map[string]interface{}", i)
		}
		if user["password"] != "[REDACTED]" {
			t.Errorf("users[%d][\"password\"] = %q, want %q", i, user["password"], "[REDACTED]")
		}
		// Name should remain visible.
		if user["name"] == "[REDACTED]" {
			t.Errorf("users[%d][\"name\"] was incorrectly redacted", i)
		}
	}
}
