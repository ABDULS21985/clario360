package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

type dedupRepository interface {
	ExistsDedupKey(ctx context.Context, tenantID uuid.UUID, event *model.DataAccessEvent) (bool, error)
}

type Deduplicator struct {
	repo dedupRepository
	seen map[string]struct{}
}

func NewDeduplicator(repo dedupRepository) *Deduplicator {
	return &Deduplicator{
		repo: repo,
		seen: make(map[string]struct{}),
	}
}

func (d *Deduplicator) Reset() {
	d.seen = make(map[string]struct{})
}

func (d *Deduplicator) IsDuplicate(ctx context.Context, tenantID uuid.UUID, event *model.DataAccessEvent) (bool, error) {
	key := dedupKey(event)
	if _, ok := d.seen[key]; ok {
		return true, nil
	}
	exists, err := d.repo.ExistsDedupKey(ctx, tenantID, event)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	d.seen[key] = struct{}{}
	return false, nil
}

func dedupKey(event *model.DataAccessEvent) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		event.EntityID,
		event.SourceType,
		event.Action,
		event.QueryHash,
		event.EventTimestamp.UTC().Format(time.RFC3339Nano),
		event.SourceIP,
	)
}
