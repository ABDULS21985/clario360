package events

import "github.com/prometheus/client_golang/prometheus"

// CrossSuiteMetrics captures consumer-level metrics shared by cross-suite handlers.
type CrossSuiteMetrics struct {
	ReceivedTotal               *prometheus.CounterVec
	ProcessedTotal              *prometheus.CounterVec
	SkippedIdempotentTotal      *prometheus.CounterVec
	ProcessingDurationSeconds   *prometheus.HistogramVec
	DeadLetteredTotal           *prometheus.CounterVec
	AlertsCreatedTotal          *prometheus.CounterVec
	NotificationsTriggeredTotal *prometheus.CounterVec
	KPIUpdatesTotal             *prometheus.CounterVec
}

func NewCrossSuiteMetrics(reg prometheus.Registerer) *CrossSuiteMetrics {
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	m := &CrossSuiteMetrics{
		ReceivedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_events_received_total",
				Help: "Total cross-suite events received by consumer and source suite.",
			},
			[]string{"consumer", "source_suite", "event_type"},
		),
		ProcessedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_events_processed_total",
				Help: "Total cross-suite events processed by result.",
			},
			[]string{"consumer", "source_suite", "event_type", "result"},
		),
		SkippedIdempotentTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_events_skipped_idempotent_total",
				Help: "Total cross-suite events skipped due to idempotency.",
			},
			[]string{"consumer", "event_type"},
		),
		ProcessingDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cross_suite_event_processing_duration_seconds",
				Help:    "Cross-suite event processing latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"consumer", "event_type"},
		),
		DeadLetteredTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_events_dead_lettered_total",
				Help: "Total cross-suite events moved to a dead letter topic.",
			},
			[]string{"consumer", "event_type"},
		),
		AlertsCreatedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_alerts_created_total",
				Help: "Total alerts created by cross-suite consumers.",
			},
			[]string{"consumer", "severity"},
		),
		NotificationsTriggeredTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_notifications_triggered_total",
				Help: "Total notifications triggered by cross-suite consumers.",
			},
			[]string{"consumer", "template"},
		),
		KPIUpdatesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cross_suite_kpi_updates_total",
				Help: "Total KPI updates triggered by cross-suite consumers.",
			},
			[]string{"consumer", "kpi_name"},
		),
	}

	reg.MustRegister(
		m.ReceivedTotal,
		m.ProcessedTotal,
		m.SkippedIdempotentTotal,
		m.ProcessingDurationSeconds,
		m.DeadLetteredTotal,
		m.AlertsCreatedTotal,
		m.NotificationsTriggeredTotal,
		m.KPIUpdatesTotal,
	)

	return m
}
