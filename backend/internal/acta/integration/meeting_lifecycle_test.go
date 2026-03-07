//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func TestMeetingLifecycle_FullLifecycle(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 6, 4, []agendaSpec{
		{
			Title: "Operational Update",
			Notes: "The committee reviewed quarterly operating metrics and requested management to circulate the revised scorecard.",
		},
		{
			Title:          "Approve Governance Policy",
			Notes:          "ACTION: Legal will publish the revised governance policy by March 15, 2026.",
			RequiresVote:   true,
			VoteType:       "majority",
			VotesFor:       3,
			VotesAgainst:   1,
			VotesAbstained: 0,
		},
	})

	if fixture.Meeting.Status != model.MeetingStatusCompleted {
		t.Fatalf("meeting status = %s, want %s", fixture.Meeting.Status, model.MeetingStatusCompleted)
	}
	if fixture.Meeting.QuorumMet == nil || !*fixture.Meeting.QuorumMet {
		t.Fatalf("expected completed meeting quorum to be met: %+v", fixture.Meeting)
	}
	if fixture.Meeting.PresentCount != 4 {
		t.Fatalf("meeting present count = %d, want 4", fixture.Meeting.PresentCount)
	}
	if fixture.Meeting.QuorumRequired != 4 {
		t.Fatalf("meeting quorum required = %d, want 4", fixture.Meeting.QuorumRequired)
	}
	if fixture.AgendaItems[1].VoteResult == nil || *fixture.AgendaItems[1].VoteResult != model.VoteResultApproved {
		t.Fatalf("agenda vote result = %v, want %s", fixture.AgendaItems[1].VoteResult, model.VoteResultApproved)
	}

	completedEvent := h.waitForTenantEventType(t, "com.clario360.acta.meeting.completed")
	var completedPayload map[string]any
	if err := json.Unmarshal(completedEvent.Data, &completedPayload); err != nil {
		t.Fatalf("unmarshal meeting.completed payload: %v", err)
	}
	if completedPayload["quorum_met"] != true {
		t.Fatalf("meeting.completed quorum_met = %v, want true", completedPayload["quorum_met"])
	}

	minutes := mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	if !minutes.AIGenerated {
		t.Fatal("expected generated minutes to be flagged as ai_generated")
	}

	minutes = mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/submit", fixture.Meeting.ID), nil), http.StatusOK)
	if minutes.Status != model.MinutesStatusReview {
		t.Fatalf("minutes status after submit = %s, want %s", minutes.Status, model.MinutesStatusReview)
	}

	minutes = mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/approve", fixture.Meeting.ID), nil), http.StatusOK)
	if minutes.Status != model.MinutesStatusApproved {
		t.Fatalf("minutes status after approve = %s, want %s", minutes.Status, model.MinutesStatusApproved)
	}

	minutes = mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/publish", fixture.Meeting.ID), nil), http.StatusOK)
	if minutes.Status != model.MinutesStatusPublished {
		t.Fatalf("minutes status after publish = %s, want %s", minutes.Status, model.MinutesStatusPublished)
	}

	meeting := h.getMeeting(t, fixture.Meeting.ID)
	if !meeting.HasMinutes {
		t.Fatal("expected meeting.HasMinutes to be true after generation")
	}
	if meeting.MinutesStatus == nil || *meeting.MinutesStatus != string(model.MinutesStatusPublished) {
		t.Fatalf("meeting minutes status = %v, want %s", meeting.MinutesStatus, model.MinutesStatusPublished)
	}

	if event := h.waitForTenantEventType(t, "com.clario360.acta.meeting.scheduled"); event == nil {
		t.Fatal("expected meeting.scheduled event")
	}
	if event := h.waitForTenantEventType(t, "com.clario360.acta.meeting.started"); event == nil {
		t.Fatal("expected meeting.started event")
	}
	if event := h.waitForTenantEventType(t, "com.clario360.acta.agenda.voted"); event == nil {
		t.Fatal("expected agenda.voted event")
	}
	if event := h.waitForTenantEventType(t, "com.clario360.acta.minutes.generated"); event == nil {
		t.Fatal("expected minutes.generated event")
	}
	if event := h.waitForTenantEventType(t, "com.clario360.acta.minutes.approved"); event == nil {
		t.Fatal("expected minutes.approved event")
	}
	if event := h.waitForTenantEventType(t, "com.clario360.acta.minutes.published"); event == nil {
		t.Fatal("expected minutes.published event")
	}
}

