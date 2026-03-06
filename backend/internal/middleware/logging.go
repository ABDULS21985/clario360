package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/observability"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	bytesWritten int
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Logging logs every HTTP request with structured fields.
func Logging(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapped := wrapResponseWriter(w)

			// Inject logger into context
			ctx := observability.WithLogger(r.Context(), logger)
			r = r.WithContext(ctx)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			var event *zerolog.Event
			switch {
			case wrapped.status >= 500:
				event = logger.Error()
			case wrapped.status >= 400:
				event = logger.Warn()
			default:
				event = logger.Info()
			}

			event.
				Str("request_id", GetRequestID(r.Context())).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Int("status", wrapped.status).
				Int("bytes", wrapped.bytesWritten).
				Dur("duration", duration).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("http request")
		})
	}
}
