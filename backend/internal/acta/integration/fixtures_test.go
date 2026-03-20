//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

type agendaSpec struct {
	Title           string
	Notes           string
	RequiresVote    bool
	VoteType        string
	VotesFor        int
	VotesAgainst    int
	VotesAbstained  int
	PresenterUserID *uuid.UUID
	PresenterName   *string
	Category        string
}

type completedMeetingFixture struct {
	Committee      committeeFixture
	Meeting        model.Meeting
	AgendaItems    []model.AgendaItem
	PresentMembers []userFixture
}

func (h *actaHarness) scheduleMeeting(t *testing.T, committeeID uuid.UUID, title string, scheduledAt time.Time) model.Meeting {
	t.Helper()

	location := "Board Room A"
	return mustData[model.Meeting](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/meetings", map[string]any{
		"committee_id":     committeeID,
		"title":            title,
		"description":      "Integration test meeting",
		"scheduled_at":     scheduledAt.UTC(),
		"duration_minutes": 90,
		"location":         location,
		"location_type":    "physical",
		"virtual_platform": nil,
		"virtual_link":     nil,
		"tags":             []string{"integration"},
		"metadata":         map[string]any{"source": "integration"},
		"scheduled_end_at": scheduledAt.UTC().Add(90 * time.Minute),
	}), http.StatusCreated)
}

func (h *actaHarness) addAgendaItem(t *testing.T, meetingID uuid.UUID, spec agendaSpec) model.AgendaItem {
	t.Helper()

	category := spec.Category
	if category == "" {
		category = "regular"
		if spec.RequiresVote {
			category = "decision"
		}
	}
	body := map[string]any{
		"title":            spec.Title,
		"description":      "Agenda item for integration testing",
		"duration_minutes": 20,
		"requires_vote":    spec.RequiresVote,
		"attachment_ids":   []uuid.UUID{},
		"confidential":     false,
		"category":         category,
	}
	if spec.RequiresVote {
		body["vote_type"] = spec.VoteType
	}
	if spec.PresenterUserID != nil {
		body["presenter_user_id"] = spec.PresenterUserID
	}
	if spec.PresenterName != nil {
		body["presenter_name"] = spec.PresenterName
	}
	return mustData[model.AgendaItem](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/agenda", meetingID), body), http.StatusCreated)
}

func (h *actaHarness) recordAttendance(t *testing.T, meetingID uuid.UUID, attendees []userFixture, status string) []model.Attendee {
	t.Helper()

	requests := make([]map[string]any, 0, len(attendees))
	for _, attendee := range attendees {
		requests = append(requests, map[string]any{
			"user_id": attendee.ID,
			"status":  status,
		})
	}
	return mustData[[]model.Attendee](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/attendance/bulk", meetingID), map[string]any{
		"attendance": requests,
	}), http.StatusOK)
}

func (h *actaHarness) updateAgendaNotes(t *testing.T, meetingID, itemID uuid.UUID, notes string) model.AgendaItem {
	t.Helper()

	return mustData[model.AgendaItem](t, h.doJSON(t, http.MethodPut, fmt.Sprintf("/api/v1/acta/meetings/%s/agenda/%s/notes", meetingID, itemID), map[string]any{
		"notes": notes,
	}), http.StatusOK)
}

func (h *actaHarness) recordVote(t *testing.T, meetingID, itemID uuid.UUID, spec agendaSpec) model.AgendaItem {
	t.Helper()

	return mustData[model.AgendaItem](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/agenda/%s/vote", meetingID, itemID), map[string]any{
		"vote_type":       spec.VoteType,
		"votes_for":       spec.VotesFor,
		"votes_against":   spec.VotesAgainst,
		"votes_abstained": spec.VotesAbstained,
		"notes":           "Recorded by integration test",
	}), http.StatusOK)
}

