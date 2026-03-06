package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	mw "github.com/clario360/platform/internal/middleware"
)

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
// It excludes sensitive headers (Authorization) and request/response bodies.
func ProxyLogging(logger zerolog.Logger, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			// Extract user/tenant info from context (may be empty for public routes)
			userID := ""
			tenantID := ""
			if user := auth.UserFromContext(r.Context()); user != nil {
				userID = user.ID
				tenantID = user.TenantID
			}

			requestID := mw.GetRequestID(r.Context())

			var event *zerolog.Event
			switch {
			case wrapped.statusCode >= 500:
				event = logger.Error()
			case wrapped.statusCode >= 400:
				event = logger.Warn()
			default:
				event = logger.Info()
			}

			event.
				Str("request_id", requestID).
				Str("service", serviceName).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
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

// getClientIP extracts the real client IP from proxy headers.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Return the first IP in the chain (the original client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}
