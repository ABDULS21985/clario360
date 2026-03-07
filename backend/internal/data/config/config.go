package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPPort         string
	DBURL            string
	DBMinConns       int
	DBMaxConns       int
	RedisURL         string
	KafkaBrokers     []string
	KafkaGroupID     string
	KafkaTopic       string
	JWTPublicKeyPath string
	EncryptionKey    []byte
	EncryptionKeyID  string

	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string

	ConnectorMaxPoolSize      int
	ConnectorStatementTimeout time.Duration
	ConnectorConnectTimeout   time.Duration
	ConnectorMaxSampleRows    int
	ConnectorMaxTables        int
	ConnectorAPIRateLimit     int

	DiscoveryMaxColumns   int
	DiscoveryPIIEnabled   bool
	DiscoverySampleValues bool
}

func Load() (*Config, error) {
	cfg := &Config{
		DBURL:            os.Getenv("DATA_DB_URL"),
		RedisURL:         os.Getenv("DATA_REDIS_URL"),
		KafkaGroupID:     os.Getenv("DATA_KAFKA_GROUP_ID"),
		JWTPublicKeyPath: os.Getenv("DATA_JWT_PUBLIC_KEY_PATH"),
		HTTPPort:         envOr("DATA_HTTP_PORT", "8091"),
		DBMinConns:       envInt("DATA_DB_MIN_CONNS", 5),
		DBMaxConns:       envInt("DATA_DB_MAX_CONNS", 20),
		KafkaTopic:       envOr("DATA_KAFKA_TOPIC", "data.source.events"),

		MinIOEndpoint:  envOr("DATA_MINIO_ENDPOINT", "minio:9000"),
		MinIOAccessKey: envOr("DATA_MINIO_ACCESS_KEY", ""),
		MinIOSecretKey: envOr("DATA_MINIO_SECRET_KEY", ""),
		MinIOBucket:    envOr("DATA_MINIO_BUCKET", "clario-data"),

		ConnectorMaxPoolSize:      envInt("DATA_CONNECTOR_MAX_POOL_SIZE", 3),
		ConnectorStatementTimeout: envDuration("DATA_CONNECTOR_STATEMENT_TIMEOUT", 30*time.Second),
		ConnectorConnectTimeout:   envDuration("DATA_CONNECTOR_CONNECT_TIMEOUT", 10*time.Second),
		ConnectorMaxSampleRows:    envInt("DATA_CONNECTOR_MAX_SAMPLE_ROWS", 100),
		ConnectorMaxTables:        envInt("DATA_CONNECTOR_MAX_TABLES", 500),
		ConnectorAPIRateLimit:     envInt("DATA_CONNECTOR_API_RATE_LIMIT", 10),

		DiscoveryMaxColumns:   envInt("DATA_DISCOVERY_MAX_COLUMNS", 1000),
		DiscoveryPIIEnabled:   envBool("DATA_DISCOVERY_PII_ENABLED", true),
		DiscoverySampleValues: envBool("DATA_DISCOVERY_SAMPLE_VALUES", true),
	}

	if cfg.DBURL == "" {
		return nil, fmt.Errorf("DATA_DB_URL is required")
	}
	if _, err := url.Parse(cfg.DBURL); err != nil {
		return nil, fmt.Errorf("DATA_DB_URL is invalid: %w", err)
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("DATA_REDIS_URL is required")
	}
	if _, err := url.Parse(cfg.RedisURL); err != nil {
		return nil, fmt.Errorf("DATA_REDIS_URL is invalid: %w", err)
	}
	brokers := strings.TrimSpace(os.Getenv("DATA_KAFKA_BROKERS"))
	if brokers == "" {
		return nil, fmt.Errorf("DATA_KAFKA_BROKERS is required")
	}
	cfg.KafkaBrokers = splitAndTrimCSV(brokers)
	if cfg.KafkaGroupID == "" {
		return nil, fmt.Errorf("DATA_KAFKA_GROUP_ID is required")
	}
	if cfg.JWTPublicKeyPath == "" {
		return nil, fmt.Errorf("DATA_JWT_PUBLIC_KEY_PATH is required")
	}
	keyPEM, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read DATA_JWT_PUBLIC_KEY_PATH: %w", err)
	}
	if block, _ := pem.Decode(keyPEM); block == nil {
		return nil, fmt.Errorf("DATA_JWT_PUBLIC_KEY_PATH does not contain a valid PEM block")
	}

	encryptionKeyBase64 := os.Getenv("DATA_ENCRYPTION_KEY")
	if encryptionKeyBase64 == "" {
		return nil, fmt.Errorf("DATA_ENCRYPTION_KEY is required")
	}
	cfg.EncryptionKey, err = base64.StdEncoding.DecodeString(encryptionKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode DATA_ENCRYPTION_KEY: %w", err)
	}
	if len(cfg.EncryptionKey) != 32 {
		return nil, fmt.Errorf("DATA_ENCRYPTION_KEY must decode to exactly 32 bytes, got %d", len(cfg.EncryptionKey))
	}
	hash := sha256.Sum256(cfg.EncryptionKey)
	cfg.EncryptionKeyID = fmt.Sprintf("%x", hash[:])[:8]

	if cfg.ConnectorMaxPoolSize < 1 || cfg.ConnectorMaxPoolSize > 10 {
		return nil, fmt.Errorf("DATA_CONNECTOR_MAX_POOL_SIZE must be in [1, 10], got %d", cfg.ConnectorMaxPoolSize)
	}
	if cfg.ConnectorStatementTimeout < 5*time.Second || cfg.ConnectorStatementTimeout > 300*time.Second {
		return nil, fmt.Errorf("DATA_CONNECTOR_STATEMENT_TIMEOUT must be in [5s, 300s], got %s", cfg.ConnectorStatementTimeout)
	}
	if cfg.ConnectorConnectTimeout < 1*time.Second || cfg.ConnectorConnectTimeout > 60*time.Second {
		return nil, fmt.Errorf("DATA_CONNECTOR_CONNECT_TIMEOUT must be in [1s, 60s], got %s", cfg.ConnectorConnectTimeout)
	}
	if cfg.ConnectorMaxTables < 1 || cfg.ConnectorMaxTables > 10000 {
		return nil, fmt.Errorf("DATA_CONNECTOR_MAX_TABLES must be in [1, 10000], got %d", cfg.ConnectorMaxTables)
	}
	if cfg.ConnectorMaxSampleRows < 1 || cfg.ConnectorMaxSampleRows > 1000 {
		return nil, fmt.Errorf("DATA_CONNECTOR_MAX_SAMPLE_ROWS must be in [1, 1000], got %d", cfg.ConnectorMaxSampleRows)
	}
	if cfg.DiscoveryMaxColumns < 1 || cfg.DiscoveryMaxColumns > 5000 {
		return nil, fmt.Errorf("DATA_DISCOVERY_MAX_COLUMNS must be in [1, 5000], got %d", cfg.DiscoveryMaxColumns)
	}

	return cfg, nil
}

func splitAndTrimCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}
