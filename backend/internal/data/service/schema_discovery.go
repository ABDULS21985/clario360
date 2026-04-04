package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clario360/platform/internal/data/connector"
	datametrics "github.com/clario360/platform/internal/data/metrics"
	"github.com/clario360/platform/internal/data/model"
)

type SchemaDiscoveryService struct {
	registry *connector.ConnectorRegistry
	options  connector.DiscoveryOptions
	metrics  *datametrics.Metrics
}

func NewSchemaDiscoveryService(registry *connector.ConnectorRegistry, options connector.DiscoveryOptions, metrics *datametrics.Metrics) *SchemaDiscoveryService {
	return &SchemaDiscoveryService{registry: registry, options: options, metrics: metrics}
}

func (s *SchemaDiscoveryService) Discover(ctx context.Context, sourceType model.DataSourceType, configJSON json.RawMessage) (*model.DiscoveredSchema, *connector.SizeEstimate, error) {
	start := time.Now()
	status := "failed"
	conn, err := s.registry.Create(sourceType, configJSON)
	if err != nil {
		s.observeMetrics(sourceType, nil, start, status)
		return nil, nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}
	defer conn.Close()

	schema, err := conn.DiscoverSchema(ctx, s.options)
	if err != nil {
		s.observeMetrics(sourceType, nil, start, status)
		return nil, nil, err
	}
	size, err := conn.EstimateSize(ctx)
	if err != nil {
		s.observeMetrics(sourceType, schema, start, status)
		return nil, nil, err
	}
	status = "success"
	s.observeMetrics(sourceType, schema, start, status)
	return schema, size, nil
}

func (s *SchemaDiscoveryService) observeMetrics(sourceType model.DataSourceType, schema *model.DiscoveredSchema, started time.Time, status string) {
	if s.metrics == nil {
		return
	}
	s.metrics.DataSchemaDiscoveryTotal.WithLabelValues(string(sourceType), status).Inc()
	s.metrics.DataSchemaDiscoveryDuration.WithLabelValues(string(sourceType)).Observe(time.Since(started).Seconds())
	if schema == nil {
		return
	}
	s.metrics.DataSchemaTablesDiscovered.Observe(float64(schema.TableCount))
	for _, table := range schema.Tables {
		for _, column := range table.Columns {
			if column.InferredPII && column.InferredPIIType != "" {
				s.metrics.DataSchemaPIIColumnsDetected.WithLabelValues(column.InferredPIIType).Inc()
			}
		}
	}
}
