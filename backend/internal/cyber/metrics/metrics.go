package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the cyber service.
// Each instance uses a private registry to avoid duplicate registration
// panics when multiple tests or service instances run in the same process.
type Metrics struct {
	// Asset counters
	AssetsTotal        *prometheus.GaugeVec
	AssetsCreated      *prometheus.CounterVec
	AssetsDeleted      *prometheus.CounterVec
	AssetsBulkImported *prometheus.CounterVec

	// Vulnerability counters
	VulnerabilitiesTotal  *prometheus.GaugeVec
	VulnerabilitiesOpened *prometheus.CounterVec
	VulnerabilitiesResolved *prometheus.CounterVec

	// Scan metrics
	ScansTotal     *prometheus.CounterVec
	ScanDuration   *prometheus.HistogramVec
	ScanAssetsFound *prometheus.HistogramVec
	ScanErrors     *prometheus.CounterVec

	// Enrichment metrics
	EnrichmentTotal    *prometheus.CounterVec
	EnrichmentDuration *prometheus.HistogramVec
	EnrichmentErrors   *prometheus.CounterVec

	// Classification metrics
	ClassificationsTotal  *prometheus.CounterVec
	ClassificationChanged *prometheus.CounterVec

	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Kafka metrics
	EventsPublished *prometheus.CounterVec
	EventsConsumed  *prometheus.CounterVec
	EventErrors     *prometheus.CounterVec

	// CTEM metrics
	CTEMAssessmentsTotal           *prometheus.CounterVec
	CTEMAssessmentsActive          prometheus.Gauge
	CTEMAssessmentDuration         *prometheus.HistogramVec
	CTEMPhaseDuration              *prometheus.HistogramVec
	CTEMFindingsTotal              *prometheus.CounterVec
	CTEMFindingsByStatus           *prometheus.GaugeVec
	CTEMRemediationGroupsTotal     *prometheus.CounterVec
	CTEMRemediationGroupsByStatus  *prometheus.GaugeVec
	CTEMExposureScore              *prometheus.GaugeVec
	CTEMExposureScoreComponent     *prometheus.GaugeVec
	CTEMAttackPathsFound           prometheus.Counter
	CTEMScopeResolutionDuration    prometheus.Histogram
	CTEMDiscoveryDuration          prometheus.Histogram
	CTEMPrioritizationDuration     prometheus.Histogram
	CTEMValidationDuration         prometheus.Histogram
	CTEMMobilizationDuration       prometheus.Histogram

	Registry *prometheus.Registry
}

