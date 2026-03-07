package connector

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

type ConnectorRegistry struct {
	factories map[model.DataSourceType]ConnectorFactory
}

func NewConnectorRegistry(limits ConnectorLimits, logger zerolog.Logger) *ConnectorRegistry {
	options := FactoryOptions{
		Limits: limits,
		Logger: logger,
	}

	return &ConnectorRegistry{
		factories: map[model.DataSourceType]ConnectorFactory{
			model.DataSourceTypePostgreSQL: func(config json.RawMessage) (Connector, error) {
				return NewPostgresConnector(config, options)
			},
			model.DataSourceTypeMySQL: func(config json.RawMessage) (Connector, error) {
				return NewMySQLConnector(config, options)
			},
			model.DataSourceTypeAPI: func(config json.RawMessage) (Connector, error) {
				return NewAPIConnector(config, options)
			},
			model.DataSourceTypeCSV: func(config json.RawMessage) (Connector, error) {
				return NewCSVConnector(config, options)
			},
			model.DataSourceTypeS3: func(config json.RawMessage) (Connector, error) {
				return NewS3Connector(config, options)
			},
		},
	}
}

func (r *ConnectorRegistry) Create(sourceType model.DataSourceType, configJSON json.RawMessage) (Connector, error) {
	factory, ok := r.factories[sourceType]
	if !ok {
		return nil, fmt.Errorf("unsupported source type: %s", sourceType)
	}
	connector, err := factory(configJSON)
	if err != nil {
		return nil, fmt.Errorf("create %s connector: %w", sourceType, err)
	}
	return connector, nil
}
