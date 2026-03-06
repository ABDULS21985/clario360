package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GatewayMetrics holds Prometheus metrics specific to the API gateway.
type GatewayMetrics struct {
	RequestsTotal        *prometheus.CounterVec
	RequestDuration      *prometheus.HistogramVec
	ActiveConnections    *prometheus.GaugeVec
	CircuitBreakerState  *prometheus.GaugeVec
	RateLimitExceeded    *prometheus.CounterVec
}

// NewGatewayMetrics creates and registers all gateway-specific Prometheus metrics.
func NewGatewayMetrics() *GatewayMetrics {
	return &GatewayMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "clario360",
				Subsystem: "gateway",
				Name:      "requests_total",
				Help:      "Total number of requests proxied through the gateway.",
			},
			[]string{"service", "method", "status_code", "tenant_id"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "clario360",
				Subsystem: "gateway",
				Name:      "request_duration_seconds",
				Help:      "Request latency through the gateway in seconds.",
				Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service", "method"},
		),
		ActiveConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "clario360",
				Subsystem: "gateway",
				Name:      "active_connections",
				Help:      "Number of active connections per backend service.",
			},
			[]string{"service"},
		),
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "clario360",
				Subsystem: "gateway",
				Name:      "circuit_breaker_state",
				Help:      "Circuit breaker state per service (0=closed, 1=half-open, 2=open).",
			},
			[]string{"service"},
		),
		RateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "clario360",
				Subsystem: "gateway",
				Name:      "rate_limit_exceeded_total",
				Help:      "Total number of rate limit exceeded responses.",
			},
			[]string{"tenant_id", "endpoint_group"},
		),
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture the status code.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if !w.written {
		w.statusCode = code
		w.written = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.statusCode = http.StatusOK
		w.written = true
	}
	return w.ResponseWriter.Write(b)
}

// ProxyMetrics collects Prometheus metrics per proxied request.
func ProxyMetrics(metrics *GatewayMetrics, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			metrics.ActiveConnections.WithLabelValues(serviceName).Inc()
			defer metrics.ActiveConnections.WithLabelValues(serviceName).Dec()

			wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			tenantID := r.Header.Get("X-Tenant-ID")
			if tenantID == "" {
				tenantID = "unknown"
			}

			metrics.RequestsTotal.WithLabelValues(
				serviceName,
				r.Method,
				strconv.Itoa(wrapped.statusCode),
				tenantID,
			).Inc()

			metrics.RequestDuration.WithLabelValues(
				serviceName,
				r.Method,
			).Observe(duration)
		})
	}
}
