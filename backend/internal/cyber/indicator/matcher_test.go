package indicator

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

type fakeIndicatorRepo struct {
	items []*model.ThreatIndicator
}

func (f *fakeIndicatorRepo) ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ThreatIndicator, error) {
	return f.items, nil
}

func TestMatcherIPAndCIDR(t *testing.T) {
	tenantID := uuid.New()
	matcher := NewMatcher(&fakeIndicatorRepo{
		items: []*model.ThreatIndicator{
			{ID: uuid.New(), TenantID: tenantID, Type: model.IndicatorTypeIP, Value: "8.8.8.8", Active: true},
			{ID: uuid.New(), TenantID: tenantID, Type: model.IndicatorTypeCIDR, Value: "10.0.0.0/8", Active: true},
		},
	}, zerolog.New(io.Discard))
	if err := matcher.Load(context.Background(), tenantID); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	sourceIP := "8.8.8.8"
	destIP := "10.0.1.5"
	event := &model.SecurityEvent{TenantID: tenantID, SourceIP: &sourceIP, DestIP: &destIP}
	matches := matcher.Match(event)
	if len(matches) != 2 {
		t.Fatalf("expected 2 indicator matches, got %d", len(matches))
	}
}
