package security_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	security "github.com/clario360/platform/internal/security"
)

// newTestSessionManager creates a SessionManager backed by miniredis.
func newTestSessionManager(t *testing.T, cfg *security.SessionConfig) (*security.SessionManager, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()
	sm := security.NewSessionManager(rdb, cfg, metrics, logger)
	return sm, mr
}

// fakeRequest creates a minimal http.Request for session validation.
func fakeRequest() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	r.Header.Set("User-Agent", "TestBrowser/1.0")
	r.Header.Set("Accept-Language", "en-US")
	return r
}

func TestSession_CreateSucceeds(t *testing.T) {
	cfg := &security.SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     5,
		RotateOnAuth:      true,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, _ := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	err := sm.CreateSession(ctx, "session-001", "user-1", "tenant-1", r)
	if err != nil {
		t.Fatalf("expected session creation to succeed, got: %v", err)
	}
}

func TestSession_ValidateValidSession(t *testing.T) {
	cfg := &security.SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     5,
		RotateOnAuth:      true,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, _ := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	err := sm.CreateSession(ctx, "session-valid", "user-1", "tenant-1", r)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	err = sm.ValidateSession(ctx, "session-valid", r)
	if err != nil {
		t.Fatalf("expected valid session to pass validation, got: %v", err)
	}
}

func TestSession_IdleTimeout(t *testing.T) {
	cfg := &security.SessionConfig{
		IdleTimeout:       5 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     5,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, mr := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	err := sm.CreateSession(ctx, "session-idle", "user-1", "tenant-1", r)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// Fast-forward past idle timeout
	mr.FastForward(6 * time.Minute)

	// We need to update the session data's last_active to reflect the old time.
	// miniredis FastForward only advances key TTLs, not the data itself.
	// Instead, we manually set the session data with old last_active.
	oldInfo := security.SessionInfo{
		UserID:     "user-1",
		TenantID:   "tenant-1",
		CreatedAt:  time.Now().Add(-10 * time.Minute),
		LastActive: time.Now().Add(-6 * time.Minute), // past idle timeout
	}
	data, _ := json.Marshal(oldInfo)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdb.Set(ctx, "session:session-idle", data, 24*time.Hour)

	err = sm.ValidateSession(ctx, "session-idle", r)
	if err == nil {
		t.Fatal("expected idle-expired session to be rejected, got nil")
	}
	if !errors.Is(err, security.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got: %v", err)
	}
}

func TestSession_AbsoluteMaxAge(t *testing.T) {
	cfg := &security.SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    1 * time.Hour,
		MaxConcurrent:     5,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, mr := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	err := sm.CreateSession(ctx, "session-abs", "user-1", "tenant-1", r)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// Set session data with old creation time (past absolute max age)
	// but recent last_active to ensure it is the absolute timeout that triggers
	oldInfo := security.SessionInfo{
		UserID:     "user-1",
		TenantID:   "tenant-1",
		CreatedAt:  time.Now().Add(-2 * time.Hour), // past absolute max
		LastActive: time.Now(),                     // recently active
	}
	data, _ := json.Marshal(oldInfo)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rdb.Set(ctx, "session:session-abs", data, 24*time.Hour)
	_ = mr // keep reference

	err = sm.ValidateSession(ctx, "session-abs", r)
	if err == nil {
		t.Fatal("expected absolute-expired session to be rejected, got nil")
	}
	if !errors.Is(err, security.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got: %v", err)
	}
}

func TestSession_ConcurrentLimit(t *testing.T) {
	maxConcurrent := 2
	cfg := &security.SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     maxConcurrent,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, _ := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	userID := "user-concurrent"
	tenantID := "tenant-1"

	// Create sessions up to the limit
	for i := 0; i < maxConcurrent; i++ {
		sessionID := fmt.Sprintf("session-c-%d", i)
		err := sm.CreateSession(ctx, sessionID, userID, tenantID, r)
		if err != nil {
			t.Fatalf("session %d creation should succeed, got: %v", i, err)
		}
	}

	// The next session should be rejected
	err := sm.CreateSession(ctx, "session-c-excess", userID, tenantID, r)
	if err == nil {
		t.Fatal("expected concurrent session limit error, got nil")
	}
	if !errors.Is(err, security.ErrConcurrentSession) {
		t.Fatalf("expected ErrConcurrentSession, got: %v", err)
	}
}

func TestSession_DestroyClears(t *testing.T) {
	cfg := &security.SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     5,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, _ := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	sessionID := "session-destroy"
	userID := "user-1"

	err := sm.CreateSession(ctx, sessionID, userID, "tenant-1", r)
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	// Validate works before destruction
	err = sm.ValidateSession(ctx, sessionID, r)
	if err != nil {
		t.Fatalf("session should be valid before destroy: %v", err)
	}

	// Destroy the session
	sm.DestroySession(ctx, sessionID, userID)

	// Validation should now fail
	err = sm.ValidateSession(ctx, sessionID, r)
	if err == nil {
		t.Fatal("expected destroyed session to fail validation, got nil")
	}
	if !errors.Is(err, security.ErrSessionExpired) {
		t.Fatalf("expected ErrSessionExpired, got: %v", err)
	}
}

func TestSession_DestroyAllSessions(t *testing.T) {
	cfg := &security.SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     10,
		BindToIP:          false,
		BindToFingerprint: false,
	}
	sm, _ := newTestSessionManager(t, cfg)
	ctx := context.Background()
	r := fakeRequest()

	userID := "user-destroy-all"
	tenantID := "tenant-1"

	// Create multiple sessions
	sessionIDs := []string{"session-da-1", "session-da-2", "session-da-3"}
	for _, sid := range sessionIDs {
		err := sm.CreateSession(ctx, sid, userID, tenantID, r)
		if err != nil {
			t.Fatalf("create session %s failed: %v", sid, err)
		}
	}

	// Verify all sessions are valid
	for _, sid := range sessionIDs {
		err := sm.ValidateSession(ctx, sid, r)
		if err != nil {
			t.Fatalf("session %s should be valid before destroy all: %v", sid, err)
		}
	}

	// Destroy all sessions for the user
	sm.DestroyAllSessions(ctx, userID)

	// All sessions should now be invalid
	for _, sid := range sessionIDs {
		err := sm.ValidateSession(ctx, sid, r)
		if err == nil {
			t.Fatalf("expected session %s to be invalid after DestroyAllSessions, got nil", sid)
		}
		if !errors.Is(err, security.ErrSessionExpired) {
			t.Fatalf("expected ErrSessionExpired for session %s, got: %v", sid, err)
		}
	}
}

func TestSession_NilRedisFailsOpen(t *testing.T) {
	cfg := security.DefaultSessionConfig()
	reg := prometheus.NewRegistry()
	metrics := security.NewMetrics(reg)
	logger := zerolog.Nop()

	sm := security.NewSessionManager(nil, cfg, metrics, logger)
	ctx := context.Background()
	r := fakeRequest()

	// All operations should succeed silently with nil Redis
	err := sm.CreateSession(ctx, "session-nil", "user-1", "tenant-1", r)
	if err != nil {
		t.Fatalf("expected nil Redis create to succeed, got: %v", err)
	}

	err = sm.ValidateSession(ctx, "session-nil", r)
	if err != nil {
		t.Fatalf("expected nil Redis validate to succeed, got: %v", err)
	}

	// Destroy should not panic
	sm.DestroySession(ctx, "session-nil", "user-1")
	sm.DestroyAllSessions(ctx, "user-1")
}
