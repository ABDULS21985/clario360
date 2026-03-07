package dashboard

import "testing"

func TestCoverageOnlyCellKnownTechnique(t *testing.T) {
	t.Parallel()

	cell := coverageOnlyCell("T1059")
	if cell.TechniqueName == "" || cell.TechniqueName == "T1059" {
		t.Fatalf("expected resolved technique name, got %#v", cell)
	}
	if cell.TacticID == "" {
		t.Fatalf("expected tactic id for known technique, got %#v", cell)
	}
}

func TestFirstText(t *testing.T) {
	t.Parallel()

	if got := firstText("", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}
