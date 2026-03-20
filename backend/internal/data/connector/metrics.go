package connector

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type ConnectorMetrics struct {
	OperationDuration    *prometheus.HistogramVec
	OperationErrorsTotal *prometheus.CounterVec
	FetchRowsTotal       *prometheus.CounterVec
	FetchBytesTotal      *prometheus.CounterVec
	ActiveConnections    *prometheus.GaugeVec
	SchemaTablesFound    *prometheus.CounterVec
	PIIColumnsDetected   *prometheus.CounterVec
	AccessEventsTotal    *prometheus.CounterVec
	DSPMFilesScanned     *prometheus.CounterVec
}

var (
	connectorMetricsOnce       sync.Once
	connectorMetricsRegisterer prometheus.Registerer = prometheus.DefaultRegisterer
	connectorMetricsInstance   *ConnectorMetrics
)

func SetMetricsRegisterer(registerer prometheus.Registerer) {
	if registerer != nil {
		connectorMetricsRegisterer = registerer
	}
}

func getConnectorMetrics() *ConnectorMetrics {
	connectorMetricsOnce.Do(func() {
		metrics := &ConnectorMetrics{
			OperationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
				Name:    "connector_operation_duration_seconds",
				Help:    "Duration of connector operations by connector type and operation.",
				Buckets: prometheus.DefBuckets,
			}, []string{"connector_type", "operation"}),
			OperationErrorsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_operation_errors_total",
				Help: "Total connector operation errors by connector type, operation, and error code.",
			}, []string{"connector_type", "operation", "error_code"}),
			FetchRowsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_fetch_rows_total",
				Help: "Total rows fetched by connector type.",
			}, []string{"connector_type"}),
			FetchBytesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_fetch_bytes_total",
				Help: "Approximate bytes fetched by connector type.",
			}, []string{"connector_type"}),
			ActiveConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: "connector_active_connections",
				Help: "Currently active connector instances by type.",
			}, []string{"connector_type"}),
			SchemaTablesFound: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_schema_tables_discovered",
				Help: "Tables discovered during schema discovery by connector type.",
			}, []string{"connector_type"}),
			PIIColumnsDetected: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_pii_columns_detected",
				Help: "PII columns detected during schema discovery by connector type and pii type.",
			}, []string{"connector_type", "pii_type"}),
			AccessEventsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_access_events_collected",
				Help: "Access events collected from security-aware connectors.",
			}, []string{"connector_type"}),
			DSPMFilesScanned: prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: "connector_dspm_files_scanned",
				Help: "Files scanned by connector type and detected format.",
			}, []string{"connector_type", "format"}),
		}
		connectorMetricsRegisterer.MustRegister(
			metrics.OperationDuration,
			metrics.OperationErrorsTotal,
			metrics.FetchRowsTotal,
			metrics.FetchBytesTotal,
			metrics.ActiveConnections,
			metrics.SchemaTablesFound,
			metrics.PIIColumnsDetected,
			metrics.AccessEventsTotal,
			metrics.DSPMFilesScanned,
		)
		connectorMetricsInstance = metrics
	})
	return connectorMetricsInstance
}
