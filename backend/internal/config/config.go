package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server        ServerConfig        `mapstructure:"server"`
	Database      DatabaseConfig      `mapstructure:"database"`
	Redis         RedisConfig         `mapstructure:"redis"`
	Kafka         KafkaConfig         `mapstructure:"kafka"`
	Auth          AuthConfig          `mapstructure:"auth"`
	Observability ObservabilityConfig `mapstructure:"observability"`
	MinIO         MinIOConfig         `mapstructure:"minio"`
	Encryption    EncryptionConfig    `mapstructure:"encryption"`
}

type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type KafkaConfig struct {
	Brokers         []string `mapstructure:"brokers"`
	GroupID         string   `mapstructure:"group_id"`
	AutoOffsetReset string   `mapstructure:"auto_offset_reset"`
}

type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	JWTIssuer       string        `mapstructure:"jwt_issuer"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
	BcryptCost      int           `mapstructure:"bcrypt_cost"`
}

type ObservabilityConfig struct {
	LogLevel     string `mapstructure:"log_level"`
	LogFormat    string `mapstructure:"log_format"`
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
	ServiceName  string `mapstructure:"service_name"`
	MetricsPort  int    `mapstructure:"metrics_port"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	UseSSL    bool   `mapstructure:"use_ssl"`
	Bucket    string `mapstructure:"bucket"`
}

type EncryptionConfig struct {
	Key string `mapstructure:"key"`
}

// Load reads configuration from environment variables, config files, and defaults.
func Load() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	// Environment variable binding
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindEnvVars(v)

	// Optional YAML config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/clario360")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", 15*time.Second)
	v.SetDefault("server.write_timeout", 15*time.Second)
	v.SetDefault("server.shutdown_timeout", 30*time.Second)

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "clario")
	v.SetDefault("database.password", "clario_dev_pass")
	v.SetDefault("database.name", "clario360")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// Redis
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Kafka
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.group_id", "clario360")
	v.SetDefault("kafka.auto_offset_reset", "earliest")

	// Auth
	v.SetDefault("auth.jwt_secret", "change-me-in-production-use-256-bit-key")
	v.SetDefault("auth.jwt_issuer", "clario360")
	v.SetDefault("auth.access_token_ttl", 15*time.Minute)
	v.SetDefault("auth.refresh_token_ttl", 7*24*time.Hour)
	v.SetDefault("auth.bcrypt_cost", 12)

	// Observability
	v.SetDefault("observability.log_level", "debug")
	v.SetDefault("observability.log_format", "json")
	v.SetDefault("observability.otlp_endpoint", "http://localhost:4317")
	v.SetDefault("observability.service_name", "clario360")
	v.SetDefault("observability.metrics_port", 9090)

	// MinIO
	v.SetDefault("minio.endpoint", "localhost:9000")
	v.SetDefault("minio.access_key", "clario_minio")
	v.SetDefault("minio.secret_key", "clario_minio_secret")
	v.SetDefault("minio.use_ssl", false)
	v.SetDefault("minio.bucket", "clario360")

	// Encryption
	v.SetDefault("encryption.key", "0123456789abcdef0123456789abcdef")
}

func bindEnvVars(v *viper.Viper) {
	bindings := map[string]string{
		"server.host":               "SERVER_HOST",
		"server.port":               "SERVER_PORT",
		"server.read_timeout":       "SERVER_READ_TIMEOUT",
		"server.write_timeout":      "SERVER_WRITE_TIMEOUT",
		"server.shutdown_timeout":   "SERVER_SHUTDOWN_TIMEOUT",
		"database.host":             "DATABASE_HOST",
		"database.port":             "DATABASE_PORT",
		"database.user":             "DATABASE_USER",
		"database.password":         "DATABASE_PASSWORD",
		"database.name":             "DATABASE_NAME",
		"database.ssl_mode":         "DATABASE_SSL_MODE",
		"database.max_open_conns":   "DATABASE_MAX_OPEN_CONNS",
		"database.max_idle_conns":   "DATABASE_MAX_IDLE_CONNS",
		"database.conn_max_lifetime": "DATABASE_CONN_MAX_LIFETIME",
		"redis.host":                "REDIS_HOST",
		"redis.port":                "REDIS_PORT",
		"redis.password":            "REDIS_PASSWORD",
		"redis.db":                  "REDIS_DB",
		"kafka.brokers":             "KAFKA_BROKERS",
		"kafka.group_id":            "KAFKA_GROUP_ID",
		"kafka.auto_offset_reset":   "KAFKA_AUTO_OFFSET_RESET",
		"auth.jwt_secret":           "AUTH_JWT_SECRET",
		"auth.jwt_issuer":           "AUTH_JWT_ISSUER",
		"auth.access_token_ttl":     "AUTH_JWT_ACCESS_TOKEN_TTL",
		"auth.refresh_token_ttl":    "AUTH_JWT_REFRESH_TOKEN_TTL",
		"auth.bcrypt_cost":          "AUTH_BCRYPT_COST",
		"observability.log_level":   "LOG_LEVEL",
		"observability.log_format":  "LOG_FORMAT",
		"observability.otlp_endpoint": "OTEL_EXPORTER_OTLP_ENDPOINT",
		"observability.service_name": "OTEL_SERVICE_NAME",
		"observability.metrics_port": "METRICS_PORT",
		"minio.endpoint":            "MINIO_ENDPOINT",
		"minio.access_key":          "MINIO_ACCESS_KEY",
		"minio.secret_key":          "MINIO_SECRET_KEY",
		"minio.use_ssl":             "MINIO_USE_SSL",
		"minio.bucket":              "MINIO_BUCKET",
		"encryption.key":            "ENCRYPTION_KEY",
	}
	for key, env := range bindings {
		_ = v.BindEnv(key, env)
	}
}
