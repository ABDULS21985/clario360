package metrics

import "github.com/prometheus/client_golang/prometheus"

// KafkaMetrics holds all standard Kafka Prometheus metrics.
type KafkaMetrics struct {
	ProducedTotal    *prometheus.CounterVec
	ProduceDuration  *prometheus.HistogramVec
	ConsumedTotal    *prometheus.CounterVec
	ConsumeDuration  *prometheus.HistogramVec
	ConsumerLag      *prometheus.GaugeVec
	RebalanceTotal   *prometheus.CounterVec
}

func newKafkaMetrics(reg *prometheus.Registry, serviceName string) *KafkaMetrics {
	m := &KafkaMetrics{
		ProducedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total number of Kafka messages produced.",
		}, []string{"topic", "status", "service"}),

		ProduceDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "kafka_produce_duration_seconds",
			Help:    "Kafka message produce latency in seconds.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5},
		}, []string{"topic", "service"}),

		ConsumedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed.",
		}, []string{"topic", "group", "status", "service"}),

		ConsumeDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "kafka_consume_duration_seconds",
			Help:    "Duration of Kafka message processing.",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.5, 1.0, 2.5},
		}, []string{"topic", "group", "service"}),

		ConsumerLag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Kafka consumer lag per topic, partition, and group.",
		}, []string{"topic", "partition", "group", "service"}),

		RebalanceTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "kafka_consumer_rebalance_total",
			Help: "Total number of Kafka consumer group rebalances.",
		}, []string{"group", "service"}),
	}

	reg.MustRegister(m.ProducedTotal)
	reg.MustRegister(m.ProduceDuration)
	reg.MustRegister(m.ConsumedTotal)
	reg.MustRegister(m.ConsumeDuration)
	reg.MustRegister(m.ConsumerLag)
	reg.MustRegister(m.RebalanceTotal)

	return m
}
