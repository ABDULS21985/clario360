package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/workflow/expression"
	"github.com/clario360/platform/internal/workflow/model"
)

// ---------- Circuit Breaker ----------

const (
	cbStateClosed   = "closed"
	cbStateOpen     = "open"
	cbStateHalfOpen = "half_open"
)

// circuitBreaker is a simple per-service circuit breaker that protects downstream
// services from being overwhelmed when they are failing. It transitions through
// three states: closed (normal), open (failing), and half-open (probing).
type circuitBreaker struct {
	mu           sync.Mutex
	state        string
	failures     int
	successes    int
	maxFailures  int           // consecutive failures before opening (default 3)
	resetTimeout time.Duration // how long to stay open before probing (default 30s)
	lastFailure  time.Time
	halfOpenMax  int // successful requests needed to close again (default 5)
}

// newCircuitBreaker returns a circuit breaker with sensible defaults.
func newCircuitBreaker() *circuitBreaker {
	return &circuitBreaker{
		state:        cbStateClosed,
		maxFailures:  3,
		resetTimeout: 30 * time.Second,
		halfOpenMax:  5,
	}
}

// Allow returns true if a request should be allowed through.
// In the open state it checks whether the reset timeout has elapsed; if so it
// transitions to half-open and allows a probe request.
func (cb *circuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbStateClosed:
		return true
	case cbStateHalfOpen:
		return true
	case cbStateOpen:
		if time.Since(cb.lastFailure) >= cb.resetTimeout {
			cb.state = cbStateHalfOpen
			cb.successes = 0
			return true
		}
		return false
	default:
		return true
	}
}

// RecordSuccess records a successful request. In half-open state, after enough
// successes the breaker closes again.
func (cb *circuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbStateHalfOpen:
		cb.successes++
		if cb.successes >= cb.halfOpenMax {
			cb.state = cbStateClosed
			cb.failures = 0
			cb.successes = 0
		}
	case cbStateClosed:
		cb.failures = 0
	}
}

// RecordFailure records a failed request. In closed or half-open state, after
// enough failures the breaker opens.
func (cb *circuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()

	switch cb.state {
	case cbStateClosed:
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.state = cbStateOpen
		}
	case cbStateHalfOpen:
		// Any failure in half-open immediately re-opens.
		cb.state = cbStateOpen
		cb.failures = cb.maxFailures
	}
}

// ---------- Service Task Executor ----------

// ServiceTaskExecutor calls external HTTP services as part of a workflow step.
// It supports variable substitution in URLs and request bodies, per-service circuit
// breakers, and configurable retry with exponential backoff.
type ServiceTaskExecutor struct {
	httpClient  *http.Client
	serviceURLs map[string]string // service name -> base URL
	resolver    *expression.VariableResolver
	logger      zerolog.Logger
	breakers    map[string]*circuitBreaker
	mu          sync.RWMutex
}

// NewServiceTaskExecutor creates a ServiceTaskExecutor with the given service URL map.
func NewServiceTaskExecutor(serviceURLs map[string]string, logger zerolog.Logger) *ServiceTaskExecutor {
	return &ServiceTaskExecutor{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		serviceURLs: serviceURLs,
		resolver:    expression.NewVariableResolver(),
		logger:      logger.With().Str("executor", "service_task").Logger(),
		breakers:    make(map[string]*circuitBreaker),
	}
}

