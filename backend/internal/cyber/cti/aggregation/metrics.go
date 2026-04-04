package aggregation

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds Prometheus metrics for CTI aggregation jobs.
type Metrics struct {
	Registry prometheus.Gatherer

	Duration  *prometheus.HistogramVec
	RunsTotal *prometheus.CounterVec
	Errors    *prometheus.CounterVec
	LastRun   *prometheus.GaugeVec
	RiskScore *prometheus.GaugeVec
	Events24h *prometheus.GaugeVec
}

// NewMetrics creates a new per-instance Prometheus registry with CTI aggregation metrics.
func NewMetrics(parent *prometheus.Registry) *Metrics {
	reg := parent
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	f := promauto.With(reg)

	m := &Metrics{
		Registry: reg,

		Duration: f.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "clario360",
			Subsystem: "cti_aggregation",
			Name:      "duration_seconds",
			Help:      "Duration of CTI aggregation jobs in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.05, 2, 10),
		}, []string{"tenant_id", "aggregation_type"}),

		RunsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "cti_aggregation",
			Name:      "runs_total",
			Help:      "Total number of CTI aggregation runs",
		}, []string{"scope"}),

		Errors: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "clario360",
			Subsystem: "cti_aggregation",
			Name:      "errors_total",
			Help:      "Total number of CTI aggregation errors",
		}, []string{"tenant_id", "aggregation_type"}),

		LastRun: f.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "clario360",
			Subsystem: "cti_aggregation",
			Name:      "last_run_timestamp",
			Help:      "Unix timestamp of the last aggregation run per tenant",
		}, []string{"tenant_id"}),

		RiskScore: f.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "clario360",
			Subsystem: "cti",
			Name:      "risk_score",
			Help:      "Current CTI risk score per tenant (0-100)",
		}, []string{"tenant_id"}),

		Events24h: f.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "clario360",
			Subsystem: "cti",
			Name:      "events_24h",
			Help:      "Threat events observed in the last 24 hours per tenant",
		}, []string{"tenant_id"}),
	}
	return m
}
