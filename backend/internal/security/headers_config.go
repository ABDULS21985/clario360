package security

import (
	"fmt"
	"net/url"
	"strings"
)

// SecurityHeadersConfig provides per-environment header policy configuration.
type SecurityHeadersConfig struct {
	Environment       string
	AllowedOrigins    []string
	CSPReportURI      string
	HSTSMaxAge        int
	HSTSPreload       bool
	FrameAncestors    string
	CustomCSPDirectives map[string]string
	EnableCOEP        bool // Cross-Origin-Embedder-Policy
	EnableCOOP        bool // Cross-Origin-Opener-Policy
	EnableCORP        bool // Cross-Origin-Resource-Policy
	StaticAssetPaths  []string
	StaticMaxAge      int // seconds
}

// DefaultProductionHeadersConfig returns production-safe header defaults.
func DefaultProductionHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Environment:    "production",
		HSTSMaxAge:     31536000, // 1 year
		HSTSPreload:    true,
		FrameAncestors: "'none'",
		EnableCOEP:     true,
		EnableCOOP:     true,
		EnableCORP:     true,
		StaticMaxAge:   86400,
	}
}

// DefaultDevelopmentHeadersConfig returns development-friendly header defaults.
func DefaultDevelopmentHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Environment:    "development",
		HSTSMaxAge:     0,
		HSTSPreload:    false,
		FrameAncestors: "'none'",
		EnableCOEP:     false,
		EnableCOOP:     false,
		EnableCORP:     false,
	}
}

// HeadersConfigFromConfig derives a SecurityHeadersConfig from the central Config.
func HeadersConfigFromConfig(cfg *Config) *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Environment:         cfg.Environment,
		AllowedOrigins:      cfg.AllowedOrigins,
		CSPReportURI:        cfg.CSPReportURI,
		HSTSMaxAge:          cfg.HSTSMaxAge,
		HSTSPreload:         cfg.HSTSPreload,
		FrameAncestors:      cfg.FrameAncestors,
		CustomCSPDirectives: cfg.CustomCSPDirectives,
		EnableCOEP:          cfg.EnableCOEP,
		EnableCOOP:          cfg.EnableCOOP,
		EnableCORP:          cfg.EnableCORP,
	}
}

// BuildCSP assembles the full CSP string from config fields + custom directives.
func (c *SecurityHeadersConfig) BuildCSP() string {
	directives := map[string]string{
		"default-src":     "'self'",
		"script-src":      "'self'",
		"style-src":       "'self' 'unsafe-inline'",
		"img-src":         "'self' data: https:",
		"font-src":        "'self'",
		"connect-src":     "'self' wss:",
		"worker-src":      "'self' blob:",
		"frame-ancestors": c.FrameAncestors,
		"base-uri":        "'self'",
		"form-action":     "'self'",
		"object-src":      "'none'",
	}

	if c.Environment == "development" {
		directives["script-src"] = "'self' 'unsafe-eval'"
	} else {
		directives["upgrade-insecure-requests"] = ""
	}

	for k, v := range c.CustomCSPDirectives {
		if strings.ContainsAny(k, ";\n\r") || strings.ContainsAny(v, ";\n\r") {
			continue
		}
		directives[k] = v
	}

	var parts []string
	// Use a deterministic order for testability
	orderedKeys := []string{
		"default-src", "script-src", "style-src", "img-src", "font-src",
		"connect-src", "worker-src", "frame-ancestors", "base-uri",
		"form-action", "object-src", "upgrade-insecure-requests",
	}
	seen := make(map[string]bool)
	for _, key := range orderedKeys {
		if value, ok := directives[key]; ok {
			seen[key] = true
			if value == "" {
				parts = append(parts, key)
			} else {
				parts = append(parts, key+" "+value)
			}
		}
	}
	// Add any custom directives not in the ordered list
	for key, value := range directives {
		if seen[key] {
			continue
		}
		if value == "" {
			parts = append(parts, key)
		} else {
			parts = append(parts, key+" "+value)
		}
	}

	csp := strings.Join(parts, "; ")

	if c.CSPReportURI != "" && c.Environment != "development" {
		csp += "; report-uri " + c.CSPReportURI
	}

	return csp
}

// ValidateHeadersConfig checks config consistency.
func (c *SecurityHeadersConfig) ValidateHeadersConfig() error {
	if c.Environment == "production" {
		if c.HSTSMaxAge <= 0 {
			return fmt.Errorf("security: HSTSMaxAge must be > 0 in production")
		}
		for _, origin := range c.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("security: wildcard origin not allowed in production")
			}
		}
	}

	if c.CSPReportURI != "" {
		if _, err := url.Parse(c.CSPReportURI); err != nil {
			return fmt.Errorf("security: invalid CSPReportURI: %w", err)
		}
	}

	if c.FrameAncestors == "" {
		return fmt.Errorf("security: FrameAncestors must not be empty")
	}

	return nil
}

// IsStaticAsset returns true if the path matches a known static asset pattern.
func (c *SecurityHeadersConfig) IsStaticAsset(path string) bool {
	for _, prefix := range c.StaticAssetPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
