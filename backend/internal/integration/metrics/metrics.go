package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	IntegrationDeliveriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_deliveries_total",
			Help: "Total integration deliveries by type and status.",
		},
		[]string{"type", "status"},
	)
	IntegrationDeliveryLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "integration_delivery_latency_seconds",
			Help:    "Latency of outbound integration deliveries.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)
	IntegrationDeliveryRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_delivery_retries_total",
			Help: "Total integration delivery retries.",
		},
		[]string{"type"},
	)
	IntegrationDeliveryQueueSize = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "integration_delivery_queue_size",
			Help: "Current integration delivery queue size.",
		},
		[]string{"type"},
	)
	IntegrationDeliveryRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "integration_delivery_rate",
			Help: "Observed integration delivery rate per minute.",
		},
		[]string{"integration_id"},
	)
	IntegrationErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_errors_total",
			Help: "Total integration errors by type and error category.",
		},
		[]string{"type", "error_category"},
	)
	IntegrationErrorThresholdTriggersTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_error_threshold_triggers_total",
			Help: "Number of integrations auto-disabled after crossing the failure threshold.",
		},
		[]string{"type"},
	)
	IntegrationBotCommandsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_bot_commands_total",
			Help: "Total bot commands executed by platform, command, and status.",
		},
		[]string{"platform", "command", "status"},
	)
	IntegrationWebhookVerificationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_webhook_verifications_total",
			Help: "Inbound webhook verification attempts by platform and result.",
		},
		[]string{"platform", "result"},
	)
	IntegrationTicketLinksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_ticket_links_total",
			Help: "Total external ticket links created by system and direction.",
		},
		[]string{"external_system", "direction"},
	)
	IntegrationTicketSyncsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "integration_ticket_syncs_total",
			Help: "Total ticket sync attempts by external system, direction, and status.",
		},
		[]string{"external_system", "direction", "status"},
	)
	IntegrationActiveTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "integration_active_total",
			Help: "Current number of integrations by type and status.",
		},
		[]string{"type", "status"},
	)
	IntegrationUserMappingsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "integration_user_mappings_total",
			Help: "Current number of cached user mappings by platform.",
		},
		[]string{"platform"},
	)
)

func init() {
	prometheus.MustRegister(
		IntegrationDeliveriesTotal,
		IntegrationDeliveryLatencySeconds,
		IntegrationDeliveryRetriesTotal,
		IntegrationDeliveryQueueSize,
		IntegrationDeliveryRate,
		IntegrationErrorsTotal,
		IntegrationErrorThresholdTriggersTotal,
		IntegrationBotCommandsTotal,
		IntegrationWebhookVerificationsTotal,
		IntegrationTicketLinksTotal,
		IntegrationTicketSyncsTotal,
		IntegrationActiveTotal,
		IntegrationUserMappingsTotal,
	)
}
