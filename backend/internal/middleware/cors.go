package middleware

import "net/http"

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a sensible default CORS configuration for development.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-Tenant-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

// CORS returns a CORS middleware with the given configuration.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool, len(cfg.AllowedOrigins))
	for _, o := range cfg.AllowedOrigins {
		allowedOrigins[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if allowedOrigins[origin] || containsWildcard(cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			if cfg.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", joinStrings(cfg.AllowedMethods))
				w.Header().Set("Access-Control-Allow-Headers", joinStrings(cfg.AllowedHeaders))
				if cfg.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", intToStr(cfg.MaxAge))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			if len(cfg.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", joinStrings(cfg.ExposedHeaders))
			}

			next.ServeHTTP(w, r)
		})
	}
}

func containsWildcard(origins []string) bool {
	for _, o := range origins {
		if o == "*" {
			return true
		}
	}
	return false
}

func joinStrings(ss []string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
