package proxy

import (
	"sync"
	"time"
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed   CircuitState = 0
	CircuitHalfOpen CircuitState = 1
	CircuitOpen     CircuitState = 2
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitHalfOpen:
		return "half-open"
	case CircuitOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig holds circuit breaker settings.
type CircuitBreakerConfig struct {
	FailureThreshold   int           // Consecutive failures to open (default: 5)
	FailureRateWindow  time.Duration // Window for failure rate calculation (default: 10s)
	FailureRatePercent float64       // Failure rate % to open (default: 50)
	OpenTimeout        time.Duration // Duration to stay open before half-open (default: 30s)
	HalfOpenSuccesses  int           // Successes in half-open to close (default: 3)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:   5,
		FailureRateWindow:  10 * time.Second,
		FailureRatePercent: 50,
		OpenTimeout:        30 * time.Second,
		HalfOpenSuccesses:  3,
	}
}

// CircuitBreaker implements the circuit breaker pattern for a single backend service.
type CircuitBreaker struct {
	mu                  sync.RWMutex
	state               CircuitState
	cfg                 CircuitBreakerConfig
	consecutiveFailures int
	consecutiveSuccess  int
	windowRequests      []requestResult
	lastStateChange     time.Time
}

type requestResult struct {
	at      time.Time
	success bool
}

// NewCircuitBreaker creates a new circuit breaker with the given config.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:           CircuitClosed,
		cfg:             cfg,
		lastStateChange: time.Now(),
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// Allow checks if a request should be allowed through.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastStateChange) >= cb.cfg.OpenTimeout {
			cb.transitionTo(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return false
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures = 0
	cb.windowRequests = append(cb.windowRequests, requestResult{at: time.Now(), success: true})

	state := cb.currentState()
	if state == CircuitHalfOpen {
		cb.consecutiveSuccess++
		if cb.consecutiveSuccess >= cb.cfg.HalfOpenSuccesses {
			cb.transitionTo(CircuitClosed)
		}
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures++
	cb.consecutiveSuccess = 0
	cb.windowRequests = append(cb.windowRequests, requestResult{at: time.Now(), success: false})

	state := cb.currentState()
	switch state {
	case CircuitClosed:
		if cb.consecutiveFailures >= cb.cfg.FailureThreshold || cb.failureRateExceeded() {
			cb.transitionTo(CircuitOpen)
		}
	case CircuitHalfOpen:
		cb.transitionTo(CircuitOpen)
	}
}

// currentState returns the effective state (may auto-transition open→half-open).
func (cb *CircuitBreaker) currentState() CircuitState {
	if cb.state == CircuitOpen && time.Since(cb.lastStateChange) >= cb.cfg.OpenTimeout {
		cb.transitionTo(CircuitHalfOpen)
	}
	return cb.state
}

func (cb *CircuitBreaker) transitionTo(state CircuitState) {
	cb.state = state
	cb.lastStateChange = time.Now()
	if state == CircuitClosed {
		cb.consecutiveFailures = 0
		cb.consecutiveSuccess = 0
		cb.windowRequests = nil
	} else if state == CircuitHalfOpen {
		cb.consecutiveSuccess = 0
	}
}

// failureRateExceeded checks if the failure rate in the window exceeds the threshold.
func (cb *CircuitBreaker) failureRateExceeded() bool {
	cutoff := time.Now().Add(-cb.cfg.FailureRateWindow)

	// Prune old entries
	recent := cb.windowRequests[:0]
	for _, r := range cb.windowRequests {
		if r.at.After(cutoff) {
			recent = append(recent, r)
		}
	}
	cb.windowRequests = recent

	if len(recent) < 5 {
		// Need at least 5 requests to evaluate rate
		return false
	}

	failures := 0
	for _, r := range recent {
		if !r.success {
			failures++
		}
	}

	rate := float64(failures) / float64(len(recent)) * 100
	return rate >= cb.cfg.FailureRatePercent
}