func TestMeetingLifecycle_WithoutQuorumCreatesComplianceFinding(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	committee := h.createCommittee(t, "Quorum Failure Committee", 6)
	meeting := h.scheduleMeeting(t, committee.Committee.ID, "Quorum Failure Meeting", time.Now().UTC().Add(24*time.Hour))
	h.recordAttendance(t, meeting.ID, committee.Members[:2], "present")

	meeting = mustData[model.Meeting](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/start", meeting.ID), nil), http.StatusOK)
	if meeting.QuorumMet == nil || *meeting.QuorumMet {
		t.Fatalf("meeting quorum on start = %v, want false", meeting.QuorumMet)
	}

	meeting = mustData[model.Meeting](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/end", meeting.ID), nil), http.StatusOK)
	if meeting.QuorumMet == nil || *meeting.QuorumMet {
		t.Fatalf("meeting quorum on end = %v, want false", meeting.QuorumMet)
	}

	results := mustPaginated[model.ComplianceCheck](t, h.doJSON(t, http.MethodGet, "/api/v1/acta/compliance/results?check_type=quorum_compliance&status=non_compliant", nil), http.StatusOK)
	found := false
	for _, check := range results.Data {
		if fmt.Sprint(check.Evidence["meeting_id"]) == meeting.ID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected quorum non-compliance result for meeting %s, got %+v", meeting.ID, results.Data)
	}
}

func TestMeetingLifecycle_Cancel(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	committee := h.createCommittee(t, "Cancellation Committee", 4)
	meeting := h.scheduleMeeting(t, committee.Committee.ID, "Meeting To Cancel", time.Now().UTC().Add(72*time.Hour))

	cancelled := mustData[model.Meeting](t, h.doJSON(t, http.MethodDelete, fmt.Sprintf("/api/v1/acta/meetings/%s", meeting.ID), map[string]any{
		"reason": "External auditor conflict",
	}), http.StatusOK)
	if cancelled.Status != model.MeetingStatusCancelled {
		t.Fatalf("cancelled meeting status = %s, want %s", cancelled.Status, model.MeetingStatusCancelled)
	}
	if cancelled.CancellationReason == nil || *cancelled.CancellationReason != "External auditor conflict" {
		t.Fatalf("cancellation reason = %v, want External auditor conflict", cancelled.CancellationReason)
	}

	event := h.waitForTenantEventType(t, "com.clario360.acta.meeting.cancelled")
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("unmarshal meeting.cancelled payload: %v", err)
	}
	if payload["reason"] != "External auditor conflict" {
		t.Fatalf("cancelled event reason = %v, want External auditor conflict", payload["reason"])
	}
}

func TestMeetingLifecycle_Postpone(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	committee := h.createCommittee(t, "Postponement Committee", 4)
	meeting := h.scheduleMeeting(t, committee.Committee.ID, "Meeting To Postpone", time.Now().UTC().Add(72*time.Hour))
	newDate := meeting.ScheduledAt.Add(7 * 24 * time.Hour)

	postponed := mustData[model.Meeting](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/postpone", meeting.ID), map[string]any{
		"new_scheduled_at":     newDate,
		"new_scheduled_end_at": newDate.Add(90 * time.Minute),
		"reason":               "Board materials delayed",
	}), http.StatusOK)
	if postponed.Status != model.MeetingStatusPostponed {
		t.Fatalf("postponed meeting status = %s, want %s", postponed.Status, model.MeetingStatusPostponed)
	}
	if !postponed.ScheduledAt.Equal(newDate.UTC()) {
		t.Fatalf("postponed meeting scheduled_at = %s, want %s", postponed.ScheduledAt, newDate.UTC())
	}

	event := h.waitForTenantEventType(t, "com.clario360.acta.meeting.postponed")
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("unmarshal meeting.postponed payload: %v", err)
	}
	if fmt.Sprint(payload["id"]) != meeting.ID.String() {
		t.Fatalf("postponed event id = %v, want %s", payload["id"], meeting.ID)
	}
}
