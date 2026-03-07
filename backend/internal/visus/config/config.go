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
	HTTPPort                int
	AdminPort               int
	DBURL                   string
	DBMinConns              int
	DBMaxConns              int
	RedisAddr               string
	RedisPassword           string
	RedisDB                 int
	KafkaBrokers            []string
	KafkaGroupID            string
	KafkaTopic              string
	JWTPublicKeyPath        string
	JWTPrivateKeyPath       string
	RateLimitPerMinute      int
	SeedDemoData            bool
	DemoTenantID            string
	DemoUserID              string
	DashboardCacheTTL       time.Duration
	SuiteCacheTTL           time.Duration
	SuiteTimeout            time.Duration
	SuiteMaxRetries         int
	CircuitThreshold        int
	CircuitReset            time.Duration
	SchedulerInterval       time.Duration
	ReportSchedulerInterval time.Duration
	SuiteCyberURL           string
	SuiteDataURL            string
	SuiteActaURL            string
	SuiteLexURL             string
	ServiceAccountToken     string
	ServiceAccountUserID    string
	ServiceAccountEmail     string
	ServiceTokenTTL         time.Duration
}

func Default() *Config {
	return &Config{
		HTTPPort:                8088,
		AdminPort:               9088,
		DBMinConns:              5,
		DBMaxConns:              20,
		RedisAddr:               "localhost:6379",
		RedisDB:                 0,
		KafkaGroupID:            "visus-service",
		KafkaTopic:              "enterprise.visus.events",
		RateLimitPerMinute:      300,
		DashboardCacheTTL:       time.Minute,
		SuiteCacheTTL:           time.Minute,
		SuiteTimeout:            5 * time.Second,
		SuiteMaxRetries:         3,
		CircuitThreshold:        5,
		CircuitReset:            time.Minute,
		SchedulerInterval:       time.Minute,
		ReportSchedulerInterval: time.Minute,
		SuiteCyberURL:           "http://localhost:8090",
		SuiteDataURL:            "http://localhost:8091",
		SuiteActaURL:            "http://localhost:8092",
		SuiteLexURL:             "http://localhost:8093",
		ServiceAccountUserID:    "00000000-0000-0000-0000-000000000360",
		ServiceAccountEmail:     "service-visus@clario.local",
		ServiceTokenTTL:         10 * time.Minute,
	}
}

