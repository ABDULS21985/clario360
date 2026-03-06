package metrics

import "github.com/prometheus/client_golang/prometheus"

// DBMetrics holds all standard database Prometheus metrics.
type DBMetrics struct {
	QueriesTotal  *prometheus.CounterVec
	QueryDuration *prometheus.HistogramVec

	// Connection pool gauges (updated via PgxPoolCollector).
	ConnectionsActive   *prometheus.GaugeVec
	ConnectionsIdle     *prometheus.GaugeVec
	ConnectionsMax      *prometheus.GaugeVec
	ConnectionsWait     *prometheus.CounterVec
	ConnectionsWaitDur  *prometheus.CounterVec
}

func newDBMetrics(reg *prometheus.Registry, serviceName string) *DBMetrics {
	m := &DBMetrics{
		QueriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries.",
		}, []string{"operation", "status", "service"}),

		QueryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query latency in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		}, []string{"operation", "service"}),

		ConnectionsActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections.",
		}, []string{"service"}),

		ConnectionsIdle: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle database connections.",
		}, []string{"service"}),

		ConnectionsMax: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "db_connections_max",
			Help: "Maximum number of database connections.",
		}, []string{"service"}),

		ConnectionsWait: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_connections_wait_total",
			Help: "Total number of times a connection was waited for.",
		}, []string{"service"}),

		ConnectionsWaitDur: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_connections_wait_duration_seconds_total",
			Help: "Total time spent waiting for database connections.",
		}, []string{"service"}),
	}

	reg.MustRegister(m.QueriesTotal)
	reg.MustRegister(m.QueryDuration)
	reg.MustRegister(m.ConnectionsActive)
	reg.MustRegister(m.ConnectionsIdle)
	reg.MustRegister(m.ConnectionsMax)
	reg.MustRegister(m.ConnectionsWait)
	reg.MustRegister(m.ConnectionsWaitDur)

	return m
}
