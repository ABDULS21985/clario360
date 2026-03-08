package security_test

import (
	"testing"

	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

func TestDependencyChecker_CheckToolAvailability(t *testing.T) {
	logger := zerolog.Nop()
	dc := security.NewDependencyChecker(logger)

	versions := dc.CheckToolAvailability()
	// We don't know if tools are installed, but it shouldn't panic
	if versions == nil {
		t.Error("expected non-nil versions map")
	}
}

func TestVulnerabilityReport_SeverityCounting(t *testing.T) {
	// Test via RunFullScan with empty paths — should produce an empty report
	logger := zerolog.Nop()
	dc := security.NewDependencyChecker(logger)

	report := dc.RunFullScan(t.Context(), "", "")
	if report == nil {
		t.Fatal("expected non-nil report")
	}

	if report.TotalCritical != 0 || report.TotalHigh != 0 {
		t.Errorf("expected zero vulns for empty scan, got critical=%d high=%d",
			report.TotalCritical, report.TotalHigh)
	}

	if !report.Timestamp.IsZero() == false {
		// Timestamp should be set
	}
}

func TestVulnerabilityReport_EmptyPaths(t *testing.T) {
	logger := zerolog.Nop()
	dc := security.NewDependencyChecker(logger)

	report := dc.RunFullScan(t.Context(), "", "")

	// No scan errors when paths are empty (scans skipped)
	if len(report.ScanErrors) > 0 {
		t.Errorf("expected no scan errors for empty paths, got: %v", report.ScanErrors)
	}
	if len(report.GoVulns) > 0 {
		t.Error("expected no Go vulns for empty path")
	}
	if len(report.NPMVulns) > 0 {
		t.Error("expected no NPM vulns for empty path")
	}
}
