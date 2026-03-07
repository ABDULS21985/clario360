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

// Config contains acta-service specific settings layered on top of the shared platform config.
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
	OverdueCheckInterval      time.Duration
	MeetingReminderInterval   time.Duration
	ComplianceCheckInterval   time.Duration
	ComplianceCheckHourUTC    int
	SeedDemoData              bool
}

func Load(base *appconfig.Config) (*Config, error) {
	if base == nil {
		return nil, fmt.Errorf("base config is required")
	}

	cfg := &Config{
		HTTPPort:                envInt("ACTA_HTTP_PORT", 8086),
		AdminPort:               envInt("ACTA_ADMIN_PORT", 9086),
		DBURL:                   envOr("ACTA_DB_URL", buildDBURL(base, "acta_db")),
		DBMinConns:              envInt("ACTA_DB_MIN_CONNS", max(base.Database.MaxIdleConns, 5)),
		DBMaxConns:              envInt("ACTA_DB_MAX_CONNS", max(base.Database.MaxOpenConns, 20)),
		RedisAddr:               envOr("ACTA_REDIS_ADDR", base.Redis.Addr()),
		RedisPassword:           envOr("ACTA_REDIS_PASSWORD", base.Redis.Password),
		RedisDB:                 envInt("ACTA_REDIS_DB", base.Redis.DB),
		KafkaBrokers:            splitCSV(envOr("ACTA_KAFKA_BROKERS", strings.Join(base.Kafka.Brokers, ","))),
		KafkaGroupID:            envOr("ACTA_KAFKA_GROUP_ID", "acta-service"),
		KafkaTopic:              envOr("ACTA_KAFKA_TOPIC", "enterprise.acta.events"),
		JWTPublicKeyPath:        os.Getenv("ACTA_JWT_PUBLIC_KEY_PATH"),
		RateLimitPerMinute:      envInt("ACTA_RATE_LIMIT_PER_MINUTE", 300),
		DashboardCacheTTL:       envDuration("ACTA_DASHBOARD_CACHE_TTL", 60*time.Second),
		OverdueCheckInterval:    envDuration("ACTA_OVERDUE_CHECK_INTERVAL", time.Hour),
		MeetingReminderInterval: envDuration("ACTA_MEETING_REMINDER_INTERVAL", 15*time.Minute),
		ComplianceCheckInterval: envDuration("ACTA_COMPLIANCE_CHECK_INTERVAL", 24*time.Hour),
		ComplianceCheckHourUTC:  envInt("ACTA_COMPLIANCE_CHECK_HOUR_UTC", 6),
		SeedDemoData:            envBool("ACTA_SEED_DEMO_DATA", false),
	}

	if _, err := url.Parse(cfg.DBURL); err != nil {
		return nil, fmt.Errorf("ACTA_DB_URL is invalid: %w", err)
	}
	if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
		return nil, fmt.Errorf("ACTA_HTTP_PORT must be between 1 and 65535")
	}
	if cfg.AdminPort < 1 || cfg.AdminPort > 65535 {
		return nil, fmt.Errorf("ACTA_ADMIN_PORT must be between 1 and 65535")
	}
	if cfg.DBMinConns < 1 {
		return nil, fmt.Errorf("ACTA_DB_MIN_CONNS must be >= 1")
	}
	if cfg.DBMaxConns < cfg.DBMinConns {
		return nil, fmt.Errorf("ACTA_DB_MAX_CONNS must be >= ACTA_DB_MIN_CONNS")
	}
	if cfg.RateLimitPerMinute < 1 {
		return nil, fmt.Errorf("ACTA_RATE_LIMIT_PER_MINUTE must be >= 1")
	}
	if cfg.DashboardCacheTTL <= 0 {
		return nil, fmt.Errorf("ACTA_DASHBOARD_CACHE_TTL must be > 0")
	}
	if cfg.OverdueCheckInterval <= 0 || cfg.MeetingReminderInterval <= 0 || cfg.ComplianceCheckInterval <= 0 {
		return nil, fmt.Errorf("scheduler intervals must be > 0")
	}
	if cfg.ComplianceCheckHourUTC < 0 || cfg.ComplianceCheckHourUTC > 23 {
		return nil, fmt.Errorf("ACTA_COMPLIANCE_CHECK_HOUR_UTC must be between 0 and 23")
	}
	if cfg.JWTPublicKeyPath != "" {
		if _, err := os.Stat(cfg.JWTPublicKeyPath); err != nil {
			return nil, fmt.Errorf("ACTA_JWT_PUBLIC_KEY_PATH: %w", err)
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
