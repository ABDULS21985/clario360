package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the cyber service.
type Config struct {
	// Server
	HTTPPort string

	// Database
	DBURL     string
	DBMinConn int
	DBMaxConn int

	// Redis
	RedisURL string

	// Kafka
	KafkaBrokers []string
	KafkaGroupID string
	KafkaTopic   string

	// JWT
	JWTPublicKeyPath string

	// Scanner settings
	ScanNetworkWorkers    int
	ScanNetworkTimeoutSec int
	ScanNetworkMaxIPs     int
	ScanDefaultPorts      []int

	// Enrichment
	EnrichmentDNSTimeoutSec int
	EnrichmentCVEEnabled    bool
	EnrichmentGeoEnabled    bool
	EnrichmentGeoDBPath     string

	// Classification
	ClassifyOnCreate bool
	ClassifyOnScan   bool
}

// Load reads configuration from environment variables. Returns an error if
// any required variable is missing or any value fails validation.
func Load() (*Config, error) {
	c := &Config{}

	// Required vars
	c.DBURL = os.Getenv("CYBER_DB_URL")
	if c.DBURL == "" {
		return nil, fmt.Errorf("CYBER_DB_URL is required")
	}
	c.RedisURL = os.Getenv("CYBER_REDIS_URL")
	if c.RedisURL == "" {
		return nil, fmt.Errorf("CYBER_REDIS_URL is required")
	}
	brokers := os.Getenv("CYBER_KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("CYBER_KAFKA_BROKERS is required")
	}
	c.KafkaBrokers = strings.Split(brokers, ",")
	c.KafkaGroupID = os.Getenv("CYBER_KAFKA_GROUP_ID")
	if c.KafkaGroupID == "" {
		return nil, fmt.Errorf("CYBER_KAFKA_GROUP_ID is required")
	}
	c.JWTPublicKeyPath = os.Getenv("CYBER_JWT_PUBLIC_KEY_PATH")
	if c.JWTPublicKeyPath == "" {
		return nil, fmt.Errorf("CYBER_JWT_PUBLIC_KEY_PATH is required")
	}
	if _, err := os.Stat(c.JWTPublicKeyPath); err != nil {
		return nil, fmt.Errorf("CYBER_JWT_PUBLIC_KEY_PATH %q: %w", c.JWTPublicKeyPath, err)
	}

	// Optional with defaults
	c.HTTPPort = envOr("CYBER_HTTP_PORT", "8090")
	c.DBMinConn = envInt("CYBER_DB_MIN_CONNS", 5)
	c.DBMaxConn = envInt("CYBER_DB_MAX_CONNS", 20)
	c.KafkaTopic = envOr("CYBER_KAFKA_TOPIC", "cyber.asset.events")

	// Scanner
	c.ScanNetworkWorkers = envInt("CYBER_SCAN_NETWORK_WORKERS", 100)
	c.ScanNetworkTimeoutSec = envInt("CYBER_SCAN_NETWORK_TIMEOUT_SEC", 2)
	c.ScanNetworkMaxIPs = envInt("CYBER_SCAN_NETWORK_MAX_IPS", 65536)
	defaultPortsStr := envOr("CYBER_SCAN_DEFAULT_PORTS", "22,80,443,3306,5432,8080,8443,3389,445,161")
	ports, err := parsePorts(defaultPortsStr)
	if err != nil {
		return nil, fmt.Errorf("CYBER_SCAN_DEFAULT_PORTS: %w", err)
	}
	c.ScanDefaultPorts = ports

	// Enrichment
	c.EnrichmentDNSTimeoutSec = envInt("CYBER_ENRICHMENT_DNS_TIMEOUT_SEC", 5)
	c.EnrichmentCVEEnabled = envBool("CYBER_ENRICHMENT_CVE_ENABLED", true)
	c.EnrichmentGeoEnabled = envBool("CYBER_ENRICHMENT_GEO_ENABLED", false)
	c.EnrichmentGeoDBPath = envOr("CYBER_ENRICHMENT_GEO_DB_PATH", "/data/GeoLite2-City.mmdb")

	// Classification
	c.ClassifyOnCreate = envBool("CYBER_CLASSIFY_ON_CREATE", true)
	c.ClassifyOnScan = envBool("CYBER_CLASSIFY_ON_SCAN", true)

	// Validate ranges
	if c.ScanNetworkWorkers < 1 || c.ScanNetworkWorkers > 500 {
		return nil, fmt.Errorf("CYBER_SCAN_NETWORK_WORKERS must be in [1, 500], got %d", c.ScanNetworkWorkers)
	}
	if c.ScanNetworkTimeoutSec < 1 || c.ScanNetworkTimeoutSec > 30 {
		return nil, fmt.Errorf("CYBER_SCAN_NETWORK_TIMEOUT_SEC must be in [1, 30], got %d", c.ScanNetworkTimeoutSec)
	}
	if c.ScanNetworkMaxIPs < 1 || c.ScanNetworkMaxIPs > 1048576 {
		return nil, fmt.Errorf("CYBER_SCAN_NETWORK_MAX_IPS must be in [1, 1048576], got %d", c.ScanNetworkMaxIPs)
	}
	for _, p := range c.ScanDefaultPorts {
		if p < 1 || p > 65535 {
			return nil, fmt.Errorf("CYBER_SCAN_DEFAULT_PORTS: port %d out of range [1, 65535]", p)
		}
	}

	return c, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func parsePorts(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	ports := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", p, err)
		}
		ports = append(ports, n)
	}
	return ports, nil
}
