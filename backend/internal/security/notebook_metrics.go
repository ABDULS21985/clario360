package security

import "github.com/prometheus/client_golang/prometheus"

type NotebookMetrics struct {
	ServersActive         *prometheus.GaugeVec
	ServerStartTotal      *prometheus.CounterVec
	ServerStopTotal       *prometheus.CounterVec
	ServerStartDuration   *prometheus.HistogramVec
	ServerUptimeSeconds   *prometheus.HistogramVec
	SDKAPICallsTotal      *prometheus.CounterVec
	SparkJobsTotal        *prometheus.CounterVec
	DataQueriesTotal      *prometheus.CounterVec
}

func NewNotebookMetrics(reg prometheus.Registerer) *NotebookMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	m := &NotebookMetrics{
		ServersActive: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "notebook_servers_active",
			Help: "Currently active notebook servers by profile.",
		}, []string{"profile"}),
		ServerStartTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notebook_server_start_total",
			Help: "Notebook server starts by profile.",
		}, []string{"profile"}),
		ServerStopTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notebook_server_stop_total",
			Help: "Notebook server stops by profile and reason.",
		}, []string{"profile", "reason"}),
		ServerStartDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "notebook_server_start_duration_seconds",
			Help:    "Notebook server start duration by profile.",
			Buckets: prometheus.DefBuckets,
		}, []string{"profile"}),
		ServerUptimeSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "notebook_server_uptime_seconds",
			Help:    "Notebook server uptime distribution by profile.",
			Buckets: prometheus.ExponentialBuckets(60, 2, 10),
		}, []string{"profile"}),
		SDKAPICallsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notebook_sdk_api_calls_total",
			Help: "Notebook SDK API calls by endpoint and status.",
		}, []string{"endpoint", "status"}),
		SparkJobsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notebook_spark_jobs_total",
			Help: "Notebook Spark jobs by execution status.",
		}, []string{"status"}),
		DataQueriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "notebook_data_queries_total",
			Help: "Notebook data queries by source family.",
		}, []string{"source"}),
	}

	reg.MustRegister(
		m.ServersActive,
		m.ServerStartTotal,
		m.ServerStopTotal,
		m.ServerStartDuration,
		m.ServerUptimeSeconds,
		m.SDKAPICallsTotal,
		m.SparkJobsTotal,
		m.DataQueriesTotal,
	)

	return m
}
