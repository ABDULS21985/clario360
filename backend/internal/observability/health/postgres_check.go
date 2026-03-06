package health

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresHealthChecker pings the database and reports pool statistics.
type PostgresHealthChecker struct {
	pool *pgxpool.Pool
}

// NewPostgresHealthChecker creates a PostgreSQL health checker.
func NewPostgresHealthChecker(pool *pgxpool.Pool) *PostgresHealthChecker {
	return &PostgresHealthChecker{pool: pool}
}

// Name returns "postgres".
func (h *PostgresHealthChecker) Name() string { return "postgres" }

// Check pings the database and checks connection pool utilization.
//
// Logic:
//  1. pool.Ping — if error → "unhealthy"
//  2. pool.Stat — if acquired >= 90% of max → "degraded"
//  3. Otherwise → "healthy"
func (h *PostgresHealthChecker) Check(ctx context.Context) HealthResult {
	if err := h.pool.Ping(ctx); err != nil {
		return HealthResult{
			Status: "unhealthy",
			Error:  fmt.Sprintf("ping failed: %s", err.Error()),
		}
	}

	stat := h.pool.Stat()
	details := map[string]interface{}{
		"active_connections": stat.AcquiredConns(),
		"idle_connections":   stat.IdleConns(),
		"max_connections":    stat.MaxConns(),
		"total_connections":  stat.TotalConns(),
	}

	// Check pool utilization: if >= 90% of max, report degraded.
	if stat.MaxConns() > 0 && float64(stat.TotalConns()) >= 0.9*float64(stat.MaxConns()) {
		return HealthResult{
			Status:  "degraded",
			Details: details,
			Error:   "connection pool utilization >= 90%",
		}
	}

	return HealthResult{
		Status:  "healthy",
		Details: details,
	}
}
