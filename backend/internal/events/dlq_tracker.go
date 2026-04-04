package events

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const dlqPendingPrefix = "dlq:pending:"

// DLQTracker stores per-service DLQ counts in Redis so services can expose
// operational status without scanning Kafka offsets on every request.
type DLQTracker struct {
	redis *redis.Client
}

func NewDLQTracker(redisClient *redis.Client) *DLQTracker {
	return &DLQTracker{redis: redisClient}
}

func (t *DLQTracker) Increment(ctx context.Context, service, _ string) error {
	if t == nil || t.redis == nil || strings.TrimSpace(service) == "" {
		return nil
	}
	return t.redis.Incr(ctx, dlqPendingPrefix+service).Err()
}

func (t *DLQTracker) Count(ctx context.Context, service string) (int64, error) {
	if t == nil || t.redis == nil || strings.TrimSpace(service) == "" {
		return 0, nil
	}

	count, err := t.redis.Get(ctx, dlqPendingPrefix+service).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return count, err
}

func DLQCountHandler(service string, tracker *DLQTracker, logger zerolog.Logger) http.HandlerFunc {
	service = strings.TrimSpace(service)

	return func(w http.ResponseWriter, r *http.Request) {
		count, err := tracker.Count(r.Context(), service)
		if err != nil {
			logger.Error().Err(err).Str("service", service).Msg("failed to read dlq count")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   "failed to read dlq count",
				"service": service,
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service":       service,
			"pending_count": count,
		})
	}
}
