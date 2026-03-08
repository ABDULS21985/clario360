package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	CommitteesTotal                  *prometheus.GaugeVec
	MeetingsTotal                    *prometheus.CounterVec
	MeetingsActive                   prometheus.Gauge
	MeetingDurationMinutes           prometheus.Histogram
	MeetingQuorumMetTotal            *prometheus.CounterVec
	AttendanceRate                   *prometheus.GaugeVec
	AgendaItemsTotal                 prometheus.Counter
	AgendaVotesTotal                 *prometheus.CounterVec
	MinutesGenerationDurationSeconds prometheus.Histogram
	MinutesStatus                    *prometheus.GaugeVec
	ActionItemsTotal                 *prometheus.GaugeVec
	ActionItemsOverdue               *prometheus.GaugeVec
	ActionItemsCompletionDays        prometheus.Histogram
	ComplianceScore                  *prometheus.GaugeVec
	ComplianceChecksTotal            *prometheus.CounterVec

	// Monitoring alert metrics (used by clario360-alerts.yaml)
	OverdueActionItems prometheus.Gauge // clario360_acta_overdue_action_items
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		CommitteesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "acta_committees_total",
			Help: "Current number of committees by tenant, type, and status.",
		}, []string{"tenant_id", "type", "status"}),
		MeetingsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acta_meetings_total",
			Help: "Total number of meetings transitioned into a status.",
		}, []string{"status"}),
		MeetingsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "acta_meetings_active",
			Help: "Current number of meetings in progress.",
		}),
		MeetingDurationMinutes: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "acta_meeting_duration_minutes",
			Help:    "Observed meeting durations in minutes.",
			Buckets: []float64{15, 30, 45, 60, 90, 120, 180, 240, 360, 480},
		}),
		MeetingQuorumMetTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acta_meeting_quorum_met_total",
			Help: "Count of completed meetings by quorum outcome.",
		}, []string{"met"}),
		AttendanceRate: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "acta_attendance_rate",
			Help: "Attendance rate percentage by committee.",
		}, []string{"committee_id"}),
		AgendaItemsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "acta_agenda_items_total",
			Help: "Total agenda items created.",
		}),
		AgendaVotesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acta_agenda_votes_total",
			Help: "Total votes recorded by result.",
		}, []string{"result"}),
		MinutesGenerationDurationSeconds: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "acta_minutes_generation_duration_seconds",
			Help:    "Duration of deterministic minutes generation.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2},
		}),
		MinutesStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "acta_minutes_status",
			Help: "Current count of minutes by status.",
		}, []string{"status"}),
		ActionItemsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "acta_action_items_total",
			Help: "Current action item counts by tenant and status.",
		}, []string{"tenant_id", "status"}),
		ActionItemsOverdue: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "acta_action_items_overdue",
			Help: "Current overdue action items by tenant.",
		}, []string{"tenant_id"}),
		ActionItemsCompletionDays: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "acta_action_items_completion_days",
			Help:    "Days taken to complete action items.",
			Buckets: []float64{1, 3, 7, 14, 30, 60, 90, 180},
		}),
		ComplianceScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "acta_compliance_score",
			Help: "Latest compliance score by tenant.",
		}, []string{"tenant_id"}),
		ComplianceChecksTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "acta_compliance_checks_total",
			Help: "Compliance checks recorded by type and status.",
		}, []string{"check_type", "status"}),
		OverdueActionItems: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "clario360_acta_overdue_action_items",
			Help: "Total overdue board action items across all tenants (used by Prometheus alerting rules).",
		}),
	}

	reg.MustRegister(
		m.CommitteesTotal,
		m.MeetingsTotal,
		m.MeetingsActive,
		m.MeetingDurationMinutes,
		m.MeetingQuorumMetTotal,
		m.AttendanceRate,
		m.AgendaItemsTotal,
		m.AgendaVotesTotal,
		m.MinutesGenerationDurationSeconds,
		m.MinutesStatus,
		m.ActionItemsTotal,
		m.ActionItemsOverdue,
		m.ActionItemsCompletionDays,
		m.ComplianceScore,
		m.ComplianceChecksTotal,
		m.OverdueActionItems,
	)

	return m
}
