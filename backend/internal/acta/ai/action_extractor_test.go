package ai

import "testing"

func TestExtractActionMarker(t *testing.T) {
	extractor := NewActionExtractor()
	items := extractor.Extract("Budget Review", "ACTION: John to review budget by March 15, 2026.")
	if len(items) != 1 {
		t.Fatalf("Extract returned %d items, want 1", len(items))
	}
	if items[0].AssignedTo != "Unspecified" {
		t.Fatalf("AssignedTo = %s, want Unspecified", items[0].AssignedTo)
	}
	if items[0].DueDate == nil || items[0].DueDate.Format("2006-01-02") != "2026-03-15" {
		t.Fatalf("DueDate = %v, want 2026-03-15", items[0].DueDate)
	}
}

func TestExtractWillPattern(t *testing.T) {
	extractor := NewActionExtractor()
	items := extractor.Extract("Reporting", "Sarah will prepare the report by March 15, 2026.")
	if len(items) != 1 {
		t.Fatalf("Extract returned %d items, want 1", len(items))
	}
	if items[0].AssignedTo != "Sarah" {
		t.Fatalf("AssignedTo = %s, want Sarah", items[0].AssignedTo)
	}
}

func TestExtractAgreedPattern(t *testing.T) {
	extractor := NewActionExtractor()
	items := extractor.Extract("Policy", "It was agreed that the team would update the policy.")
	if len(items) != 1 {
		t.Fatalf("Extract returned %d items, want 1", len(items))
	}
	if items[0].Title != "the team would update the policy" {
		t.Fatalf("Title = %q, want %q", items[0].Title, "the team would update the policy")
	}
}

func TestExtractPriorityInference(t *testing.T) {
	extractor := NewActionExtractor()
	items := extractor.Extract("Incident", "ACTION: Musa will urgently close the critical incident.")
	if len(items) != 1 {
		t.Fatalf("Extract returned %d items, want 1", len(items))
	}
	if items[0].Priority != "high" {
		t.Fatalf("Priority = %s, want high", items[0].Priority)
	}
}

func TestExtractNoActions(t *testing.T) {
	extractor := NewActionExtractor()
	items := extractor.Extract("Overview", "The committee reviewed the pack and noted progress.")
	if len(items) != 0 {
		t.Fatalf("Extract returned %d items, want 0", len(items))
	}
}

func TestExtractDeduplication(t *testing.T) {
	extractor := NewActionExtractor()
	items := extractor.Extract("Budget", "Sarah will prepare the report. Sarah will prepare the report by March 15, 2026.")
	if len(items) != 1 {
		t.Fatalf("Extract returned %d items, want 1", len(items))
	}
	if items[0].DueDate == nil {
		t.Fatal("expected deduplicated action to retain due date")
	}
}
