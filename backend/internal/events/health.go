package events

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/config"
)

// HealthStatus represents the health of a Kafka component.
type HealthStatus struct {
	Status    string            `json:"status"`    // "healthy", "unhealthy", "degraded"
	Service   string            `json:"service"`
	Component string            `json:"component"` // "broker", "consumer_group", "producer"
	Uptime    string            `json:"uptime,omitempty"`
	Error     string            `json:"error,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

// HealthChecker checks Kafka broker connectivity and consumer lag.
type HealthChecker struct {
	brokers []string
	groupID string
	logger  zerolog.Logger
}

// NewHealthChecker creates a new Kafka health checker.
func NewHealthChecker(cfg config.KafkaConfig, logger zerolog.Logger) *HealthChecker {
	return &HealthChecker{
		brokers: cfg.Brokers,
		groupID: cfg.GroupID,
		logger:  logger,
	}
}

// CheckBrokers verifies that Kafka brokers are reachable.
func (h *HealthChecker) CheckBrokers(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Component: "kafka_brokers",
		Details:   make(map[string]string),
	}

	reachable := 0
	for _, addr := range h.brokers {
		broker := sarama.NewBroker(addr)
		conf := sarama.NewConfig()
		conf.Net.DialTimeout = 5 * time.Second

		err := broker.Open(conf)
		if err != nil {
			status.Details[addr] = fmt.Sprintf("connect failed: %v", err)
			continue
		}

		connected, err := broker.Connected()
		_ = broker.Close()
		if err != nil || !connected {
			status.Details[addr] = "not connected"
			continue
		}

		status.Details[addr] = "connected"
		reachable++
	}

	switch {
	case reachable == len(h.brokers):
		status.Status = "healthy"
	case reachable > 0:
		status.Status = "degraded"
		status.Error = fmt.Sprintf("%d/%d brokers reachable", reachable, len(h.brokers))
	default:
		status.Status = "unhealthy"
		status.Error = "no brokers reachable"
	}

	return status
}

// ConsumerLag returns the lag (difference between latest offset and committed offset)
// for each partition of the given topics.
func (h *HealthChecker) ConsumerLag(ctx context.Context, topics []string) (map[string]map[int32]int64, error) {
	conf := sarama.NewConfig()
	conf.Net.DialTimeout = 5 * time.Second

	client, err := sarama.NewClient(h.brokers, conf)
	if err != nil {
		return nil, fmt.Errorf("creating sarama client: %w", err)
	}
	defer client.Close()

	admin, err := sarama.NewClusterAdminFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("creating cluster admin: %w", err)
	}
	defer admin.Close()

	// Get committed offsets for the consumer group
	offsetManager, err := sarama.NewOffsetManagerFromClient(h.groupID, client)
	if err != nil {
		return nil, fmt.Errorf("creating offset manager: %w", err)
	}
	defer offsetManager.Close()

	result := make(map[string]map[int32]int64)

	for _, topic := range topics {
		partitions, err := client.Partitions(topic)
		if err != nil {
			h.logger.Warn().Err(err).Str("topic", topic).Msg("failed to get partitions")
			continue
		}

		topicLag := make(map[int32]int64)
		for _, partition := range partitions {
			// Get latest offset
			latestOffset, err := client.GetOffset(topic, partition, sarama.OffsetNewest)
			if err != nil {
				h.logger.Warn().Err(err).Str("topic", topic).Int32("partition", partition).
					Msg("failed to get latest offset")
				continue
			}

			// Get committed offset
			pom, err := offsetManager.ManagePartition(topic, partition)
			if err != nil {
				h.logger.Warn().Err(err).Str("topic", topic).Int32("partition", partition).
					Msg("failed to manage partition offset")
				continue
			}
			committed, _ := pom.NextOffset()
			pom.Close()

			if committed < 0 {
				// No committed offset; lag is the full topic size
				topicLag[partition] = latestOffset
			} else {
				topicLag[partition] = latestOffset - committed
			}
		}

		result[topic] = topicLag
	}

	return result, nil
}

// Check performs a comprehensive health check: broker connectivity + consumer lag.
func (h *HealthChecker) Check(ctx context.Context, topics []string) HealthStatus {
	brokerStatus := h.CheckBrokers(ctx)
	if brokerStatus.Status == "unhealthy" {
		return brokerStatus
	}

	// Check consumer lag
	lag, err := h.ConsumerLag(ctx, topics)
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to check consumer lag")
		return HealthStatus{
			Status:    "degraded",
			Component: "kafka",
			Error:     fmt.Sprintf("broker: %s, lag check failed: %v", brokerStatus.Status, err),
			Details:   brokerStatus.Details,
		}
	}

	// Calculate total lag
	var totalLag int64
	details := make(map[string]string)
	for topic, partitions := range lag {
		var topicLag int64
		for _, l := range partitions {
			topicLag += l
		}
		totalLag += topicLag
		details[fmt.Sprintf("lag.%s", topic)] = fmt.Sprintf("%d", topicLag)
	}

	for k, v := range brokerStatus.Details {
		details[k] = v
	}

	status := "healthy"
	if totalLag > 10000 {
		status = "degraded"
	}

	return HealthStatus{
		Status:    status,
		Component: "kafka",
		Details:   details,
	}
}
