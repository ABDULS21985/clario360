package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type VCISOMetrics struct {
	MessagesTotal            *prometheus.CounterVec
	IntentClassifiedTotal    *prometheus.CounterVec
	IntentConfidence         *prometheus.HistogramVec
	UnknownIntentsTotal      prometheus.Counter
	ToolExecutionsTotal      *prometheus.CounterVec
	ToolLatencySeconds       *prometheus.HistogramVec
	ToolTimeoutsTotal        *prometheus.CounterVec
	ConversationsActive      prometheus.Gauge
	ConversationsTotal       prometheus.Counter
	DashboardsCreatedTotal   prometheus.Counter
	SuggestionsServedTotal   prometheus.Counter
	PermissionDenialsTotal   *prometheus.CounterVec
	ContextResolutionsTotal  *prometheus.CounterVec
}

func NewMetrics(reg prometheus.Registerer) *VCISOMetrics {
	factory := promauto.With(reg)
	return &VCISOMetrics{
		MessagesTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "messages_total",
			Help:      "Total vCISO chat messages by role and classified intent.",
		}, []string{"role", "intent"}),
		IntentClassifiedTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "intent_classified_total",
			Help:      "Total deterministic vCISO intent classifications by method.",
		}, []string{"intent", "method"}),
		IntentConfidence: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "vciso",
			Name:      "intent_confidence",
			Help:      "Distribution of vCISO intent-classification confidence scores.",
			Buckets:   []float64{0, 0.3, 0.5, 0.65, 0.8, 0.9, 1},
		}, []string{"intent"}),
		UnknownIntentsTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "unknown_intents_total",
			Help:      "Total vCISO user messages that fell back to the unknown intent.",
		}),
		ToolExecutionsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "tool_executions_total",
			Help:      "Total vCISO tool executions by tool and completion status.",
		}, []string{"tool_name", "status"}),
		ToolLatencySeconds: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "vciso",
			Name:      "tool_latency_seconds",
			Help:      "Latency of vCISO tool execution in seconds.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10},
		}, []string{"tool_name"}),
		ToolTimeoutsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "tool_timeouts_total",
			Help:      "Total vCISO tool executions that hit the 10-second timeout.",
		}, []string{"tool_name"}),
		ConversationsActive: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: "vciso",
			Name:      "conversations_active",
			Help:      "Approximate number of active vCISO conversations observed since process start.",
		}),
		ConversationsTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "conversations_total",
			Help:      "Total vCISO conversations created.",
		}),
		DashboardsCreatedTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "dashboards_created_total",
			Help:      "Total dashboards created by the vCISO dashboard-builder tool.",
		}),
		SuggestionsServedTotal: factory.NewCounter(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "suggestions_served_total",
			Help:      "Total suggestion lists served by the vCISO suggestion engine.",
		}),
		PermissionDenialsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "permission_denials_total",
			Help:      "Total vCISO tool requests denied due to missing permissions.",
		}, []string{"tool_name"}),
		ContextResolutionsTotal: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: "vciso",
			Name:      "context_resolutions_total",
			Help:      "Total context-based entity resolution outcomes.",
		}, []string{"resolution_type"}),
	}
}
