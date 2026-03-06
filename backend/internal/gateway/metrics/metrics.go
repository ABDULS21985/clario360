package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GatewayMetrics holds all Prometheus metrics for the API gateway.
// IMPORTANT: tenant_id is intentionally NOT a label on any metric. With potentially
// thousands of tenants this would cause cardinality explosion and OOM in Prometheus.
// Per-tenant analytics should use log aggregation (ELK/Loki) instead.
type GatewayMetrics struct {
	// Request throughput
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	RequestSize     *prometheus.HistogramVec
	ResponseSize    *prometheus.HistogramVec
	ActiveRequests  *prometheus.GaugeVec

	// Circuit breaker
	CircuitBreakerState  *prometheus.GaugeVec
	CircuitBreakerTrips  *prometheus.CounterVec

	// Rate limiting
	RateLimitExceeded    *prometheus.CounterVec
	RateLimitRedisErrors prometheus.Counter

	// WebSocket
	WebSocketConnectionsActive *prometheus.GaugeVec
	WebSocketConnectionsTotal  *prometheus.CounterVec
	WebSocketDuration          *prometheus.HistogramVec

	// Auth
	AuthFailures *prometheus.CounterVec

	// Upstream errors
	UpstreamErrors *prometheus.CounterVec
}

// NewGatewayMetrics creates and registers all gateway Prometheus metrics.
func NewGatewayMetrics() *GatewayMetrics {
	return &GatewayMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gw_requests_total",
				Help: "Total gateway requests by service, method, and status code.",
			},
			[]string{"service", "method", "status_code"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gw_request_duration_seconds",
				Help:    "Request latency through the gateway.",
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
			},
			[]string{"service", "method"},
		),
		RequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gw_request_size_bytes",
				Help:    "Request body size in bytes.",
				Buckets: []float64{100, 1000, 10_000, 100_000, 1_000_000, 10_000_000},
			},
			[]string{"service"},
		),
		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gw_response_size_bytes",
				Help:    "Response body size in bytes.",
				Buckets: []float64{100, 1000, 10_000, 100_000, 1_000_000, 10_000_000},
			},
			[]string{"service"},
		),
		ActiveRequests: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gw_active_requests",
				Help: "Number of active in-flight requests per backend service.",
			},
			[]string{"service"},
		),
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gw_circuit_breaker_state",
				Help: "Circuit breaker state per service (0=closed, 1=half-open, 2=open).",
			},
			[]string{"service"},
		),
		CircuitBreakerTrips: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gw_circuit_breaker_trips_total",
				Help: "Number of times a circuit breaker has opened.",
			},
			[]string{"service"},
		),
		RateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gw_ratelimit_exceeded_total",
				Help: "Number of requests rejected due to rate limiting.",
			},
			[]string{"group"},
		),
		RateLimitRedisErrors: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "gw_ratelimit_redis_failures_total",
				Help: "Number of Redis failures during rate limit checks (fail-open).",
			},
		),
		WebSocketConnectionsActive: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gw_websocket_connections_active",
				Help: "Number of active WebSocket connections per backend service.",
			},
			[]string{"service"},
		),
		WebSocketConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gw_websocket_connections_total",
				Help: "Total WebSocket connections established per backend service.",
			},
			[]string{"service"},
		),
		WebSocketDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gw_websocket_duration_seconds",
				Help:    "WebSocket connection duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service"},
		),
		AuthFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gw_auth_failures_total",
				Help: "Number of authentication failures at the gateway.",
			},
			[]string{"reason"}, // "expired", "invalid", "missing", "api_key"
		),
		UpstreamErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gw_upstream_errors_total",
				Help: "Number of upstream errors by service and error type.",
			},
			[]string{"service", "error_type"}, // "timeout", "connection_refused", "5xx"
		),
	}
}