func (h *actaHarness) completeMeeting(t *testing.T, memberCount, presentCount int, agendaSpecs []agendaSpec) completedMeetingFixture {
	t.Helper()

	committee := h.createCommittee(t, fmt.Sprintf("Integration Committee %s", uuid.NewString()), memberCount)
	meeting := h.scheduleMeeting(t, committee.Committee.ID, "Integration Meeting", time.Now().UTC().Add(48*time.Hour))

	agendaItems := make([]model.AgendaItem, 0, len(agendaSpecs))
	for _, spec := range agendaSpecs {
		if spec.PresenterUserID == nil {
			spec.PresenterUserID = &committee.Members[0].ID
		}
		if spec.PresenterName == nil {
			spec.PresenterName = &committee.Members[0].Name
		}
		item := h.addAgendaItem(t, meeting.ID, spec)
		agendaItems = append(agendaItems, item)
	}

	if presentCount > len(committee.Members) {
		presentCount = len(committee.Members)
	}
	presentMembers := append([]userFixture(nil), committee.Members[:presentCount]...)
	h.recordAttendance(t, meeting.ID, presentMembers, "present")

	meeting = mustData[model.Meeting](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/start", meeting.ID), nil), http.StatusOK)
	for idx, item := range agendaItems {
		spec := agendaSpecs[idx]
		if spec.Notes != "" {
			item = h.updateAgendaNotes(t, meeting.ID, item.ID, spec.Notes)
		}
		if spec.RequiresVote {
			if spec.VotesFor == 0 && spec.VotesAgainst == 0 && spec.VotesAbstained == 0 {
				spec.VotesFor = presentCount - 1
				spec.VotesAgainst = 1
			}
			item = h.recordVote(t, meeting.ID, item.ID, spec)
		}
		agendaItems[idx] = item
	}
	endResp := h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/end", meeting.ID), nil)
	if endResp.StatusCode != http.StatusOK {
		body := readBody(t, endResp.Body)
		if _, err := h.env.app.MeetingService.EndMeeting(context.Background(), h.tenantID, h.userID, meeting.ID); err != nil {
			t.Fatalf("end meeting status = %d, body=%s, service error=%v", endResp.StatusCode, body, err)
		}
		t.Fatalf("end meeting status = %d, body=%s", endResp.StatusCode, body)
	}
	meeting = mustData[model.Meeting](t, endResp, http.StatusOK)

	return completedMeetingFixture{
		Committee:      committee,
		Meeting:        meeting,
		AgendaItems:    agendaItems,
		PresentMembers: presentMembers,
	}
}

func (h *actaHarness) createActionItem(t *testing.T, meetingID, committeeID uuid.UUID, assignedTo uuid.UUID, assigneeName, title string, dueDate time.Time) model.ActionItem {
	t.Helper()

	return mustData[model.ActionItem](t, h.doJSON(t, http.MethodPost, "/api/v1/acta/action-items", map[string]any{
		"meeting_id":    meetingID,
		"committee_id":  committeeID,
		"title":         title,
		"description":   "Action item created by integration test",
		"priority":      "medium",
		"assigned_to":   assignedTo,
		"assignee_name": assigneeName,
		"due_date":      dueDate.UTC(),
		"tags":          []string{"integration"},
		"metadata":      map[string]any{"source": "integration"},
	}), http.StatusCreated)
}

func (h *actaHarness) setMeetingWindow(t *testing.T, meetingID uuid.UUID, scheduledAt, actualStartAt, actualEndAt time.Time) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := h.env.db.Exec(ctx, `
		UPDATE meetings
		SET scheduled_at = $3,
		    scheduled_end_at = $4,
		    actual_start_at = $5,
		    actual_end_at = $6,
		    updated_at = now()
		WHERE tenant_id = $1
		  AND id = $2`,
		h.tenantID,
		meetingID,
		scheduledAt.UTC(),
		actualEndAt.UTC(),
		actualStartAt.UTC(),
		actualEndAt.UTC(),
	)
	if err != nil {
		t.Fatalf("set meeting window: %v", err)
	}
}

func (h *actaHarness) setActionItemDueDate(t *testing.T, actionItemID uuid.UUID, dueDate time.Time) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := h.env.db.Exec(ctx, `
		UPDATE action_items
		SET due_date = $3,
		    original_due_date = $3,
		    updated_at = now()
		WHERE tenant_id = $1
		  AND id = $2`,
		h.tenantID,
		actionItemID,
		dueDate.UTC(),
	)
	if err != nil {
		t.Fatalf("set action item due date: %v", err)
	}
}
