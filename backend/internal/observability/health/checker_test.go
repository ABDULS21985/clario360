package health

import (
	"context"
	"testing"
	"time"
)

// mockChecker is a simple HealthChecker for testing.
type mockChecker struct {
	name   string
	status string
	delay  time.Duration
	err    string
}

func (m *mockChecker) Name() string { return m.name }

func (m *mockChecker) Check(ctx context.Context) HealthResult {
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return HealthResult{
				Status: "unhealthy",
				Error:  ctx.Err().Error(),
			}
		}
	}
	return HealthResult{
		Status: m.status,
		Error:  m.err,
	}
}

func TestCompositeHealthChecker_AllHealthy(t *testing.T) {
	checkers := []HealthChecker{
		&mockChecker{name: "postgres", status: "healthy"},
		&mockChecker{name: "redis", status: "healthy"},
		&mockChecker{name: "kafka", status: "healthy"},
	}

	composite := NewCompositeHealthChecker(2*time.Second, checkers...)
	result := composite.CheckAll(context.Background())

	if result.Status != "healthy" {
		t.Errorf("overall status = %q, want %q", result.Status, "healthy")
	}
	if len(result.Checks) != 3 {
		t.Errorf("expected 3 check results, got %d", len(result.Checks))
	}

	for name, check := range result.Checks {
		if check.Status != "healthy" {
			t.Errorf("check %q status = %q, want %q", name, check.Status, "healthy")
		}
	}
}

func TestCompositeHealthChecker_OneDegraded(t *testing.T) {
	checkers := []HealthChecker{
		&mockChecker{name: "postgres", status: "healthy"},
		&mockChecker{name: "redis", status: "degraded"},
		&mockChecker{name: "kafka", status: "healthy"},
	}

	composite := NewCompositeHealthChecker(2*time.Second, checkers...)
	result := composite.CheckAll(context.Background())

	if result.Status != "degraded" {
		t.Errorf("overall status = %q, want %q", result.Status, "degraded")
	}

	if result.Checks["redis"].Status != "degraded" {
		t.Errorf("redis check status = %q, want %q", result.Checks["redis"].Status, "degraded")
	}
}

func TestCompositeHealthChecker_OneUnhealthy(t *testing.T) {
	checkers := []HealthChecker{
		&mockChecker{name: "postgres", status: "healthy"},
		&mockChecker{name: "redis", status: "degraded"},
		&mockChecker{name: "kafka", status: "unhealthy", err: "connection refused"},
	}

	composite := NewCompositeHealthChecker(2*time.Second, checkers...)
	result := composite.CheckAll(context.Background())

	if result.Status != "unhealthy" {
		t.Errorf("overall status = %q, want %q", result.Status, "unhealthy")
	}

	if result.Checks["kafka"].Status != "unhealthy" {
		t.Errorf("kafka check status = %q, want %q", result.Checks["kafka"].Status, "unhealthy")
	}
	if result.Checks["kafka"].Error != "connection refused" {
		t.Errorf("kafka check error = %q, want %q", result.Checks["kafka"].Error, "connection refused")
	}
}

func TestCompositeHealthChecker_Timeout(t *testing.T) {
	checkers := []HealthChecker{
		&mockChecker{name: "postgres", status: "healthy"},
		&mockChecker{name: "slow-service", status: "healthy", delay: 5 * time.Second},
	}

	// Use a short timeout so the slow checker times out.
	composite := NewCompositeHealthChecker(100*time.Millisecond, checkers...)
	result := composite.CheckAll(context.Background())

	if result.Status != "unhealthy" {
		t.Errorf("overall status = %q, want %q (slow checker should timeout)", result.Status, "unhealthy")
	}

	slowCheck, ok := result.Checks["slow-service"]
	if !ok {
		t.Fatal("missing result for 'slow-service'")
	}
	if slowCheck.Status != "unhealthy" {
		t.Errorf("slow-service status = %q, want %q", slowCheck.Status, "unhealthy")
	}
}

func TestCompositeHealthChecker_Concurrent(t *testing.T) {
	// Each checker takes 100ms. If they run concurrently, total time should be
	// approximately 100ms, not 300ms.
	delay := 100 * time.Millisecond
	checkers := []HealthChecker{
		&mockChecker{name: "postgres", status: "healthy", delay: delay},
		&mockChecker{name: "redis", status: "healthy", delay: delay},
		&mockChecker{name: "kafka", status: "healthy", delay: delay},
	}

	composite := NewCompositeHealthChecker(2*time.Second, checkers...)

	start := time.Now()
	result := composite.CheckAll(context.Background())
	elapsed := time.Since(start)

	if result.Status != "healthy" {
		t.Errorf("overall status = %q, want %q", result.Status, "healthy")
	}

	// If executed sequentially, this would take >= 300ms.
	// With concurrency, it should complete in roughly 100-150ms.
	// Use 250ms as upper bound to allow for scheduling jitter.
	if elapsed >= 250*time.Millisecond {
		t.Errorf("checks took %v, expected < 250ms (checks should run concurrently)", elapsed)
	}
}

func TestCompositeHealthChecker_DefaultTimeout(t *testing.T) {
	composite := NewCompositeHealthChecker(0)
	// Should not panic and should use 2s default.
	result := composite.CheckAll(context.Background())

	if result.Status != "healthy" {
		t.Errorf("overall status = %q, want %q (no checkers means healthy)", result.Status, "healthy")
	}
}
