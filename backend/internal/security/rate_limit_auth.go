package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// AuthRateLimiter provides enhanced rate limiting for authentication endpoints.
type AuthRateLimiter struct {
	redis   *redis.Client
	metrics *Metrics
	logger  zerolog.Logger
	config  *AuthRateLimitConfig
}

// AuthRateLimitConfig configures auth-specific rate limits.
type AuthRateLimitConfig struct {
	LoginPerEmail         int
	LoginPerIP            int
	LoginWindow           time.Duration
	RegisterPerIP         int
	RegisterWindow        time.Duration
	PasswordResetPerEmail int
	PasswordResetPerIP    int
	PasswordResetWindow   time.Duration
	MFAPerSession         int
	MFAWindow             time.Duration
	LockoutThreshold      int
	LockoutDuration       time.Duration
	EscalationThreshold   int
	EscalationWindow      time.Duration
}

// DefaultAuthRateLimitConfig returns production-safe defaults.
func DefaultAuthRateLimitConfig() *AuthRateLimitConfig {
	return &AuthRateLimitConfig{
		LoginPerEmail:         5,
		LoginPerIP:            20,
		LoginWindow:           15 * time.Minute,
		RegisterPerIP:         3,
		RegisterWindow:        time.Hour,
		PasswordResetPerEmail: 3,
		PasswordResetPerIP:    10,
		PasswordResetWindow:   time.Hour,
		MFAPerSession:         5,
		MFAWindow:             15 * time.Minute,
		LockoutThreshold:      5,
		LockoutDuration:       15 * time.Minute,
		EscalationThreshold:   20,
		EscalationWindow:      time.Hour,
	}
}

// NewAuthRateLimiter creates a new auth rate limiter.
func NewAuthRateLimiter(rdb *redis.Client, cfg *AuthRateLimitConfig, metrics *Metrics, logger zerolog.Logger) *AuthRateLimiter {
	return &AuthRateLimiter{
		redis:   rdb,
		metrics: metrics,
		logger:  logger.With().Str("component", "auth_rate_limiter").Logger(),
		config:  cfg,
	}
}

// RateLimitError represents a rate limit exceeded condition.
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
	Limit      int
	Remaining  int
	Reset      time.Time
}

func (e *RateLimitError) Error() string { return e.Message }

// CheckLoginRate checks login rate limits for both email and IP dimensions.
func (l *AuthRateLimiter) CheckLoginRate(ctx context.Context, email, ip string) error {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — login rate limiting disabled (fail-open)")
		return nil
	}

	// Check account lockout first
	locked, err := l.isAccountLocked(ctx, email)
	if err != nil {
		l.logger.Error().Err(err).Msg("failed to check account lockout")
		return nil // Fail open
	}
	if locked {
		return ErrAccountLocked
	}

	// Check per-email limit
	emailHash := hashDimension("email", email)
	if err := l.slidingWindowCheck(ctx, "auth:login:email:"+emailHash,
		l.config.LoginPerEmail, l.config.LoginWindow, "login_email"); err != nil {
		return err
	}

	// Check per-IP limit
	ipHash := hashDimension("ip", ip)
	if err := l.slidingWindowCheck(ctx, "auth:login:ip:"+ipHash,
		l.config.LoginPerIP, l.config.LoginWindow, "login_ip"); err != nil {
		return err
	}

	return nil
}

// RecordLoginFailure records a failed login for lockout tracking.
func (l *AuthRateLimiter) RecordLoginFailure(ctx context.Context, email, ip string) {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — login failure not recorded (fail-open)")
		return
	}

	emailHash := hashDimension("email", email)
	key := "auth:login:failures:" + emailHash

	pipe := l.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, l.config.LockoutDuration)
	_, _ = pipe.Exec(ctx)

	// Check if lockout threshold reached
	count, err := l.redis.Get(ctx, key).Int()
	if err == nil && count >= l.config.LockoutThreshold {
		l.lockAccount(ctx, emailHash)
		if l.metrics != nil {
			l.metrics.AccountLockouts.Inc()
		}
		l.logger.Warn().
			Str("email_hash", emailHash).
			Str("ip", ip).
			Int("failures", count).
			Msg("account locked due to repeated failures")
	}

	// Check escalation threshold
	ipHash := hashDimension("ip", ip)
	l.checkEscalation(ctx, ipHash, ip)
}

// RecordLoginSuccess clears failure counters for the email.
func (l *AuthRateLimiter) RecordLoginSuccess(ctx context.Context, email string) {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — login success not recorded (fail-open)")
		return
	}
	emailHash := hashDimension("email", email)
	l.redis.Del(ctx, "auth:login:failures:"+emailHash)
}

