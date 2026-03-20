package security

import (
	"crypto/hmac"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

// CSRFConfig configures CSRF protection middleware.
type CSRFConfig struct {
	CookieName     string
	HeaderName     string
	CookieDomain   string
	CookieSecure   bool
	CookieSameSite http.SameSite
	MaxAge         int      // seconds
	ExemptPaths    []string // paths exempt from CSRF (webhooks, health)
	ExemptMethods  []string // default: GET, HEAD, OPTIONS
}

// DefaultCSRFConfig returns sensible defaults for CSRF protection.
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		CookieName:     "clario360_csrf",
		HeaderName:     "X-CSRF-Token",
		CookieSecure:   true,
		CookieSameSite: http.SameSiteStrictMode,
		MaxAge:         86400,
		ExemptMethods:  []string{http.MethodGet, http.MethodHead, http.MethodOptions},
		ExemptPaths: []string{
			"/api/v1/webhooks/",
			"/api/v1/health",
			"/healthz",
			"/readyz",
		},
	}
}

// CSRFProtection returns middleware implementing the Double-Submit Cookie pattern.
func CSRFProtection(cfg *CSRFConfig, secLogger *SecurityLogger, logger zerolog.Logger, metrics *Metrics) func(http.Handler) http.Handler {
	exemptMethods := make(map[string]bool, len(cfg.ExemptMethods))
	for _, m := range cfg.ExemptMethods {
		exemptMethods[strings.ToUpper(m)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check exempt methods
			if exemptMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Check exempt paths
			for _, path := range cfg.ExemptPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 3. Check if API key authentication (not cookie-based)
			if r.Header.Get("Authorization") != "" && !hasCookieAuth(r, cfg.CookieName) {
				// API key or Bearer token without CSRF cookie — likely programmatic access
				next.ServeHTTP(w, r)
				return
			}

			// 4. Extract CSRF cookie
			cookie, err := r.Cookie(cfg.CookieName)
			if err != nil {
				secLogger.LogFromRequest(r, EventCSRFFailure, SeverityMedium,
					"CSRF cookie missing", true)
				if metrics != nil {
					metrics.CSRFFailures.WithLabelValues("cookie_missing").Inc()
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "CSRF_MISSING",
					"message": "CSRF token cookie not found. Please refresh the page.",
				})
				return
			}

			// 5. Extract CSRF header
			headerToken := r.Header.Get(cfg.HeaderName)
			if headerToken == "" {
				secLogger.LogFromRequest(r, EventCSRFFailure, SeverityMedium,
					"CSRF header missing", true)
				if metrics != nil {
					metrics.CSRFFailures.WithLabelValues("header_missing").Inc()
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "CSRF_HEADER_MISSING",
					"message": "X-CSRF-Token header is required for this request.",
				})
				return
			}

			// 6. Constant-time comparison
			if !hmac.Equal([]byte(cookie.Value), []byte(headerToken)) {
				secLogger.LogFromRequest(r, EventCSRFFailure, SeverityHigh,
					"CSRF token mismatch", true)
				if metrics != nil {
					metrics.CSRFFailures.WithLabelValues("mismatch").Inc()
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code":    "CSRF_INVALID",
					"message": "CSRF token validation failed. Please refresh the page and try again.",
				})
				return
			}

			// 7. Pass through
			next.ServeHTTP(w, r)
		})
	}
}

// hasCookieAuth checks if the request has the CSRF cookie (indicating browser-based auth).
func hasCookieAuth(r *http.Request, cookieName string) bool {
	_, err := r.Cookie(cookieName)
	return err == nil
}
