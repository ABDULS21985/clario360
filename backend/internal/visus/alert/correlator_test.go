package alert

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
	"github.com/clario360/platform/internal/visus/model"
)

type fakeCorrelatorKPIRepo struct {
	items []model.KPIDefinition
}

func (f *fakeCorrelatorKPIRepo) List(ctx context.Context, tenantID uuid.UUID, page, perPage int, sortCol, sortDir, search, suite string, enabled *bool) ([]model.KPIDefinition, int, error) {
	return f.items, len(f.items), nil
}

type fakeCorrelatorSnapshotRepo struct {
	items map[uuid.UUID][]model.KPISnapshot
}

func (f *fakeCorrelatorSnapshotRepo) ListByKPI(ctx context.Context, tenantID, kpiID uuid.UUID, query model.KPIQuery) ([]model.KPISnapshot, error) {
	return f.items[kpiID], nil
}

type fakeCorrelatorAlertRepo struct {
	criticalSuites []string
	recent         map[string][]model.ExecutiveAlert
}

func (f *fakeCorrelatorAlertRepo) CountCriticalSuites(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	return f.criticalSuites, nil
}

func (f *fakeCorrelatorAlertRepo) ListRecentBySource(ctx context.Context, tenantID uuid.UUID, sourceSuite string, since time.Time, severity *string) ([]model.ExecutiveAlert, error) {
	return f.recent[sourceSuite], nil
}

type fakeCorrelatorSuiteClient struct {
	meta    aggregator.FetchMetadata
	payload map[string]any
}

func (f *fakeCorrelatorSuiteClient) Fetch(ctx context.Context, suite, endpoint string, tenantID uuid.UUID, target interface{}) aggregator.FetchMetadata {
	if ptr, ok := target.(*map[string]any); ok && f.payload != nil {
		*ptr = f.payload
	}
	return f.meta
}

func TestCorrelate_DegradingTrend(t *testing.T) {
	tenantID := uuid.New()
	kpiID := uuid.New()
	correlator := NewCorrelator(
		&fakeCorrelatorKPIRepo{items: []model.KPIDefinition{{ID: kpiID, Name: "Pipeline Success Rate", Direction: model.KPIDirectionHigherIsBetter, Suite: model.KPISuiteData}}},
		&fakeCorrelatorSnapshotRepo{items: map[uuid.UUID][]model.KPISnapshot{
			kpiID: {
				{Value: 70},
				{Value: 80},
				{Value: 90},
			},
		}},
		&fakeCorrelatorAlertRepo{recent: map[string][]model.ExecutiveAlert{}},
		&fakeCorrelatorSuiteClient{meta: aggregator.FetchMetadata{Status: "unavailable"}},
	)

	patterns, err := correlator.DetectPatterns(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
	if !strings.Contains(patterns[0].Title, "degrading") {
		t.Fatalf("expected degrading trend pattern, got %+v", patterns[0])
	}
}

func TestCorrelate_MultiSuiteIssue(t *testing.T) {
	correlator := NewCorrelator(
		&fakeCorrelatorKPIRepo{},
		&fakeCorrelatorSnapshotRepo{},
		&fakeCorrelatorAlertRepo{criticalSuites: []string{"cyber", "data"}, recent: map[string][]model.ExecutiveAlert{}},
		&fakeCorrelatorSuiteClient{meta: aggregator.FetchMetadata{Status: "unavailable"}},
	)

	patterns, err := correlator.DetectPatterns(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(patterns))
	}
	if patterns[0].Category != model.AlertCategoryStrategic {
		t.Fatalf("expected strategic alert, got %s", patterns[0].Category)
	}
}

func TestCorrelate_NoPattern(t *testing.T) {
	tenantID := uuid.New()
	kpiID := uuid.New()
	correlator := NewCorrelator(
		&fakeCorrelatorKPIRepo{items: []model.KPIDefinition{{ID: kpiID, Name: "Healthy KPI", Direction: model.KPIDirectionLowerIsBetter, Suite: model.KPISuiteCyber}}},
		&fakeCorrelatorSnapshotRepo{items: map[uuid.UUID][]model.KPISnapshot{
			kpiID: {
				{Value: 41},
				{Value: 42},
				{Value: 43},
			},
		}},
		&fakeCorrelatorAlertRepo{recent: map[string][]model.ExecutiveAlert{}},
		&fakeCorrelatorSuiteClient{meta: aggregator.FetchMetadata{Status: "unavailable"}},
	)

	patterns, err := correlator.DetectPatterns(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patterns) != 0 {
		t.Fatalf("expected no patterns, got %+v", patterns)
	}
}
