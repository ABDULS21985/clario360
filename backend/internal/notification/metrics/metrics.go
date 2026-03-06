package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	NotificationsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "created_total",
			Help:      "Total notifications created, by type and category.",
		},
		[]string{"type", "category"},
	)

	DeliveriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "deliveries_total",
			Help:      "Total delivery attempts, by channel and status.",
		},
		[]string{"channel", "status"},
	)

	DeliveryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "notification",
			Name:      "delivery_duration_seconds",
			Help:      "Time to deliver a notification per channel.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"channel"},
	)

	WSConnectionsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "notification",
			Name:      "ws_connections_active",
			Help:      "Current active WebSocket connections by tenant.",
		},
		[]string{"tenant_id"},
	)

	WSConnectionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "ws_connections_total",
			Help:      "Total WebSocket connections established.",
		},
	)

	WSMessagesSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "ws_messages_sent_total",
			Help:      "Total WebSocket messages sent.",
		},
	)

	EmailSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "email_sent_total",
			Help:      "Total emails sent, by provider and status.",
		},
		[]string{"provider", "status"},
	)

	WebhookDeliveries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "webhook_deliveries_total",
			Help:      "Total webhook delivery attempts, by status.",
		},
		[]string{"status"},
	)

	ConsumerEventsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "consumer_events_processed_total",
			Help:      "Total events consumed from Kafka, by topic and result.",
		},
		[]string{"topic", "result"},
	)

	DuplicatesSkipped = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "duplicates_skipped_total",
			Help:      "Total duplicate notifications skipped via dedup.",
		},
	)

	RateLimitRejected = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "rate_limit_rejected_total",
			Help:      "Total requests rejected by rate limiter.",
		},
	)

	DigestsSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "notification",
			Name:      "digests_sent_total",
			Help:      "Total digest emails sent, by frequency.",
		},
		[]string{"frequency"},
	)
)

func init() {
	prometheus.MustRegister(
		NotificationsCreated,
		DeliveriesTotal,
		DeliveryDuration,
		WSConnectionsActive,
		WSConnectionsTotal,
		WSMessagesSent,
		EmailSent,
		WebhookDeliveries,
		ConsumerEventsProcessed,
		DuplicatesSkipped,
		RateLimitRejected,
		DigestsSent,
	)
}
