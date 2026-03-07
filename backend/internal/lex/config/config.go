package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	appconfig "github.com/clario360/platform/internal/config"
)

type Config struct {
	HTTPPort                  int
	AdminPort                 int
	DBURL                     string
	DBMinConns                int
	DBMaxConns                int
	RedisAddr                 string
	RedisPassword             string
	RedisDB                   int
	KafkaBrokers              []string
	KafkaGroupID              string
	KafkaTopic                string
	JWTPublicKeyPath          string
	RateLimitPerMinute        int
	DashboardCacheTTL         time.Duration
	ExpiryMonitorInterval     time.Duration
	ComplianceMonitorInterval time.Duration
	RenewalReminderInterval   time.Duration
	SeedDemoData              bool
	OrgJurisdiction           string
}

func Default() *Config {
	return &Config{
		HTTPPort:                  8087,
		AdminPort:                 9087,
		DBMinConns:                5,
		DBMaxConns:                20,
		RedisAddr:                 "localhost:6379",
		RedisDB:                   0,
		KafkaGroupID:              "lex-service",
		KafkaTopic:                "enterprise.lex.events",
		RateLimitPerMinute:        300,
		DashboardCacheTTL:         60 * time.Second,
		ExpiryMonitorInterval:     time.Hour,
		ComplianceMonitorInterval: 6 * time.Hour,
		RenewalReminderInterval:   6 * time.Hour,
		OrgJurisdiction:           "Saudi Arabia",
	}
}

func Load(base *appconfig.Config) (*Config, error) {
	if base == nil {
		return nil, fmt.Errorf("base config is required")
	}

	cfg := Default()
	cfg.HTTPPort = envInt("LEX_HTTP_PORT", cfg.HTTPPort)
	cfg.AdminPort = envInt("LEX_ADMIN_PORT", cfg.AdminPort)
	cfg.DBURL = envOr("LEX_DB_URL", buildDBURL(base, "lex_db"))
	cfg.DBMinConns = envInt("LEX_DB_MIN_CONNS", max(base.Database.MaxIdleConns, cfg.DBMinConns))
	cfg.DBMaxConns = envInt("LEX_DB_MAX_CONNS", max(base.Database.MaxOpenConns, cfg.DBMaxConns))
	cfg.RedisAddr = envOr("LEX_REDIS_ADDR", base.Redis.Addr())
	cfg.RedisPassword = envOr("LEX_REDIS_PASSWORD", base.Redis.Password)
	cfg.RedisDB = envInt("LEX_REDIS_DB", base.Redis.DB)
	cfg.KafkaBrokers = splitCSV(envOr("LEX_KAFKA_BROKERS", strings.Join(base.Kafka.Brokers, ",")))
	cfg.KafkaGroupID = envOr("LEX_KAFKA_GROUP_ID", cfg.KafkaGroupID)
	cfg.KafkaTopic = envOr("LEX_KAFKA_TOPIC", cfg.KafkaTopic)
	cfg.JWTPublicKeyPath = os.Getenv("LEX_JWT_PUBLIC_KEY_PATH")
	cfg.RateLimitPerMinute = envInt("LEX_RATE_LIMIT_PER_MINUTE", cfg.RateLimitPerMinute)
	cfg.DashboardCacheTTL = envDuration("LEX_DASHBOARD_CACHE_TTL", cfg.DashboardCacheTTL)
	cfg.ExpiryMonitorInterval = envDuration("LEX_EXPIRY_MONITOR_INTERVAL", cfg.ExpiryMonitorInterval)
	cfg.ComplianceMonitorInterval = envDuration("LEX_COMPLIANCE_MONITOR_INTERVAL", cfg.ComplianceMonitorInterval)
	cfg.RenewalReminderInterval = envDuration("LEX_RENEWAL_REMINDER_INTERVAL", cfg.RenewalReminderInterval)
	cfg.SeedDemoData = envBool("LEX_SEED_DEMO_DATA", false)
	cfg.OrgJurisdiction = envOr("LEX_ORG_JURISDICTION", cfg.OrgJurisdiction)

	if _, err := url.Parse(cfg.DBURL); err != nil {
		return nil, fmt.Errorf("LEX_DB_URL is invalid: %w", err)
	}
	if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
		return nil, fmt.Errorf("LEX_HTTP_PORT must be between 1 and 65535")
	}
	if cfg.AdminPort < 1 || cfg.AdminPort > 65535 {
		return nil, fmt.Errorf("LEX_ADMIN_PORT must be between 1 and 65535")
	}
	if cfg.DBMinConns < 1 {
		return nil, fmt.Errorf("LEX_DB_MIN_CONNS must be >= 1")
	}
	if cfg.DBMaxConns < cfg.DBMinConns {
		return nil, fmt.Errorf("LEX_DB_MAX_CONNS must be >= LEX_DB_MIN_CONNS")
	}
	if cfg.RateLimitPerMinute < 1 {
		return nil, fmt.Errorf("LEX_RATE_LIMIT_PER_MINUTE must be >= 1")
	}
	if cfg.DashboardCacheTTL <= 0 {
		return nil, fmt.Errorf("LEX_DASHBOARD_CACHE_TTL must be > 0")
	}
	if cfg.ExpiryMonitorInterval <= 0 || cfg.ComplianceMonitorInterval <= 0 || cfg.RenewalReminderInterval <= 0 {
		return nil, fmt.Errorf("scheduler intervals must be > 0")
	}
	if cfg.JWTPublicKeyPath != "" {
		if _, err := os.Stat(cfg.JWTPublicKeyPath); err != nil {
			return nil, fmt.Errorf("LEX_JWT_PUBLIC_KEY_PATH: %w", err)
		}
	}
	return cfg, nil
}

func buildDBURL(base *appconfig.Config, dbName string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		base.Database.User,
		base.Database.Password,
		base.Database.Host,
		base.Database.Port,
		dbName,
		base.Database.SSLMode,
	)
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			return parsed
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			return parsed
		}
	}
	return fallback
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
