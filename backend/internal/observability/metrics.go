package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the application.
type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec
	DBQueryDuration     *prometheus.HistogramVec
	DBConnectionsOpen   prometheus.Gauge
	KafkaMessagesTotal  *prometheus.CounterVec
	KafkaConsumerLag    *prometheus.GaugeVec
	ActiveRequests      prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics.
func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		HTTPResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "http_response_size_bytes",
				Help:      "HTTP response size in bytes.",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			},
			[]string{"method", "path"},
		),
		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "db_query_duration_seconds",
				Help:      "Database query latency in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"query"},
		),
		DBConnectionsOpen: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "db_connections_open",
				Help:      "Number of open database connections.",
			},
		),
		KafkaMessagesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "kafka_messages_total",
				Help:      "Total number of Kafka messages produced/consumed.",
			},
			[]string{"topic", "direction"},
		),
		KafkaConsumerLag: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "kafka_consumer_lag",
				Help:      "Kafka consumer lag per partition.",
			},
			[]string{"topic", "partition"},
		),
		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "http_active_requests",
				Help:      "Number of currently active HTTP requests.",
			},
		),
	}
}
