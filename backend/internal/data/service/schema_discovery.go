package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
)

type SchemaDiscoveryService struct {
	registry *connector.ConnectorRegistry
	options  connector.DiscoveryOptions
}

func NewSchemaDiscoveryService(registry *connector.ConnectorRegistry, options connector.DiscoveryOptions) *SchemaDiscoveryService {
	return &SchemaDiscoveryService{registry: registry, options: options}
}

func (s *SchemaDiscoveryService) Discover(ctx context.Context, sourceType model.DataSourceType, configJSON json.RawMessage) (*model.DiscoveredSchema, *connector.SizeEstimate, error) {
	conn, err := s.registry.Create(sourceType, configJSON)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}
	defer conn.Close()

	schema, err := conn.DiscoverSchema(ctx, s.options)
	if err != nil {
		return nil, nil, err
	}
	size, err := conn.EstimateSize(ctx)
	if err != nil {
		return nil, nil, err
	}
	return schema, size, nil
}
