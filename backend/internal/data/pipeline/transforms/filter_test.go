package transforms

import "testing"

func TestApplyFilter(t *testing.T) {
	rows := []map[string]any{
		{"amount": 50.0, "status": "inactive", "type": "A"},
		{"amount": 150.0, "status": "active", "type": "A"},
		{"amount": 200.0, "status": "deleted", "type": "C"},
		{"amount": nil, "status": "active", "type": "B"},
	}

	t.Run("simple_comparison", func(t *testing.T) {
		got, stats, err := ApplyFilter(rows, FilterConfig{Expression: "amount > 100"})
		if err != nil {
			t.Fatalf("ApplyFilter() error = %v", err)
		}
		if len(got) != 2 || stats.FilteredRows != 2 {
			t.Fatalf("ApplyFilter() rows = %#v, stats = %+v", got, stats)
		}
	})

	t.Run("string_equals", func(t *testing.T) {
		got, _, err := ApplyFilter(rows, FilterConfig{Expression: "status == 'active'"})
		if err != nil {
			t.Fatalf("ApplyFilter() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("ApplyFilter() len = %d, want 2", len(got))
		}
	})

	t.Run("boolean_and", func(t *testing.T) {
		got, _, err := ApplyFilter(rows, FilterConfig{Expression: "amount > 0 AND status == 'active'"})
		if err != nil {
			t.Fatalf("ApplyFilter() error = %v", err)
		}
		if len(got) != 1 || got[0]["amount"] != 150.0 {
			t.Fatalf("ApplyFilter() rows = %#v", got)
		}
	})

	t.Run("boolean_or", func(t *testing.T) {
		got, _, err := ApplyFilter(rows, FilterConfig{Expression: "type == 'A' OR type == 'B'"})
		if err != nil {
			t.Fatalf("ApplyFilter() error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("ApplyFilter() len = %d, want 3", len(got))
		}
	})

	t.Run("not_operator", func(t *testing.T) {
		got, _, err := ApplyFilter(rows, FilterConfig{Expression: "NOT (status == 'deleted')"})
		if err != nil {
			t.Fatalf("ApplyFilter() error = %v", err)
		}
		if len(got) != 3 {
			t.Fatalf("ApplyFilter() len = %d, want 3", len(got))
		}
	})

	t.Run("null_column_filtered_out", func(t *testing.T) {
		got, _, err := ApplyFilter(rows, FilterConfig{Expression: "amount != null"})
		if err != nil {
			t.Fatalf("ApplyFilter() error = %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("ApplyFilter() len = %d, want 0 because null comparisons propagate", len(got))
		}
	})
}
