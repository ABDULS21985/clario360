package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds audit-service specific configuration.
type Config struct {
	HTTPPort            int
	BatchSize           int
	BatchWindowMs       int
	DBMinConns          int
	DBMaxConns          int
	ExportAsyncThreshold int
	MinIOEndpoint       string
	MinIOBucket         string
	MinIOAccessKey      string
	MinIOSecretKey      string
	MinIOUseSSL         bool
	RateLimitPerMinute  int
	BatchWindow         time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		HTTPPort:             8084,
		BatchSize:            500,
		BatchWindowMs:        500,
		DBMinConns:           5,
		DBMaxConns:           20,
		ExportAsyncThreshold: 100000,
		MinIOEndpoint:        "minio:9000",
		MinIOBucket:          "audit-exports",
		RateLimitPerMinute:   100,
	}
}

// LoadFromEnv overlays environment variable values onto the default config.
func LoadFromEnv() *Config {
	cfg := DefaultConfig()

	if v := os.Getenv("AUDIT_HTTP_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.HTTPPort = n
		}
	}
	if v := os.Getenv("AUDIT_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.BatchSize = n
		}
	}
	if v := os.Getenv("AUDIT_BATCH_WINDOW_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.BatchWindowMs = n
		}
	}
	if v := os.Getenv("AUDIT_DB_MIN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DBMinConns = n
		}
	}
	if v := os.Getenv("AUDIT_DB_MAX_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DBMaxConns = n
		}
	}
	if v := os.Getenv("AUDIT_EXPORT_ASYNC_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.ExportAsyncThreshold = n
		}
	}
	if v := os.Getenv("AUDIT_MINIO_ENDPOINT"); v != "" {
		cfg.MinIOEndpoint = v
	}
	if v := os.Getenv("AUDIT_MINIO_BUCKET"); v != "" {
		cfg.MinIOBucket = v
	}
	if v := os.Getenv("AUDIT_MINIO_ACCESS_KEY"); v != "" {
		cfg.MinIOAccessKey = v
	}
	if v := os.Getenv("AUDIT_MINIO_SECRET_KEY"); v != "" {
		cfg.MinIOSecretKey = v
	}
	if v := os.Getenv("AUDIT_MINIO_USE_SSL"); strings.EqualFold(v, "true") {
		cfg.MinIOUseSSL = true
	}
	if v := os.Getenv("AUDIT_RATELIMIT_PER_MINUTE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RateLimitPerMinute = n
		}
	}

	cfg.BatchWindow = time.Duration(cfg.BatchWindowMs) * time.Millisecond

	return cfg
}

// Validate checks that configuration values are within acceptable bounds.
func (c *Config) Validate() error {
	if c.BatchSize < 1 || c.BatchSize > 2000 {
		return fmt.Errorf("batch_size must be between 1 and 2000, got %d", c.BatchSize)
	}
	if c.BatchWindowMs < 100 || c.BatchWindowMs > 5000 {
		return fmt.Errorf("batch_window_ms must be between 100 and 5000, got %d", c.BatchWindowMs)
	}
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("http_port must be between 1 and 65535, got %d", c.HTTPPort)
	}
	if c.DBMinConns < 1 {
		return fmt.Errorf("db_min_conns must be >= 1, got %d", c.DBMinConns)
	}
	if c.DBMaxConns < c.DBMinConns {
		return fmt.Errorf("db_max_conns (%d) must be >= db_min_conns (%d)", c.DBMaxConns, c.DBMinConns)
	}
	c.BatchWindow = time.Duration(c.BatchWindowMs) * time.Millisecond
	return nil
}
