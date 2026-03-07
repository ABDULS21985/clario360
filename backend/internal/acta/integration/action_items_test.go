//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	actascheduler "github.com/clario360/platform/internal/acta/scheduler"
	"github.com/clario360/platform/internal/acta/model"
)

func TestActionItems_CreateFromExtracted(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Risk Register Review",
			Notes: "ACTION: Musa to refresh the risk register by April 25, 2026.",
		},
		{
			Title: "Vendor Assurance",
			Notes: "Sarah will prepare the vendor assurance summary before the next meeting.",
		},
	})

	minutes := h.mustData[model.MeetingMinutes](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/meetings/%s/minutes/generate", fixture.Meeting.ID), nil), http.StatusOK)
	if len(minutes.AIActionItems) == 0 {
		t.Fatal("expected generated minutes to contain extracted action items")
	}

	created, err := h.env.app.ActionItemService.CreateFromExtracted(context.Background(), h.tenantID, h.userID, fixture.Meeting.ID, fixture.Committee.Committee.ID, &fixture.AgendaItems[0].ID, minutes.AIActionItems)
	if err != nil {
		t.Fatalf("CreateFromExtracted() error = %v", err)
	}
	if len(created) != len(minutes.AIActionItems) {
		t.Fatalf("created action items = %d, want %d", len(created), len(minutes.AIActionItems))
	}

	list := h.mustPaginated[model.ActionItem](t, h.doJSON(t, http.MethodGet, fmt.Sprintf("/api/v1/acta/action-items?meeting_id=%s", fixture.Meeting.ID), nil), http.StatusOK)
	if len(list.Data) != len(minutes.AIActionItems) {
		t.Fatalf("meeting action items listed = %d, want %d", len(list.Data), len(minutes.AIActionItems))
	}
}

func TestActionItems_OverdueDetection(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Overdue Detection Item",
			Notes: "The committee directed that management resolve the outstanding issue immediately.",
		},
	})

	assigneeID := fixture.Committee.Members[1].ID
	assigneeName := fixture.Committee.Members[1].Name
	actionItem := h.createActionItem(t, fixture.Meeting.ID, fixture.Committee.Committee.ID, assigneeID, assigneeName, "Resolve overdue issue", time.Now().UTC().Add(-24*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- actascheduler.NewOverdueChecker(h.env.app.ActionItemService, 50*time.Millisecond, h.env.logger).Run(ctx)
	}()

	var updated model.ActionItem
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		updated = h.getActionItem(t, actionItem.ID)
		if updated.Status == model.ActionItemStatusOverdue {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("overdue checker run error = %v", err)
	}
	if updated.Status != model.ActionItemStatusOverdue {
		t.Fatalf("action item status = %s, want %s", updated.Status, model.ActionItemStatusOverdue)
	}

	event := h.waitForTenantEventType(t, "com.clario360.acta.action_item.overdue")
	var payload map[string]any
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		t.Fatalf("unmarshal action_item.overdue payload: %v", err)
	}
	if fmt.Sprint(payload["id"]) != actionItem.ID.String() {
		t.Fatalf("overdue event id = %v, want %s", payload["id"], actionItem.ID)
	}
}

func TestActionItems_ExtendDueDate(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "Extension Item",
			Notes: "The committee asked for additional supporting evidence before closure.",
		},
	})

	assigneeID := fixture.Committee.Members[1].ID
	assigneeName := fixture.Committee.Members[1].Name
	actionItem := h.createActionItem(t, fixture.Meeting.ID, fixture.Committee.Committee.ID, assigneeID, assigneeName, "Collect further evidence", time.Now().UTC().Add(48*time.Hour))
	newDueDate := actionItem.DueDate.Add(7 * 24 * time.Hour)

	extended := h.mustData[model.ActionItem](t, h.doJSON(t, http.MethodPost, fmt.Sprintf("/api/v1/acta/action-items/%s/extend", actionItem.ID), map[string]any{
		"new_due_date": newDueDate,
		"reason":       "Awaiting independent assurance evidence",
	}), http.StatusOK)
	if extended.ExtendedCount != 1 {
		t.Fatalf("extended_count = %d, want 1", extended.ExtendedCount)
	}
	if extended.ExtensionReason == nil || *extended.ExtensionReason != "Awaiting independent assurance evidence" {
		t.Fatalf("extension_reason = %v, want Awaiting independent assurance evidence", extended.ExtensionReason)
	}
	if !extended.DueDate.Equal(newDueDate.UTC()) {
		t.Fatalf("extended due date = %s, want %s", extended.DueDate, newDueDate.UTC())
	}
}

func TestActionItems_MyActionItems(t *testing.T) {
	t.Parallel()

	h := newActaHarness(t)
	fixture := h.completeMeeting(t, 5, 4, []agendaSpec{
		{
			Title: "My Action Items",
			Notes: "The committee closed out housekeeping matters.",
		},
	})

	assigneeID := uuid.New()
	assigneeToken := h.tokenForUser(t, assigneeID, "tenant_admin")
	for idx := 0; idx < 3; idx++ {
		h.createActionItem(t, fixture.Meeting.ID, fixture.Committee.Committee.ID, assigneeID, "Action Owner", fmt.Sprintf("Personal action item %d", idx+1), time.Now().UTC().AddDate(0, 0, idx+2))
	}

	myItems := h.mustData[[]model.ActionItem](t, h.doJSONWithToken(t, assigneeToken, http.MethodGet, "/api/v1/acta/action-items/my", nil), http.StatusOK)
	if len(myItems) != 3 {
		t.Fatalf("my action items count = %d, want 3", len(myItems))
	}
}
