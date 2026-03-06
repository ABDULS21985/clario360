package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// UploadGuard enforces file size limits on upload requests.
// It checks Content-Length before reading the body and wraps the body with MaxBytesReader.
func UploadGuard(maxSizeMB int) func(http.Handler) http.Handler {
	maxBytes := int64(maxSizeMB) * 1024 * 1024

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to upload endpoints (POST with content)
			if r.Method != http.MethodPost && r.Method != http.MethodPut {
				next.ServeHTTP(w, r)
				return
			}

			// Pre-check Content-Length header if present
			if cl := r.Header.Get("Content-Length"); cl != "" {
				size, err := strconv.ParseInt(cl, 10, 64)
				if err == nil && size > maxBytes {
					writeUploadError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE",
						"file exceeds maximum upload size", r)
					return
				}
			}

			// Wrap body with MaxBytesReader as a safety net
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			next.ServeHTTP(w, r)
		})
	}
}

func writeUploadError(w http.ResponseWriter, status int, code, message string, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code":       code,
			"message":    message,
			"request_id": r.Header.Get("X-Request-ID"),
		},
	})
}
