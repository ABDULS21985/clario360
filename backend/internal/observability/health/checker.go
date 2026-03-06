package health

import (
	"context"
	"sync"
	"time"
)

// HealthChecker is the interface each dependency probe must implement.
type HealthChecker interface {
	// Name returns the dependency name (e.g., "postgres", "redis", "kafka").
	Name() string
	// Check probes the dependency and returns a result.
	Check(ctx context.Context) HealthResult
}

// HealthResult describes the outcome of a single health check.
type HealthResult struct {
	Status    string                 `json:"status"`              // "healthy", "degraded", "unhealthy"
	LatencyMs int64                  `json:"latency_ms"`
	Error     string                 `json:"error,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// CompositeResult is the aggregated result of all health checks.
type CompositeResult struct {
	Status string                   `json:"status"`
	Checks map[string]HealthResult  `json:"checks"`
}

// CompositeHealthChecker runs multiple health checks concurrently.
type CompositeHealthChecker struct {
	checkers []HealthChecker
	timeout  time.Duration
}

// NewCompositeHealthChecker creates a composite checker with a per-check timeout.
// Default timeout is 2 seconds if zero is provided.
func NewCompositeHealthChecker(timeout time.Duration, checkers ...HealthChecker) *CompositeHealthChecker {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &CompositeHealthChecker{
		checkers: checkers,
		timeout:  timeout,
	}
}

// CheckAll runs ALL checks concurrently, each with its own timeout.
// Returns overall status:
//   - "healthy"   if all checks are healthy
//   - "degraded"  if any check is degraded but none are unhealthy
//   - "unhealthy" if any check is unhealthy
func (c *CompositeHealthChecker) CheckAll(ctx context.Context) CompositeResult {
	results := make(map[string]HealthResult, len(c.checkers))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, checker := range c.checkers {
		checker := checker
		wg.Add(1)
		go func() {
			defer wg.Done()

			checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
			defer cancel()

			start := time.Now()
			result := checker.Check(checkCtx)
			result.LatencyMs = time.Since(start).Milliseconds()

			mu.Lock()
			results[checker.Name()] = result
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Determine overall status.
	overall := "healthy"
	for _, r := range results {
		switch r.Status {
		case "unhealthy":
			overall = "unhealthy"
		case "degraded":
			if overall != "unhealthy" {
				overall = "degraded"
			}
		}
	}

	return CompositeResult{
		Status: overall,
		Checks: results,
	}
}
