package security

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// APISecurityConfig configures the API security middleware.
type APISecurityConfig struct {
	MaxBodySize       int64
	RequireJSON       bool
	SanitizeInput     bool
	MaxPerPage        int
	EndpointOverrides map[string]EndpointSecurityConfig
}

// EndpointSecurityConfig provides per-endpoint security overrides.
type EndpointSecurityConfig struct {
	MaxBodySize   int64
	AllowedFields []string
}

// DefaultAPISecurityConfig returns production-safe defaults.
func DefaultAPISecurityConfig() *APISecurityConfig {
	return &APISecurityConfig{
		MaxBodySize:   10 * 1024 * 1024, // 10MB
		RequireJSON:   true,
		SanitizeInput: true,
		MaxPerPage:    100,
	}
}

// APISecurityMiddleware wraps multiple API security checks into a single middleware:
// 1. Request body size limit
// 2. Content-Type validation
// 3. UUID parameter validation
// 4. Pagination limit enforcement
func APISecurityMiddleware(cfg *APISecurityConfig, secLogger *SecurityLogger, logger zerolog.Logger, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Body size limit
			maxBody := cfg.MaxBodySize
			for prefix, override := range cfg.EndpointOverrides {
				if strings.HasPrefix(r.URL.Path, prefix) && override.MaxBodySize > 0 {
					maxBody = override.MaxBodySize
					break
				}
			}
			if r.ContentLength > maxBody {
				secLogger.LogFromRequest(r, EventContentTypeReject, SeverityLow,
					"request body exceeds maximum size", true)
				writeJSONError(w, http.StatusRequestEntityTooLarge, "BODY_TOO_LARGE",
					"request body exceeds maximum allowed size")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)

			// 2. Content-Type validation for state-changing methods
			if cfg.RequireJSON && isStateChangingMethod(r.Method) {
				ct := r.Header.Get("Content-Type")
				if ct != "" && !isAcceptableContentType(ct) {
					secLogger.LogFromRequest(r, EventContentTypeReject, SeverityMedium,
						"invalid content type: "+ct, true)
					if metrics != nil {
						metrics.InvalidContentType.Inc()
					}
					writeJSONError(w, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE",
						"Content-Type must be application/json or multipart/form-data")
					return
				}
			}

			// 3. UUID parameter validation
			rctx := chi.RouteContext(r.Context())
			if rctx != nil {
				for i, key := range rctx.URLParams.Keys {
					if isUUIDParam(key) && i < len(rctx.URLParams.Values) {
						val := rctx.URLParams.Values[i]
						if val != "" {
							if err := ValidateUUID(val); err != nil {
								writeJSONError(w, http.StatusBadRequest, "INVALID_PARAMETER",
									"parameter '"+key+"' must be a valid UUID")
								return
							}
						}
					}
				}
			}

			// 4. Pagination limit enforcement
			if r.Method == http.MethodGet {
				if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
					perPage, err := strconv.Atoi(perPageStr)
					if err != nil || perPage > cfg.MaxPerPage || perPage < 1 {
						writeJSONError(w, http.StatusBadRequest, "INVALID_PAGINATION",
							"per_page must be between 1 and "+strconv.Itoa(cfg.MaxPerPage))
						return
					}
				}
				if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
					limit, err := strconv.Atoi(limitStr)
					if err != nil || limit > cfg.MaxPerPage || limit < 1 {
						writeJSONError(w, http.StatusBadRequest, "INVALID_PAGINATION",
							"limit must be between 1 and "+strconv.Itoa(cfg.MaxPerPage))
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isStateChangingMethod returns true for methods that modify server state.
func isStateChangingMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut ||
		method == http.MethodPatch || method == http.MethodDelete
}

// isAcceptableContentType checks if the content type is safe for API requests.
func isAcceptableContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	return strings.HasPrefix(ct, "application/json") ||
		strings.HasPrefix(ct, "multipart/form-data")
}

// isUUIDParam checks if a URL param name likely represents a UUID.
func isUUIDParam(name string) bool {
	return strings.HasSuffix(name, "ID") || strings.HasSuffix(name, "_id") ||
		strings.HasSuffix(name, "Id") || name == "id"
}

// ContentTypeEnforcement returns middleware that enforces Content-Type on POST/PUT/PATCH.
func ContentTypeEnforcement(secLogger *SecurityLogger, metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isStateChangingMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			ct := r.Header.Get("Content-Type")
			if ct == "" && r.ContentLength > 0 {
				secLogger.LogFromRequest(r, EventContentTypeReject, SeverityLow,
					"missing Content-Type on request with body", true)
				if metrics != nil {
					metrics.InvalidContentType.Inc()
				}
				writeJSONError(w, http.StatusUnsupportedMediaType, "MISSING_CONTENT_TYPE",
					"Content-Type header is required for requests with a body")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
