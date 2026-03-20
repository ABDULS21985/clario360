package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type fakeAlertEventService struct {
	created map[string]*model.Alert
	order   []*model.Alert
}

func newFakeAlertEventService() *fakeAlertEventService {
	return &fakeAlertEventService{created: map[string]*model.Alert{}}
}

func (f *fakeAlertEventService) CreateFromEvent(_ context.Context, alert *model.Alert) (*model.Alert, error) {
	cloned := *alert
	if cloned.ID == uuid.Nil {
		cloned.ID = uuid.New()
	}
	f.created[cloned.ID.String()] = &cloned
	f.order = append(f.order, &cloned)
	return &cloned, nil
}

func (f *fakeAlertEventService) FindRecentEventAlert(_ context.Context, _ uuid.UUID, source, metadataKey, metadataValue string, _ time.Duration) (*model.Alert, error) {
	for _, alert := range f.created {
		if alert.Source != source {
			continue
		}
		var metadata map[string]any
		if err := json.Unmarshal(alert.Metadata, &metadata); err != nil {
			continue
		}
		if fmt.Sprintf("%v", metadata[metadataKey]) == metadataValue {
			cloned := *alert
			return &cloned, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeAlertEventService) UpdateEventAlert(_ context.Context, alert *model.Alert) (*model.Alert, error) {
	cloned := *alert
	f.created[cloned.ID.String()] = &cloned
	return &cloned, nil
}

func newIAMConsumer(t *testing.T) (*IAMEventConsumer, *fakeAlertEventService, *miniredis.Miniredis, *redis.Client, *time.Time, string) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})

	alerts := newFakeAlertEventService()
	now := time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC)
	current := now
	tenantID := uuid.NewString()

	consumer := NewIAMEventConsumer(
		alerts,
		client,
		events.NewIdempotencyGuard(client, time.Hour),
		nil,
		zerolog.New(nil),
		nil,
	)
	consumer.now = func() time.Time { return current }

	return consumer, alerts, server, client, &current, tenantID
}

func loginFailedEventForTest(t *testing.T, tenantID, eventID, ip string, attempt int, at time.Time) *events.Event {
	t.Helper()
	event, err := events.NewEvent("iam.user.login.failed", "iam-service", tenantID, map[string]any{
		"user_id":       "user-1",
		"email":         "user@example.com",
		"ip_address":    ip,
		"attempt_count": attempt,
		"user_agent":    "Firefox",
		"timestamp":     at,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = eventID
	return event
}

func TestBruteForce_Under5Attempts(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	ctx := context.Background()

	for idx := 0; idx < 3; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(alerts.order); got != 0 {
		t.Fatalf("expected no alerts, got %d", got)
	}
}

func TestBruteForce_At5Attempts(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	ctx := context.Background()

	for idx := 0; idx < 5; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(alerts.order); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}
	if alerts.order[0].Severity != model.SeverityHigh {
		t.Fatalf("expected high severity, got %s", alerts.order[0].Severity)
	}
}

func TestBruteForce_DifferentIPs(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	ctx := context.Background()

	for idx := 0; idx < 3; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-a-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-b-%d", idx), "10.0.0.2", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(alerts.order); got != 0 {
		t.Fatalf("expected no alerts, got %d", got)
	}
}

func TestBruteForce_WindowExpiry(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	ctx := context.Background()

	for idx := 0; idx < 3; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	*current = current.Add(6 * time.Minute)

	for idx := 3; idx < 6; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(alerts.order); got != 0 {
		t.Fatalf("expected no alerts after window expiry, got %d", got)
	}
}

func TestBruteForce_At20_Escalation(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	ctx := context.Background()

	for idx := 0; idx < 20; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(alerts.created); got != 1 {
		t.Fatalf("expected 1 deduplicated alert, got %d", got)
	}
	for _, alert := range alerts.created {
		if alert.Severity != model.SeverityCritical {
			t.Fatalf("expected critical severity after escalation, got %s", alert.Severity)
		}
	}
}

func TestBruteForce_AlertDedup(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	ctx := context.Background()

	for idx := 0; idx < 10; idx++ {
		if err := consumer.Handle(ctx, loginFailedEventForTest(t, tenantID, fmt.Sprintf("evt-%d", idx), "10.0.0.1", idx+1, *current)); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if got := len(alerts.created); got != 1 {
		t.Fatalf("expected 1 deduplicated alert, got %d", got)
	}
}

func TestMFADowngrade(t *testing.T) {
	consumer, alerts, _, _, current, tenantID := newIAMConsumer(t)
	event, err := events.NewEvent("iam.user.mfa.disabled", "iam-service", tenantID, map[string]any{
		"user_id":     "user-1",
		"email":       "user@example.com",
		"disabled_by": "admin-1",
		"reason":      "user_requested",
		"timestamp":   *current,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = "evt-mfa"

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.order); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}
	if alerts.order[0].Severity != model.SeverityMedium {
		t.Fatalf("expected medium severity, got %s", alerts.order[0].Severity)
	}
	if alerts.order[0].MITRETechniqueID == nil || *alerts.order[0].MITRETechniqueID != "T1556.006" {
		t.Fatalf("expected MITRE technique T1556.006, got %v", alerts.order[0].MITRETechniqueID)
	}
}

func TestMalformedEvent(t *testing.T) {
	consumer, alerts, _, _, _, tenantID := newIAMConsumer(t)
	event, err := events.NewEvent("iam.user.login.failed", "iam-service", tenantID, map[string]any{
		"user_id": "user-1",
		"email":   "user@example.com",
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = "evt-malformed"

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.order); got != 0 {
		t.Fatalf("expected no alerts, got %d", got)
	}
}
