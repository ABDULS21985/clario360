package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func TestComplianceScore(t *testing.T) {
	committeeID := uuid.New()
	checks := []model.ComplianceCheck{
		{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			CommitteeID: &committeeID,
			CheckType:   model.ComplianceCheckMeetingFrequency,
			Status:      model.ComplianceStatusCompliant,
			Severity:    model.ComplianceSeverityCritical,
			CheckedAt:   time.Now().UTC(),
		},
		{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			CommitteeID: &committeeID,
			CheckType:   model.ComplianceCheckMinutesCompletion,
			Status:      model.ComplianceStatusCompliant,
			Severity:    model.ComplianceSeverityHigh,
			CheckedAt:   time.Now().UTC(),
		},
		{
			ID:          uuid.New(),
			TenantID:    uuid.New(),
			CommitteeID: &committeeID,
			CheckType:   model.ComplianceCheckActionTracking,
			Status:      model.ComplianceStatusWarning,
			Severity:    model.ComplianceSeverityMedium,
			CheckedAt:   time.Now().UTC(),
		},
	}

	score := complianceScore(checks)
	if score < 83.3 || score > 83.34 {
		t.Fatalf("complianceScore = %v, want approximately 83.33", score)
	}
}

func TestLastCompletedPeriodMonthly(t *testing.T) {
	start, end, applicable := lastCompletedPeriod(model.MeetingFrequencyMonthly, time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC))
	if !applicable {
		t.Fatal("lastCompletedPeriod(monthly) applicable = false, want true")
	}
	if got := start.Format("2006-01-02"); got != "2026-02-01" {
		t.Fatalf("monthly start = %s, want 2026-02-01", got)
	}
	if got := end.Format("2006-01-02"); got != "2026-03-01" {
		t.Fatalf("monthly end = %s, want 2026-03-01", got)
	}
}
