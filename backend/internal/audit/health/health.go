package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Checker performs health checks against audit service dependencies.
type Checker struct {
	db           *pgxpool.Pool
	rdb          *redis.Client
	kafkaBrokers []string
	logger       zerolog.Logger
}

// NewChecker creates a new health Checker.
func NewChecker(db *pgxpool.Pool, rdb *redis.Client, kafkaBrokers []string, logger zerolog.Logger) *Checker {
	return &Checker{
		db:           db,
		rdb:          rdb,
		kafkaBrokers: kafkaBrokers,
		logger:       logger,
	}
}

// CheckResult holds the result of a single dependency check.
type CheckResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// HealthResponse is the response body for health endpoints.
type HealthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks,omitempty"`
}

// LivenessHandler returns a handler for GET /healthz.
// Always returns 200 if the process is running.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "alive"})
	}
}

// ReadinessHandler returns a handler for GET /readyz.
// Checks PostgreSQL, Redis, and Kafka connectivity.
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]CheckResult)
		allOK := true

		// Check PostgreSQL
		pgCheck := c.checkPostgres(r.Context())
		checks["postgres"] = pgCheck
		if !pgCheck.OK {
			allOK = false
		}

		// Check Redis
		redisCheck := c.checkRedis(r.Context())
		checks["redis"] = redisCheck
		if !redisCheck.OK {
			allOK = false
		}

		// Check Kafka
		kafkaCheck := c.checkKafka(r.Context())
		checks["kafka"] = kafkaCheck
		if !kafkaCheck.OK {
			allOK = false
		}

		status := "ready"
		httpStatus := http.StatusOK
		if !allOK {
			status = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(HealthResponse{
			Status: status,
			Checks: checks,
		})
	}
}

func (c *Checker) checkPostgres(ctx context.Context) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := c.db.Ping(checkCtx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{OK: false, LatencyMs: latency, Error: err.Error()}
	}
	return CheckResult{OK: true, LatencyMs: latency}
}

func (c *Checker) checkRedis(ctx context.Context) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := c.rdb.Ping(checkCtx).Err()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{OK: false, LatencyMs: latency, Error: err.Error()}
	}
	return CheckResult{OK: true, LatencyMs: latency}
}

func (c *Checker) checkKafka(ctx context.Context) CheckResult {
	if len(c.kafkaBrokers) == 0 {
		return CheckResult{OK: false, Error: "no brokers configured"}
	}

	start := time.Now()
	dialer := &net.Dialer{Timeout: 2 * time.Second}

	conn, err := dialer.DialContext(ctx, "tcp", c.kafkaBrokers[0])
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{OK: false, LatencyMs: latency, Error: fmt.Sprintf("broker %s: %s", c.kafkaBrokers[0], err.Error())}
	}
	conn.Close()

	return CheckResult{OK: true, LatencyMs: latency}
}