func Load(base *appconfig.Config) (*Config, error) {
	if base == nil {
		return nil, fmt.Errorf("base config is required")
	}

	cfg := Default()
	cfg.HTTPPort = envInt("VISUS_HTTP_PORT", cfg.HTTPPort)
	cfg.AdminPort = envInt("VISUS_ADMIN_PORT", cfg.AdminPort)
	cfg.DBURL = envOr("VISUS_DB_URL", buildDBURL(base, "visus_db"))
	cfg.DBMinConns = envInt("VISUS_DB_MIN_CONNS", max(base.Database.MaxIdleConns, cfg.DBMinConns))
	cfg.DBMaxConns = envInt("VISUS_DB_MAX_CONNS", max(base.Database.MaxOpenConns, cfg.DBMaxConns))
	cfg.RedisAddr = envOr("VISUS_REDIS_ADDR", base.Redis.Addr())
	cfg.RedisPassword = envOr("VISUS_REDIS_PASSWORD", base.Redis.Password)
	cfg.RedisDB = envInt("VISUS_REDIS_DB", base.Redis.DB)
	cfg.KafkaBrokers = splitCSV(envOr("VISUS_KAFKA_BROKERS", strings.Join(base.Kafka.Brokers, ",")))
	cfg.KafkaGroupID = envOr("VISUS_KAFKA_GROUP_ID", cfg.KafkaGroupID)
	cfg.KafkaTopic = envOr("VISUS_KAFKA_TOPIC", cfg.KafkaTopic)
	cfg.JWTPublicKeyPath = envOr("VISUS_JWT_PUBLIC_KEY_PATH", "")
	cfg.JWTPrivateKeyPath = envOr("VISUS_JWT_PRIVATE_KEY_PATH", "")
	cfg.RateLimitPerMinute = envInt("VISUS_RATE_LIMIT_PER_MINUTE", cfg.RateLimitPerMinute)
	cfg.SeedDemoData = envBool("VISUS_SEED_DEMO_DATA", false)
	cfg.DemoTenantID = envOr("VISUS_DEMO_TENANT_ID", "99999999-9999-9999-9999-999999999999")
	cfg.DemoUserID = envOr("VISUS_DEMO_USER_ID", "99999999-9999-9999-9999-999999999001")
	cfg.DashboardCacheTTL = envDuration("VISUS_DASHBOARD_CACHE_TTL", cfg.DashboardCacheTTL)
	cfg.SuiteCacheTTL = envDuration("VISUS_SUITE_CACHE_TTL", cfg.SuiteCacheTTL)
	cfg.SuiteTimeout = envDuration("VISUS_SUITE_TIMEOUT", cfg.SuiteTimeout)
	cfg.SuiteMaxRetries = envInt("VISUS_SUITE_MAX_RETRIES", cfg.SuiteMaxRetries)
	cfg.CircuitThreshold = envInt("VISUS_CIRCUIT_THRESHOLD", cfg.CircuitThreshold)
	cfg.CircuitReset = envDuration("VISUS_CIRCUIT_RESET", cfg.CircuitReset)
	cfg.SchedulerInterval = envDuration("VISUS_SCHEDULER_INTERVAL", cfg.SchedulerInterval)
	cfg.ReportSchedulerInterval = envDuration("VISUS_REPORT_SCHEDULER_INTERVAL", cfg.ReportSchedulerInterval)
	cfg.SuiteCyberURL = envOr("VISUS_SUITE_CYBER_URL", cfg.SuiteCyberURL)
	cfg.SuiteDataURL = envOr("VISUS_SUITE_DATA_URL", cfg.SuiteDataURL)
	cfg.SuiteActaURL = envOr("VISUS_SUITE_ACTA_URL", cfg.SuiteActaURL)
	cfg.SuiteLexURL = envOr("VISUS_SUITE_LEX_URL", cfg.SuiteLexURL)
	cfg.ServiceAccountToken = envOr("VISUS_SERVICE_ACCOUNT_TOKEN", "")
	cfg.ServiceAccountUserID = envOr("VISUS_SERVICE_ACCOUNT_USER_ID", cfg.ServiceAccountUserID)
	cfg.ServiceAccountEmail = envOr("VISUS_SERVICE_ACCOUNT_EMAIL", cfg.ServiceAccountEmail)
	cfg.ServiceTokenTTL = envDuration("VISUS_SERVICE_TOKEN_TTL", cfg.ServiceTokenTTL)

	if _, err := url.Parse(cfg.DBURL); err != nil {
		return nil, fmt.Errorf("VISUS_DB_URL is invalid: %w", err)
	}
	if cfg.HTTPPort < 1 || cfg.HTTPPort > 65535 {
		return nil, fmt.Errorf("VISUS_HTTP_PORT must be between 1 and 65535")
	}
	if cfg.AdminPort < 1 || cfg.AdminPort > 65535 {
		return nil, fmt.Errorf("VISUS_ADMIN_PORT must be between 1 and 65535")
	}
	if cfg.DBMinConns < 1 {
		return nil, fmt.Errorf("VISUS_DB_MIN_CONNS must be >= 1")
	}
	if cfg.DBMaxConns < cfg.DBMinConns {
		return nil, fmt.Errorf("VISUS_DB_MAX_CONNS must be >= VISUS_DB_MIN_CONNS")
	}
	if cfg.RateLimitPerMinute < 1 {
		return nil, fmt.Errorf("VISUS_RATE_LIMIT_PER_MINUTE must be >= 1")
	}
	if cfg.SuiteTimeout <= 0 {
		return nil, fmt.Errorf("VISUS_SUITE_TIMEOUT must be > 0")
	}
	if cfg.SuiteMaxRetries < 1 {
		return nil, fmt.Errorf("VISUS_SUITE_MAX_RETRIES must be >= 1")
	}
	if cfg.CircuitThreshold < 1 {
		return nil, fmt.Errorf("VISUS_CIRCUIT_THRESHOLD must be >= 1")
	}
	if cfg.CircuitReset <= 0 {
		return nil, fmt.Errorf("VISUS_CIRCUIT_RESET must be > 0")
	}
	if cfg.SchedulerInterval <= 0 || cfg.ReportSchedulerInterval <= 0 {
		return nil, fmt.Errorf("scheduler intervals must be > 0")
	}
	if cfg.JWTPublicKeyPath != "" {
		if _, err := os.Stat(cfg.JWTPublicKeyPath); err != nil {
			return nil, fmt.Errorf("VISUS_JWT_PUBLIC_KEY_PATH: %w", err)
		}
	}
	if cfg.JWTPrivateKeyPath != "" {
		if _, err := os.Stat(cfg.JWTPrivateKeyPath); err != nil {
			return nil, fmt.Errorf("VISUS_JWT_PRIVATE_KEY_PATH: %w", err)
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

func envBool(key string, fallback bool) bool {
	if raw := strings.TrimSpace(os.Getenv(key)); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
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