// getBreaker returns the circuit breaker for a service, creating one if needed.
func (e *ServiceTaskExecutor) getBreaker(service string) *circuitBreaker {
	e.mu.RLock()
	cb, ok := e.breakers[service]
	e.mu.RUnlock()
	if ok {
		return cb
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	// Double-check after acquiring write lock.
	if cb, ok = e.breakers[service]; ok {
		return cb
	}
	cb = newCircuitBreaker()
	e.breakers[service] = cb
	return cb
}

// Execute performs an HTTP service call based on the step configuration.
//
// Expected step.Config keys:
//   - service (string, required): service name to look up base URL
//   - method (string, required): HTTP method (GET, POST, PUT, DELETE, PATCH)
//   - url (string, required): URL path (appended to base URL), may contain ${...} references
//   - body (map, optional): request body with possible ${...} references
//   - headers (map[string]string, optional): additional HTTP headers
//   - timeout_seconds (float64, optional): per-request timeout, default 30s
//   - retry (map, optional): {"max_attempts": int, "backoff_ms": int}
func (e *ServiceTaskExecutor) Execute(ctx context.Context, instance *model.WorkflowInstance, step *model.StepDefinition, exec *model.StepExecution) (*ExecutionResult, error) {
	// Extract configuration.
	service, err := configString(step.Config, "service")
	if err != nil {
		return nil, fmt.Errorf("service_task %s: %w", step.ID, err)
	}
	method, err := configString(step.Config, "method")
	if err != nil {
		return nil, fmt.Errorf("service_task %s: %w", step.ID, err)
	}
	urlPath, err := configString(step.Config, "url")
	if err != nil {
		return nil, fmt.Errorf("service_task %s: %w", step.ID, err)
	}

	// Look up base URL.
	baseURL, ok := e.serviceURLs[service]
	if !ok {
		return nil, fmt.Errorf("service_task %s: unknown service %q", step.ID, service)
	}

	// Build data context for variable resolution.
	dataCtx := buildDataContext(instance)

	// Resolve variables in URL path.
	resolvedURL, err := e.resolver.Resolve(urlPath, dataCtx)
	if err != nil {
		return nil, fmt.Errorf("service_task %s: resolving url: %w", step.ID, err)
	}
	fullURL := baseURL + fmt.Sprintf("%v", resolvedURL)

	// Resolve variables in body if present.
	var bodyBytes []byte
	if bodyRaw, ok := step.Config["body"]; ok && bodyRaw != nil {
		resolvedBody, err := e.resolver.Resolve(bodyRaw, dataCtx)
		if err != nil {
			return nil, fmt.Errorf("service_task %s: resolving body: %w", step.ID, err)
		}
		bodyBytes, err = json.Marshal(resolvedBody)
		if err != nil {
			return nil, fmt.Errorf("service_task %s: marshaling body: %w", step.ID, err)
		}
	}

	// Parse retry config.
	maxAttempts := 1
	backoffMs := 500
	if retryRaw, ok := step.Config["retry"]; ok {
		if retryMap, ok := retryRaw.(map[string]interface{}); ok {
			if v, ok := retryMap["max_attempts"]; ok {
				maxAttempts = toInt(v)
			}
			if v, ok := retryMap["backoff_ms"]; ok {
				backoffMs = toInt(v)
			}
		}
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	// Parse per-request timeout.
	requestTimeout := 30 * time.Second
	if v, ok := step.Config["timeout_seconds"]; ok {
		if seconds := toFloat(v); seconds > 0 {
			requestTimeout = time.Duration(seconds * float64(time.Second))
		}
	}

	// Resolve optional headers.
	headers := make(map[string]string)
	if hdrsRaw, ok := step.Config["headers"]; ok {
		if hdrsMap, ok := hdrsRaw.(map[string]interface{}); ok {
			for k, v := range hdrsMap {
				resolved, err := e.resolver.Resolve(v, dataCtx)
				if err != nil {
					return nil, fmt.Errorf("service_task %s: resolving header %q: %w", step.ID, k, err)
				}
				headers[k] = fmt.Sprintf("%v", resolved)
			}
		}
	}

	// Check circuit breaker.
	cb := e.getBreaker(service)

	// Execute with retry.
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: backoff_ms * 2^(attempt-1), capped at 30s.
			delay := time.Duration(backoffMs) * time.Millisecond
			for i := 1; i < attempt; i++ {
				delay *= 2
			}
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}
			e.logger.Debug().
				Str("step_id", step.ID).
				Int("attempt", attempt+1).
				Dur("backoff", delay).
				Msg("retrying service call")
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("service_task %s: context cancelled during retry backoff: %w", step.ID, ctx.Err())
			case <-time.After(delay):
			}
		}

		if !cb.Allow() {
			lastErr = fmt.Errorf("service_task %s: circuit_open for service %q", step.ID, service)
			e.logger.Warn().
				Str("step_id", step.ID).
				Str("service", service).
				Msg("circuit breaker open, skipping request")
			continue
		}

		result, retryable, err := e.doRequest(ctx, method, fullURL, bodyBytes, headers, requestTimeout, step.ID)
		if err == nil {
			cb.RecordSuccess()
			e.logger.Info().
				Str("step_id", step.ID).
				Str("service", service).
				Str("method", method).
				Str("url", fullURL).
				Int("attempt", attempt+1).
				Msg("service call succeeded")
			return result, nil
		}

		lastErr = err
		if !retryable {
			// 4xx errors are not retryable.
			e.logger.Error().
				Err(err).
				Str("step_id", step.ID).
				Str("service", service).
				Int("attempt", attempt+1).
				Msg("service call failed with non-retryable error")
			return nil, err
		}

		// 5xx or transport error: record failure for circuit breaker.
		cb.RecordFailure()
		e.logger.Warn().
			Err(err).
			Str("step_id", step.ID).
			Str("service", service).
			Int("attempt", attempt+1).
			Int("max_attempts", maxAttempts).
			Msg("service call failed, will retry")
	}

	return nil, fmt.Errorf("service_task %s: all %d attempts exhausted: %w", step.ID, maxAttempts, lastErr)
}

