package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics holds all standard HTTP Prometheus metrics.
type HTTPMetrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	RequestSize      *prometheus.HistogramVec
	ResponseSize     *prometheus.HistogramVec
	ActiveRequests   *prometheus.GaugeVec
	PanicsTotal      *prometheus.CounterVec
}

func newHTTPMetrics(reg *prometheus.Registry, serviceName string) *HTTPMetrics {
	m := &HTTPMetrics{
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		}, []string{"method", "path", "status_code", "service"}),

		RequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}, []string{"method", "path", "service"}),

		RequestSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request body size in bytes.",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		}, []string{"method", "path", "service"}),

		ResponseSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response body size in bytes.",
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		}, []string{"method", "path", "service"}),

		ActiveRequests: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of currently active HTTP requests.",
		}, []string{"service"}),

		PanicsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_panics_total",
			Help: "Total number of recovered panics in HTTP handlers.",
		}, []string{"service"}),
	}

	reg.MustRegister(m.RequestsTotal)
	reg.MustRegister(m.RequestDuration)
	reg.MustRegister(m.RequestSize)
	reg.MustRegister(m.ResponseSize)
	reg.MustRegister(m.ActiveRequests)
	reg.MustRegister(m.PanicsTotal)

	return m
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code and bytes written.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += n
	return n, err
}

func (w *metricsResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// ChiMetricsMiddleware returns chi-compatible middleware that instruments every HTTP request.
//
// CRITICAL: Uses chi.RouteContext to get the route pattern (not the raw URL),
// preventing high-cardinality label explosion from dynamic path parameters.
func ChiMetricsMiddleware(m *HTTPMetrics, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			m.ActiveRequests.WithLabelValues(serviceName).Inc()

			wrapped := &metricsResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			m.ActiveRequests.WithLabelValues(serviceName).Dec()

			duration := time.Since(start).Seconds()

			// Use the route pattern from chi, not the raw URL.
			routePattern := routePatternFromRequest(r)

			method := r.Method
			statusCode := strconv.Itoa(wrapped.statusCode)

			m.RequestsTotal.WithLabelValues(method, routePattern, statusCode, serviceName).Inc()
			m.RequestDuration.WithLabelValues(method, routePattern, serviceName).Observe(duration)

			if r.ContentLength > 0 {
				m.RequestSize.WithLabelValues(method, routePattern, serviceName).Observe(float64(r.ContentLength))
			}

			m.ResponseSize.WithLabelValues(method, routePattern, serviceName).Observe(float64(wrapped.bytesWritten))
		})
	}
}

// routePatternFromRequest extracts the chi route pattern from the request context.
// Falls back to "unknown" if no route pattern is available.
func routePatternFromRequest(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}
	return "unknown"
}
