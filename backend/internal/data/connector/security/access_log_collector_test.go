package security

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	dataconnector "github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
)

func TestAccessLogCollectorCollectAll(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	repo := &fakeSourceRepo{
		active: map[uuid.UUID][]*repository.SourceRecord{
			tenantID: {
				{
					Source: &model.DataSource{
						ID:       sourceID,
						TenantID: tenantID,
						Name:     "warehouse",
						Type:     model.DataSourceTypeClickHouse,
						Status:   model.DataSourceStatusActive,
					},
					EncryptedConfig: []byte(`{"host":"ignored"}`),
				},
			},
		},
	}
	registry := &fakeRegistry{
		connector: &fakeSecurityConnector{
			events: []dataconnector.DataAccessEvent{
				{
					Timestamp:    time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
					User:         "analyst",
					Action:       "query",
					Database:     "warehouse",
					Table:        "events",
					QueryHash:    "hash",
					QueryPreview: "SELECT * FROM events",
					SourceType:   "clickhouse",
					SourceID:     sourceID,
					TenantID:     tenantID,
				},
			},
		},
	}
	publisher := &fakePublisher{}
	collector := NewAccessLogCollector(
		registry,
		repo,
		fakeDecryptor{},
		rdb,
		publisher,
		zerolog.Nop(),
		prometheus.NewRegistry(),
	)

	events, err := collector.CollectAll(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("CollectAll() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	if len(publisher.published) != 1 {
		t.Fatalf("published events = %d, want 1", len(publisher.published))
	}
	if got, err := server.Get(lastCollectionKey(sourceID)); err != nil || got == "" {
		t.Fatal("expected last collection cursor to be stored in redis")
	}
}

func TestAccessLogCollectorMarksSourceDegradedAfterRepeatedFailures(t *testing.T) {
	tenantID := uuid.New()
	sourceID := uuid.New()
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	repo := &fakeSourceRepo{
		active: map[uuid.UUID][]*repository.SourceRecord{
			tenantID: {
				{
					Source: &model.DataSource{
						ID:       sourceID,
						TenantID: tenantID,
						Name:     "warehouse",
						Type:     model.DataSourceTypeClickHouse,
						Status:   model.DataSourceStatusActive,
					},
					EncryptedConfig: []byte(`{"host":"ignored"}`),
				},
			},
		},
	}
	registry := &fakeRegistry{connector: &fakeBrokenSecurityConnector{}}
	publisher := &fakePublisher{}
	collector := NewAccessLogCollector(
		registry,
		repo,
		fakeDecryptor{},
		rdb,
		publisher,
		zerolog.Nop(),
		prometheus.NewRegistry(),
	)

	for i := 0; i < 5; i++ {
		_, _ = collector.CollectAll(context.Background(), tenantID)
	}
	if repo.statusUpdates != 1 {
		t.Fatalf("statusUpdates = %d, want 1", repo.statusUpdates)
	}
	if repo.lastStatus != model.DataSourceStatusError {
		t.Fatalf("lastStatus = %s, want error", repo.lastStatus)
	}
	if len(publisher.published) == 0 {
		t.Fatal("expected degraded event to be published")
	}
}

type fakeDecryptor struct{}

func (fakeDecryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	return append([]byte(nil), ciphertext...), nil
}

type fakeSourceRepo struct {
	active        map[uuid.UUID][]*repository.SourceRecord
	statusUpdates int
	lastStatus    model.DataSourceStatus
}

func (f *fakeSourceRepo) ListActive(_ context.Context, tenantID uuid.UUID) ([]*repository.SourceRecord, error) {
	return f.active[tenantID], nil
}

func (f *fakeSourceRepo) ListActiveTenants(context.Context) ([]uuid.UUID, error) {
	tenantIDs := make([]uuid.UUID, 0, len(f.active))
	for tenantID := range f.active {
		tenantIDs = append(tenantIDs, tenantID)
	}
	return tenantIDs, nil
}

func (f *fakeSourceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ uuid.UUID, status model.DataSourceStatus, _ *string) error {
	f.statusUpdates++
	f.lastStatus = status
	return nil
}

type fakeRegistry struct {
	connector dataconnector.Connector
}

func (f *fakeRegistry) CreateWithSourceContext(_ model.DataSourceType, _ json.RawMessage, _ uuid.UUID, _ uuid.UUID) (dataconnector.Connector, error) {
	return f.connector, nil
}

type fakeSecurityConnector struct {
	events []dataconnector.DataAccessEvent
}

func (f *fakeSecurityConnector) TestConnection(context.Context) (*dataconnector.ConnectionTestResult, error) {
	return &dataconnector.ConnectionTestResult{Success: true}, nil
}

func (f *fakeSecurityConnector) DiscoverSchema(context.Context, dataconnector.DiscoveryOptions) (*model.DiscoveredSchema, error) {
	return &model.DiscoveredSchema{}, nil
}

func (f *fakeSecurityConnector) FetchData(context.Context, string, dataconnector.FetchParams) (*dataconnector.DataBatch, error) {
	return &dataconnector.DataBatch{}, nil
}

func (f *fakeSecurityConnector) ReadQuery(context.Context, string, []any) (*dataconnector.DataBatch, error) {
	return &dataconnector.DataBatch{}, nil
}

func (f *fakeSecurityConnector) WriteData(context.Context, string, []map[string]any, dataconnector.WriteParams) (*dataconnector.WriteResult, error) {
	return &dataconnector.WriteResult{}, nil
}

func (f *fakeSecurityConnector) EstimateSize(context.Context) (*dataconnector.SizeEstimate, error) {
	return &dataconnector.SizeEstimate{}, nil
}

func (f *fakeSecurityConnector) QueryAccessLogs(context.Context, time.Time) ([]dataconnector.DataAccessEvent, error) {
	return f.events, nil
}

func (f *fakeSecurityConnector) ListDataLocations(context.Context) ([]dataconnector.DataLocation, error) {
	return nil, nil
}

func (f *fakeSecurityConnector) Close() error { return nil }

type fakeBrokenSecurityConnector struct {
	fakeSecurityConnector
}

func (f *fakeBrokenSecurityConnector) QueryAccessLogs(context.Context, time.Time) ([]dataconnector.DataAccessEvent, error) {
	return nil, context.DeadlineExceeded
}

type fakePublisher struct {
	published []*events.Event
}

func (f *fakePublisher) Publish(_ context.Context, _ string, event *events.Event) error {
	f.published = append(f.published, event)
	return nil
}
