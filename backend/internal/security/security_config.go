package security

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config is the central security configuration for all security middleware.
// Loaded from environment variables / Vault at service startup.
type Config struct {
	Environment string // "development", "staging", "production"

	// CSRF
	CSRFCookieName   string
	CSRFHeaderName   string
	CSRFCookieDomain string
	CSRFCookieSecure bool
	CSRFMaxAge       int // seconds

	// Headers
	AllowedOrigins      []string
	CSPReportURI        string
	HSTSMaxAge          int
	HSTSPreload         bool
	FrameAncestors      string
	CustomCSPDirectives map[string]string
	EnableCOEP          bool
	EnableCOOP          bool
	EnableCORP          bool

	// Rate Limiting — Auth
	LoginPerEmail         int
	LoginPerIP            int
	LoginWindow           time.Duration
	RegisterPerIP         int
	RegisterWindow        time.Duration
	PasswordResetPerEmail int
	PasswordResetPerIP    int
	PasswordResetWindow   time.Duration
	MFAPerSession         int
	MFAWindow             time.Duration
	LockoutThreshold      int
	LockoutDuration       time.Duration
	EscalationThreshold   int
	EscalationWindow      time.Duration

	// Rate Limiting — API
	APIDefaultPerMinute int
	APIBurstMultiplier  float64

	// Session
	SessionIdleTimeout    time.Duration
	SessionAbsoluteMax    time.Duration
	MaxConcurrentSessions int

	// File Upload
	MaxUploadSize       int64    // bytes
	AllowedMIMETypes    []string // e.g., "application/pdf", "image/png"
	AllowedExtensions   []string // e.g., ".pdf", ".png"
	QuarantinePath      string
	VirusScanEnabled    bool
	VirusScanEndpoint   string

	// Sanitizer
	MaxStringLength   int
	MaxJSONDepth      int
	MaxJSONSize       int
	MaxFilenameLength int

	// SSRF
	SSRFAllowedHosts []string
	SSRFBlockPrivate bool

	// Logging
	SecurityLogPath   string
	EnableTamperProof bool
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() *Config {
	return &Config{
		Environment: "production",

		// CSRF
		CSRFCookieName:   "clario360_csrf",
		CSRFHeaderName:   "X-CSRF-Token",
		CSRFCookieSecure: true,
		CSRFMaxAge:       86400,

		// Headers
		HSTSMaxAge:     31536000,
		HSTSPreload:    true,
		FrameAncestors: "'none'",
		EnableCOEP:     true,
		EnableCOOP:     true,
		EnableCORP:     true,

		// Rate Limiting — Auth
		LoginPerEmail:         5,
		LoginPerIP:            20,
		LoginWindow:           15 * time.Minute,
		RegisterPerIP:         3,
		RegisterWindow:        time.Hour,
		PasswordResetPerEmail: 3,
		PasswordResetPerIP:    10,
		PasswordResetWindow:   time.Hour,
		MFAPerSession:         5,
		MFAWindow:             15 * time.Minute,
		LockoutThreshold:      5,
		LockoutDuration:       15 * time.Minute,
		EscalationThreshold:   20,
		EscalationWindow:      time.Hour,

		// Rate Limiting — API
		APIDefaultPerMinute: 100,
		APIBurstMultiplier:  2.0,

		// Session
		SessionIdleTimeout:    30 * time.Minute,
		SessionAbsoluteMax:    24 * time.Hour,
		MaxConcurrentSessions: 5,

		// File Upload
		MaxUploadSize:    50 * 1024 * 1024, // 50MB
		AllowedMIMETypes: []string{"application/pdf", "image/png", "image/jpeg", "image/gif", "text/csv", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		AllowedExtensions: []string{".pdf", ".png", ".jpg", ".jpeg", ".gif", ".csv", ".xlsx", ".docx"},
		QuarantinePath:   "/tmp/clario360-quarantine",

		// Sanitizer
		MaxStringLength:   10000,
		MaxJSONDepth:      10,
		MaxJSONSize:       1 * 1024 * 1024, // 1MB
		MaxFilenameLength: 255,

		// SSRF
		SSRFBlockPrivate: true,
	}
}

// DevelopmentConfig returns development-friendly defaults.
func DevelopmentConfig() *Config {
	cfg := DefaultConfig()
	cfg.Environment = "development"
	cfg.CSRFCookieSecure = false
	cfg.HSTSMaxAge = 0
	cfg.HSTSPreload = false
	cfg.EnableCOEP = false // Allow cross-origin in dev for HMR
	cfg.LoginPerEmail = 50
	cfg.LoginPerIP = 200
	cfg.RegisterPerIP = 50
	cfg.LockoutThreshold = 50
	cfg.EscalationThreshold = 100
	cfg.MaxConcurrentSessions = 20
	return cfg
}

// Validate checks config consistency and returns errors for invalid settings.
func (c *Config) Validate() error {
	if c.Environment == "" {
		return fmt.Errorf("security: environment must be set")
	}
	if c.Environment != "development" && c.Environment != "staging" && c.Environment != "production" {
		return fmt.Errorf("security: invalid environment %q", c.Environment)
	}

	if c.Environment == "production" {
		if c.HSTSMaxAge <= 0 {
			return fmt.Errorf("security: HSTSMaxAge must be > 0 in production")
		}
		for _, origin := range c.AllowedOrigins {
			if origin == "*" {
				return fmt.Errorf("security: wildcard origin not allowed in production")
			}
		}
		if !c.CSRFCookieSecure {
			return fmt.Errorf("security: CSRF cookie must be secure in production")
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

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsDevelopment returns true if running in development environment.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// BuildCSP assembles the full Content-Security-Policy string.
func (c *Config) BuildCSP() string {
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

	if c.IsDevelopment() {
		directives["script-src"] = "'self' 'unsafe-eval'"
	} else {
		directives["upgrade-insecure-requests"] = ""
	}

	// Apply custom overrides
	for k, v := range c.CustomCSPDirectives {
		if strings.ContainsAny(k, ";\n\r") || strings.ContainsAny(v, ";\n\r") {
			continue // Skip directives that could cause injection
		}
		directives[k] = v
	}

	var parts []string
	for directive, value := range directives {
		if value == "" {
			parts = append(parts, directive)
		} else {
			parts = append(parts, directive+" "+value)
		}
	}

	csp := strings.Join(parts, "; ")

	if c.CSPReportURI != "" && !c.IsDevelopment() {
		csp += "; report-uri " + c.CSPReportURI
	}

	return csp
}
