package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type SuiteCacheRecord struct {
	TenantID       uuid.UUID
	Suite          string
	Endpoint       string
	ResponseData   map[string]any
	FetchedAt      time.Time
	TTLSeconds     int
	FetchLatencyMS *int
}

type SuiteCacheRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewSuiteCacheRepository(db *pgxpool.Pool, logger zerolog.Logger) *SuiteCacheRepository {
	return &SuiteCacheRepository{db: db, logger: logger.With().Str("repo", "visus_suite_cache").Logger()}
}

func (r *SuiteCacheRepository) Upsert(ctx context.Context, item *SuiteCacheRecord) error {
	if item == nil {
		return ErrValidation
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO visus_suite_cache (tenant_id, suite, endpoint, response_data, fetched_at, ttl_seconds, fetch_latency_ms)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (tenant_id, suite, endpoint)
		DO UPDATE SET response_data = EXCLUDED.response_data,
		              fetched_at = EXCLUDED.fetched_at,
		              ttl_seconds = EXCLUDED.ttl_seconds,
		              fetch_latency_ms = EXCLUDED.fetch_latency_ms`,
		item.TenantID, item.Suite, item.Endpoint, marshalJSON(item.ResponseData), item.FetchedAt, item.TTLSeconds, item.FetchLatencyMS,
	)
	return wrapErr("upsert suite cache", err)
}

func (r *SuiteCacheRepository) Get(ctx context.Context, tenantID uuid.UUID, suite, endpoint string) (*SuiteCacheRecord, error) {
	row := r.db.QueryRow(ctx, `
		SELECT tenant_id, suite, endpoint, response_data, fetched_at, ttl_seconds, fetch_latency_ms
		FROM visus_suite_cache
		WHERE tenant_id = $1 AND suite = $2 AND endpoint = $3`, tenantID, suite, endpoint)
	var raw []byte
	item := &SuiteCacheRecord{}
	if err := row.Scan(&item.TenantID, &item.Suite, &item.Endpoint, &raw, &item.FetchedAt, &item.TTLSeconds, &item.FetchLatencyMS); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, wrapErr("get suite cache", err)
	}
	item.ResponseData = unmarshalMap(raw)
	return item, nil
}
