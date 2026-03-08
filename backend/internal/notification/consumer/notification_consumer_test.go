package consumer

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/notification/service"
	"github.com/rs/zerolog"
)

type fakeNotificationService struct {
	requests []service.CreateNotificationRequest
}

func (f *fakeNotificationService) CreateNotification(_ context.Context, req service.CreateNotificationRequest) error {
	f.requests = append(f.requests, req)
	return nil
}

type fakeRecipientResolver struct {
	roles  map[string][]string
	emails map[string]string
}

func (f *fakeRecipientResolver) ResolveByRoles(_ context.Context, _ string, roles []string) ([]string, error) {
	var out []string
	for _, role := range roles {
		out = append(out, f.roles[role]...)
	}
	return uniqueStrings(out), nil
}

func (f *fakeRecipientResolver) GetUserEmail(_ context.Context, userID string) (string, error) {
	return f.emails[userID], nil
}

func testNotificationEvent(eventType string, data map[string]interface{}) *events.Event {
	payload, _ := json.Marshal(data)
	return &events.Event{
		ID:       "evt-1",
		Type:     eventType,
		Source:   "clario360/test-service",
		TenantID: "tenant-1",
		Data:     payload,
	}
}

func TestCriticalAlert_NotifiesSecurityManager(t *testing.T) {
	notifs := &fakeNotificationService{}
	resolver := &fakeRecipientResolver{
		roles:  map[string][]string{"security-manager": {"manager-1"}},
		emails: map[string]string{"manager-1": "manager@example.com"},
	}
	consumer := NewNotificationConsumer(nil, notifs, resolver, nil, nil, zerolog.New(io.Discard))

	err := consumer.handleEvent(context.Background(), testNotificationEvent("com.clario360.cyber.alert.created", map[string]interface{}{
		"id":       "alert-1",
		"title":    "Critical alert",
		"severity": "critical",
	}))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(notifs.requests) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifs.requests))
	}
	req := notifs.requests[0]
	if req.UserID != "manager-1" {
		t.Fatalf("expected security manager recipient, got %s", req.UserID)
	}
	if req.Priority != "critical" {
		t.Fatalf("expected critical priority, got %s", req.Priority)
	}
	if len(req.Channels) != 3 {
		t.Fatalf("expected 3 channels, got %v", req.Channels)
	}
}

func TestPipelineFailed_NotifiesOwner(t *testing.T) {
	notifs := &fakeNotificationService{}
	consumer := NewNotificationConsumer(nil, notifs, &fakeRecipientResolver{}, nil, nil, zerolog.New(io.Discard))

	err := consumer.handleEvent(context.Background(), testNotificationEvent("com.clario360.data.pipeline.run.failed", map[string]interface{}{
		"pipeline_id":   "pipe-1",
		"pipeline_name": "Daily Sync",
		"created_by":    "owner-1",
		"error_message": "database timeout",
	}))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(notifs.requests) != 1 || notifs.requests[0].UserID != "owner-1" {
		t.Fatalf("expected pipeline owner notification, got %+v", notifs.requests)
	}
}

func TestMeetingScheduled_NotifiesCommittee(t *testing.T) {
	notifs := &fakeNotificationService{}
	resolver := &fakeRecipientResolver{
		emails: map[string]string{
			"user-1": "user1@example.com",
			"user-2": "user2@example.com",
		},
	}
	consumer := NewNotificationConsumer(nil, notifs, resolver, nil, nil, zerolog.New(io.Discard))

	err := consumer.handleEvent(context.Background(), testNotificationEvent("com.clario360.acta.meeting.scheduled", map[string]interface{}{
		"id":           "meeting-1",
		"title":        "Quarterly Governance Review",
		"scheduled_at": "2026-03-08T10:00:00Z",
		"attendee_ids": []string{"user-1", "user-2"},
	}))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(notifs.requests) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifs.requests))
	}
}

func TestContractExpiring7d_NotifiesOwnerAndManager(t *testing.T) {
	notifs := &fakeNotificationService{}
	resolver := &fakeRecipientResolver{
		roles: map[string][]string{"legal-manager": {"manager-1"}},
		emails: map[string]string{
			"owner-1":   "owner@example.com",
			"manager-1": "manager@example.com",
		},
	}
	consumer := NewNotificationConsumer(nil, notifs, resolver, nil, nil, zerolog.New(io.Discard))

	err := consumer.handleEvent(context.Background(), testNotificationEvent("com.clario360.lex.contract.expiring", map[string]interface{}{
		"id":                "contract-1",
		"title":             "Master Services Agreement",
		"owner_user_id":     "owner-1",
		"days_until_expiry": 7,
	}))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(notifs.requests) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifs.requests))
	}
	for _, req := range notifs.requests {
		if req.Priority != "critical" {
			t.Fatalf("expected critical priority for all recipients, got %+v", req)
		}
	}
}

func TestMalware_NotifiesUploaderAndAdmin(t *testing.T) {
	notifs := &fakeNotificationService{}
	resolver := &fakeRecipientResolver{
		roles: map[string][]string{"tenant-admin": {"admin-1"}},
		emails: map[string]string{
			"uploader-1": "uploader@example.com",
			"admin-1":    "admin@example.com",
		},
	}
	consumer := NewNotificationConsumer(nil, notifs, resolver, nil, nil, zerolog.New(io.Discard))

	err := consumer.handleEvent(context.Background(), testNotificationEvent("com.clario360.file.scan.infected", map[string]interface{}{
		"file_id":     "file-1",
		"virus_name":  "eicar",
		"uploaded_by": "uploader-1",
	}))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(notifs.requests) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(notifs.requests))
	}
}
