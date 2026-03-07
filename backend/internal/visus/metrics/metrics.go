package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	DashboardsTotal                 *prometheus.GaugeVec
	WidgetsTotal                    *prometheus.GaugeVec
	KPIsTotal                       *prometheus.GaugeVec
	KPISnapshotsTotal               *prometheus.CounterVec
	KPISnapshotDurationSeconds      *prometheus.HistogramVec
	KPIThresholdBreachesTotal       *prometheus.CounterVec
	ExecutiveAlertsTotal            *prometheus.GaugeVec
	ExecutiveAlertsCreatedTotal     *prometheus.CounterVec
	ReportsTotal                    *prometheus.GaugeVec
	ReportGenerationDurationSeconds prometheus.Histogram
	SuiteFetchTotal                 *prometheus.CounterVec
	SuiteFetchDurationSeconds       *prometheus.HistogramVec
	SuiteCircuitBreakerState        *prometheus.GaugeVec
	ExecutiveViewDurationSeconds    prometheus.Histogram
	WidgetDataFetchDurationSeconds  *prometheus.HistogramVec
}

func New(registerer prometheus.Registerer) *Metrics {
	if registerer == nil {
		registerer = prometheus.NewRegistry()
	}
	factory := promauto.With(registerer)

	return &Metrics{
		DashboardsTotal: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "visus",
			Name:      "dashboards_total",
			Help:      "Current dashboards by tenant and visibility.",
		}, []string{"tenant_id", "visibility"}),
		WidgetsTotal: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "visus",
			Name:      "widgets_total",
			Help:      "Current widgets by tenant and type.",
		}, []string{"tenant_id", "type"}),
		KPIsTotal: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "visus",
			Name:      "kpis_total",
			Help:      "Current KPIs by tenant, suite, and enabled state.",
		}, []string{"tenant_id", "suite", "enabled"}),
		KPISnapshotsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "visus",
			Name:      "kpi_snapshots_total",
			Help:      "Total KPI snapshots recorded.",
		}, []string{"suite", "status"}),
		KPISnapshotDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "visus",
			Name:      "kpi_snapshot_duration_seconds",
			Help:      "KPI snapshot duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"suite"}),
		KPIThresholdBreachesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "visus",
			Name:      "kpi_threshold_breaches_total",
			Help:      "Total KPI threshold breaches.",
		}, []string{"suite", "status"}),
		ExecutiveAlertsTotal: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "visus",
			Name:      "executive_alerts_total",
			Help:      "Current executive alerts by tenant, category, severity, and status.",
		}, []string{"tenant_id", "category", "severity", "status"}),
		ExecutiveAlertsCreatedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "visus",
			Name:      "executive_alerts_created_total",
			Help:      "Total executive alerts created.",
		}, []string{"category", "source_type"}),
		ReportsTotal: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "visus",
			Name:      "reports_total",
			Help:      "Current report definitions by tenant and type.",
		}, []string{"tenant_id", "report_type"}),
		ReportGenerationDurationSeconds: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: "visus",
			Name:      "report_generation_duration_seconds",
			Help:      "Duration of report generation.",
			Buckets:   prometheus.DefBuckets,
		}),
		SuiteFetchTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "visus",
			Name:      "suite_fetch_total",
			Help:      "Total cross-suite fetches by suite and result status.",
		}, []string{"suite", "status"}),
		SuiteFetchDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "visus",
			Name:      "suite_fetch_duration_seconds",
			Help:      "Cross-suite fetch duration in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"suite"}),
		SuiteCircuitBreakerState: factory.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "visus",
			Name:      "suite_circuit_breaker_state",
			Help:      "Circuit breaker state by suite. 0=closed, 1=open.",
		}, []string{"suite"}),
		ExecutiveViewDurationSeconds: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: "visus",
			Name:      "executive_view_duration_seconds",
			Help:      "Duration of executive view generation.",
			Buckets:   prometheus.DefBuckets,
		}),
		WidgetDataFetchDurationSeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "visus",
			Name:      "widget_data_fetch_duration_seconds",
			Help:      "Duration of widget data resolution in seconds.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"widget_type"}),
	}
}
