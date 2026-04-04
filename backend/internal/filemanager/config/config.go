package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/observability/bootstrap"
	"github.com/clario360/platform/internal/observability/tracing"
)

// Config holds the complete file service configuration.
type Config struct {
	ServiceConfig *bootstrap.ServiceConfig

	// MinIO
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOUseSSL    bool
	MinIORegion    string
	BucketPrefix   string

	// ClamAV
	ClamAVAddress    string
	ClamAVTimeout    time.Duration
	ClamAVMaxSizeMB  int

	// Encryption
	EncryptionMasterKey []byte
	EncryptionKeyID     string

	// Upload limits
	MaxUploadSizeMB int

	// Presigned URLs
	PresignedURLExpiry time.Duration

	// Quarantine
	QuarantineBucket string

	// Lifecycle
	TempExpiryDays int

	// JWT
	JWTPublicKeyPath string
}

// Load reads configuration from environment variables with validation.
func Load() (*Config, error) {
	var missing []string

	getRequired := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	getOptional := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}

	getOptionalInt := func(key string, fallback int) int {
		if v := os.Getenv(key); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
		return fallback
	}

	cfg := &Config{}

	// Required
	dbURL := getRequired("FILE_DB_URL")
	redisURL := getRequired("FILE_REDIS_URL")
	kafkaBrokers := getRequired("FILE_KAFKA_BROKERS")
	kafkaGroupID := getRequired("FILE_KAFKA_GROUP_ID")
	cfg.JWTPublicKeyPath = getRequired("FILE_JWT_PUBLIC_KEY_PATH")
	cfg.MinIOEndpoint = getRequired("FILE_MINIO_ENDPOINT")
	cfg.MinIOAccessKey = getRequired("FILE_MINIO_ACCESS_KEY")
	cfg.MinIOSecretKey = getRequired("FILE_MINIO_SECRET_KEY")
	masterKeyB64 := getRequired("FILE_ENCRYPTION_MASTER_KEY")

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	// Decode and validate master key
	masterKey, err := base64.StdEncoding.DecodeString(masterKeyB64)
	if err != nil {
		return nil, fmt.Errorf("FILE_ENCRYPTION_MASTER_KEY: invalid base64: %w", err)
	}
	if len(masterKey) != 32 {
		return nil, fmt.Errorf("FILE_ENCRYPTION_MASTER_KEY: must be exactly 32 bytes, got %d", len(masterKey))
	}
	cfg.EncryptionMasterKey = masterKey

	// Optional with defaults
	port := getOptionalInt("FILE_HTTP_PORT", 8091)
	dbMinConns := getOptionalInt("FILE_DB_MIN_CONNS", 5)
	dbMaxConns := getOptionalInt("FILE_DB_MAX_CONNS", 20)
	if dbMinConns < 1 {
		return nil, fmt.Errorf("FILE_DB_MIN_CONNS must be >= 1")
	}
	if dbMaxConns < dbMinConns {
		return nil, fmt.Errorf("FILE_DB_MAX_CONNS must be >= FILE_DB_MIN_CONNS")
	}
	cfg.MinIOUseSSL = getOptional("FILE_MINIO_USE_SSL", "false") == "true"
	cfg.MinIORegion = getOptional("FILE_MINIO_REGION", "us-east-1")
	cfg.BucketPrefix = getOptional("FILE_MINIO_BUCKET_PREFIX", "clario360")
	cfg.ClamAVAddress = getOptional("FILE_CLAMAV_ADDRESS", "clamd:3310")
	cfg.EncryptionKeyID = getOptional("FILE_ENCRYPTION_KEY_ID", "kek-v1")
	cfg.QuarantineBucket = getOptional("FILE_QUARANTINE_BUCKET", "clario360-quarantine")
	cfg.TempExpiryDays = getOptionalInt("FILE_TEMP_EXPIRY_DAYS", 7)

	clamTimeoutSec := getOptionalInt("FILE_CLAMAV_TIMEOUT_SEC", 120)
	if clamTimeoutSec < 10 || clamTimeoutSec > 600 {
		return nil, fmt.Errorf("FILE_CLAMAV_TIMEOUT_SEC must be between 10 and 600")
	}
	cfg.ClamAVTimeout = time.Duration(clamTimeoutSec) * time.Second

	cfg.ClamAVMaxSizeMB = getOptionalInt("FILE_CLAMAV_MAX_SCAN_SIZE_MB", 100)

	cfg.MaxUploadSizeMB = getOptionalInt("FILE_MAX_UPLOAD_SIZE_MB", 100)
	if cfg.MaxUploadSizeMB < 1 || cfg.MaxUploadSizeMB > 500 {
		return nil, fmt.Errorf("FILE_MAX_UPLOAD_SIZE_MB must be between 1 and 500")
	}

	presignedMin := getOptionalInt("FILE_PRESIGNED_URL_EXPIRY_MIN", 15)
	if presignedMin < 1 || presignedMin > 60 {
		return nil, fmt.Errorf("FILE_PRESIGNED_URL_EXPIRY_MIN must be between 1 and 60")
	}
	cfg.PresignedURLExpiry = time.Duration(presignedMin) * time.Minute

	env := getOptional("FILE_ENVIRONMENT", "development")
	logLevel := getOptional("FILE_LOG_LEVEL", "info")

	// Build bootstrap ServiceConfig
	brokers := strings.Split(kafkaBrokers, ",")
	for i := range brokers {
		brokers[i] = strings.TrimSpace(brokers[i])
	}

	cfg.ServiceConfig = &bootstrap.ServiceConfig{
		Name:        "file-service",
		Version:     "1.0.0",
		Environment: env,
		Port:        port,
		AdminPort:   port + 1000, // 9091
		LogLevel:    logLevel,
		DB: &bootstrap.DBConfig{
			URL:               dbURL,
			MinConns:          dbMinConns,
			MaxConns:          dbMaxConns,
			MaxConnLife:       time.Hour,
			MaxConnIdle:       30 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		Redis: &bootstrap.RedisConfig{
			Addr: redisURL,
		},
		Kafka: &bootstrap.KafkaConfig{
			Brokers: brokers,
			GroupID: kafkaGroupID,
		},
		Tracing: tracing.TracerConfig{
			Enabled:     getOptional("FILE_TRACING_ENABLED", "true") == "true",
			Endpoint:    getOptional("FILE_TRACING_ENDPOINT", "jaeger:4317"),
			ServiceName: "file-service",
			Version:     "1.0.0",
			Environment: env,
			SampleRate:  1.0,
			Insecure:    true,
		},
		ShutdownTimeout: 30 * time.Second,
		ReadTimeout:     60 * time.Second, // longer for file uploads
		WriteTimeout:    120 * time.Second,
	}

	return cfg, nil
}
