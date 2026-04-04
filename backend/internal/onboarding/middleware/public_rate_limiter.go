package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type PublicRateLimitConfig struct {
	RequestsPerWindow int
	Window            time.Duration
	KeyPrefix         string
}

func NewPublicRateLimiter(redisClient *redis.Client, cfg PublicRateLimitConfig) func(http.Handler) http.Handler {
	if cfg.RequestsPerWindow <= 0 {
		cfg.RequestsPerWindow = 10
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if strings.TrimSpace(cfg.KeyPrefix) == "" {
		cfg.KeyPrefix = "ratelimit:onboarding:public"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if redisClient == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := publicRateLimitKey(cfg.KeyPrefix, r)
			count, ttl, err := incrementPublicCounter(r.Context(), redisClient, key, cfg.Window)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			remaining := cfg.RequestsPerWindow - count
			if remaining < 0 {
				remaining = 0
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerWindow))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
			w.Header().Set("X-Captcha-Ready", "true")

			if count > cfg.RequestsPerWindow {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status":  429,
					"code":    "RATE_LIMITED",
					"message": "too many requests, please try again later",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func incrementPublicCounter(ctx context.Context, redisClient *redis.Client, key string, window time.Duration) (int, time.Duration, error) {
	pipe := redisClient.TxPipeline()
	incr := pipe.Incr(ctx, key)
	ttl := pipe.TTL(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, 0, err
	}

	expiry := ttl.Val()
	if expiry <= 0 {
		expiry = window
	}

	return int(incr.Val()), expiry, nil
}

func publicRateLimitKey(prefix string, r *http.Request) string {
	material := strings.Join([]string{
		clientIP(r),
		strings.ToUpper(r.Method),
		r.URL.Path,
	}, ":")
	sum := sha256.Sum256([]byte(material))
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(sum[:]))
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
