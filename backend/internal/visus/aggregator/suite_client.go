package aggregator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	visusmetrics "github.com/clario360/platform/internal/visus/metrics"
)

type FetchMetadata struct {
	Status      string
	Latency     time.Duration
	Error       error
	LastSuccess time.Time
}

type CircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	openUntil   time.Time
	lastSuccess time.Time
}

func (c *CircuitBreaker) IsOpen(now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.openUntil.IsZero() && now.Before(c.openUntil)
}

func (c *CircuitBreaker) OnSuccess(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = 0
	c.openUntil = time.Time{}
	c.lastSuccess = now
}

func (c *CircuitBreaker) OnFailure(now time.Time, threshold int, reset time.Duration) (opened bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures++
	if c.failures >= threshold {
		c.openUntil = now.Add(reset)
		opened = true
	}
	return opened
}

func (c *CircuitBreaker) LastSuccess() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastSuccess
}

type ServiceTokenProvider struct {
	staticToken string
	jwtMgr      *auth.JWTManager
	userID      string
	email       string
	tokenTTL    time.Duration
	mu          sync.Mutex
	cache       map[uuid.UUID]cachedToken
}

type cachedToken struct {
	token     string
	expiresAt time.Time
}

func NewServiceTokenProvider(staticToken string, jwtMgr *auth.JWTManager, userID, email string, tokenTTL time.Duration) *ServiceTokenProvider {
	if tokenTTL <= 0 {
		tokenTTL = 10 * time.Minute
	}
	return &ServiceTokenProvider{
		staticToken: strings.TrimSpace(staticToken),
		jwtMgr:      jwtMgr,
		userID:      userID,
		email:       email,
		tokenTTL:    tokenTTL,
		cache:       map[uuid.UUID]cachedToken{},
	}
}

func (p *ServiceTokenProvider) Token(ctx context.Context, tenantID uuid.UUID) (string, error) {
	_ = ctx
	if p.staticToken != "" {
		return p.staticToken, nil
	}
	if p.jwtMgr == nil {
		return "", fmt.Errorf("service account token not configured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if existing, ok := p.cache[tenantID]; ok && time.Until(existing.expiresAt) > time.Minute {
		return existing.token, nil
	}
	pair, err := p.jwtMgr.GenerateTokenPair(p.userID, tenantID.String(), p.email, []string{"service:visus"})
	if err != nil {
		return "", fmt.Errorf("generate service account token: %w", err)
	}
	p.cache[tenantID] = cachedToken{
		token:     pair.AccessToken,
		expiresAt: pair.ExpiresAt,
	}
	return pair.AccessToken, nil
}

type SuiteClient struct {
	httpClient       *http.Client
	baseURLs         map[string]string
	tokenProvider    *ServiceTokenProvider
	circuitBreakers  map[string]*CircuitBreaker
	cache            *SuiteCache
	timeout          time.Duration
	maxRetries       int
	circuitThreshold int
	circuitReset     time.Duration
	logger           zerolog.Logger
	metrics          *visusmetrics.Metrics
}

func NewSuiteClient(baseURLs map[string]string, tokenProvider *ServiceTokenProvider, cache *SuiteCache, timeout time.Duration, maxRetries, circuitThreshold int, circuitReset time.Duration, metrics *visusmetrics.Metrics, logger zerolog.Logger) *SuiteClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if maxRetries < 1 {
		maxRetries = 3
	}
	if circuitThreshold < 1 {
		circuitThreshold = 5
	}
	if circuitReset <= 0 {
		circuitReset = time.Minute
	}
	breakers := make(map[string]*CircuitBreaker, len(baseURLs))
	for suite := range baseURLs {
		breakers[suite] = &CircuitBreaker{}
	}
	return &SuiteClient{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURLs:         baseURLs,
		tokenProvider:    tokenProvider,
		circuitBreakers:  breakers,
		cache:            cache,
		timeout:          timeout,
		maxRetries:       maxRetries,
		circuitThreshold: circuitThreshold,
		circuitReset:     circuitReset,
		logger:           logger.With().Str("component", "visus_suite_client").Logger(),
		metrics:          metrics,
	}
}

