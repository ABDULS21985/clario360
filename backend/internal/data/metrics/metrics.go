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
	LineageNodesTotal                *prometheus.GaugeVec
	LineageEdgesTotal                *prometheus.GaugeVec
	LineageGraphDepthMax             *prometheus.GaugeVec
	LineageImpactAnalysisDuration    prometheus.Histogram
	LineageGraphBuildDuration        prometheus.Histogram
	DarkDataScansTotal               *prometheus.CounterVec
	DarkDataScanDurationSeconds      prometheus.Histogram
	DarkDataAssetsTotal              *prometheus.GaugeVec
	DarkDataPIIAssetsTotal           *prometheus.GaugeVec
	DarkDataHighRiskTotal            *prometheus.GaugeVec
	DarkDataTotalSizeBytes           *prometheus.GaugeVec
	AnalyticsQueriesTotal            *prometheus.CounterVec
	AnalyticsQueryDurationSeconds    *prometheus.HistogramVec
	AnalyticsRowsReturnedTotal       prometheus.Counter
	AnalyticsPIIMaskingAppliedTotal  prometheus.Counter
	AnalyticsQueryErrorsTotal        prometheus.Counter
	AnalyticsSavedQueriesTotal       *prometheus.GaugeVec
	AnalyticsAuditLogTotal           prometheus.Counter
	DataDashboardRequestDuration     prometheus.Histogram
	DataDashboardCacheHitTotal       prometheus.Counter
	DataDashboardCacheMissTotal      prometheus.Counter
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
		LineageNodesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lineage_nodes_total",
			Help: "Lineage graph nodes by tenant and type.",
		}, []string{"tenant_id", "type"}),
		LineageEdgesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lineage_edges_total",
			Help: "Lineage graph edges by tenant and relationship.",
		}, []string{"tenant_id", "relationship"}),
		LineageGraphDepthMax: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lineage_graph_depth_max",
			Help: "Maximum lineage graph depth by tenant.",
		}, []string{"tenant_id"}),
		LineageImpactAnalysisDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "lineage_impact_analysis_duration_seconds",
			Help:    "Duration of lineage impact analysis requests.",
			Buckets: prometheus.DefBuckets,
		}),
		LineageGraphBuildDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "lineage_graph_build_duration_seconds",
			Help:    "Duration of lineage graph builds.",
			Buckets: prometheus.DefBuckets,
		}),
		DarkDataScansTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "darkdata_scans_total",
			Help: "Dark data scans by status.",
		}, []string{"status"}),
		DarkDataScanDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "darkdata_scan_duration_seconds",
			Help:    "Duration of dark data scans.",
			Buckets: prometheus.DefBuckets,
		}),
		DarkDataAssetsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "darkdata_assets_total",
			Help: "Dark data assets by tenant and governance status.",
		}, []string{"tenant_id", "governance_status"}),
		DarkDataPIIAssetsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "darkdata_pii_assets_total",
			Help: "Dark data assets containing PII by tenant.",
		}, []string{"tenant_id"}),
		DarkDataHighRiskTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "darkdata_high_risk_total",
			Help: "High-risk dark data assets by tenant.",
		}, []string{"tenant_id"}),
		DarkDataTotalSizeBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "darkdata_total_size_bytes",
			Help: "Total estimated size of dark data assets by tenant.",
		}, []string{"tenant_id"}),
		AnalyticsQueriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "analytics_queries_total",
			Help: "Analytics queries executed by classification and PII access.",
		}, []string{"classification", "pii_accessed"}),
		AnalyticsQueryDurationSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "analytics_query_duration_seconds",
			Help:    "Analytics query execution duration by classification.",
			Buckets: prometheus.DefBuckets,
		}, []string{"classification"}),
		AnalyticsRowsReturnedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analytics_rows_returned_total",
			Help: "Total analytics rows returned.",
		}),
		AnalyticsPIIMaskingAppliedTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analytics_pii_masking_applied_total",
			Help: "Number of analytics executions with PII masking applied.",
		}),
		AnalyticsQueryErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analytics_query_errors_total",
			Help: "Analytics query execution errors.",
		}),
		AnalyticsSavedQueriesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "analytics_saved_queries_total",
			Help: "Saved analytics queries by tenant.",
		}, []string{"tenant_id"}),
		AnalyticsAuditLogTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "analytics_audit_log_total",
			Help: "Analytics audit-log entries recorded.",
		}),
		DataDashboardRequestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "dashboard_request_duration_seconds",
			Help:    "Duration of Data Suite dashboard requests.",
			Buckets: prometheus.DefBuckets,
		}),
		DataDashboardCacheHitTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dashboard_cache_hit_total",
			Help: "Data Suite dashboard cache hits.",
		}),
		DataDashboardCacheMissTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "dashboard_cache_miss_total",
			Help: "Data Suite dashboard cache misses.",
		}),
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
		m.LineageNodesTotal,
		m.LineageEdgesTotal,
		m.LineageGraphDepthMax,
		m.LineageImpactAnalysisDuration,
		m.LineageGraphBuildDuration,
		m.DarkDataScansTotal,
		m.DarkDataScanDurationSeconds,
		m.DarkDataAssetsTotal,
		m.DarkDataPIIAssetsTotal,
		m.DarkDataHighRiskTotal,
		m.DarkDataTotalSizeBytes,
		m.AnalyticsQueriesTotal,
		m.AnalyticsQueryDurationSeconds,
		m.AnalyticsRowsReturnedTotal,
		m.AnalyticsPIIMaskingAppliedTotal,
		m.AnalyticsQueryErrorsTotal,
		m.AnalyticsSavedQueriesTotal,
		m.AnalyticsAuditLogTotal,
		m.DataDashboardRequestDuration,
		m.DataDashboardCacheHitTotal,
		m.DataDashboardCacheMissTotal,
	)

	return m
}
