package middleware

import (
	"net/http"

	"github.com/redis/go-redis/v9"

	sharedmw "github.com/clario360/platform/internal/middleware"
)

func RateLimiter(rdb *redis.Client) func(http.Handler) http.Handler {
	return sharedmw.RateLimit(rdb, sharedmw.RateLimitConfig{
		RequestsPerMinute: 600,
		Burst:             120,
		Window:            sharedmw.DefaultRateLimitConfig().Window,
	})
}
