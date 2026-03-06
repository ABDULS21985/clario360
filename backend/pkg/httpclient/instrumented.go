package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/clario360/platform/internal/observability/metrics"
)

// InstrumentedClient is an HTTP client with OTel trace propagation and Prometheus metrics.
type InstrumentedClient struct {
	client  *http.Client
	tracer  trace.Tracer
	metrics *instrumentedClientMetrics
	logger  zerolog.Logger
}

type instrumentedClientMetrics struct {
	requestDuration *metrics.Metrics
	serviceName     string
}

// InstrumentedClientOption configures the instrumented client.
type InstrumentedClientOption func(*instrumentedClientConfig)

type instrumentedClientConfig struct {
	timeout         time.Duration
	maxIdleConns    int
	idleConnTimeout time.Duration
}

// WithInstrumentedTimeout sets the HTTP client timeout.
func WithInstrumentedTimeout(d time.Duration) InstrumentedClientOption {
	return func(c *instrumentedClientConfig) { c.timeout = d }
}

// WithMaxIdleConnsOption sets the max idle connections.
func WithMaxIdleConnsOption(n int) InstrumentedClientOption {
	return func(c *instrumentedClientConfig) { c.maxIdleConns = n }
}

// WithIdleConnTimeoutOption sets the idle connection timeout.
func WithIdleConnTimeoutOption(d time.Duration) InstrumentedClientOption {
	return func(c *instrumentedClientConfig) { c.idleConnTimeout = d }
}

// NewInstrumentedClient creates an HTTP client instrumented with OTel tracing and Prometheus metrics.
func NewInstrumentedClient(m *metrics.Metrics, logger zerolog.Logger, opts ...InstrumentedClientOption) *InstrumentedClient {
	cfg := &instrumentedClientConfig{
		timeout:         30 * time.Second,
		maxIdleConns:    100,
		idleConnTimeout: 90 * time.Second,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.maxIdleConns,
		IdleConnTimeout:     cfg.idleConnTimeout,
		MaxIdleConnsPerHost: 10,
	}

	return &InstrumentedClient{
		client: &http.Client{
			Timeout:   cfg.timeout,
			Transport: transport,
		},
		tracer: otel.Tracer("httpclient"),
		metrics: &instrumentedClientMetrics{
			requestDuration: m,
		},
		logger: logger,
	}
}

// Do executes an HTTP request with trace propagation and metrics.
func (c *InstrumentedClient) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Start span.
	host := req.URL.Host
	spanName := fmt.Sprintf("http.client %s %s", req.Method, host)

	ctx, span := c.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", redactURL(req.URL)),
			attribute.String("net.peer.name", host),
		),
	)
	defer span.End()

	// Inject W3C TraceContext headers into outbound request.
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	req = req.WithContext(ctx)

	start := time.Now()
	resp, err := c.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.logger.Warn().
			Err(err).
			Str("method", req.Method).
			Str("host", host).
			Dur("duration", duration).
			Msg("outbound HTTP request failed")
		return nil, err
	}

	statusCode := strconv.Itoa(resp.StatusCode)
	span.SetAttributes(
		attribute.Int("http.status_code", resp.StatusCode),
		attribute.Int64("http.response_content_length", resp.ContentLength),
	)

	if resp.StatusCode >= 500 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", resp.StatusCode))
	}

	c.logger.Debug().
		Str("method", req.Method).
		Str("host", host).
		Str("status", statusCode).
		Dur("duration", duration).
		Msg("outbound HTTP request completed")

	return resp, nil
}

// Get performs an instrumented HTTP GET request.
func (c *InstrumentedClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	return c.Do(ctx, req)
}

// StandardClient returns the underlying *http.Client for use with libraries
// that require a standard client (e.g., gRPC, third-party SDKs).
func (c *InstrumentedClient) StandardClient() *http.Client {
	return c.client
}

// redactURL removes query parameters from the URL to prevent logging tokens/PII.
// Returns only scheme://host/path.
func redactURL(u *url.URL) string {
	clean := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}
	return clean.String()
}
