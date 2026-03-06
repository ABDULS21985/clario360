package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// CheckResult holds the result of a single dependency health check.
type CheckResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

// HealthResponse is the JSON response body for health endpoints.
type HealthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]CheckResult `json:"checks,omitempty"`
}

// Checker performs health checks against workflow engine dependencies.
type Checker struct {
	pool   *pgxpool.Pool
	rdb    *redis.Client
	logger zerolog.Logger
}

// NewChecker creates a new health Checker.
func NewChecker(pool *pgxpool.Pool, rdb *redis.Client, logger zerolog.Logger) *Checker {
	return &Checker{
		pool:   pool,
		rdb:    rdb,
		logger: logger,
	}
}

// CheckPostgres pings the PostgreSQL connection pool and returns the result.
func CheckPostgres(ctx context.Context, pool *pgxpool.Pool) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := pool.Ping(checkCtx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{OK: false, LatencyMs: latency, Error: err.Error()}
	}
	return CheckResult{OK: true, LatencyMs: latency}
}

// CheckRedis pings the Redis client and returns the result.
func CheckRedis(ctx context.Context, rdb *redis.Client) CheckResult {
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := rdb.Ping(checkCtx).Err()
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return CheckResult{OK: false, LatencyMs: latency, Error: err.Error()}
	}
	return CheckResult{OK: true, LatencyMs: latency}
}

// HealthHandler returns an http.HandlerFunc for the /healthz liveness endpoint.
// Liveness checks only confirm the process is running; they do not probe
// external dependencies.
func HealthHandler(pool *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "alive"})
	}
}

// ReadyHandler returns an http.HandlerFunc for the /readyz readiness endpoint.
// Readiness checks probe all required dependencies (PostgreSQL, Redis) and
// return 503 Service Unavailable if any check fails.
func ReadyHandler(pool *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]CheckResult)
		allOK := true

		pgCheck := CheckPostgres(r.Context(), pool)
		checks["postgres"] = pgCheck
		if !pgCheck.OK {
			allOK = false
		}

		redisCheck := CheckRedis(r.Context(), rdb)
		checks["redis"] = redisCheck
		if !redisCheck.OK {
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

// LivenessHandler is an alias for HealthHandler, provided for naming
// consistency with Kubernetes probe conventions.
func (c *Checker) LivenessHandler() http.HandlerFunc {
	return HealthHandler(c.pool, c.rdb)
}

// ReadinessHandler delegates to ReadyHandler using the Checker's dependencies.
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return ReadyHandler(c.pool, c.rdb)
}
