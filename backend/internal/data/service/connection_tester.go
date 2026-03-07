package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/clario360/platform/internal/data/connector"
	"github.com/clario360/platform/internal/data/model"
)

type ConnectionTester struct {
	registry *connector.ConnectorRegistry
}

func NewConnectionTester(registry *connector.ConnectorRegistry) *ConnectionTester {
	return &ConnectionTester{registry: registry}
}

func (t *ConnectionTester) Test(ctx context.Context, sourceType model.DataSourceType, configJSON json.RawMessage) (*connector.ConnectionTestResult, error) {
	conn, err := t.registry.Create(sourceType, configJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnsupportedType, err)
	}
	defer conn.Close()
	result, err := conn.TestConnection(ctx)
	if err != nil {
		return nil, err
	}
	return result, nil
}
