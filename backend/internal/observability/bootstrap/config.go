package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/observability/tracing"
)

// ServiceConfig is the unified configuration for any Clario 360 service.
type ServiceConfig struct {
	// Identity
	Name        string
	Version     string
	Environment string // "production", "staging", "development"

	// HTTP
	Port      int // main API port (default 8080)
	AdminPort int // metrics + health + pprof port (default 9090)

	// Logging
	LogLevel        string // debug, info, warn, error
	DebugSampleRate int    // 1-in-N debug logs in production

	// Database (nil-safe: omit if service doesn't need DB)
	DB *DBConfig

	// Redis (nil-safe)
	Redis *RedisConfig

	// Kafka (nil-safe)
	Kafka *KafkaConfig

	// Tracing
	Tracing tracing.TracerConfig

	// Profiling
	EnablePprof bool

	// Graceful shutdown
	ShutdownTimeout time.Duration // default 30s

	// HTTP server timeouts
	ReadTimeout  time.Duration // default 15s
	WriteTimeout time.Duration // default 15s
}

// DBConfig holds PostgreSQL connection configuration.
type DBConfig struct {
	URL             string        // postgres://...
	MinConns        int           // default 5
	MaxConns        int           // default 20
	MaxConnLife     time.Duration // default 1h
	MaxConnIdle     time.Duration // default 30m
	HealthCheckPeriod time.Duration // default 1m
}

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Addr     string // host:port
	Password string
	DB       int
}

// KafkaConfig holds Kafka configuration.
type KafkaConfig struct {
	Brokers []string
	GroupID string
}

// LoadConfig reads ALL configuration values from environment variables.
// Returns error listing ALL missing required variables (not just the first).
func LoadConfig() (*ServiceConfig, error) {
	var missing []string

	name := getEnv("SERVICE_NAME", "")
	if name == "" {
		missing = append(missing, "SERVICE_NAME")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	cfg := &ServiceConfig{
		Name:            name,
		Version:         getEnv("SERVICE_VERSION", "0.0.0"),
		Environment:     getEnv("ENVIRONMENT", "development"),
		Port:            getEnvInt("PORT", 8080),
		AdminPort:       getEnvInt("ADMIN_PORT", 9090),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		DebugSampleRate: getEnvInt("DEBUG_SAMPLE_RATE", 100),
		EnablePprof:     getEnvBool("ENABLE_PPROF", false),
		ShutdownTimeout: getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		ReadTimeout:     getEnvDuration("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getEnvDuration("WRITE_TIMEOUT", 15*time.Second),
	}

	// Tracing config.
	cfg.Tracing = tracing.TracerConfig{
		Enabled:     getEnvBool("TRACING_ENABLED", true),
		Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		ServiceName: name,
		Version:     cfg.Version,
		Environment: cfg.Environment,
		SampleRate:  getEnvFloat("TRACING_SAMPLE_RATE", 0.1),
		Insecure:    getEnvBool("TRACING_INSECURE", true),
	}

	// Database config (optional).
	if dbURL := getEnv("DATABASE_URL", ""); dbURL != "" {
		cfg.DB = &DBConfig{
			URL:               dbURL,
			MinConns:          getEnvInt("DATABASE_MIN_CONNS", 5),
			MaxConns:          getEnvInt("DATABASE_MAX_CONNS", 20),
			MaxConnLife:       getEnvDuration("DATABASE_MAX_CONN_LIFETIME", 1*time.Hour),
			MaxConnIdle:       getEnvDuration("DATABASE_MAX_CONN_IDLE", 30*time.Minute),
			HealthCheckPeriod: getEnvDuration("DATABASE_HEALTH_CHECK_PERIOD", 1*time.Minute),
		}
	}

	// Redis config (optional).
	if redisAddr := getEnv("REDIS_ADDR", ""); redisAddr != "" {
		cfg.Redis = &RedisConfig{
			Addr:     redisAddr,
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		}
	}

	// Kafka config (optional).
	if brokers := getEnv("KAFKA_BROKERS", ""); brokers != "" {
		cfg.Kafka = &KafkaConfig{
			Brokers: strings.Split(brokers, ","),
			GroupID: getEnv("KAFKA_GROUP_ID", name),
		}
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return defaultVal
}