func (c *SuiteClient) Fetch(ctx context.Context, suite, endpoint string, tenantID uuid.UUID, target interface{}) FetchMetadata {
	baseURL, ok := c.baseURLs[suite]
	if !ok || strings.TrimSpace(baseURL) == "" {
		return FetchMetadata{Status: "unavailable", Error: fmt.Errorf("suite %q not configured", suite)}
	}
	breaker := c.circuitBreakers[suite]
	now := time.Now().UTC()
	if breaker != nil && breaker.IsOpen(now) {
		if c.metrics != nil && c.metrics.SuiteCircuitBreakerState != nil {
			c.metrics.SuiteCircuitBreakerState.WithLabelValues(suite).Set(1)
		}
		c.logger.Warn().Str("suite", suite).Msg("circuit open for suite, using cache")
		if payload, ok, err := c.cache.Get(ctx, tenantID, suite, endpoint); err == nil && ok {
			if decodeErr := decodeIntoTarget(payload, target); decodeErr == nil {
				if c.metrics != nil && c.metrics.SuiteFetchTotal != nil {
					c.metrics.SuiteFetchTotal.WithLabelValues(suite, "cached").Inc()
				}
				return FetchMetadata{Status: "cached", LastSuccess: breaker.LastSuccess()}
			}
		}
		return FetchMetadata{Status: "unavailable", LastSuccess: breaker.LastSuccess(), Error: fmt.Errorf("circuit open")}
	}
	if c.metrics != nil && c.metrics.SuiteCircuitBreakerState != nil {
		c.metrics.SuiteCircuitBreakerState.WithLabelValues(suite).Set(0)
	}

	endpoint = normalizeEndpoint(suite, endpoint)
	token, err := c.tokenProvider.Token(ctx, tenantID)
	if err != nil {
		return FetchMetadata{Status: "unavailable", Error: err}
	}

	var lastErr error
	start := time.Now()
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		latencyStart := time.Now()
		payload, fetchErr := c.fetchOnce(ctx, baseURL, endpoint, token, tenantID)
		latency := time.Since(latencyStart)
		if c.metrics != nil && c.metrics.SuiteFetchDurationSeconds != nil {
			c.metrics.SuiteFetchDurationSeconds.WithLabelValues(suite).Observe(latency.Seconds())
		}
		if fetchErr == nil {
			if err := decodeIntoTarget(payload, target); err != nil {
				lastErr = err
				break
			}
			if c.cache != nil {
				_ = c.cache.Set(ctx, tenantID, suite, endpoint, payload, latency)
			}
			if breaker != nil {
				breaker.OnSuccess(time.Now().UTC())
			}
			if c.metrics != nil && c.metrics.SuiteFetchTotal != nil {
				c.metrics.SuiteFetchTotal.WithLabelValues(suite, "success").Inc()
			}
			return FetchMetadata{Status: "live", Latency: time.Since(start), LastSuccess: time.Now().UTC()}
		}
		lastErr = fetchErr
		if attempt < c.maxRetries-1 {
			backoff := time.Second
			if attempt == 1 {
				backoff = 2 * time.Second
			}
			select {
			case <-ctx.Done():
				return FetchMetadata{Status: "unavailable", Error: ctx.Err()}
			case <-time.After(backoff):
			}
		}
	}

	if breaker != nil {
		opened := breaker.OnFailure(time.Now().UTC(), c.circuitThreshold, c.circuitReset)
		if opened {
			c.logger.Warn().Str("suite", suite).Err(lastErr).Msg("suite circuit opened after repeated failures")
			if c.metrics != nil && c.metrics.SuiteCircuitBreakerState != nil {
				c.metrics.SuiteCircuitBreakerState.WithLabelValues(suite).Set(1)
			}
		}
	}
	if c.metrics != nil && c.metrics.SuiteFetchTotal != nil {
		c.metrics.SuiteFetchTotal.WithLabelValues(suite, "failure").Inc()
	}
	if payload, ok, err := c.cache.Get(ctx, tenantID, suite, endpoint); err == nil && ok {
		if decodeErr := decodeIntoTarget(payload, target); decodeErr == nil {
			if c.metrics != nil && c.metrics.SuiteFetchTotal != nil {
				c.metrics.SuiteFetchTotal.WithLabelValues(suite, "cached").Inc()
			}
			return FetchMetadata{Status: "cached", Latency: time.Since(start), LastSuccess: breaker.LastSuccess(), Error: lastErr}
		}
	}
	return FetchMetadata{Status: "unavailable", Latency: time.Since(start), LastSuccess: breaker.LastSuccess(), Error: lastErr}
}

func (c *SuiteClient) fetchOnce(ctx context.Context, baseURL, endpoint, token string, tenantID uuid.UUID) (map[string]any, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/") + endpoint)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Tenant-ID", tenantID.String())
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("suite request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("suite request rejected with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func normalizeEndpoint(suite, endpoint string) string {
	if strings.TrimSpace(endpoint) == "" {
		return "/api/v1/" + suite
	}
	if strings.HasPrefix(endpoint, "/api/") {
		return endpoint
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	return "/api/v1/" + suite + endpoint
}

func decodeIntoTarget(payload map[string]any, target interface{}) error {
	if target == nil {
		return nil
	}
	switch typed := target.(type) {
	case *map[string]any:
		*typed = payload
		return nil
	default:
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		return json.Unmarshal(raw, target)
	}
}

func IsUnavailable(err error) bool {
	return err != nil && !errors.Is(err, context.Canceled)
}
