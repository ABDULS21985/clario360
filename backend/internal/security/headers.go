package security

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// SecurityHeaders returns middleware that sets all required HTTP security headers.
// This middleware MUST be the FIRST middleware after Recovery in the middleware chain.
// Headers are set BEFORE the handler executes — they apply even if the handler panics.
func SecurityHeaders(cfg *SecurityHeadersConfig, logger zerolog.Logger, metrics *Metrics) func(http.Handler) http.Handler {
	csp := cfg.BuildCSP()
	permissionsPolicy := buildPermissionsPolicy()
	isDev := cfg.Environment == "development"

	// Log configured headers at startup
	log := logger.With().Str("component", "security_headers").Logger()
	log.Info().
		Str("environment", cfg.Environment).
		Str("csp", csp).
		Int("hsts_max_age", cfg.HSTSMaxAge).
		Bool("hsts_preload", cfg.HSTSPreload).
		Str("frame_ancestors", cfg.FrameAncestors).
		Bool("coep", cfg.EnableCOEP).
		Bool("coop", cfg.EnableCOOP).
		Bool("corp", cfg.EnableCORP).
		Msg("security headers configured")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// 1. HSTS
			if !isDev {
				if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
					hstsValue := fmt.Sprintf("max-age=%d; includeSubDomains", cfg.HSTSMaxAge)
					if cfg.HSTSPreload {
						hstsValue += "; preload"
					}
					h.Set("Strict-Transport-Security", hstsValue)
				}
			}

			// 2. MIME sniffing prevention
			h.Set("X-Content-Type-Options", "nosniff")

			// 3. XSS Protection (disabled — CSP is the modern layer)
			h.Set("X-XSS-Protection", "0")

			// 4. Clickjacking prevention
			h.Set("X-Frame-Options", "DENY")

			// 5. Content Security Policy
			h.Set("Content-Security-Policy", csp)

			// 6. Referrer Policy
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// 7. Permissions Policy
			h.Set("Permissions-Policy", permissionsPolicy)

			// 8. Cache-Control for API responses
			if strings.HasPrefix(r.URL.Path, "/api/") {
				h.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
				h.Set("Pragma", "no-cache")
				h.Set("Expires", "0")
			} else if cfg.IsStaticAsset(r.URL.Path) && cfg.StaticMaxAge > 0 {
				h.Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cfg.StaticMaxAge))
			}

			// 9. Server identification removal
			h.Del("Server")
			h.Del("X-Powered-By")
			h.Del("X-AspNet-Version")
			h.Del("X-AspNetMvc-Version")

			// 10. Cross-Origin headers
			if cfg.EnableCOEP {
				h.Set("Cross-Origin-Embedder-Policy", "require-corp")
			}
			if cfg.EnableCOOP {
				h.Set("Cross-Origin-Opener-Policy", "same-origin")
			}
			if cfg.EnableCORP {
				h.Set("Cross-Origin-Resource-Policy", "same-origin")
			}

			if metrics != nil {
				metrics.SecurityHeadersApplied.Inc()
			}

			next.ServeHTTP(w, r)
		})
	}
}

// buildPermissionsPolicy constructs the Permissions-Policy header value.
func buildPermissionsPolicy() string {
	policies := []string{
		"camera=()",
		"microphone=()",
		"geolocation=()",
		"payment=()",
		"usb=()",
		"magnetometer=()",
		"gyroscope=()",
		"accelerometer=()",
		"interest-cohort=()",
	}
	return strings.Join(policies, ", ")
}

// CSPReportHandler returns an HTTP handler that receives CSP violation reports.
func CSPReportHandler(logger zerolog.Logger, metrics *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		logger.Warn().
			Str("client_ip", extractClientIP(r)).
			Str("request_id", r.Header.Get("X-Request-ID")).
			Msg("CSP violation report received")

		if metrics != nil {
			metrics.CSPViolations.Inc()
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
