//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func TestCompliance_RunChecks(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Compliance Seed Item",
			Notes: "The committee reviewed governance matters and assigned follow-up work.",
		},
	})

	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/meetings/"+fixture.Meeting.ID.String()+"/minutes/generate", nil), http.StatusOK)
	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/meetings/"+fixture.Meeting.ID.String()+"/minutes/submit", nil), http.StatusOK)
	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/meetings/"+fixture.Meeting.ID.String()+"/minutes/approve", nil), http.StatusOK)
	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/meetings/"+fixture.Meeting.ID.String()+"/minutes/publish", nil), http.StatusOK)

	// Move the completed meeting outside the last completed monthly period so meeting-frequency becomes non-compliant.
	actualEnd := time.Now().UTC().AddDate(0, -2, 0)
	actualStart := actualEnd.Add(-90 * time.Minute)
	h.setMeetingWindow(t, fixture.Meeting.ID, actualStart, actualStart, actualEnd)

	for idx := 0; idx < 2; idx++ {
		h.createActionItem(
			t,
			fixture.Meeting.ID,
			fixture.Committee.Committee.ID,
			fixture.Committee.Members[idx+1].ID,
			fixture.Committee.Members[idx+1].Name,
			"Overdue compliance action",
			time.Now().UTC().AddDate(0, 0, -(idx+3)),
		)
	}

	report := mustData[model.ComplianceReport](t, h.doJSON(t, http.MethodGet, "/api/v1/acta/compliance/run", nil), http.StatusOK)

	var (
		meetingFrequencyFound bool
		actionTrackingFound   bool
	)
	for _, result := range report.Results {
		if result.CommitteeID != nil && *result.CommitteeID == fixture.Committee.Committee.ID && result.CheckType == model.ComplianceCheckMeetingFrequency && result.Status == model.ComplianceStatusNonCompliant {
			meetingFrequencyFound = true
		}
		if result.CommitteeID != nil && *result.CommitteeID == fixture.Committee.Committee.ID && result.CheckType == model.ComplianceCheckActionTracking && result.Status == model.ComplianceStatusWarning {
			actionTrackingFound = true
		}
	}
	if !meetingFrequencyFound {
		t.Fatalf("expected meeting_frequency non_compliant result in %+v", report.Results)
	}
	if !actionTrackingFound {
		t.Fatalf("expected action_item_tracking warning result in %+v", report.Results)
	}
}

func TestCompliance_Score(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	now := time.Now().UTC()
	checks := []model.ComplianceCheck{
		{
			ID:          uuid.New(),
			TenantID:    h.tenantID,
			CheckType:   model.ComplianceCheckMeetingFrequency,
			CheckName:   "Meeting frequency",
			Status:      model.ComplianceStatusCompliant,
			Severity:    model.ComplianceSeverityHigh,
			Description: "High severity compliant check",
			Evidence:    map[string]any{"slot": 1},
			PeriodStart: now.AddDate(0, -1, 0),
			PeriodEnd:   now,
			CheckedAt:   now,
			CheckedBy:   "system",
			CreatedAt:   now,
		},
		{
			ID:          uuid.New(),
			TenantID:    h.tenantID,
			CheckType:   model.ComplianceCheckQuorumCompliance,
			CheckName:   "Quorum compliance",
			Status:      model.ComplianceStatusCompliant,
			Severity:    model.ComplianceSeverityMedium,
			Description: "Medium severity compliant check",
			Evidence:    map[string]any{"slot": 2},
			PeriodStart: now.AddDate(0, -1, 0),
			PeriodEnd:   now,
			CheckedAt:   now,
			CheckedBy:   "system",
			CreatedAt:   now,
		},
		{
			ID:          uuid.New(),
			TenantID:    h.tenantID,
			CheckType:   model.ComplianceCheckActionTracking,
			CheckName:   "Action tracking",
			Status:      model.ComplianceStatusWarning,
			Severity:    model.ComplianceSeverityLow,
			Description: "Low severity warning check",
			Evidence:    map[string]any{"slot": 3},
			PeriodStart: now.AddDate(0, -1, 0),
			PeriodEnd:   now,
			CheckedAt:   now,
			CheckedBy:   "system",
			CreatedAt:   now,
		},
		{
			ID:          uuid.New(),
			TenantID:    h.tenantID,
			CheckType:   model.ComplianceCheckMinutesCompletion,
			CheckName:   "Minutes completion",
			Status:      model.ComplianceStatusNonCompliant,
			Severity:    model.ComplianceSeverityCritical,
			Description: "Critical severity non-compliant check",
			Evidence:    map[string]any{"slot": 4},
			PeriodStart: now.AddDate(0, -1, 0),
			PeriodEnd:   now,
			CheckedAt:   now,
			CheckedBy:   "system",
			CreatedAt:   now,
		},
	}

	if err := h.env.store.InsertComplianceChecks(context.Background(), h.env.store.DB(), checks); err != nil {
		t.Fatalf("InsertComplianceChecks() error = %v", err)
	}

	scorePayload := mustData[map[string]float64](t, h.doJSON(t, http.MethodGet, "/api/v1/acta/compliance/score", nil), http.StatusOK)
	score := scorePayload["score"]
	want := (3.0 / 6.5) * 100.0
	if diff := score - want; diff < -0.001 || diff > 0.001 {
		t.Fatalf("compliance score = %.6f, want %.6f", score, want)
	}
}
