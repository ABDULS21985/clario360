package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// Timeout sets a per-request deadline. The timeout is looked up from routeOverrides
// by longest-prefix match; if no override matches, defaultTimeout is used.
// The proxy's Transport.ResponseHeaderTimeout handles upstream timeouts;
// this middleware catches cases where the overall handler is slow to complete.
func Timeout(defaultTimeout time.Duration, routeOverrides map[string]time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timeout := defaultTimeout
			if routeOverrides != nil {
				bestLen := 0
				for prefix, d := range routeOverrides {
					if strings.HasPrefix(r.URL.Path, prefix) && len(prefix) > bestLen {
						bestLen = len(prefix)
						timeout = d
					}
				}
			}

			if timeout <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)

			// Run the handler in a goroutine so we can detect deadline exceeded.
			done := make(chan struct{})
			go func() {
				defer close(done)
				next.ServeHTTP(w, r)
			}()

			select {
			case <-done:
				// Handler finished before timeout — all good.
			case <-ctx.Done():
				reqID := r.Header.Get("X-Request-ID")
				writeGWError(w, http.StatusGatewayTimeout, "GATEWAY_TIMEOUT",
					"request processing exceeded the time limit", reqID)
			}
		})
	}
}
