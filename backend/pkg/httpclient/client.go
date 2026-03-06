package httpclient

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// Config holds HTTP client configuration.
type Config struct {
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

// DefaultConfig returns sensible defaults for an instrumented HTTP client.
func DefaultConfig() Config {
	return Config{
		Timeout:        30 * time.Second,
		MaxRetries:     3,
		RetryBaseDelay: 200 * time.Millisecond,
		RetryMaxDelay:  5 * time.Second,
	}
}

// Client is an instrumented HTTP client with retries and circuit breaker.
type Client struct {
	client *http.Client
	config Config
	logger zerolog.Logger

	// Circuit breaker state
	failures    int
	lastFailure time.Time
	cbThreshold int
	cbTimeout   time.Duration
}

// New creates a new instrumented HTTP client.
func New(cfg Config, logger zerolog.Logger) *Client {
	return &Client{
		client: &http.Client{
			Timeout: cfg.Timeout,
		},
		config:      cfg,
		logger:      logger,
		cbThreshold: 5,
		cbTimeout:   30 * time.Second,
	}
}

// Do executes an HTTP request with retries and circuit breaker logic.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Circuit breaker check
	if c.isCircuitOpen() {
		return nil, fmt.Errorf("circuit breaker is open, request blocked")
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := c.backoffDelay(attempt)
			c.logger.Debug().
				Int("attempt", attempt).
				Dur("delay", delay).
				Str("url", req.URL.String()).
				Msg("retrying request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		start := time.Now()
		resp, err = c.client.Do(req.WithContext(ctx))
		duration := time.Since(start)

		if err != nil {
			c.recordFailure()
			c.logger.Warn().
				Err(err).
				Str("method", req.Method).
				Str("url", req.URL.String()).
				Dur("duration", duration).
				Int("attempt", attempt).
				Msg("request failed")
			continue
		}

		// Don't retry on non-retryable status codes
		if !isRetryable(resp.StatusCode) {
			c.recordSuccess()
			c.logger.Debug().
				Str("method", req.Method).
				Str("url", req.URL.String()).
				Int("status", resp.StatusCode).
				Dur("duration", duration).
				Msg("request completed")
			return resp, nil
		}

		// Close body before retry
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		c.recordFailure()
		c.logger.Warn().
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Int("status", resp.StatusCode).
			Dur("duration", duration).
			Int("attempt", attempt).
			Msg("retryable status code")
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, err)
	}
	return resp, nil
}

// Get performs an HTTP GET request.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	return c.Do(ctx, req)
}

func (c *Client) backoffDelay(attempt int) time.Duration {
	delay := time.Duration(float64(c.config.RetryBaseDelay) * math.Pow(2, float64(attempt-1)))
	if delay > c.config.RetryMaxDelay {
		delay = c.config.RetryMaxDelay
	}
	return delay
}

func (c *Client) isCircuitOpen() bool {
	if c.failures >= c.cbThreshold {
		if time.Since(c.lastFailure) < c.cbTimeout {
			return true
		}
		// Reset circuit after timeout
		c.failures = 0
	}
	return false
}

func (c *Client) recordFailure() {
	c.failures++
	c.lastFailure = time.Now()
}

func (c *Client) recordSuccess() {
	c.failures = 0
}

func isRetryable(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
