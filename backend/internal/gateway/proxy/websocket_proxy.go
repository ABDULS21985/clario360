package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/gateway/metrics"
)

// WebSocketProxy authenticates and proxies WebSocket connections to a backend service.
// Authentication happens BEFORE the WebSocket upgrade is performed.
type WebSocketProxy struct {
	target         *url.URL
	jwtMgr         *auth.JWTManager
	limiter        WSLimiter
	allowedOrigins []*regexp.Regexp
	metrics        *metrics.GatewayMetrics
	logger         zerolog.Logger
}

// WSLimiter is a minimal interface for WebSocket rate limiting.
type WSLimiter interface {
	CheckWS(userID string) (allowed bool, remaining int)
}

// NewWebSocketProxy creates a WebSocket proxy that authenticates before upgrading.
func NewWebSocketProxy(
	target *url.URL,
	jwtMgr *auth.JWTManager,
	allowedOrigins []string,
	limiter WSLimiter,
	gwMetrics *metrics.GatewayMetrics,
	logger zerolog.Logger,
) *WebSocketProxy {
	compiled := compileOrigins(allowedOrigins)
	return &WebSocketProxy{
		target:         target,
		jwtMgr:         jwtMgr,
		limiter:        limiter,
		allowedOrigins: compiled,
		metrics:        gwMetrics,
		logger:         logger,
	}
}

// ServeHTTP implements http.Handler. It validates the JWT from ?token=<JWT> query param,
// applies rate limiting, dials the backend WebSocket, upgrades the client connection,
// and pipes messages bidirectionally.
func (p *WebSocketProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	serviceName := p.target.Host

	// ── 1. AUTHENTICATE BEFORE UPGRADE ──────────────────────────────────────────
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Also accept bearer token via Authorization header for non-browser clients.
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}
	if tokenStr == "" {
		writeWSError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}

	claims, err := p.jwtMgr.ValidateAccessToken(tokenStr)
	if err != nil {
		writeWSError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
		return
	}

	// ── 2. RATE LIMIT (per user_id) ──────────────────────────────────────────────
	if p.limiter != nil {
		if allowed, _ := p.limiter.CheckWS(claims.UserID); !allowed {
			w.Header().Set("Retry-After", "60")
			writeWSError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many websocket connections")
			return
		}
	}

	// ── 3. DIAL BACKEND ──────────────────────────────────────────────────────────
	backendURL := buildBackendWSURL(p.target, r)

	reqID, _ := r.Context().Value(requestIDKey).(string)

	backendHeaders := http.Header{
		"X-Tenant-ID":  []string{claims.TenantID},
		"X-User-ID":    []string{claims.UserID},
		"X-User-Email": []string{claims.Email},
		"X-Request-ID": []string{reqID},
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Subprotocols:     websocket.Subprotocols(r),
	}

	backendConn, _, err := dialer.Dial(backendURL, backendHeaders)
	if err != nil {
		p.logger.Error().Err(err).Str("service", serviceName).Str("url", backendURL).Msg("websocket dial backend failed")
		writeWSError(w, http.StatusBadGateway, "BAD_GATEWAY", "upstream service unavailable")
		return
	}

	// ── 4. UPGRADE CLIENT CONNECTION ─────────────────────────────────────────────
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     p.checkOrigin,
	}

	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		p.logger.Error().Err(err).Msg("websocket upgrade client connection failed")
		backendConn.Close()
		return
	}

	// Track metrics.
	if p.metrics != nil {
		p.metrics.WebSocketConnectionsActive.WithLabelValues(serviceName).Inc()
		p.metrics.WebSocketConnectionsTotal.WithLabelValues(serviceName).Inc()
	}

	// ── 5. BIDIRECTIONAL PIPE ────────────────────────────────────────────────────
	start := time.Now()
	var once sync.Once
	closeAll := func() {
		once.Do(func() {
			clientConn.Close()
			backendConn.Close()
		})
	}
	defer closeAll()

	done := make(chan struct{})

	// Client → Backend
	go func() {
		defer func() { close(done) }()
		for {
			msgType, msg, err := clientConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					p.logger.Debug().Err(err).Msg("client websocket read error")
				}
				// Forward close to backend.
				_ = backendConn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := backendConn.WriteMessage(msgType, msg); err != nil {
				p.logger.Debug().Err(err).Msg("backend websocket write error")
				return
			}
		}
	}()

	// Backend → Client
	go func() {
		for {
			msgType, msg, err := backendConn.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					p.logger.Debug().Err(err).Msg("backend websocket read error")
				}
				_ = clientConn.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			if err := clientConn.WriteMessage(msgType, msg); err != nil {
				p.logger.Debug().Err(err).Msg("client websocket write error")
				return
			}
		}
	}()

	// Wait for client→backend goroutine to exit (either side closed).
	<-done

	// ── 6. CLEANUP ───────────────────────────────────────────────────────────────
	duration := time.Since(start).Seconds()
	if p.metrics != nil {
		p.metrics.WebSocketConnectionsActive.WithLabelValues(serviceName).Dec()
		p.metrics.WebSocketDuration.WithLabelValues(serviceName).Observe(duration)
	}
	p.logger.Debug().
		Str("service", serviceName).
		Str("user_id", claims.UserID).
		Str("tenant_id", claims.TenantID).
		Float64("duration_seconds", duration).
		Msg("websocket connection closed")
}

// buildBackendWSURL constructs the backend WebSocket URL, stripping the ?token param.
func buildBackendWSURL(target *url.URL, r *http.Request) string {
	scheme := "ws"
	if target.Scheme == "https" {
		scheme = "wss"
	}

	// Copy and sanitize query params — remove "token" so JWT is not forwarded.
	q := r.URL.Query()
	q.Del("token")

	path := r.URL.Path
	queryStr := q.Encode()

	result := fmt.Sprintf("%s://%s%s", scheme, target.Host, path)
	if queryStr != "" {
		result += "?" + queryStr
	}
	return result
}

// checkOrigin validates the WebSocket connection origin against the allowed list.
func (p *WebSocketProxy) checkOrigin(r *http.Request) bool {
	if len(p.allowedOrigins) == 0 {
		return true // permissive in development
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	for _, re := range p.allowedOrigins {
		if re.MatchString(origin) {
			return true
		}
	}
	return false
}

// compileOrigins converts origin patterns to regexps.
// Patterns like "https://*.clario360.com" are converted to regex.
func compileOrigins(patterns []string) []*regexp.Regexp {
	var out []*regexp.Regexp
	for _, p := range patterns {
		if p == "*" {
			// Wildcard: match everything (only in dev — production config.Validate() rejects this).
			re := regexp.MustCompile(`.*`)
			out = append(out, re)
			continue
		}
		// Escape for regex, then replace \* with a subdomain pattern.
		escaped := regexp.QuoteMeta(p)
		// "https://\*\.clario360\.com" → "^https://[a-zA-Z0-9-]+\.clario360\.com$"
		pattern := strings.Replace(escaped, `\*`, `[a-zA-Z0-9-]+`, 1)
		re, err := regexp.Compile("^" + pattern + "$")
		if err != nil {
			continue
		}
		out = append(out, re)
	}
	return out
}

func writeWSError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}
