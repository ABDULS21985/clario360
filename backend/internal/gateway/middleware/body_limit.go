package middleware

import (
	"net/http"
	"strings"
)

// BodyLimit enforces maximum request body size per route.
// It checks Content-Length pre-read to reject oversized requests immediately,
// then wraps the body with http.MaxBytesReader for streaming enforcement.
func BodyLimit(defaultMaxMB int, routeOverrides map[string]int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			maxMB := defaultMaxMB
			if routeOverrides != nil {
				// Match by longest prefix.
				bestLen := 0
				for prefix, mb := range routeOverrides {
					if strings.HasPrefix(r.URL.Path, prefix) && len(prefix) > bestLen {
						bestLen = len(prefix)
						maxMB = mb
					}
				}
			}

			maxBytes := int64(maxMB) * 1024 * 1024

			// Fast-path: reject by Content-Length before reading any bytes.
			// This protects against slow-loris upload attacks.
			if r.ContentLength > maxBytes {
				reqID := getReqID(r)
				writeGWError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE",
					"request body exceeds the maximum allowed size", reqID)
				return
			}

			// Enforce actual body size during streaming.
			if maxBytes > 0 {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}

			next.ServeHTTP(w, r)
		})
	}
}
