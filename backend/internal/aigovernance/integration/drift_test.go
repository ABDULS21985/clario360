//go:build integration

package integration

import (
	"testing"
	"time"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestDriftDetectionNoDrift(t *testing.T) {
	h := newHarness(t)
	slug := "integration-drift-none-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	version := h.createRuleVersion(t, model.ID, "drift")
	h.promote(t, model.ID, version.ID, nil, false)
	h.promote(t, model.ID, version.ID, nil, false)
	version = h.currentProduction(t, model.ID)

	promotedAt := time.Now().UTC().AddDate(0, 0, -20)
	version = h.backdateProductionVersion(t, version, promotedAt)
	stable := []float64{0.80, 0.82, 0.84, 0.86, 0.88, 0.81, 0.83, 0.85}
	h.insertPredictionLogs(t, version, promotedAt.Add(12*time.Hour), stable)
	h.insertPredictionLogs(t, version, time.Now().UTC().AddDate(0, 0, -6), stable)

	report, err := h.env.driftSvc.RunVersion(h.env.ctx, version, "7d")
	if err != nil {
		t.Fatalf("RunVersion() error = %v", err)
	}
	if report.OutputPSI == nil {
		t.Fatal("output PSI is nil")
	}
	if *report.OutputPSI >= 0.10 {
		t.Fatalf("output PSI = %.4f, want < 0.10", *report.OutputPSI)
	}
	if report.OutputDriftLevel != aigovmodel.DriftLevelNone {
		t.Fatalf("output drift level = %s, want none", report.OutputDriftLevel)
	}
}

func TestDriftDetectionSignificantDrift(t *testing.T) {
	h := newHarness(t)
	slug := "integration-drift-sig-" + h.tenantID.String()[:8]
	model := h.registerRuleModel(t, slug, aigovmodel.RiskTierMedium)

	version := h.createRuleVersion(t, model.ID, "drift")
	h.promote(t, model.ID, version.ID, nil, false)
	h.promote(t, model.ID, version.ID, nil, false)
	version = h.currentProduction(t, model.ID)

	promotedAt := time.Now().UTC().AddDate(0, 0, -20)
	version = h.backdateProductionVersion(t, version, promotedAt)
	h.insertPredictionLogs(t, version, promotedAt.Add(12*time.Hour), []float64{0.82, 0.84, 0.86, 0.88, 0.90, 0.83, 0.85, 0.87})
	h.insertPredictionLogs(t, version, time.Now().UTC().AddDate(0, 0, -6), []float64{0.32, 0.34, 0.36, 0.38, 0.40, 0.33, 0.35, 0.37})

	report, err := h.env.driftSvc.RunVersion(h.env.ctx, version, "7d")
	if err != nil {
		t.Fatalf("RunVersion() error = %v", err)
	}
	if report.OutputPSI == nil {
		t.Fatal("output PSI is nil")
	}
	if *report.OutputPSI <= 0.25 {
		t.Fatalf("output PSI = %.4f, want > 0.25", *report.OutputPSI)
	}
	if report.OutputDriftLevel != aigovmodel.DriftLevelSignificant {
		t.Fatalf("output drift level = %s, want significant", report.OutputDriftLevel)
	}
}
