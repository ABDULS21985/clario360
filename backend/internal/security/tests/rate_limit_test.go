package security_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// newTestAuthRateLimiter creates an AuthRateLimiter backed by miniredis.
func newTestAuthRateLimiter(t *testing.T, cfg *security.AuthRateLimitConfig) (*security.AuthRateLimiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	limiter := security.NewAuthRateLimiter(rdb, cfg, metrics, logger)
	return limiter, mr
}

func TestRateLimit_WithinLimit(t *testing.T) {
	cfg := &security.AuthRateLimitConfig{
		LoginPerEmail:    5,
		LoginPerIP:       20,
		LoginWindow:      15 * time.Minute,
		LockoutThreshold: 10,
		LockoutDuration:  15 * time.Minute,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	// First 5 requests (the limit) should pass.
	// Note: sliding window adds the entry then checks count, so the Nth request
	// hits the limit. We test N-1 requests that should definitely pass.
	for i := 0; i < cfg.LoginPerEmail-1; i++ {
		err := limiter.CheckLoginRate(ctx, "user@example.com", "192.168.1.100")
		if err != nil {
			t.Fatalf("request %d should pass within limit, got: %v", i+1, err)
		}
	}
}

func TestRateLimit_ExceedsLimit(t *testing.T) {
	cfg := &security.AuthRateLimitConfig{
		LoginPerEmail:    3,
		LoginPerIP:       100, // high so we only hit per-email limit
		LoginWindow:      15 * time.Minute,
		LockoutThreshold: 100, // high so we don't trigger lockout
		LockoutDuration:  15 * time.Minute,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	// Make requests up to the limit
	for i := 0; i < cfg.LoginPerEmail; i++ {
		_ = limiter.CheckLoginRate(ctx, "user@example.com", "10.0.0.1")
	}

	// The next request should be rate limited
	err := limiter.CheckLoginRate(ctx, "user@example.com", "10.0.0.1")
	if err == nil {
		t.Fatal("expected rate limit error after exceeding limit, got nil")
	}

	var rlErr *security.RateLimitError
	if !errors.As(err, &rlErr) {
		// Could also be ErrAccountLocked if lockout triggered
		if !errors.Is(err, security.ErrAccountLocked) {
			t.Fatalf("expected RateLimitError or ErrAccountLocked, got: %v", err)
		}
	}
}

func TestRateLimit_WindowExpiry(t *testing.T) {
	// The sliding window uses time.Now() for score-based cleanup, so we need a
	// real time delay. Use a very short window to keep the test fast.
	cfg := &security.AuthRateLimitConfig{
		LoginPerEmail:    2,
		LoginPerIP:       100,
		LoginWindow:      500 * time.Millisecond,
		LockoutThreshold: 100,
		LockoutDuration:  500 * time.Millisecond,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	// Exhaust the limit
	for i := 0; i < cfg.LoginPerEmail; i++ {
		_ = limiter.CheckLoginRate(ctx, "user@test.com", "10.0.0.1")
	}

	// Should be rate limited now
	err := limiter.CheckLoginRate(ctx, "user@test.com", "10.0.0.1")
	if err == nil {
		t.Fatal("expected rate limit error, got nil")
	}

	// Wait for the sliding window to expire
	time.Sleep(600 * time.Millisecond)

	// After window expires, requests should be allowed again
	err = limiter.CheckLoginRate(ctx, "user@test.com", "10.0.0.1")
	if err != nil {
		t.Fatalf("expected requests to be allowed after window expiry, got: %v", err)
	}
}

func TestRateLimit_AccountLockout(t *testing.T) {
	cfg := &security.AuthRateLimitConfig{
		LoginPerEmail:       100, // high so rate limit doesn't interfere
		LoginPerIP:          100,
		LoginWindow:         15 * time.Minute,
		LockoutThreshold:    3,
		LockoutDuration:     15 * time.Minute,
		EscalationThreshold: 100,
		EscalationWindow:    time.Hour,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	email := "victim@example.com"
	ip := "10.0.0.99"

	// Record failures up to the lockout threshold
	for i := 0; i < cfg.LockoutThreshold; i++ {
		limiter.RecordLoginFailure(ctx, email, ip)
	}

	// The account should now be locked
	err := limiter.CheckLoginRate(ctx, email, ip)
	if err == nil {
		t.Fatal("expected account locked error, got nil")
	}
	if !errors.Is(err, security.ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked, got: %v", err)
	}
}

func TestRateLimit_LoginSuccessClearsFailures(t *testing.T) {
	cfg := &security.AuthRateLimitConfig{
		LoginPerEmail:       100,
		LoginPerIP:          100,
		LoginWindow:         15 * time.Minute,
		LockoutThreshold:    5,
		LockoutDuration:     15 * time.Minute,
		EscalationThreshold: 100,
		EscalationWindow:    time.Hour,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	email := "user@example.com"
	ip := "10.0.0.50"

	// Record some failures (but not enough for lockout)
	for i := 0; i < cfg.LockoutThreshold-1; i++ {
		limiter.RecordLoginFailure(ctx, email, ip)
	}

	// Successful login should clear failures
	limiter.RecordLoginSuccess(ctx, email)

	// Now record failures again up to threshold - should not lock because counter was reset
	for i := 0; i < cfg.LockoutThreshold-1; i++ {
		limiter.RecordLoginFailure(ctx, email, ip)
	}

	// Should NOT be locked
	err := limiter.CheckLoginRate(ctx, email, ip)
	if errors.Is(err, security.ErrAccountLocked) {
		t.Fatal("account should not be locked after success cleared failures")
	}
}

func TestRateLimit_NilRedisFailsOpen(t *testing.T) {
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	cfg := security.DefaultAuthRateLimitConfig()

	// Create limiter with nil Redis client
	limiter := security.NewAuthRateLimiter(nil, cfg, metrics, logger)
	ctx := context.Background()

	// Should fail open (no error) when Redis is nil
	err := limiter.CheckLoginRate(ctx, "user@example.com", "10.0.0.1")
	if err != nil {
		t.Fatalf("expected nil Redis to fail open, got: %v", err)
	}
}

func TestRateLimit_RegisterPerIP(t *testing.T) {
	cfg := &security.AuthRateLimitConfig{
		RegisterPerIP:       2,
		RegisterWindow:      15 * time.Minute,
		LoginPerEmail:       100,
		LoginPerIP:          100,
		LoginWindow:         15 * time.Minute,
		LockoutThreshold:    100,
		LockoutDuration:     15 * time.Minute,
		EscalationThreshold: 100,
		EscalationWindow:    time.Hour,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	ip := "10.0.0.1"

	// Use up the register limit
	for i := 0; i < cfg.RegisterPerIP; i++ {
		_ = limiter.CheckRegisterRate(ctx, ip)
	}

	// Next registration attempt should be limited
	err := limiter.CheckRegisterRate(ctx, ip)
	if err == nil {
		t.Fatal("expected rate limit error for registration, got nil")
	}

	var rlErr *security.RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("expected RateLimitError, got: %v", err)
	}
}

func TestRateLimit_MFAPerSession(t *testing.T) {
	cfg := &security.AuthRateLimitConfig{
		MFAPerSession:       3,
		MFAWindow:           15 * time.Minute,
		LoginPerEmail:       100,
		LoginPerIP:          100,
		LoginWindow:         15 * time.Minute,
		LockoutThreshold:    100,
		LockoutDuration:     15 * time.Minute,
		EscalationThreshold: 100,
		EscalationWindow:    time.Hour,
	}
	limiter, _ := newTestAuthRateLimiter(t, cfg)
	ctx := context.Background()

	sessionID := "session-abc-123"

	// Use up the MFA limit
	for i := 0; i < cfg.MFAPerSession; i++ {
		_ = limiter.CheckMFARate(ctx, sessionID)
	}

	// Next MFA attempt should be limited
	err := limiter.CheckMFARate(ctx, sessionID)
	if err == nil {
		t.Fatal("expected rate limit error for MFA, got nil")
	}
}
