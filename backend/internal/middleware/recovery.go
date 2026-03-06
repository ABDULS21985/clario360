package middleware

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/observability"
)

// Recovery catches panics in HTTP handlers, logs the stack trace,
// and returns a 500 Internal Server Error response.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger := observability.LoggerFromContext(r.Context())
				logger.Error().
					Str("request_id", GetRequestID(r.Context())).
					Interface("panic", rec).
					Str("stack", string(debug.Stack())).
					Msg("panic recovered")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"status":  500,
					"code":    "INTERNAL_ERROR",
					"message": "an unexpected error occurred",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RecoveryWithLogger creates a recovery middleware with a specific logger.
func RecoveryWithLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error().
						Str("request_id", GetRequestID(r.Context())).
						Interface("panic", rec).
						Str("stack", string(debug.Stack())).
						Msg("panic recovered")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"status":  500,
						"code":    "INTERNAL_ERROR",
						"message": "an unexpected error occurred",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
