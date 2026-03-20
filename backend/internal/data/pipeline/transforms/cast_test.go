package transforms

import (
	"testing"
	"time"
)

func TestApplyCast(t *testing.T) {
	t.Run("string_to_int", func(t *testing.T) {
		rows := []map[string]any{{"amount": "123"}, {"amount": "abc"}}
		got, stats, err := ApplyCast(rows, CastConfig{Column: "amount", ToType: "integer"})
		if err != nil {
			t.Fatalf("ApplyCast() error = %v", err)
		}
		if got[0]["amount"] != int64(123) {
			t.Fatalf("ApplyCast() first amount = %#v, want 123", got[0]["amount"])
		}
		if got[1]["amount"] != nil {
			t.Fatalf("ApplyCast() second amount = %#v, want nil", got[1]["amount"])
		}
		if stats.ErrorRows != 1 {
			t.Fatalf("ApplyCast() ErrorRows = %d, want 1", stats.ErrorRows)
		}
	})

	t.Run("string_to_float", func(t *testing.T) {
		rows := []map[string]any{{"amount": "45.67"}}
		got, _, err := ApplyCast(rows, CastConfig{Column: "amount", ToType: "float"})
		if err != nil {
			t.Fatalf("ApplyCast() error = %v", err)
		}
		if got[0]["amount"] != 45.67 {
			t.Fatalf("ApplyCast() amount = %#v, want 45.67", got[0]["amount"])
		}
	})

	t.Run("string_to_bool", func(t *testing.T) {
		rows := []map[string]any{{"flag": "true"}, {"flag": "0"}, {"flag": "maybe"}}
		got, stats, err := ApplyCast(rows, CastConfig{Column: "flag", ToType: "boolean"})
		if err != nil {
			t.Fatalf("ApplyCast() error = %v", err)
		}
		if got[0]["flag"] != true || got[1]["flag"] != false || got[2]["flag"] != nil {
			t.Fatalf("ApplyCast() flags = %#v", got)
		}
		if stats.ErrorRows != 1 {
			t.Fatalf("ApplyCast() ErrorRows = %d, want 1", stats.ErrorRows)
		}
	})

	t.Run("string_to_datetime", func(t *testing.T) {
		rows := []map[string]any{{"created_at": "2026-03-07T10:30:00Z"}}
		got, _, err := ApplyCast(rows, CastConfig{Column: "created_at", ToType: "datetime"})
		if err != nil {
			t.Fatalf("ApplyCast() error = %v", err)
		}
		value, ok := got[0]["created_at"].(time.Time)
		if !ok {
			t.Fatalf("ApplyCast() created_at type = %T, want time.Time", got[0]["created_at"])
		}
		want := time.Date(2026, 3, 7, 10, 30, 0, 0, time.UTC)
		if !value.Equal(want) {
			t.Fatalf("ApplyCast() created_at = %s, want %s", value, want)
		}
	})

	t.Run("int_to_string", func(t *testing.T) {
		rows := []map[string]any{{"value": 42}}
		got, _, err := ApplyCast(rows, CastConfig{Column: "value", ToType: "string"})
		if err != nil {
			t.Fatalf("ApplyCast() error = %v", err)
		}
		if got[0]["value"] != "42" {
			t.Fatalf("ApplyCast() value = %#v, want \"42\"", got[0]["value"])
		}
	})

	t.Run("null_input", func(t *testing.T) {
		rows := []map[string]any{{"value": nil}}
		got, stats, err := ApplyCast(rows, CastConfig{Column: "value", ToType: "float"})
		if err != nil {
			t.Fatalf("ApplyCast() error = %v", err)
		}
		if got[0]["value"] != nil {
			t.Fatalf("ApplyCast() value = %#v, want nil", got[0]["value"])
		}
		if stats.ErrorRows != 0 {
			t.Fatalf("ApplyCast() ErrorRows = %d, want 0", stats.ErrorRows)
		}
	})
}
