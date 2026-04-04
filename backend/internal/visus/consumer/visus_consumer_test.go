package consumer

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/visus/model"
	repostore "github.com/clario360/platform/internal/visus/repository"
)

type fakeExecutiveAlertCreator struct {
	alerts []*model.ExecutiveAlert
}

func (f *fakeExecutiveAlertCreator) Create(_ context.Context, alert *model.ExecutiveAlert) (*model.ExecutiveAlert, error) {
	cloned := *alert
	if cloned.ID == uuid.Nil {
		cloned.ID = uuid.New()
	}
	f.alerts = append(f.alerts, &cloned)
	return &cloned, nil
}

type fakeKPIStore struct {
	items []model.KPIDefinition
}

func (f *fakeKPIStore) ListEnabled(_ context.Context, _ uuid.UUID) ([]model.KPIDefinition, error) {
	return append([]model.KPIDefinition(nil), f.items...), nil
}

func (f *fakeKPIStore) UpdateSnapshotState(_ context.Context, _ uuid.UUID, id uuid.UUID, at time.Time, value float64, status model.KPIStatus) error {
	for idx := range f.items {
		if f.items[idx].ID == id {
			f.items[idx].LastSnapshotAt = &at
			f.items[idx].LastValue = &value
			f.items[idx].LastStatus = &status
			return nil
		}
	}
	return repostore.ErrNotFound
}

type fakeSnapshotStore struct {
	latest  map[uuid.UUID]*model.KPISnapshot
	created []*model.KPISnapshot
}

func (f *fakeSnapshotStore) Create(_ context.Context, item *model.KPISnapshot) (*model.KPISnapshot, error) {
	cloned := *item
	if cloned.ID == uuid.Nil {
		cloned.ID = uuid.New()
	}
	if f.latest == nil {
		f.latest = map[uuid.UUID]*model.KPISnapshot{}
	}
	f.latest[item.KPIID] = &cloned
	f.created = append(f.created, &cloned)
	return &cloned, nil
}

func (f *fakeSnapshotStore) LatestByKPI(_ context.Context, _ uuid.UUID, kpiID uuid.UUID) (*model.KPISnapshot, error) {
	if snapshot, ok := f.latest[kpiID]; ok {
		cloned := *snapshot
		return &cloned, nil
	}
	return nil, repostore.ErrNotFound
}

func newVisusConsumerForTest(t *testing.T) (*VisusConsumer, *fakeExecutiveAlertCreator, *fakeKPIStore, *fakeSnapshotStore, *redis.Client, *miniredis.Miniredis, string) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
		server.Close()
	})

	alerts := &fakeExecutiveAlertCreator{}
	kpiID := uuid.New()
	kpis := &fakeKPIStore{items: []model.KPIDefinition{
		{
			ID:                kpiID,
			Name:              "Security Risk Score",
			Direction:         model.KPIDirectionLowerIsBetter,
			WarningThreshold:  ptrFloat(60),
			CriticalThreshold: ptrFloat(80),
		},
		{
			ID:                uuid.New(),
			Name:              "Data Quality Score",
			Direction:         model.KPIDirectionHigherIsBetter,
			WarningThreshold:  ptrFloat(85),
			CriticalThreshold: ptrFloat(70),
		},
		{
			ID:                uuid.New(),
			Name:              "Open Contradictions",
			Direction:         model.KPIDirectionLowerIsBetter,
			WarningThreshold:  ptrFloat(10),
			CriticalThreshold: ptrFloat(20),
		},
	}}
	snapshots := &fakeSnapshotStore{latest: map[uuid.UUID]*model.KPISnapshot{}}
	tenantID := uuid.NewString()

	consumer := NewVisusConsumer(zerolog.New(nil))
	consumer.alerts = alerts
	consumer.kpis = kpis
	consumer.snapshots = snapshots
	consumer.redis = client
	consumer.now = func() time.Time { return time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC) }

	return consumer, alerts, kpis, snapshots, client, server, tenantID
}

func ptrFloat(value float64) *float64 {
	return &value
}

func TestCriticalAlert_CreatesExecutiveAlert(t *testing.T) {
	consumer, alerts, _, _, _, _, tenantID := newVisusConsumerForTest(t)
	event, err := events.NewEvent("cyber.alert.created", "cyber-service", tenantID, map[string]any{
		"id":                   "alert-1",
		"title":                "Critical Ransomware Alert",
		"severity":             "critical",
		"confidence_score":     0.97,
		"affected_asset_count": 4,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.alerts); got != 1 {
		t.Fatalf("expected 1 executive alert, got %d", got)
	}
}

func TestHighAlert_Ignored(t *testing.T) {
	consumer, alerts, _, _, _, _, tenantID := newVisusConsumerForTest(t)
	event, err := events.NewEvent("cyber.alert.created", "cyber-service", tenantID, map[string]any{
		"id":                   "alert-1",
		"title":                "High Alert",
		"severity":             "high",
		"affected_asset_count": 1,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(alerts.alerts); got != 0 {
		t.Fatalf("expected no executive alerts, got %d", got)
	}
}

func TestRiskScore_SignificantIncrease(t *testing.T) {
	consumer, alerts, _, snapshots, _, _, tenantID := newVisusConsumerForTest(t)
	event, err := events.NewEvent("cyber.risk.score_calculated", "cyber-service", tenantID, map[string]any{
		"score":          78.0,
		"previous_score": 60.0,
		"grade":          "C",
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(snapshots.created); got != 1 {
		t.Fatalf("expected 1 KPI snapshot, got %d", got)
	}
	if got := len(alerts.alerts); got != 1 {
		t.Fatalf("expected 1 executive alert, got %d", got)
	}
}

func TestRiskScore_SmallChange(t *testing.T) {
	consumer, alerts, _, snapshots, _, _, tenantID := newVisusConsumerForTest(t)
	event, err := events.NewEvent("cyber.risk.score_calculated", "cyber-service", tenantID, map[string]any{
		"score":          63.0,
		"previous_score": 60.0,
		"grade":          "B",
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}

	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if got := len(snapshots.created); got != 1 {
		t.Fatalf("expected 1 KPI snapshot, got %d", got)
	}
	if got := len(alerts.alerts); got != 0 {
		t.Fatalf("expected no executive alert, got %d", got)
	}
}

func TestLineageUpdate_InvalidatesCache(t *testing.T) {
	consumer, _, _, _, client, _, tenantID := newVisusConsumerForTest(t)
	key := "visus:lineage_graph:" + tenantID
	if err := client.Set(context.Background(), key, "cached", time.Hour).Err(); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	event, err := events.NewEvent("data.lineage.graph_updated", "data-service", tenantID, map[string]any{
		"graph_version": "v2",
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	if err := consumer.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if _, err := client.Get(context.Background(), key).Result(); err == nil {
		t.Fatal("expected lineage cache key to be deleted")
	}
}
