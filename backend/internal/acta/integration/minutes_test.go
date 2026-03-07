//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func TestMinutes_AIGeneration(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 6, 5, []agendaSpec{
		{
			Title: "Approve Prior Minutes",
			Notes: "The committee confirmed the prior minutes without amendment.",
		},
		{
			Title:          "Cyber Update",
			Notes:          "Sarah will prepare the follow-up cyber resilience report by April 20, 2026.",
			RequiresVote:   true,
			VoteType:       "majority",
			VotesFor:       4,
			VotesAgainst:   1,
			VotesAbstained: 0,
		},
		{
			Title: "Risk Register Review",
			Notes: "ACTION: Musa to refresh the top risks dashboard before the next meeting.",
		},
		{
			Title: "Policy Exception",
			Notes: "It was agreed that the compliance team would update the exception register by May 1, 2026.",
		},
	})

	minutes := mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	if !minutes.AIGenerated {
		t.Fatal("expected minutes to be AI generated")
	}
	if len(minutes.AIActionItems) == 0 {
		t.Fatal("expected AI-generated minutes to include extracted action items")
	}
	if !strings.Contains(minutes.Content, "# Minutes of the") {
		t.Fatalf("minutes content missing title header: %s", minutes.Content)
	}
	if !strings.Contains(minutes.Content, "## Attendance") {
		t.Fatalf("minutes content missing attendance section: %s", minutes.Content)
	}
	if !strings.Contains(minutes.Content, "## Agenda Items") {
		t.Fatalf("minutes content missing agenda section: %s", minutes.Content)
	}
	if !strings.Contains(minutes.Content, "**Vote:** majority") {
		t.Fatalf("minutes content missing vote section: %s", minutes.Content)
	}
	if !strings.Contains(minutes.Content, "## Action Items") {
		t.Fatalf("minutes content missing action item section: %s", minutes.Content)
	}
	if minutes.AISummary == nil || !strings.Contains(*minutes.AISummary, fixture.Committee.Committee.Name) {
		t.Fatalf("minutes ai summary = %v, want committee name %q", minutes.AISummary, fixture.Committee.Committee.Name)
	}
}

func TestMinutes_ApprovalFlow(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Approval Item",
			Notes: "The committee approved the control framework and asked management to circulate the revised policy pack.",
		},
	})

	minutes := mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	minutes = mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/submit", fixture.Meeting.ID), nil), http.StatusOK)
	if minutes.Status != model.MinutesStatusReview {
		t.Fatalf("minutes status after submit = %s, want %s", minutes.Status, model.MinutesStatusReview)
	}

	minutes = mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/approve", fixture.Meeting.ID), nil), http.StatusOK)
	if minutes.Status != model.MinutesStatusApproved {
		t.Fatalf("minutes status after approve = %s, want %s", minutes.Status, model.MinutesStatusApproved)
	}
	if minutes.ApprovedBy == nil || *minutes.ApprovedBy != h.userID {
		t.Fatalf("minutes approved_by = %v, want %s", minutes.ApprovedBy, h.userID)
	}

	minutes = mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/publish", fixture.Meeting.ID), nil), http.StatusOK)
	if minutes.Status != model.MinutesStatusPublished {
		t.Fatalf("minutes status after publish = %s, want %s", minutes.Status, model.MinutesStatusPublished)
	}
}

func TestMinutes_NonChairApprovalRejected(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Non-Chair Approval Item",
			Notes: "The committee reviewed the annual governance calendar.",
		},
	})

	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/submit", fixture.Meeting.ID), nil), http.StatusOK)

	nonChairToken := h.tokenForUser(t, fixture.Committee.Members[1].ID, "tenant_admin")
	errResp := mustError(t, h.doJSONWithToken(t, nonChairToken, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/approve", fixture.Meeting.ID), nil), http.StatusForbidden)
	if errResp.Error.Message != "only the committee chair can approve minutes" {
		t.Fatalf("approve error message = %q, want chair-only message", errResp.Error.Message)
	}
}

func TestMinutes_VersioningAfterApproval(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Versioned Minutes Item",
			Notes: "ACTION: Governance office to circulate the signed minutes pack by June 1, 2026.",
		},
	})

	minutes := mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/submit", fixture.Meeting.ID), nil), http.StatusOK)
	mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/approve", fixture.Meeting.ID), nil), http.StatusOK)

	updated := mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes", fixture.Meeting.ID), map[string]any{
		"content": minutes.Content + "\n\nAdditional correction recorded after chair approval.\n",
	}), http.StatusOK)
	if updated.Version != 2 {
		t.Fatalf("updated minutes version = %d, want 2", updated.Version)
	}
	if updated.Status != model.MinutesStatusDraft {
		t.Fatalf("updated minutes status = %s, want %s", updated.Status, model.MinutesStatusDraft)
	}
	if updated.PreviousVersionID == nil || *updated.PreviousVersionID != minutes.ID {
		t.Fatalf("updated previous_version_id = %v, want %s", updated.PreviousVersionID, minutes.ID)
	}

	versions := mustData[[]model.MeetingMinutes](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/versions", fixture.Meeting.ID), nil), http.StatusOK)
	if len(versions) != 2 {
		t.Fatalf("minutes versions length = %d, want 2", len(versions))
	}
}

func TestMinutes_GenerationUnder500Milliseconds(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	agenda := make([]agendaSpec, 0, 15)
	for idx := 0; idx < 15; idx++ {
		agenda = append(agenda, agendaSpec{
			Title: fmt.Sprintf("Performance Agenda %02d", idx+1),
			Notes: fmt.Sprintf("ACTION: Member %02d will update the governance workstream register before July %d, 2026.", idx+1, idx+10),
		})
	}
	fixture := h.completeMeeting(t, 50, 50, agenda)

	start := time.Now()
	minutes := mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	duration := time.Since(start)

	if minutes.ID == uuid.Nil {
		t.Fatal("expected generated minutes to be persisted")
	}
	if duration >= 500*time.Millisecond {
		t.Fatalf("minutes generation duration = %s, want < 500ms", duration)
	}
}
