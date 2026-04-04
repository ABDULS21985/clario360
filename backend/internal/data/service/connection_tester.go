package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/clario360/platform/internal/data/connector"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	"github.com/clario360/platform/internal/data/model"
)

type ConnectionTester struct {
	registry *connector.ConnectorRegistry
	metrics  *datametrics.Metrics
}

func NewConnectionTester(registry *connector.ConnectorRegistry, metrics *datametrics.Metrics) *ConnectionTester {
	return &ConnectionTester{registry: registry, metrics: metrics}
}

func (t *ConnectionTester) Test(ctx context.Context, sourceType model.DataSourceType, configJSON json.RawMessage) (*connector.ConnectionTestResult, error) {
	start := time.Now()
	conn, err := t.registry.Create(sourceType, configJSON)
	if err != nil {
		if t.metrics != nil {
			t.metrics.DataConnectionTestTotal.WithLabelValues(string(sourceType), strconv.FormatBool(false)).Inc()
			t.metrics.DataConnectionTestLatencySeconds.WithLabelValues(string(sourceType)).Observe(time.Since(start).Seconds())
		}
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}
	defer conn.Close()
	result, err := conn.TestConnection(ctx)
	if t.metrics != nil {
		success := err == nil && result != nil && result.Success
		t.metrics.DataConnectionTestTotal.WithLabelValues(string(sourceType), strconv.FormatBool(success)).Inc()
		t.metrics.DataConnectionTestLatencySeconds.WithLabelValues(string(sourceType)).Observe(time.Since(start).Seconds())
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}
