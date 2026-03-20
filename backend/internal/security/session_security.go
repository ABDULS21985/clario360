package security

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// SessionManager provides session fixation prevention, idle timeout,
// and concurrent session control.
type SessionManager struct {
	redis   *redis.Client
	metrics *Metrics
	logger  zerolog.Logger
	config  *SessionConfig
}

// SessionConfig configures session security controls.
type SessionConfig struct {
	IdleTimeout       time.Duration
	AbsoluteMaxAge    time.Duration
	MaxConcurrent     int
	RotateOnAuth      bool // Rotate session ID on authentication state change
	BindToIP          bool // Bind session to client IP (strict)
	BindToFingerprint bool // Bind to user-agent fingerprint
}

// DefaultSessionConfig returns production-safe defaults.
func DefaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		IdleTimeout:       30 * time.Minute,
		AbsoluteMaxAge:    24 * time.Hour,
		MaxConcurrent:     5,
		RotateOnAuth:      true,
		BindToIP:          false, // Too strict for mobile/corporate NAT
		BindToFingerprint: true,
	}
}

// SessionInfo represents stored session metadata.
type SessionInfo struct {
	UserID      string    `json:"user_id"`
	TenantID    string    `json:"tenant_id"`
	CreatedAt   time.Time `json:"created_at"`
	LastActive  time.Time `json:"last_active"`
	ClientIP    string    `json:"client_ip"`
	UserAgent   string    `json:"user_agent"`
	Fingerprint string    `json:"fingerprint"`
}

// NewSessionManager creates a new session manager.
func NewSessionManager(rdb *redis.Client, cfg *SessionConfig, metrics *Metrics, logger zerolog.Logger) *SessionManager {
	return &SessionManager{
		redis:   rdb,
		metrics: metrics,
		logger:  logger.With().Str("component", "session_security").Logger(),
		config:  cfg,
	}
}

// ValidateSession checks session validity: idle timeout, absolute expiry,
// and optional IP/fingerprint binding.
func (sm *SessionManager) ValidateSession(ctx context.Context, sessionID string, r *http.Request) error {
	if sm.redis == nil {
		sm.logger.Warn().Msg("Redis unavailable — session validation disabled (fail-open)")
		return nil
	}

	key := "session:" + sessionID
	data, err := sm.redis.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return ErrSessionExpired
		}
		sm.logger.Error().Err(err).Msg("failed to read session")
		return nil // Fail open
	}

	var info SessionInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return ErrSessionExpired
	}

	now := time.Now()

	// Check absolute expiry
	if now.Sub(info.CreatedAt) > sm.config.AbsoluteMaxAge {
		sm.redis.Del(ctx, key)
		if sm.metrics != nil {
			sm.metrics.SessionEvents.WithLabelValues("absolute_expired").Inc()
		}
		return ErrSessionExpired
	}

	// Check idle timeout
	if now.Sub(info.LastActive) > sm.config.IdleTimeout {
		sm.redis.Del(ctx, key)
		if sm.metrics != nil {
			sm.metrics.SessionEvents.WithLabelValues("idle_expired").Inc()
		}
		return ErrSessionExpired
	}

	// Check IP binding
	if sm.config.BindToIP && info.ClientIP != extractClientIP(r) {
		if sm.metrics != nil {
			sm.metrics.SessionEvents.WithLabelValues("ip_mismatch").Inc()
		}
		return ErrSessionFixation
	}

	// Check fingerprint binding
	if sm.config.BindToFingerprint {
		fp := generateFingerprint(r)
		if info.Fingerprint != "" && info.Fingerprint != fp {
			if sm.metrics != nil {
				sm.metrics.SessionEvents.WithLabelValues("fingerprint_mismatch").Inc()
			}
			sm.logger.Warn().
				Str("session_id_hash", hashDimension("session", sessionID)).
				Msg("session fingerprint mismatch — possible session hijacking")
			return ErrSessionFixation
		}
	}

	// Update last active
	info.LastActive = now
	updated, _ := json.Marshal(info)
	sm.redis.Set(ctx, key, updated, sm.config.AbsoluteMaxAge)

	return nil
}

