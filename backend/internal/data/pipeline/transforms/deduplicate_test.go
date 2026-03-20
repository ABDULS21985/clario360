package transforms

import "testing"

func TestApplyDeduplicate(t *testing.T) {
	rows := []map[string]any{
		{"email": "a@example.com", "updated_at": "2026-03-01T10:00:00Z", "tenant": "t1"},
		{"email": "a@example.com", "updated_at": "2026-03-02T10:00:00Z", "tenant": "t1"},
		{"email": "b@example.com", "updated_at": "2026-03-01T11:00:00Z", "tenant": "t1"},
	}

	t.Run("keep_latest", func(t *testing.T) {
		got, stats, err := ApplyDeduplicate(rows, DeduplicateConfig{
			KeyColumns: []string{"email"},
			Keep:       "latest",
			OrderBy:    "updated_at",
		})
		if err != nil {
			t.Fatalf("ApplyDeduplicate() error = %v", err)
		}
		if len(got) != 2 || stats.DedupedRows != 1 {
			t.Fatalf("ApplyDeduplicate() rows = %#v, stats = %+v", got, stats)
		}
	})

	t.Run("keep_first", func(t *testing.T) {
		got, _, err := ApplyDeduplicate(rows, DeduplicateConfig{
			KeyColumns: []string{"email"},
			Keep:       "first",
			OrderBy:    "updated_at",
		})
		if err != nil {
			t.Fatalf("ApplyDeduplicate() error = %v", err)
		}
		var found bool
		for _, row := range got {
			if row["email"] == "a@example.com" && row["updated_at"] == "2026-03-01T10:00:00Z" {
				found = true
			}
		}
		if !found {
			t.Fatalf("ApplyDeduplicate() did not keep earliest row: %#v", got)
		}
	})

	t.Run("composite_key", func(t *testing.T) {
		compositeRows := []map[string]any{
			{"email": "a@example.com", "tenant": "t1", "updated_at": "2026-03-01T10:00:00Z"},
			{"email": "a@example.com", "tenant": "t2", "updated_at": "2026-03-01T10:00:00Z"},
			{"email": "a@example.com", "tenant": "t1", "updated_at": "2026-03-03T10:00:00Z"},
		}
		got, _, err := ApplyDeduplicate(compositeRows, DeduplicateConfig{
			KeyColumns: []string{"email", "tenant"},
			Keep:       "latest",
			OrderBy:    "updated_at",
		})
		if err != nil {
			t.Fatalf("ApplyDeduplicate() error = %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("ApplyDeduplicate() len = %d, want 2", len(got))
		}
	})

	t.Run("no_duplicates", func(t *testing.T) {
		uniqueRows := []map[string]any{
			{"email": "a@example.com"},
			{"email": "b@example.com"},
		}
		got, stats, err := ApplyDeduplicate(uniqueRows, DeduplicateConfig{
			KeyColumns: []string{"email"},
			Keep:       "latest",
		})
		if err != nil {
			t.Fatalf("ApplyDeduplicate() error = %v", err)
		}
		if len(got) != 2 || stats.DedupedRows != 0 {
			t.Fatalf("ApplyDeduplicate() rows = %#v, stats = %+v", got, stats)
		}
	})
}
