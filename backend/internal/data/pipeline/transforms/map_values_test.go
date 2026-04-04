package transforms

import "testing"

func TestApplyMapValues(t *testing.T) {
	t.Run("known_value", func(t *testing.T) {
		rows := []map[string]any{{"status": "A"}}
		got, _, err := ApplyMapValues(rows, MapValuesConfig{
			Column:  "status",
			Mapping: map[string]any{"A": "active"},
		})
		if err != nil {
			t.Fatalf("ApplyMapValues() error = %v", err)
		}
		if got[0]["status"] != "active" {
			t.Fatalf("ApplyMapValues() status = %#v, want active", got[0]["status"])
		}
	})

	t.Run("unknown_with_default", func(t *testing.T) {
		rows := []map[string]any{{"status": "X"}}
		got, _, err := ApplyMapValues(rows, MapValuesConfig{
			Column:  "status",
			Mapping: map[string]any{"A": "active"},
			Default: "unknown",
		})
		if err != nil {
			t.Fatalf("ApplyMapValues() error = %v", err)
		}
		if got[0]["status"] != "unknown" {
			t.Fatalf("ApplyMapValues() status = %#v, want unknown", got[0]["status"])
		}
	})

	t.Run("unknown_no_default", func(t *testing.T) {
		rows := []map[string]any{{"status": "X"}}
		got, _, err := ApplyMapValues(rows, MapValuesConfig{
			Column:  "status",
			Mapping: map[string]any{"A": "active"},
		})
		if err != nil {
			t.Fatalf("ApplyMapValues() error = %v", err)
		}
		if got[0]["status"] != "X" {
			t.Fatalf("ApplyMapValues() status = %#v, want X", got[0]["status"])
		}
	})
}