// doRequest performs a single HTTP request. Returns the execution result on success,
// a retryable flag, and an error. Only 5xx responses are considered retryable.
func (e *ServiceTaskExecutor) doRequest(ctx context.Context, method, url string, body []byte, headers map[string]string, timeout time.Duration, stepID string) (*ExecutionResult, bool, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(reqCtx, method, url, bodyReader)
	if err != nil {
		return nil, false, fmt.Errorf("creating request: %w", err)
	}

	// Set default content type for requests with a body.
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		// Transport errors are retryable.
		return nil, true, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		return nil, true, fmt.Errorf("reading response body: %w", err)
	}

	// 2xx: success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		output := make(map[string]interface{})
		output["status_code"] = resp.StatusCode

		if len(respBody) > 0 {
			var parsed interface{}
			if err := json.Unmarshal(respBody, &parsed); err != nil {
				// Non-JSON response: store as raw string.
				output["response"] = string(respBody)
			} else {
				output["response"] = parsed
			}
		}

		return &ExecutionResult{Output: output}, false, nil
	}

	// 4xx: client error, not retryable
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, false, fmt.Errorf("service returned %d: %s", resp.StatusCode, string(respBody))
	}

	// 5xx: server error, retryable
	return nil, true, fmt.Errorf("service returned %d: %s", resp.StatusCode, string(respBody))
}

// ---------- helpers ----------

// buildDataContext creates the standard data context used for variable resolution
// from a workflow instance. The structure is:
//
//	{"variables": ..., "steps": ..., "trigger": {"data": ...}}
func buildDataContext(instance *model.WorkflowInstance) map[string]interface{} {
	ctx := map[string]interface{}{
		"variables": instance.Variables,
		"steps":     instance.StepOutputs,
	}

	// Parse trigger data if present.
	trigger := map[string]interface{}{}
	if len(instance.TriggerData) > 0 {
		var triggerData interface{}
		if err := json.Unmarshal(instance.TriggerData, &triggerData); err == nil {
			trigger["data"] = triggerData
		} else {
			trigger["data"] = map[string]interface{}{}
		}
	} else {
		trigger["data"] = map[string]interface{}{}
	}
	ctx["trigger"] = trigger

	return ctx
}

// configString extracts a required string value from a config map.
func configString(config map[string]interface{}, key string) (string, error) {
	v, ok := config[key]
	if !ok {
		return "", fmt.Errorf("missing required config key %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("config key %q must be a string, got %T", key, v)
	}
	return s, nil
}

// configStringOptional extracts an optional string value from a config map.
func configStringOptional(config map[string]interface{}, key string) string {
	v, ok := config[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// toInt converts a numeric interface{} value to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

// toFloat converts a numeric interface{} value to float64.
func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}