// New creates a new Metrics instance using a private Prometheus registry.
// Using a private registry ensures no conflicts when tests create multiple instances.
func New() *Metrics {
	reg := prometheus.NewRegistry()
	factory := promauto.With(reg)

	m := &Metrics{Registry: reg}

	m.AssetsTotal = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "cyber",
		Name:      "assets_total",
		Help:      "Current number of assets in the inventory by type and criticality.",
	}, []string{"tenant_id", "asset_type", "criticality"})

	m.AssetsCreated = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "assets_created_total",
		Help:      "Total number of assets created.",
	}, []string{"tenant_id", "asset_type", "discovery_source"})

	m.AssetsDeleted = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "assets_deleted_total",
		Help:      "Total number of assets soft-deleted.",
	}, []string{"tenant_id", "asset_type"})

	m.AssetsBulkImported = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "assets_bulk_imported_total",
		Help:      "Total number of assets imported via bulk import.",
	}, []string{"tenant_id"})

	m.VulnerabilitiesTotal = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "cyber",
		Name:      "vulnerabilities_total",
		Help:      "Current number of open vulnerabilities by severity.",
	}, []string{"tenant_id", "severity", "status"})

	m.VulnerabilitiesOpened = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "vulnerabilities_opened_total",
		Help:      "Total number of vulnerabilities recorded.",
	}, []string{"tenant_id", "severity", "source"})

	m.VulnerabilitiesResolved = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "vulnerabilities_resolved_total",
		Help:      "Total number of vulnerabilities marked resolved or mitigated.",
	}, []string{"tenant_id", "severity"})

	m.ScansTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "scans_total",
		Help:      "Total number of discovery scans executed.",
	}, []string{"tenant_id", "scan_type", "status"})

	m.ScanDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cyber",
		Name:      "scan_duration_seconds",
		Help:      "Duration of discovery scans in seconds.",
		Buckets:   []float64{1, 5, 15, 30, 60, 120, 300, 600, 900, 1800},
	}, []string{"tenant_id", "scan_type"})

	m.ScanAssetsFound = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cyber",
		Name:      "scan_assets_found",
		Help:      "Number of assets discovered per scan.",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 12),
	}, []string{"tenant_id", "scan_type"})

	m.ScanErrors = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "scan_errors_total",
		Help:      "Total number of errors encountered during scans.",
	}, []string{"tenant_id", "scan_type", "error_type"})

	m.EnrichmentTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "enrichment_total",
		Help:      "Total number of enrichment operations executed.",
	}, []string{"tenant_id", "enricher", "status"})

	m.EnrichmentDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cyber",
		Name:      "enrichment_duration_seconds",
		Help:      "Duration of enrichment operations.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"tenant_id", "enricher"})

	m.EnrichmentErrors = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "enrichment_errors_total",
		Help:      "Total number of enrichment failures.",
	}, []string{"tenant_id", "enricher", "error_type"})

	m.ClassificationsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "classifications_total",
		Help:      "Total number of asset classifications performed.",
	}, []string{"tenant_id", "rule_name"})

	m.ClassificationChanged = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "classification_changed_total",
		Help:      "Total number of assets whose criticality changed after classification.",
	}, []string{"tenant_id", "from_criticality", "to_criticality"})

	m.HTTPRequestsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "http_requests_total",
		Help:      "Total number of HTTP requests received.",
	}, []string{"method", "path", "status_code"})

	m.HTTPRequestDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "cyber",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path"})

	m.EventsPublished = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "events_published_total",
		Help:      "Total number of events published to Kafka.",
	}, []string{"topic", "event_type"})

	m.EventsConsumed = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "events_consumed_total",
		Help:      "Total number of events consumed from Kafka.",
	}, []string{"topic", "event_type"})

	m.EventErrors = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "cyber",
		Name:      "event_errors_total",
		Help:      "Total number of event processing errors.",
	}, []string{"topic", "event_type", "error_type"})

	m.CTEMAssessmentsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ctem",
		Name:      "assessments_total",
		Help:      "Total number of CTEM assessments processed by terminal status.",
	}, []string{"status"})

	m.CTEMAssessmentsActive = factory.NewGauge(prometheus.GaugeOpts{
		Namespace: "ctem",
		Name:      "assessments_active",
		Help:      "Currently active CTEM assessments.",
	})

	m.CTEMAssessmentDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "assessment_duration_seconds",
		Help:      "End-to-end CTEM assessment duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"status"})

	m.CTEMPhaseDuration = factory.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "phase_duration_seconds",
		Help:      "Phase duration in seconds by phase name.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"phase"})

	m.CTEMFindingsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ctem",
		Name:      "findings_total",
		Help:      "Total number of CTEM findings by type, severity, and priority group.",
	}, []string{"type", "severity", "priority_group"})

	m.CTEMFindingsByStatus = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ctem",
		Name:      "findings_by_status",
		Help:      "Current CTEM findings grouped by status.",
	}, []string{"status"})

	m.CTEMRemediationGroupsTotal = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ctem",
		Name:      "remediation_groups_total",
		Help:      "Total remediation groups generated by type and effort.",
	}, []string{"type", "effort"})

	m.CTEMRemediationGroupsByStatus = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ctem",
		Name:      "remediation_groups_by_status",
		Help:      "Current remediation groups grouped by status.",
	}, []string{"status"})

	m.CTEMExposureScore = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ctem",
		Name:      "exposure_score",
		Help:      "Current exposure score by tenant.",
	}, []string{"tenant_id"})

	m.CTEMExposureScoreComponent = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ctem",
		Name:      "exposure_score_component",
		Help:      "Exposure score components.",
	}, []string{"component"})

	m.CTEMAttackPathsFound = factory.NewCounter(prometheus.CounterOpts{
		Namespace: "ctem",
		Name:      "attack_paths_found",
		Help:      "Total attack paths discovered across assessments.",
	})

	m.CTEMScopeResolutionDuration = factory.NewHistogram(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "scope_resolution_duration_seconds",
		Help:      "Duration of scope resolution.",
		Buckets:   prometheus.DefBuckets,
	})

	m.CTEMDiscoveryDuration = factory.NewHistogram(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "discovery_duration_seconds",
		Help:      "Duration of CTEM discovery phase.",
		Buckets:   prometheus.DefBuckets,
	})

	m.CTEMPrioritizationDuration = factory.NewHistogram(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "prioritization_duration_seconds",
		Help:      "Duration of CTEM prioritization phase.",
		Buckets:   prometheus.DefBuckets,
	})

	m.CTEMValidationDuration = factory.NewHistogram(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "validation_duration_seconds",
		Help:      "Duration of CTEM validation phase.",
		Buckets:   prometheus.DefBuckets,
	})

	m.CTEMMobilizationDuration = factory.NewHistogram(prometheus.HistogramOpts{
		Namespace: "ctem",
		Name:      "mobilization_duration_seconds",
		Help:      "Duration of CTEM mobilization phase.",
		Buckets:   prometheus.DefBuckets,
	})

	return m
}
