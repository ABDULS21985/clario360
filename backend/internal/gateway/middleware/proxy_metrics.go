package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/clario360/platform/internal/gateway/metrics"
)

// metricsResponseWriter wraps http.ResponseWriter to capture status code and bytes.
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
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
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// ProxyMetrics collects per-service Prometheus metrics for every proxied request.
// NOTE: tenant_id is intentionally NOT a label to prevent Prometheus cardinality explosion.
func ProxyMetrics(gwMetrics *metrics.GatewayMetrics, serviceName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			gwMetrics.ActiveRequests.WithLabelValues(serviceName).Inc()
			defer gwMetrics.ActiveRequests.WithLabelValues(serviceName).Dec()

			// Track request size.
			if r.ContentLength > 0 {
				gwMetrics.RequestSize.WithLabelValues(serviceName).Observe(float64(r.ContentLength))
			}

			wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()

			gwMetrics.RequestsTotal.WithLabelValues(
				serviceName,
				r.Method,
				strconv.Itoa(wrapped.statusCode),
			).Inc()

			gwMetrics.RequestDuration.WithLabelValues(
				serviceName,
				r.Method,
			).Observe(duration)

			if wrapped.bytes > 0 {
				gwMetrics.ResponseSize.WithLabelValues(serviceName).Observe(float64(wrapped.bytes))
			}

			// Record upstream errors in a dedicated counter.
			if wrapped.statusCode >= 500 {
				gwMetrics.UpstreamErrors.WithLabelValues(serviceName, "5xx").Inc()
			} else if wrapped.statusCode == http.StatusGatewayTimeout {
				gwMetrics.UpstreamErrors.WithLabelValues(serviceName, "timeout").Inc()
			} else if wrapped.statusCode == http.StatusBadGateway {
				gwMetrics.UpstreamErrors.WithLabelValues(serviceName, "connection_refused").Inc()
			}
		})
	}
}

// NewGatewayMetrics is re-exported for convenience (actual definition is in metrics package).
func NewGatewayMetrics() *metrics.GatewayMetrics {
	return metrics.NewGatewayMetrics()
}