// CreateSession creates a new session entry.
func (sm *SessionManager) CreateSession(ctx context.Context, sessionID string, userID, tenantID string, r *http.Request) error {
	if sm.redis == nil {
		sm.logger.Warn().Msg("Redis unavailable — session creation skipped (fail-open)")
		return nil
	}

	// Enforce concurrent session limit
	if err := sm.enforceConcurrentLimit(ctx, userID); err != nil {
		return err
	}

	now := time.Now()
	info := SessionInfo{
		UserID:      userID,
		TenantID:    tenantID,
		CreatedAt:   now,
		LastActive:  now,
		ClientIP:    extractClientIP(r),
		UserAgent:   truncateString(r.UserAgent(), 256),
		Fingerprint: generateFingerprint(r),
	}

	data, err := json.Marshal(info)
	if err != nil {
		return err
	}

	key := "session:" + sessionID
	if err := sm.redis.Set(ctx, key, data, sm.config.AbsoluteMaxAge).Err(); err != nil {
		return err
	}

	// Track session in user's session set
	userKey := "user:sessions:" + userID
	sm.redis.SAdd(ctx, userKey, sessionID)
	sm.redis.Expire(ctx, userKey, sm.config.AbsoluteMaxAge)

	if sm.metrics != nil {
		sm.metrics.SessionEvents.WithLabelValues("created").Inc()
	}

	return nil
}

// DestroySession removes a session.
func (sm *SessionManager) DestroySession(ctx context.Context, sessionID, userID string) {
	if sm.redis == nil {
		sm.logger.Warn().Msg("Redis unavailable — session destroy skipped (fail-open)")
		return
	}

	sm.redis.Del(ctx, "session:"+sessionID)
	sm.redis.SRem(ctx, "user:sessions:"+userID, sessionID)

	if sm.metrics != nil {
		sm.metrics.SessionEvents.WithLabelValues("destroyed").Inc()
	}
}

// DestroyAllSessions removes all sessions for a user (e.g., password change).
func (sm *SessionManager) DestroyAllSessions(ctx context.Context, userID string) {
	if sm.redis == nil {
		sm.logger.Warn().Msg("Redis unavailable — destroy all sessions skipped (fail-open)")
		return
	}

	userKey := "user:sessions:" + userID
	sessions, err := sm.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return
	}

	for _, sid := range sessions {
		sm.redis.Del(ctx, "session:"+sid)
	}
	sm.redis.Del(ctx, userKey)

	if sm.metrics != nil {
		sm.metrics.SessionEvents.WithLabelValues("all_destroyed").Inc()
	}
}

// enforceConcurrentLimit evicts oldest sessions if limit exceeded.
func (sm *SessionManager) enforceConcurrentLimit(ctx context.Context, userID string) error {
	userKey := "user:sessions:" + userID
	sessions, err := sm.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		sm.logger.Error().Err(err).Msg("concurrent session check failed — fail-open")
		return nil
	}

	// Clean up expired sessions from the set
	var activeSessions []string
	for _, sid := range sessions {
		exists, _ := sm.redis.Exists(ctx, "session:"+sid).Result()
		if exists > 0 {
			activeSessions = append(activeSessions, sid)
		} else {
			sm.redis.SRem(ctx, userKey, sid)
		}
	}

	if len(activeSessions) >= sm.config.MaxConcurrent {
		if sm.metrics != nil {
			sm.metrics.SessionEvents.WithLabelValues("concurrent_limit").Inc()
		}
		return ErrConcurrentSession
	}

	return nil
}

// generateFingerprint creates a session fingerprint from request metadata.
func generateFingerprint(r *http.Request) string {
	return ClientIPHash(r.UserAgent() + "|" + r.Header.Get("Accept-Language"))
}

// truncateString truncates a string to maxLen characters.
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

// SessionSecurityMiddleware validates sessions on each request.
func SessionSecurityMiddleware(sm *SessionManager, secLogger *SecurityLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := auth.UserFromContext(r.Context())
			if user == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Use user ID as session identifier if no explicit session
			sessionID := r.Header.Get("X-Session-ID")
			if sessionID == "" {
				sessionID = user.ID
			}

			err := sm.ValidateSession(r.Context(), sessionID, r)
			if err != nil {
				secLogger.LogFromRequest(r, EventSessionFixation, SeverityHigh,
					fmt.Sprintf("session validation failed: %s", err), true)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "SESSION_INVALID",
					"message": "Your session has expired. Please log in again.",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
