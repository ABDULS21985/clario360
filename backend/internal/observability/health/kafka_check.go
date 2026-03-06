package health

import (
	"context"
	"fmt"
	"net"
	"time"
)

// KafkaHealthChecker verifies Kafka broker connectivity via TCP dial.
// It does NOT create a full Kafka client — just a lightweight TCP connection test.
type KafkaHealthChecker struct {
	brokers []string
}

// NewKafkaHealthChecker creates a Kafka health checker for the given broker addresses.
func NewKafkaHealthChecker(brokers []string) *KafkaHealthChecker {
	return &KafkaHealthChecker{brokers: brokers}
}

// Name returns "kafka".
func (h *KafkaHealthChecker) Name() string { return "kafka" }

// Check dials the first reachable broker via TCP.
//
// Logic:
//  1. For each broker, attempt net.DialTimeout with the context deadline (or 2s fallback).
//  2. If at least one broker is reachable → "healthy" with broker detail.
//  3. If no broker is reachable → "unhealthy" with last error.
func (h *KafkaHealthChecker) Check(ctx context.Context) HealthResult {
	if len(h.brokers) == 0 {
		return HealthResult{
			Status: "unhealthy",
			Error:  "no brokers configured",
		}
	}

	timeout := 2 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	var lastErr error
	for _, broker := range h.brokers {
		conn, err := net.DialTimeout("tcp", broker, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		conn.Close()
		return HealthResult{
			Status: "healthy",
			Details: map[string]interface{}{
				"broker": broker,
			},
		}
	}

	return HealthResult{
		Status: "unhealthy",
		Error:  fmt.Sprintf("all brokers unreachable: %s", lastErr),
	}
}
