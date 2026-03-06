package middleware

import "net/http"

// SecurityHeaders sets defensive HTTP headers on every response.
// These protect against common web vulnerabilities (XSS, clickjacking, MIME sniffing, etc.).
func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Prevent MIME type sniffing
			h.Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			h.Set("X-Frame-Options", "DENY")

			// Disable XSS filter (modern browsers; CSP is preferred)
			h.Set("X-XSS-Protection", "0")

			// API-only service — no content to render
			h.Set("Content-Security-Policy", "default-src 'none'")

			// Auth responses must never be cached
			h.Set("Cache-Control", "no-store")
			h.Set("Pragma", "no-cache")

			// Do not send referrer information
			h.Set("Referrer-Policy", "no-referrer")

			// Restrict access to browser features
			h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			// HSTS — only set if the request came over HTTPS (or behind a TLS-terminating proxy)
			if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}
