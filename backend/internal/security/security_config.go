package security

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
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
	VirusScanEnabled  bool
	ClamAVAddr        string // clamd TCP address (default "localhost:3310")
	ClamAVTimeout     time.Duration
	ClamAVMaxScanSize int64

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
		QuarantinePath:    "/tmp/clario360-quarantine",
		VirusScanEnabled:  true,
		ClamAVAddr:        "localhost:3310",
		ClamAVTimeout:     30 * time.Second,
		ClamAVMaxScanSize: 25 * 1024 * 1024,

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

// ClamAVConfigFromConfig extracts ClamAV settings from the central config.
func (c *Config) ClamAVConfigFromConfig() *ClamAVConfig {
	cfg := DefaultClamAVConfig()
	if c.ClamAVAddr != "" {
		cfg.Addr = c.ClamAVAddr
	}
	if c.ClamAVTimeout > 0 {
		cfg.Timeout = c.ClamAVTimeout
	}
	if c.ClamAVMaxScanSize > 0 {
		cfg.MaxSize = c.ClamAVMaxScanSize
	}
	return cfg
}

// ConfigFromEnv loads security configuration from environment variables,
// using DefaultConfig() as the base and overriding with any set env vars.
func ConfigFromEnv() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("SECURITY_ENVIRONMENT"); v != "" {
		cfg.Environment = v
	}

	// CSRF
	if v := os.Getenv("SECURITY_CSRF_COOKIE_NAME"); v != "" {
		cfg.CSRFCookieName = v
	}
	if v := os.Getenv("SECURITY_CSRF_HEADER_NAME"); v != "" {
		cfg.CSRFHeaderName = v
	}
	if v := os.Getenv("SECURITY_CSRF_COOKIE_DOMAIN"); v != "" {
		cfg.CSRFCookieDomain = v
	}
	if v := os.Getenv("SECURITY_CSRF_COOKIE_SECURE"); v != "" {
		cfg.CSRFCookieSecure = parseBool(v, cfg.CSRFCookieSecure)
	}

	// Headers
	if v := os.Getenv("SECURITY_ALLOWED_ORIGINS"); v != "" {
		cfg.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("SECURITY_CSP_REPORT_URI"); v != "" {
		cfg.CSPReportURI = v
	}
	if v := os.Getenv("SECURITY_HSTS_MAX_AGE"); v != "" {
		cfg.HSTSMaxAge = parseInt(v, cfg.HSTSMaxAge)
	}
	if v := os.Getenv("SECURITY_HSTS_PRELOAD"); v != "" {
		cfg.HSTSPreload = parseBool(v, cfg.HSTSPreload)
	}
	if v := os.Getenv("SECURITY_FRAME_ANCESTORS"); v != "" {
		cfg.FrameAncestors = v
	}

	// Rate Limiting — Auth
	if v := os.Getenv("SECURITY_LOGIN_PER_EMAIL"); v != "" {
		cfg.LoginPerEmail = parseInt(v, cfg.LoginPerEmail)
	}
	if v := os.Getenv("SECURITY_LOGIN_PER_IP"); v != "" {
		cfg.LoginPerIP = parseInt(v, cfg.LoginPerIP)
	}
	if v := os.Getenv("SECURITY_LOGIN_WINDOW"); v != "" {
		cfg.LoginWindow = parseDuration(v, cfg.LoginWindow)
	}
	if v := os.Getenv("SECURITY_LOCKOUT_THRESHOLD"); v != "" {
		cfg.LockoutThreshold = parseInt(v, cfg.LockoutThreshold)
	}
	if v := os.Getenv("SECURITY_LOCKOUT_DURATION"); v != "" {
		cfg.LockoutDuration = parseDuration(v, cfg.LockoutDuration)
	}
	if v := os.Getenv("SECURITY_ESCALATION_THRESHOLD"); v != "" {
		cfg.EscalationThreshold = parseInt(v, cfg.EscalationThreshold)
	}

	// Rate Limiting — API
	if v := os.Getenv("SECURITY_API_RATE_PER_MINUTE"); v != "" {
		cfg.APIDefaultPerMinute = parseInt(v, cfg.APIDefaultPerMinute)
	}
	if v := os.Getenv("SECURITY_API_BURST_MULTIPLIER"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.APIBurstMultiplier = f
		}
	}

	// Session
	if v := os.Getenv("SECURITY_SESSION_IDLE_TIMEOUT"); v != "" {
		cfg.SessionIdleTimeout = parseDuration(v, cfg.SessionIdleTimeout)
	}
	if v := os.Getenv("SECURITY_SESSION_ABSOLUTE_MAX"); v != "" {
		cfg.SessionAbsoluteMax = parseDuration(v, cfg.SessionAbsoluteMax)
	}
	if v := os.Getenv("SECURITY_MAX_CONCURRENT_SESSIONS"); v != "" {
		cfg.MaxConcurrentSessions = parseInt(v, cfg.MaxConcurrentSessions)
	}

	// File Upload
	if v := os.Getenv("SECURITY_MAX_UPLOAD_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.MaxUploadSize = n
		}
	}
	if v := os.Getenv("SECURITY_QUARANTINE_PATH"); v != "" {
		cfg.QuarantinePath = v
	}
	if v := os.Getenv("SECURITY_VIRUS_SCAN_ENABLED"); v != "" {
		cfg.VirusScanEnabled = parseBool(v, cfg.VirusScanEnabled)
	}
	if v := os.Getenv("SECURITY_CLAMAV_ADDR"); v != "" {
		cfg.ClamAVAddr = v
	}
	if v := os.Getenv("SECURITY_CLAMAV_TIMEOUT"); v != "" {
		cfg.ClamAVTimeout = parseDuration(v, cfg.ClamAVTimeout)
	}

	// Sanitizer
	if v := os.Getenv("SECURITY_MAX_STRING_LENGTH"); v != "" {
		cfg.MaxStringLength = parseInt(v, cfg.MaxStringLength)
	}
	if v := os.Getenv("SECURITY_MAX_JSON_DEPTH"); v != "" {
		cfg.MaxJSONDepth = parseInt(v, cfg.MaxJSONDepth)
	}
	if v := os.Getenv("SECURITY_MAX_JSON_SIZE"); v != "" {
		cfg.MaxJSONSize = parseInt(v, cfg.MaxJSONSize)
	}

	// SSRF
	if v := os.Getenv("SECURITY_SSRF_ALLOWED_HOSTS"); v != "" {
		cfg.SSRFAllowedHosts = strings.Split(v, ",")
	}
	if v := os.Getenv("SECURITY_SSRF_BLOCK_PRIVATE"); v != "" {
		cfg.SSRFBlockPrivate = parseBool(v, cfg.SSRFBlockPrivate)
	}

	// Logging
	if v := os.Getenv("SECURITY_LOG_PATH"); v != "" {
		cfg.SecurityLogPath = v
	}
	if v := os.Getenv("SECURITY_TAMPER_PROOF"); v != "" {
		cfg.EnableTamperProof = parseBool(v, cfg.EnableTamperProof)
	}

	return cfg
}

func parseInt(s string, fallback int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}

func parseBool(s string, fallback bool) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return fallback
}

func parseDuration(s string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return fallback
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
