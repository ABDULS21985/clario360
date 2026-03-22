package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/model"
)

func TestEnrichmentCache_NilReceiver(t *testing.T) {
	var cache *EnrichmentCache
	ctx := context.Background()
	tenantID := uuid.New()
	indicatorID := uuid.New()

	result, hit, err := cache.Get(ctx, tenantID, indicatorID)
	if err != nil || hit || result != nil {
		t.Fatal("nil cache Get should return nil, false, nil")
	}

	if err := cache.Set(ctx, tenantID, indicatorID, &model.IndicatorEnrichment{}); err != nil {
		t.Fatal("nil cache Set should return nil")
	}

	if err := cache.Invalidate(ctx, tenantID, indicatorID); err != nil {
		t.Fatal("nil cache Invalidate should return nil")
	}
}

func TestEnrichmentCache_NilRedis(t *testing.T) {
	cache := NewEnrichmentCache(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	indicatorID := uuid.New()

	result, hit, err := cache.Get(ctx, tenantID, indicatorID)
	if err != nil || hit || result != nil {
		t.Fatal("nil redis Get should return nil, false, nil")
	}

	if err := cache.Set(ctx, tenantID, indicatorID, &model.IndicatorEnrichment{}); err != nil {
		t.Fatal("nil redis Set should return nil")
	}

	if err := cache.Invalidate(ctx, tenantID, indicatorID); err != nil {
		t.Fatal("nil redis Invalidate should return nil")
	}
}

func TestEnrichmentCache_SetNilData(t *testing.T) {
	cache := NewEnrichmentCache(nil)
	ctx := context.Background()
	if err := cache.Set(ctx, uuid.New(), uuid.New(), nil); err != nil {
		t.Fatal("Set nil data should return nil")
	}
}

func TestEnrichmentCacheKey(t *testing.T) {
	tenantID := uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	indicatorID := uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
	expected := "cyber:enrichment:aaaaaaaa-0000-0000-0000-000000000001:bbbbbbbb-0000-0000-0000-000000000002"
	if got := enrichmentCacheKey(tenantID, indicatorID); got != expected {
		t.Fatalf("unexpected cache key: %s", got)
	}
}