// CheckRegisterRate checks registration rate limits per IP.
func (l *AuthRateLimiter) CheckRegisterRate(ctx context.Context, ip string) error {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — register rate limiting disabled (fail-open)")
		return nil
	}
	ipHash := hashDimension("ip", ip)
	return l.slidingWindowCheck(ctx, "auth:register:ip:"+ipHash,
		l.config.RegisterPerIP, l.config.RegisterWindow, "register")
}

// CheckPasswordResetRate checks password reset rate limits.
func (l *AuthRateLimiter) CheckPasswordResetRate(ctx context.Context, email, ip string) error {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — password reset rate limiting disabled (fail-open)")
		return nil
	}

	emailHash := hashDimension("email", email)
	if err := l.slidingWindowCheck(ctx, "auth:reset:email:"+emailHash,
		l.config.PasswordResetPerEmail, l.config.PasswordResetWindow, "reset_email"); err != nil {
		return err
	}

	ipHash := hashDimension("ip", ip)
	return l.slidingWindowCheck(ctx, "auth:reset:ip:"+ipHash,
		l.config.PasswordResetPerIP, l.config.PasswordResetWindow, "reset_ip")
}

// CheckMFARate checks MFA attempt rate limits per session.
func (l *AuthRateLimiter) CheckMFARate(ctx context.Context, sessionID string) error {
	if l.redis == nil {
		l.logger.Warn().Msg("Redis unavailable — MFA rate limiting disabled (fail-open)")
		return nil
	}
	sessionHash := hashDimension("session", sessionID)
	return l.slidingWindowCheck(ctx, "auth:mfa:session:"+sessionHash,
		l.config.MFAPerSession, l.config.MFAWindow, "mfa")
}

// slidingWindowCheck implements the Redis sorted set sliding window algorithm.
func (l *AuthRateLimiter) slidingWindowCheck(ctx context.Context, key string, limit int, window time.Duration, category string) error {
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
		l.logger.Error().Err(err).Str("key", key).Msg("rate limit check failed")
		return nil // Fail open
	}

	count := int(countCmd.Val())
	if count >= limit {
		if l.metrics != nil {
			l.metrics.RateLimitHits.WithLabelValues(category).Inc()
		}
		return &RateLimitError{
			RetryAfter: window - now.Sub(windowStart),
			Message:    "too many requests, please try again later",
			Limit:      limit,
			Remaining:  0,
			Reset:      now.Add(window),
		}
	}

	return nil
}

// isAccountLocked checks if an account is currently locked.
func (l *AuthRateLimiter) isAccountLocked(ctx context.Context, email string) (bool, error) {
	emailHash := hashDimension("email", email)
	key := "auth:lockout:" + emailHash
	exists, err := l.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

// lockAccount locks an account for the configured duration.
func (l *AuthRateLimiter) lockAccount(ctx context.Context, emailHash string) {
	key := "auth:lockout:" + emailHash
	l.redis.Set(ctx, key, "1", l.config.LockoutDuration)
}

// checkEscalation checks if IP-level attack escalation threshold is reached.
func (l *AuthRateLimiter) checkEscalation(ctx context.Context, ipHash, ip string) {
	key := "auth:escalation:" + ipHash
	pipe := l.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, l.config.EscalationWindow)
	results, err := pipe.Exec(ctx)
	if err != nil || len(results) == 0 {
		return
	}

	count, err := results[0].(*redis.IntCmd).Result()
	if err != nil {
		return
	}

	if int(count) >= l.config.EscalationThreshold {
		if l.metrics != nil {
			l.metrics.EscalationTriggers.Inc()
		}
		l.logger.Error().
			Str("ip_hash", ipHash).
			Int64("attempts", count).
			Msg("security escalation: repeated auth failures from IP")
	}
}

// hashDimension creates a SHA-256 hash of a dimension value for privacy.
func hashDimension(dimension, value string) string {
	hash := sha256.Sum256([]byte(dimension + ":" + value))
	return hex.EncodeToString(hash[:])
}

// AuthRateLimitMiddleware returns middleware for auth endpoint rate limiting.
func AuthRateLimitMiddleware(limiter *AuthRateLimiter, category string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)

			var err error
			switch category {
			case "register":
				err = limiter.CheckRegisterRate(r.Context(), ip)
			case "login":
				// Login checks require email — handled in the handler itself
				next.ServeHTTP(w, r)
				return
			default:
				next.ServeHTTP(w, r)
				return
			}

			if err != nil {
				writeRateLimitResponse(w, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeRateLimitResponse writes a 429 response with appropriate headers.
func writeRateLimitResponse(w http.ResponseWriter, err error) {
	var rlErr *RateLimitError
	if rle, ok := err.(*RateLimitError); ok {
		rlErr = rle
	} else {
		rlErr = &RateLimitError{
			RetryAfter: 60 * time.Second,
			Message:    err.Error(),
			Limit:      0,
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
		"code":    "RATE_LIMITED",
		"message": rlErr.Message,
	})
}
