package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all notification-service specific configuration.
type Config struct {
	HTTPPort int

	// Email provider: "smtp" or "sendgrid"
	EmailProvider string

	// SMTP settings
	SMTPHost       string
	SMTPPort       int
	SMTPUsername   string
	SMTPPassword   string
	SMTPFrom       string
	SMTPTLSEnabled bool

	// SendGrid settings
	SendGridAPIKey string
	SendGridFrom   string

	// WebSocket settings
	WSMaxConnectionsPerUser int
	WSPingIntervalSec       int
	WSPongTimeoutSec        int
	WSWriteTimeoutSec       int
	WSMaxMessageSizeBytes   int64
	WSAllowedOrigins        []string

	// Webhook settings
	WebhookTimeoutSec int
	WebhookMaxRetries int
	WebhookHMACSecret string

	// Digest settings
	DigestEnabled      bool
	DigestDailyUTCHour int
	DigestWeeklyDay    int // 1=Monday

	// IAM service URL for role-based recipient resolution
	IAMServiceURL string

	// Rate limiting
	RateLimitPerMinute int

	// Environment ("development" or "production")
	Environment string
}

// LoadFromEnv loads notification-specific config from environment variables.
func LoadFromEnv() *Config {
	cfg := &Config{
		HTTPPort:                envInt("NOTIF_HTTP_PORT", 8089),
		EmailProvider:           envStr("NOTIF_EMAIL_PROVIDER", "smtp"),
		SMTPHost:                envStr("NOTIF_SMTP_HOST", ""),
		SMTPPort:                envInt("NOTIF_SMTP_PORT", 587),
		SMTPUsername:            envStr("NOTIF_SMTP_USERNAME", ""),
		SMTPPassword:            envStr("NOTIF_SMTP_PASSWORD", ""),
		SMTPFrom:                envStr("NOTIF_SMTP_FROM_ADDRESS", "Clario 360 <notifications@clario360.com>"),
		SMTPTLSEnabled:          envBool("NOTIF_SMTP_TLS_ENABLED", true),
		SendGridAPIKey:          envStr("NOTIF_SENDGRID_API_KEY", ""),
		SendGridFrom:            envStr("NOTIF_SENDGRID_FROM_ADDRESS", "Clario 360 <notifications@clario360.com>"),
		WSMaxConnectionsPerUser: envInt("NOTIF_WS_MAX_CONNECTIONS_PER_USER", 5),
		WSPingIntervalSec:       envInt("NOTIF_WS_PING_INTERVAL_SEC", 30),
		WSPongTimeoutSec:        envInt("NOTIF_WS_PONG_TIMEOUT_SEC", 10),
		WSWriteTimeoutSec:       envInt("NOTIF_WS_WRITE_TIMEOUT_SEC", 10),
		WSMaxMessageSizeBytes:   envInt64("NOTIF_WS_MAX_MESSAGE_SIZE_BYTES", 4096),
		WSAllowedOrigins:        envStrSlice("NOTIF_WS_ALLOWED_ORIGINS", nil),
		WebhookTimeoutSec:       envInt("NOTIF_WEBHOOK_TIMEOUT_SEC", 10),
		WebhookMaxRetries:       envInt("NOTIF_WEBHOOK_MAX_RETRIES", 3),
		WebhookHMACSecret:       envStr("NOTIF_WEBHOOK_HMAC_SECRET", ""),
		DigestEnabled:           envBool("NOTIF_DIGEST_ENABLED", true),
		DigestDailyUTCHour:      envInt("NOTIF_DIGEST_DAILY_UTC_HOUR", 8),
		DigestWeeklyDay:         envInt("NOTIF_DIGEST_WEEKLY_DAY", 1),
		IAMServiceURL:           envStr("NOTIF_IAM_SERVICE_URL", "http://iam-service:8083"),
		RateLimitPerMinute:      envInt("NOTIF_RATE_LIMIT_PER_MINUTE", 120),
		Environment:             envStr("NOTIF_ENVIRONMENT", "development"),
	}
	return cfg
}

// Validate checks that the configuration is internally consistent.
func (c *Config) Validate() error {
	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		return fmt.Errorf("NOTIF_HTTP_PORT must be in [1, 65535], got %d", c.HTTPPort)
	}

	switch c.EmailProvider {
	case "smtp":
		if c.SMTPHost == "" {
			return fmt.Errorf("NOTIF_SMTP_HOST is required when email provider is smtp")
		}
		if c.SMTPPort < 1 || c.SMTPPort > 65535 {
			return fmt.Errorf("NOTIF_SMTP_PORT must be in [1, 65535], got %d", c.SMTPPort)
		}
	case "sendgrid":
		if c.SendGridAPIKey == "" {
			return fmt.Errorf("NOTIF_SENDGRID_API_KEY is required when email provider is sendgrid")
		}
	default:
		return fmt.Errorf("NOTIF_EMAIL_PROVIDER must be 'smtp' or 'sendgrid', got %q", c.EmailProvider)
	}

	if c.WSPingIntervalSec <= c.WSPongTimeoutSec {
		return fmt.Errorf("WS ping interval (%d) must be > pong timeout (%d)", c.WSPingIntervalSec, c.WSPongTimeoutSec)
	}

	if c.WebhookHMACSecret != "" && len(c.WebhookHMACSecret) < 32 {
		return fmt.Errorf("NOTIF_WEBHOOK_HMAC_SECRET must be >= 32 bytes if set, got %d", len(c.WebhookHMACSecret))
	}

	if c.DigestDailyUTCHour < 0 || c.DigestDailyUTCHour > 23 {
		return fmt.Errorf("NOTIF_DIGEST_DAILY_UTC_HOUR must be in [0, 23]")
	}

	if c.DigestWeeklyDay < 0 || c.DigestWeeklyDay > 6 {
		return fmt.Errorf("NOTIF_DIGEST_WEEKLY_DAY must be in [0, 6]")
	}

	return nil
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func envStrSlice(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return fallback
}
