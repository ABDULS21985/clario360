package middleware

import (
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	sharedmw "github.com/clario360/platform/internal/middleware"
)

func RateLimiter(rdb *redis.Client, requestsPerMinute int) func(http.Handler) http.Handler {
	cfg := sharedmw.RateLimitConfig{
		RequestsPerWindow: requestsPerMinute,
		Window:            time.Minute,
		KeyPrefix:         "acta:ratelimit",
	}
	return sharedmw.RateLimit(rdb, cfg)
}
