package consumer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/service"
	"github.com/clario360/platform/internal/events"
)

type fakeDSPMTriggerService struct {
	calls    int
	tenantID uuid.UUID
	userID   uuid.UUID
	actor    *service.Actor
}

func (f *fakeDSPMTriggerService) TriggerScan(_ context.Context, tenantID, userID uuid.UUID, actor *service.Actor) (*model.DSPMScan, error) {
	f.calls++
	f.tenantID = tenantID
	f.userID = userID
	f.actor = actor
	return &model.DSPMScan{ID: uuid.New(), TenantID: tenantID}, nil
}

func newDataConsumerForTest(t *testing.T) (*DataEventConsumer, *fakeAlertEventService, *fakeDSPMTriggerService, *miniredis.Miniredis, string) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})

	alerts := newFakeAlertEventService()
	dspm := &fakeDSPMTriggerService{}
	tenantID := uuid.NewString()

	consumer := NewDataEventConsumer(
		alerts,
		dspm,
		client,
		events.NewIdempotencyGuard(client, time.Hour),
		nil,
		zerolog.New(nil),
		nil,
	)

	return consumer, alerts, dspm, server, tenantID
}

func TestConnectionTestFailure_CreatesLowAlert(t *testing.T) {
	consumer, alerts, _, _, tenantID := newDataConsumerForTest(t)
	event, err := events.NewEvent("data.source.connection_tested", "data-service", tenantID, map[string]any{
		"id":         "source-1",
		"name":       "Payroll DB",
		"type":       "postgres",
		"success":    false,
		"latency_ms": 3200,
		"error":      "connection refused",
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = "evt-source-failed"

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.order); got != 1 {
		t.Fatalf("expected 1 alert, got %d", got)
	}
	if alerts.order[0].Severity != model.SeverityLow {
		t.Fatalf("expected low severity, got %s", alerts.order[0].Severity)
	}
}

func TestConnectionTestSuccess_NoAlert(t *testing.T) {
	consumer, alerts, _, _, tenantID := newDataConsumerForTest(t)
	event, err := events.NewEvent("data.source.connection_tested", "data-service", tenantID, map[string]any{
		"id":      "source-1",
		"success": true,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = "evt-source-ok"

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.order); got != 0 {
		t.Fatalf("expected no alerts, got %d", got)
	}
}

func TestDarkDataScanCompleted_TriggersDSPMScan(t *testing.T) {
	consumer, _, dspm, _, tenantID := newDataConsumerForTest(t)
	event, err := events.NewEvent("data.darkdata.scan_completed", "data-service", tenantID, map[string]any{
		"scan_id":           "scan-1",
		"assets_discovered": 12,
		"pii_found":         4,
		"high_risk":         2,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	event.ID = "evt-darkdata-1"

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if dspm.calls != 1 {
		t.Fatalf("expected 1 DSPM scan trigger, got %d", dspm.calls)
	}
	if dspm.actor == nil || dspm.actor.UserEmail != cyberDataSystemActorEmail {
		t.Fatalf("expected system actor email %q, got %+v", cyberDataSystemActorEmail, dspm.actor)
	}
}

func TestDarkDataScanCompleted_Debounced(t *testing.T) {
	consumer, _, dspm, _, tenantID := newDataConsumerForTest(t)
	ctx := context.Background()

	for idx := 0; idx < 2; idx++ {
		event, err := events.NewEvent("data.darkdata.scan_completed", "data-service", tenantID, map[string]any{
			"scan_id": fmt.Sprintf("scan-%d", idx),
		})
		if err != nil {
			t.Fatalf("NewEvent() error = %v", err)
		}
		event.ID = fmt.Sprintf("evt-darkdata-%d", idx)
		if err := consumer.Handle(ctx, event); err != nil {
			t.Fatalf("Handle() error = %v", err)
		}
	}

	if dspm.calls != 1 {
		t.Fatalf("expected debounce to limit DSPM triggers to 1, got %d", dspm.calls)
	}
}
