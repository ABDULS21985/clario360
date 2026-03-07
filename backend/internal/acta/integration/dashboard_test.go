//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/service"
)

func TestDashboard(t *testing.T) {
	t.Parallel()

	h := newDemoHarness(t)
	boardCommitteeID := uuid.MustParse("11111111-1111-1111-1111-111111112001")
	h.mustData[any](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/meetings", map[string]any{
		"committee_id":      boardCommitteeID,
		"title":             "Board Dashboard Upcoming Meeting",
		"description":       "Additional scheduled meeting for dashboard coverage",
		"scheduled_at":      time.Now().UTC().Add(7 * 24 * time.Hour),
		"scheduled_end_at":  time.Now().UTC().Add(7*24*time.Hour + 90*time.Minute),
		"duration_minutes":  90,
		"location":          "Executive Board Room",
		"location_type":     "physical",
		"tags":              []string{"dashboard"},
		"metadata":          map[string]any{"source": "dashboard-test"},
	}), http.StatusCreated)

	first := h.mustData[service.ActaDashboard](t, h.doJSON(t, http.MethodGet, "/api/v1/acta/dashboard", nil), http.StatusOK)
	second := h.mustData[service.ActaDashboard](t, h.doJSON(t, http.MethodGet, "/api/v1/acta/dashboard", nil), http.StatusOK)

	if first.KPIs.ActiveCommittees < 4 {
		t.Fatalf("active committees = %d, want at least 4", first.KPIs.ActiveCommittees)
	}
	if len(first.UpcomingMeetings) == 0 {
		t.Fatal("expected dashboard upcoming meetings to be populated")
	}
	if len(first.RecentMeetings) == 0 {
		t.Fatal("expected dashboard recent meetings to be populated")
	}
	if len(first.ActionItemsByStatus) == 0 || len(first.ActionItemsByPriority) == 0 {
		t.Fatalf("expected dashboard action item maps to be populated: %+v %+v", first.ActionItemsByStatus, first.ActionItemsByPriority)
	}
	if len(first.OverdueActionItems) == 0 {
		t.Fatal("expected dashboard overdue action items to be populated")
	}
	if len(first.ComplianceByCommittee) == 0 {
		t.Fatal("expected dashboard compliance by committee to be populated")
	}
	if first.ComplianceScore <= 0 {
		t.Fatalf("dashboard compliance score = %f, want > 0", first.ComplianceScore)
	}
	if len(first.MeetingFrequencyChart) == 0 || len(first.AttendanceRateChart) == 0 {
		t.Fatalf("expected dashboard charts to be populated: %+v %+v", first.MeetingFrequencyChart, first.AttendanceRateChart)
	}
	if len(first.RecentActivity) == 0 {
		t.Fatal("expected dashboard recent activity to be populated")
	}
	if !first.CalculatedAt.Equal(second.CalculatedAt) {
		t.Fatalf("expected cached dashboard response to preserve calculated_at, first=%s second=%s", first.CalculatedAt, second.CalculatedAt)
	}

	foundUpcoming := false
	for _, meeting := range first.UpcomingMeetings {
		if meeting.Title == "Board Dashboard Upcoming Meeting" {
			foundUpcoming = true
			break
		}
	}
	if !foundUpcoming {
		t.Fatalf("expected extra scheduled meeting in dashboard upcoming list: %+v", first.UpcomingMeetings)
	}
}
