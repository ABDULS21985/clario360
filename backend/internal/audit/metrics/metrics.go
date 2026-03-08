package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// EventsConsumed tracks total events consumed from Kafka.
	EventsConsumed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "events_consumed_total",
		Help:      "Total events consumed from Kafka by topic and status.",
	}, []string{"topic", "status"})

	// EventsIngested tracks total events inserted into the audit log.
	EventsIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "events_ingested_total",
		Help:      "Total audit events ingested by status (ok, duplicate, error).",
	}, []string{"status"})

	// BatchInsertDuration tracks the duration of batch INSERT operations.
	BatchInsertDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "audit",
		Name:      "batch_insert_duration_seconds",
		Help:      "Duration of batch INSERT operations in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	})

	// BatchSize tracks the number of records per batch insert.
	BatchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "audit",
		Name:      "batch_size",
		Help:      "Number of records per batch insert.",
		Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	// DLQPublished tracks events sent to the dead letter queue.
	DLQPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "dlq_published_total",
		Help:      "Total events published to dead letter queue.",
	}, []string{"topic", "reason"})

	// QueryDuration tracks the duration of audit query operations.
	QueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "audit",
		Name:      "query_duration_seconds",
		Help:      "Duration of audit query operations in seconds.",
		Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.2, 0.5, 1},
	}, []string{"endpoint"})

	// QueryResults tracks total query result counts.
	QueryResults = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "query_results_total",
		Help:      "Total query results returned.",
	}, []string{"endpoint"})

	// ExportDuration tracks audit export operation durations.
	ExportDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "audit",
		Name:      "export_duration_seconds",
		Help:      "Duration of export operations in seconds.",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120},
	}, []string{"format", "mode"})

	// HashChainVerifications tracks hash chain verification results.
	HashChainVerifications = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "hash_chain_verifications_total",
		Help:      "Total hash chain verifications by result.",
	}, []string{"result"})

	// PartitionsCreated tracks new partitions created.
	PartitionsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "partition_created_total",
		Help:      "Total partitions created.",
	})

	// RateLimitRejected tracks rate-limited requests.
	RateLimitRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "ratelimit_rejected_total",
		Help:      "Total requests rejected by rate limiter.",
	}, []string{"tenant_id"})

	// RateLimitRedisFailures tracks Redis failures in rate limiter.
	RateLimitRedisFailures = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "audit",
		Name:      "ratelimit_redis_failures_total",
		Help:      "Total Redis failures in the rate limiter (fail-open).",
	})

	// ConsumerLag tracks Kafka consumer lag per topic/partition.
	ConsumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "audit",
		Name:      "consumer_lag",
		Help:      "Kafka consumer lag per topic and partition.",
	}, []string{"topic", "partition"})

	// ChainValid tracks whether the audit hash chain is intact.
	// 1 = valid, 0 = broken. Used by Prometheus alerting rules.
	ChainValid = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "clario360_audit_chain_valid",
		Help: "Audit trail hash chain integrity (1=valid, 0=broken). Set by periodic verification.",
	})
)
