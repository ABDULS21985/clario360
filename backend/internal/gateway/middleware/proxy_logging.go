package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	mw "github.com/clario360/platform/internal/middleware"
)

// sensitiveQueryParams are query parameter names whose values must never appear in logs.
var sensitiveQueryParams = []string{"token", "password", "secret", "key", "api_key", "access_token"}

// loggingResponseWriter captures status code and bytes for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	written      bool
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

// ProxyLogging logs every proxied request with structured fields.
// Sensitive data (JWT, API keys, request/response bodies, sensitive query params) is NEVER logged.
func ProxyLogging(logger zerolog.Logger, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			// Extract user/tenant from context (empty for public routes).
			userID := ""
			tenantID := ""
			if user := auth.UserFromContext(r.Context()); user != nil {
				userID = user.ID
				tenantID = user.TenantID
			}

			requestID := mw.GetRequestID(r.Context())

			// Redact sensitive query parameters before logging.
			query := redactQuery(r.URL.RawQuery)

			var event *zerolog.Event
			switch {
			case wrapped.statusCode >= 500:
				event = logger.Error()
			case wrapped.statusCode >= 400:
				event = logger.Info()
			default:
				event = logger.Debug()
			}

			event.
				Str("request_id", requestID).
				Str("service", serviceName).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", query).
				Int64("content_length", r.ContentLength).
				Str("user_agent", r.UserAgent()).
				Str("ip", getClientIP(r)).
				Str("tenant_id", tenantID).
				Str("user_id", userID).
				Int("status", wrapped.statusCode).
				Int("response_bytes", wrapped.bytesWritten).
				Dur("latency", duration).
				Msg("gateway request")
		})
	}
}

// redactQuery replaces the values of sensitive query params with "[REDACTED]".
func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return "[PARSE_ERROR]"
	}

	for _, param := range sensitiveQueryParams {
		if vals.Has(param) {
			vals.Set(param, "[REDACTED]")
		}
	}
	return vals.Encode()
}

// getClientIP extracts the real client IP from proxy headers.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Return the first (original client) IP in the chain.
		if idx := strings.IndexByte(xff, ','); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
