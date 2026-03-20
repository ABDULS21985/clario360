package transforms

import "testing"

func TestApplyRenameSuccess(t *testing.T) {
	rows := []map[string]any{
		{"old": "alpha", "other": 1},
		{"old": "beta", "other": 2},
	}

	got, stats, err := ApplyRename(rows, RenameConfig{From: "old", To: "new"})
	if err != nil {
		t.Fatalf("ApplyRename() error = %v", err)
	}
	if stats.InputRows != 2 || stats.OutputRows != 2 {
		t.Fatalf("ApplyRename() stats = %+v", stats)
	}
	if got[0]["new"] != "alpha" || got[1]["new"] != "beta" {
		t.Fatalf("ApplyRename() renamed values = %#v", got)
	}
	if _, exists := got[0]["old"]; exists {
		t.Fatalf("ApplyRename() kept old key in first row: %#v", got[0])
	}
}

func TestApplyRenameColumnNotFound(t *testing.T) {
	rows := []map[string]any{{"name": "alpha"}}

	if _, _, err := ApplyRename(rows, RenameConfig{From: "old", To: "new"}); err == nil {
		t.Fatal("ApplyRename() error = nil, want non-nil")
	}
}
