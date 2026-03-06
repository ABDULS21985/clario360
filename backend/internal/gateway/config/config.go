package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GatewayConfig holds all API gateway configuration loaded from GW_* environment variables.
type GatewayConfig struct {
	// HTTP server
	HTTPPort        int
	AdminPort       int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	MaxRequestBodyMB int
	ProxyTimeout    time.Duration

	// Security
	CORSAllowedOrigins []string
	Environment        string // "production", "development", "staging"

	// Rate limits (requests per minute)
	RateLimitAuthPerMin   int
	RateLimitReadPerMin   int
	RateLimitWritePerMin  int
	RateLimitAdminPerMin  int
	RateLimitUploadPerMin int
	RateLimitWSPerMin     int

	// Circuit breaker
	CBMaxRequests      int
	CBIntervalSec      int
	CBTimeoutSec       int
	CBFailureThreshold int

	// Logging
	LogLevel string
}

// Load reads the gateway configuration from environment variables.
// Required: none (all have defaults). Validation fails on invalid values.
func Load() (*GatewayConfig, error) {
	cfg := &GatewayConfig{
		HTTPPort:              intEnv("GW_HTTP_PORT", 8080),
		AdminPort:             intEnv("GW_ADMIN_PORT", 9090),
		ReadTimeout:           durationSecEnv("GW_READ_TIMEOUT_SEC", 30),
		WriteTimeout:          durationSecEnv("GW_WRITE_TIMEOUT_SEC", 60),
		IdleTimeout:           durationSecEnv("GW_IDLE_TIMEOUT_SEC", 120),
		MaxRequestBodyMB:      intEnv("GW_MAX_REQUEST_BODY_MB", 10),
		ProxyTimeout:          durationSecEnv("GW_PROXY_TIMEOUT_SEC", 30),
		Environment:           strEnv("GW_ENVIRONMENT", "production"),
		LogLevel:              strEnv("GW_LOG_LEVEL", "info"),
		RateLimitAuthPerMin:   intEnv("GW_RATELIMIT_AUTH_PER_MIN", 20),
		RateLimitReadPerMin:   intEnv("GW_RATELIMIT_READ_PER_MIN", 2000),
		RateLimitWritePerMin:  intEnv("GW_RATELIMIT_WRITE_PER_MIN", 500),
		RateLimitAdminPerMin:  intEnv("GW_RATELIMIT_ADMIN_PER_MIN", 100),
		RateLimitUploadPerMin: intEnv("GW_RATELIMIT_UPLOAD_PER_MIN", 50),
		RateLimitWSPerMin:     intEnv("GW_RATELIMIT_WS_PER_MIN", 10),
		CBMaxRequests:         intEnv("GW_CB_MAX_REQUESTS", 5),
		CBIntervalSec:         intEnv("GW_CB_INTERVAL_SEC", 60),
		CBTimeoutSec:          intEnv("GW_CB_TIMEOUT_SEC", 30),
		CBFailureThreshold:    intEnv("GW_CB_FAILURE_THRESHOLD", 5),
	}

	// CORS origins
	originsRaw := strEnv("GW_CORS_ALLOWED_ORIGINS", "https://*.clario360.com,http://localhost:3000")
	for _, o := range strings.Split(originsRaw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			cfg.CORSAllowedOrigins = append(cfg.CORSAllowedOrigins, o)
		}
	}

	return cfg, cfg.Validate()
}

// Validate checks that all config values are within acceptable ranges.
func (c *GatewayConfig) Validate() error {
	var errs []string

	// In production, reject wildcard CORS origin.
	if strings.EqualFold(c.Environment, "production") {
		for _, o := range c.CORSAllowedOrigins {
			if o == "*" {
				errs = append(errs, "GW_CORS_ALLOWED_ORIGINS: wildcard '*' is not allowed in production")
			}
		}
	}

	// Rate limits must be ≥ 1.
	rateLimits := map[string]int{
		"GW_RATELIMIT_AUTH_PER_MIN":   c.RateLimitAuthPerMin,
		"GW_RATELIMIT_READ_PER_MIN":   c.RateLimitReadPerMin,
		"GW_RATELIMIT_WRITE_PER_MIN":  c.RateLimitWritePerMin,
		"GW_RATELIMIT_ADMIN_PER_MIN":  c.RateLimitAdminPerMin,
		"GW_RATELIMIT_UPLOAD_PER_MIN": c.RateLimitUploadPerMin,
		"GW_RATELIMIT_WS_PER_MIN":     c.RateLimitWSPerMin,
	}
	for k, v := range rateLimits {
		if v < 1 {
			errs = append(errs, fmt.Sprintf("%s must be >= 1, got %d", k, v))
		}
	}

	// Timeout ordering: read < write.
	if c.ReadTimeout >= c.WriteTimeout {
		errs = append(errs, "GW_READ_TIMEOUT_SEC must be less than GW_WRITE_TIMEOUT_SEC")
	}
	// Proxy timeout < write timeout.
	if c.ProxyTimeout >= c.WriteTimeout {
		errs = append(errs, "GW_PROXY_TIMEOUT_SEC must be less than GW_WRITE_TIMEOUT_SEC")
	}

	if len(errs) > 0 {
		return fmt.Errorf("gateway config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// IsProduction returns true when ENVIRONMENT is "production".
func (c *GatewayConfig) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

func intEnv(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func durationSecEnv(key string, defSec int) time.Duration {
	return time.Duration(intEnv(key, defSec)) * time.Second
}

func strEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
