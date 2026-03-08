package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// APIRateLimiter provides per-tenant, per-endpoint rate limiting for API endpoints.
type APIRateLimiter struct {
	redis   *redis.Client
	metrics *Metrics
	logger  zerolog.Logger
	config  *APIRateLimitConfig
}

// APIRateLimitConfig configures API rate limiting.
type APIRateLimitConfig struct {
	DefaultPerMinute int
	BurstMultiplier  float64 // Multiplier for burst allowance
	EndpointLimits   map[string]EndpointRateLimit
}

// EndpointRateLimit specifies per-endpoint rate limits.
type EndpointRateLimit struct {
	RequestsPerMinute int
	BurstSize         int
}

// DefaultAPIRateLimitConfig returns production defaults.
func DefaultAPIRateLimitConfig() *APIRateLimitConfig {
	return &APIRateLimitConfig{
		DefaultPerMinute: 100,
		BurstMultiplier:  2.0,
		EndpointLimits: map[string]EndpointRateLimit{
			"/api/v1/auth/":   {RequestsPerMinute: 20, BurstSize: 5},
			"/api/v1/upload":  {RequestsPerMinute: 50, BurstSize: 10},
			"/api/v1/export":  {RequestsPerMinute: 10, BurstSize: 3},
			"/api/v1/reports": {RequestsPerMinute: 30, BurstSize: 5},
		},
	}
}

// NewAPIRateLimiter creates a new API rate limiter.
func NewAPIRateLimiter(rdb *redis.Client, cfg *APIRateLimitConfig, metrics *Metrics, logger zerolog.Logger) *APIRateLimiter {
	return &APIRateLimiter{
		redis:   rdb,
		metrics: metrics,
		logger:  logger.With().Str("component", "api_rate_limiter").Logger(),
		config:  cfg,
	}
}

// RateLimitInfo contains rate limit state for inclusion in response headers.
type RateLimitInfo struct {
	Limit     int
	Remaining int
	Reset     time.Time
}

// CheckAPIRate checks the API rate limit for a request.
// Returns (info, nil) on success with current quota info, or (nil, error) when rate-limited.
func (l *APIRateLimiter) CheckAPIRate(ctx context.Context, tenantID, path, ip string) (*RateLimitInfo, error) {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — rate limiting disabled (fail-open)")
		return nil, nil
	}

	limit := l.config.DefaultPerMinute
	window := time.Minute

	// Check for endpoint-specific limits
	for prefix, epLimit := range l.config.EndpointLimits {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			limit = epLimit.RequestsPerMinute
			break
		}
	}

	// Use tenant ID as the primary dimension, fall back to IP
	dimension := tenantID
	if dimension == "" {
		dimension = ip
	}

	key := fmt.Sprintf("api:rate:%s:%s", hashDimension("tenant", dimension), pathCategory(path))

	now := time.Now()
	windowStart := now.Add(-window)

	pipe := l.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixMicro(), 10))
	countCmd := pipe.ZCard(ctx, key)
	member := fmt.Sprintf("%d", now.UnixNano())
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixMicro()),
		Member: member,
	})
	pipe.Expire(ctx, key, window+time.Minute)

	_, err := pipe.Exec(ctx)
	if err != nil {
		l.logger.Error().Err(err).Msg("API rate limit check failed — fail-open")
		return nil, nil // Fail open
	}

	count := int(countCmd.Val())
	remaining := limit - count - 1
	if remaining < 0 {
		remaining = 0
	}
	reset := now.Add(window)

	if count >= limit {
		if l.metrics != nil {
			l.metrics.RateLimitHits.WithLabelValues("api").Inc()
		}
		return nil, &RateLimitError{
			RetryAfter: window,
			Message:    "API rate limit exceeded, please try again later",
			Limit:      limit,
			Remaining:  0,
			Reset:      reset,
		}
	}

	return &RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}

// APIRateLimitMiddleware returns middleware for API rate limiting.
// Sets X-RateLimit-* headers on both success and 429 responses.
func APIRateLimitMiddleware(limiter *APIRateLimiter, secLogger *SecurityLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := auth.TenantFromContext(r.Context())
			ip := extractClientIP(r)

			info, err := limiter.CheckAPIRate(r.Context(), tenantID, r.URL.Path, ip)
			if err != nil {
				secLogger.LogFromRequest(r, EventRateLimited, SeverityLow,
					"API rate limit exceeded", true)

				var rlErr *RateLimitError
				if rle, ok := err.(*RateLimitError); ok {
					rlErr = rle
				} else {
					rlErr = &RateLimitError{
						RetryAfter: time.Minute,
						Message:    "rate limit exceeded",
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(int(rlErr.RetryAfter.Seconds())))
				if rlErr.Limit > 0 {
					w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rlErr.Limit))
					w.Header().Set("X-RateLimit-Remaining", "0")
					w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(rlErr.Reset.Unix(), 10))
				}
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]string{
						"code":    "RATE_LIMITED",
						"message": rlErr.Message,
					},
				})
				return
			}

			// Set rate-limit headers on successful responses
			if info != nil {
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(info.Limit))
				w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(info.Remaining))
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(info.Reset.Unix(), 10))
			}

			next.ServeHTTP(w, r)
		})
	}
}

// pathCategory normalizes a URL path to a rate limit category.
func pathCategory(path string) string {
	// Strip trailing slashes and query params for grouping
	if len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	// Group by API prefix segments (e.g., /api/v1/assets → "assets")
	segments := splitPath(path)
	if len(segments) >= 3 && segments[0] == "api" && segments[1] == "v1" {
		return segments[2]
	}
	if len(segments) >= 1 {
		return segments[0]
	}
	return "root"
}

// splitPath splits a URL path into segments.
func splitPath(path string) []string {
	var segments []string
	for _, s := range splitPathRaw(path) {
		if s != "" {
			segments = append(segments, s)
		}
	}
	return segments
}

func splitPathRaw(path string) []string {
	result := make([]string, 0, 8)
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				result = append(result, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		result = append(result, path[start:])
	}
	return result
}
