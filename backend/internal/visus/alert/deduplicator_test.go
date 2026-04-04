package alert

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

type fakeDedupRepo struct {
	match       *model.ExecutiveAlert
	findErr     error
	updated     *model.ExecutiveAlert
	incremented bool
}

func (f *fakeDedupRepo) FindDedupMatch(ctx context.Context, tenantID uuid.UUID, dedupKey string, window time.Duration) (*model.ExecutiveAlert, error) {
	return f.match, f.findErr
}

func (f *fakeDedupRepo) IncrementOccurrence(ctx context.Context, tenantID, id uuid.UUID) (*model.ExecutiveAlert, error) {
	f.incremented = true
	if f.updated != nil {
		return f.updated, nil
	}
	updated := *f.match
	updated.OccurrenceCount++
	return &updated, nil
}

func TestDedup_NewAlert(t *testing.T) {
	dedup := NewDeduplicator(&fakeDedupRepo{findErr: repository.ErrNotFound})
	alert, matched, err := dedup.CheckAndUpdate(context.Background(), uuid.New(), "kpi_breach:test", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched {
		t.Fatal("expected no dedup match")
	}
	if alert != nil {
		t.Fatalf("expected nil alert, got %+v", alert)
	}
}

func TestDedup_ExistingWithinWindow(t *testing.T) {
	existing := &model.ExecutiveAlert{ID: uuid.New(), OccurrenceCount: 1}
	repo := &fakeDedupRepo{
		match:   existing,
		updated: &model.ExecutiveAlert{ID: existing.ID, OccurrenceCount: 2},
	}
	dedup := NewDeduplicator(repo)
	alert, matched, err := dedup.CheckAndUpdate(context.Background(), uuid.New(), "kpi_breach:test", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !matched {
		t.Fatal("expected dedup match")
	}
	if !repo.incremented {
		t.Fatal("expected occurrence count increment")
	}
	if alert.OccurrenceCount != 2 {
		t.Fatalf("expected occurrence count 2, got %d", alert.OccurrenceCount)
	}
}

func TestDedup_ExistingOutsideWindow(t *testing.T) {
	dedup := NewDeduplicator(&fakeDedupRepo{findErr: repository.ErrNotFound})
	alert, matched, err := dedup.CheckAndUpdate(context.Background(), uuid.New(), "kpi_breach:test", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched || alert != nil {
		t.Fatal("expected no match for alert outside dedup window")
	}
}

func TestDedup_DismissedNotMatched(t *testing.T) {
	dedup := NewDeduplicator(&fakeDedupRepo{findErr: repository.ErrNotFound})
	alert, matched, err := dedup.CheckAndUpdate(context.Background(), uuid.New(), "kpi_breach:test", time.Hour)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if matched || alert != nil {
		t.Fatal("expected dismissed alert to be ignored by dedup")
	}
}
