package prd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPRDMatrix_Complete(t *testing.T) {
	items := Requirements()
	if len(items) == 0 {
		t.Fatal("expected PRD requirements")
	}
	for _, item := range items {
		if len(item.Prompts) == 0 {
			t.Fatalf("requirement %q has no prompt reference", item.Requirement)
		}
		if item.Status != "✅" {
			t.Fatalf("requirement %q status = %q, want ✅", item.Requirement, item.Status)
		}
	}
	expected := RenderMatrixMarkdown()
	path := filepath.Clean(filepath.Join("..", "..", "..", "docs", "prd", "PRD_COMPLIANCE_MATRIX.md"))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read matrix file: %v", err)
	}
	if string(content) != expected {
		t.Fatalf("PRD matrix markdown is out of date; regenerate %s", path)
	}
}

func TestPRDMatrix_AllSections(t *testing.T) {
	seen := map[string]bool{}
	for _, item := range Requirements() {
		seen[item.Section] = true
	}
	for _, section := range []string{
		"§1 Goal & Focus",
		"§2 Core Requirements",
		"§3 Platform Capabilities",
		"§4 Technical & Governance",
		"§5 Mandatory Integrations",
	} {
		if !seen[section] {
			t.Fatalf("missing section %s", section)
		}
	}
}
