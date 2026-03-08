package connector

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/google/uuid"
)

type ConnectorRegistry struct {
	factories map[model.DataSourceType]ConnectorFactory
	metadata  map[model.DataSourceType]ConnectorTypeMetadata
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
			model.DataSourceTypeClickHouse: func(config json.RawMessage) (Connector, error) {
				return NewClickHouseConnector(config, options)
			},
			model.DataSourceTypeImpala: func(config json.RawMessage) (Connector, error) {
				return NewImpalaConnector(config, options)
			},
			model.DataSourceTypeHive: func(config json.RawMessage) (Connector, error) {
				return NewHiveConnector(config, options)
			},
			model.DataSourceTypeHDFS: func(config json.RawMessage) (Connector, error) {
				return NewHDFSConnector(config, options)
			},
			model.DataSourceTypeSpark: func(config json.RawMessage) (Connector, error) {
				return NewSparkConnector(config, options)
			},
			model.DataSourceTypeDagster: func(config json.RawMessage) (Connector, error) {
				return NewDagsterConnector(config, options)
			},
			model.DataSourceTypeDolt: func(config json.RawMessage) (Connector, error) {
				return NewDoltConnector(config, options)
			},
		},
		metadata: defaultConnectorMetadata(),
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

func (r *ConnectorRegistry) CreateWithSourceContext(sourceType model.DataSourceType, configJSON json.RawMessage, sourceID, tenantID uuid.UUID) (Connector, error) {
	connector, err := r.Create(sourceType, configJSON)
	if err != nil {
		return nil, err
	}
	if aware, ok := connector.(SourceContextAware); ok {
		aware.SetSourceContext(sourceID, tenantID)
	}
	return connector, nil
}

func (r *ConnectorRegistry) IsRegistered(sourceType model.DataSourceType) bool {
	_, ok := r.factories[sourceType]
	return ok
}

func (r *ConnectorRegistry) ListTypes() []model.DataSourceType {
	values := make([]model.DataSourceType, 0, len(r.factories))
	for sourceType := range r.factories {
		values = append(values, sourceType)
	}
	slices.Sort(values)
	return values
}

func (r *ConnectorRegistry) TypeMetadata(sourceType model.DataSourceType) *ConnectorTypeMetadata {
	value, ok := r.metadata[sourceType]
	if !ok {
		return nil
	}
	copyValue := value
	return &copyValue
}
