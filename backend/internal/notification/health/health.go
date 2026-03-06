package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Checker performs health checks for the notification service dependencies.
type Checker struct {
	db           *pgxpool.Pool
	rdb          *redis.Client
	kafkaBrokers []string
	smtpAddr     string
	logger       zerolog.Logger
}

// NewChecker creates a new health Checker.
func NewChecker(db *pgxpool.Pool, rdb *redis.Client, kafkaBrokers []string, smtpAddr string, logger zerolog.Logger) *Checker {
	return &Checker{
		db:           db,
		rdb:          rdb,
		kafkaBrokers: kafkaBrokers,
		smtpAddr:     smtpAddr,
		logger:       logger.With().Str("component", "health").Logger(),
	}
}

// LivenessHandler returns a simple liveness probe.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	}
}

// ReadinessHandler checks all dependencies.
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{}
		healthy := true

		// PostgreSQL
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		if err := c.db.Ping(ctx); err != nil {
			checks["postgres"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["postgres"] = "ok"
		}
		cancel()

		// Redis
		ctx, cancel = context.WithTimeout(r.Context(), 2*time.Second)
		if err := c.rdb.Ping(ctx).Err(); err != nil {
			checks["redis"] = "unhealthy: " + err.Error()
			healthy = false
		} else {
			checks["redis"] = "ok"
		}
		cancel()

		// Kafka (TCP dial check)
		kafkaOK := false
		for _, broker := range c.kafkaBrokers {
			broker = strings.TrimSpace(broker)
			if broker == "" {
				continue
			}
			conn, err := net.DialTimeout("tcp", broker, 2*time.Second)
			if err == nil {
				conn.Close()
				kafkaOK = true
				break
			}
		}
		if kafkaOK {
			checks["kafka"] = "ok"
		} else {
			checks["kafka"] = fmt.Sprintf("unhealthy: cannot reach any broker in %v", c.kafkaBrokers)
			healthy = false
		}

		// SMTP (optional connectivity check)
		if c.smtpAddr != "" {
			conn, err := net.DialTimeout("tcp", c.smtpAddr, 2*time.Second)
			if err != nil {
				checks["smtp"] = "unhealthy: " + err.Error()
				// SMTP failure is degraded but not critical
			} else {
				conn.Close()
				checks["smtp"] = "ok"
			}
		}

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": map[bool]string{true: "ready", false: "not_ready"}[healthy],
			"checks": checks,
		})
	}
}
