package middleware

import (
	"net/http"

	"github.com/redis/go-redis/v9"

	sharedmw "github.com/clario360/platform/internal/middleware"
)

func RateLimiter(rdb *redis.Client) func(http.Handler) http.Handler {
	return sharedmw.RateLimit(rdb, sharedmw.RateLimitConfig{
		RequestsPerWindow: 600,
		Window:            sharedmw.DefaultRateLimitConfig().Window,
		KeyPrefix:         "data:ratelimit",
	})
}
