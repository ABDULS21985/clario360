package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	DataSourcesTotal                 *prometheus.GaugeVec
	DataSourceOperationsTotal        *prometheus.CounterVec
	DataConnectionTestTotal          *prometheus.CounterVec
	DataConnectionTestLatencySeconds *prometheus.HistogramVec
	DataSchemaDiscoveryTotal         *prometheus.CounterVec
	DataSchemaDiscoveryDuration      *prometheus.HistogramVec
	DataSchemaTablesDiscovered       prometheus.Histogram
	DataSchemaPIIColumnsDetected     *prometheus.CounterVec
	DataSyncTotal                    *prometheus.CounterVec
	DataSyncDurationSeconds          *prometheus.HistogramVec
	DataSyncRowsTotal                *prometheus.CounterVec
	DataSyncBytesTotal               *prometheus.CounterVec
	DataModelsTotal                  *prometheus.GaugeVec
	DataModelDerivationsTotal        prometheus.Counter
	DataEncryptionOperationsTotal    *prometheus.CounterVec
	DataConnectorPoolConnections     *prometheus.GaugeVec
}

func New(registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = prometheus.NewRegistry()
	}

	m := &Metrics{
		DataSourcesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "data_sources_total",
			Help: "Total data sources by tenant, type, and status.",
		}, []string{"tenant_id", "type", "status"}),
		DataSourceOperationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_source_operations_total",
			Help: "Count of data source operations by operation type.",
		}, []string{"operation"}),
		DataConnectionTestTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_connection_test_total",
			Help: "Count of connection tests by source type and outcome.",
		}, []string{"type", "success"}),
		DataConnectionTestLatencySeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "data_connection_test_latency_seconds",
			Help:    "Latency of data source connection tests.",
			Buckets: prometheus.DefBuckets,
		}, []string{"type"}),
		DataSchemaDiscoveryTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_schema_discovery_total",
			Help: "Count of schema discovery runs by source type and status.",
		}, []string{"type", "status"}),
		DataSchemaDiscoveryDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "data_schema_discovery_duration_seconds",
			Help:    "Duration of schema discovery runs.",
			Buckets: prometheus.DefBuckets,
		}, []string{"type"}),
		DataSchemaTablesDiscovered: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "data_schema_tables_discovered",
			Help:    "Number of tables discovered per schema discovery run.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
		}),
		DataSchemaPIIColumnsDetected: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_schema_pii_columns_detected",
			Help: "Count of PII columns detected during schema discovery by PII type.",
		}, []string{"pii_type"}),
		DataSyncTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_sync_total",
			Help: "Count of sync operations by source type, status, and sync type.",
		}, []string{"type", "status", "sync_type"}),
		DataSyncDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "data_sync_duration_seconds",
			Help:    "Duration of data sync operations.",
			Buckets: prometheus.DefBuckets,
		}, []string{"type"}),
		DataSyncRowsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_sync_rows_total",
			Help: "Rows processed during sync operations by source type and direction.",
		}, []string{"type", "direction"}),
		DataSyncBytesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_sync_bytes_total",
			Help: "Bytes processed during sync operations by source type.",
		}, []string{"type"}),
		DataModelsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "data_models_total",
			Help: "Total data models by tenant and status.",
		}, []string{"tenant_id", "status"}),
		DataModelDerivationsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "data_model_derivations_total",
			Help: "Count of automatic model derivations.",
		}),
		DataEncryptionOperationsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "data_encryption_operations_total",
			Help: "Count of connection-config encryption and decryption operations.",
		}, []string{"operation"}),
		DataConnectorPoolConnections: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "data_connector_pool_connections",
			Help: "Active connector pool connections by source identifier.",
		}, []string{"source_id"}),
	}

	registerer.MustRegister(
		m.DataSourcesTotal,
		m.DataSourceOperationsTotal,
		m.DataConnectionTestTotal,
		m.DataConnectionTestLatencySeconds,
		m.DataSchemaDiscoveryTotal,
		m.DataSchemaDiscoveryDuration,
		m.DataSchemaTablesDiscovered,
		m.DataSchemaPIIColumnsDetected,
		m.DataSyncTotal,
		m.DataSyncDurationSeconds,
		m.DataSyncRowsTotal,
		m.DataSyncBytesTotal,
		m.DataModelsTotal,
		m.DataModelDerivationsTotal,
		m.DataEncryptionOperationsTotal,
		m.DataConnectorPoolConnections,
	)

	return m
}
