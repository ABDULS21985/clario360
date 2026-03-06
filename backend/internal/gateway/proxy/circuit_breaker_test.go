package proxy

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StartsInClosedState(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed, got %s", cb.State())
	}
}

func TestCircuitBreaker_AllowsRequestsWhenClosed(t *testing.T) {
	cb := NewCircuitBreaker(DefaultCircuitBreakerConfig())
	if !cb.Allow() {
		t.Error("expected request to be allowed when closed")
	}
}

func TestCircuitBreaker_OpensAfterConsecutiveFailures(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 3
	cb := NewCircuitBreaker(cfg)

	for i := 0; i < 3; i++ {
		cb.Allow()
		cb.RecordFailure()
	}

	if cb.State() != CircuitOpen {
		t.Errorf("expected open after %d failures, got %s", cfg.FailureThreshold, cb.State())
	}
}

func TestCircuitBreaker_RejectsRequestsWhenOpen(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.OpenTimeout = 10 * time.Second
	cb := NewCircuitBreaker(cfg)

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	if cb.Allow() {
		t.Error("expected request to be rejected when open")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpenAfterTimeout(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.OpenTimeout = 1 * time.Millisecond
	cb := NewCircuitBreaker(cfg)

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	time.Sleep(5 * time.Millisecond)

	if !cb.Allow() {
		t.Error("expected request to be allowed in half-open state")
	}
	if cb.State() != CircuitHalfOpen {
		t.Errorf("expected half-open, got %s", cb.State())
	}
}

func TestCircuitBreaker_ClosesAfterHalfOpenSuccesses(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.OpenTimeout = 1 * time.Millisecond
	cfg.HalfOpenSuccesses = 2
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(5 * time.Millisecond)
	cb.Allow()

	// Record successes in half-open
	cb.RecordSuccess()
	cb.RecordSuccess()

	if cb.State() != CircuitClosed {
		t.Errorf("expected closed after half-open successes, got %s", cb.State())
	}
}

func TestCircuitBreaker_ReopensOnHalfOpenFailure(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 2
	cfg.OpenTimeout = 1 * time.Millisecond
	cb := NewCircuitBreaker(cfg)

	// Open the circuit
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()

	// Wait for half-open
	time.Sleep(5 * time.Millisecond)
	cb.Allow()

	// Fail in half-open
	cb.RecordFailure()

	if cb.State() != CircuitOpen {
		t.Errorf("expected open after half-open failure, got %s", cb.State())
	}
}

func TestCircuitBreaker_ResetsConsecutiveFailuresOnSuccess(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	cfg.FailureThreshold = 3
	cb := NewCircuitBreaker(cfg)

	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordFailure()
	cb.Allow()
	cb.RecordSuccess() // Reset

	cb.Allow()
	cb.RecordFailure()

	// Should still be closed (only 1 consecutive failure after reset)
	if cb.State() != CircuitClosed {
		t.Errorf("expected closed, got %s", cb.State())
	}
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "closed"},
		{CircuitHalfOpen, "half-open"},
		{CircuitOpen, "open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitState(%d).String() = %s, want %s", tt.state, got, tt.want)
		}
	}
}
