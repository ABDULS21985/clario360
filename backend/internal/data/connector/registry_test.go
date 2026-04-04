package connector

import (
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

func newTestRegistry() *ConnectorRegistry {
	return NewConnectorRegistry(ConnectorLimits{
		MaxPoolSize:      4,
		StatementTimeout: 30 * time.Second,
		ConnectTimeout:   10 * time.Second,
		MaxSampleRows:    100,
		MaxTables:        500,
		APIRateLimit:     10,
	}, zerolog.Nop())
}

func TestAllConnectorsRegistered(t *testing.T) {
	registry := newTestRegistry()
	expected := []model.DataSourceType{
		model.DataSourceTypeAPI,
		model.DataSourceTypeCSV,
		model.DataSourceTypeClickHouse,
		model.DataSourceTypeDagster,
		model.DataSourceTypeDolt,
		model.DataSourceTypeHDFS,
		model.DataSourceTypeHive,
		model.DataSourceTypeImpala,
		model.DataSourceTypeMySQL,
		model.DataSourceTypePostgreSQL,
		model.DataSourceTypeS3,
		model.DataSourceTypeSpark,
	}

	for _, sourceType := range expected {
		if !registry.IsRegistered(sourceType) {
			t.Fatalf("expected %s to be registered", sourceType)
		}
		if metadata := registry.TypeMetadata(sourceType); metadata == nil {
			t.Fatalf("expected metadata for %s", sourceType)
		}
	}
}

func TestListTypes(t *testing.T) {
	registry := newTestRegistry()
	got := registry.ListTypes()
	want := []model.DataSourceType{
		model.DataSourceTypeAPI,
		model.DataSourceTypeCSV,
		model.DataSourceTypeClickHouse,
		model.DataSourceTypeDagster,
		model.DataSourceTypeDolt,
		model.DataSourceTypeHDFS,
		model.DataSourceTypeHive,
		model.DataSourceTypeImpala,
		model.DataSourceTypeMySQL,
		model.DataSourceTypePostgreSQL,
		model.DataSourceTypeS3,
		model.DataSourceTypeSpark,
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("ListTypes() = %v, want %v", got, want)
	}
}

func TestTypeMetadata(t *testing.T) {
	registry := newTestRegistry()
	metadata := registry.TypeMetadata(model.DataSourceTypeClickHouse)
	if metadata == nil {
		t.Fatal("TypeMetadata(clickhouse) = nil")
	}
	if metadata.DisplayName != "ClickHouse" {
		t.Fatalf("DisplayName = %q, want ClickHouse", metadata.DisplayName)
	}
	if metadata.Category != "database" {
		t.Fatalf("Category = %q, want database", metadata.Category)
	}
	if !metadata.SupportsSecurity {
		t.Fatal("SupportsSecurity = false, want true")
	}
	if len(metadata.ConfigFields) == 0 {
		t.Fatal("ConfigFields is empty")
	}
}

func TestRegistry_CreateKnownConnectors(t *testing.T) {
	registry := newTestRegistry()

	tests := []struct {
		name       string
		sourceType model.DataSourceType
		config     any
		wantType   string
	}{
		{
			name:       "postgresql",
			sourceType: model.DataSourceTypePostgreSQL,
			config: model.PostgresConnectionConfig{
				Host:     "localhost",
				Port:     5432,
				Database: "app",
				Username: "user",
				Password: "secret",
				SSLMode:  "require",
			},
			wantType: "*connector.PostgresConnector",
		},
		{
			name:       "mysql",
			sourceType: model.DataSourceTypeMySQL,
			config: model.MySQLConnectionConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "app",
				Username: "user",
				Password: "secret",
			},
			wantType: "*connector.MySQLConnector",
		},
		{
			name:       "api",
			sourceType: model.DataSourceTypeAPI,
			config: model.APIConnectionConfig{
				BaseURL:               "https://localhost/data",
				AuthType:              model.APIAuthNone,
				PaginationType:        model.APIPaginationOffset,
				AllowPrivateAddresses: true,
			},
			wantType: "*connector.APIConnector",
		},
		{
			name:       "csv",
			sourceType: model.DataSourceTypeCSV,
			config: model.CSVConnectionConfig{
				MinioEndpoint: "localhost:9000",
				Bucket:        "ingest",
				FilePath:      "customers.csv",
				AccessKey:     "minio",
				SecretKey:     "secret",
			},
			wantType: "*connector.CSVConnector",
		},
		{
			name:       "s3",
			sourceType: model.DataSourceTypeS3,
			config: model.S3ConnectionConfig{
				Endpoint:  "localhost:9000",
				Bucket:    "warehouse",
				AccessKey: "minio",
				SecretKey: "secret",
			},
			wantType: "*connector.S3Connector",
		},
		{
			name:       "clickhouse",
			sourceType: model.DataSourceTypeClickHouse,
			config: model.ClickHouseConnectionConfig{
				Host:     "localhost",
				Port:     9000,
				Database: "default",
				Username: "default",
				Password: "secret",
				Protocol: "native",
			},
			wantType: "*connector.ClickHouseConnector",
		},
		{
			name:       "impala",
			sourceType: model.DataSourceTypeImpala,
			config: model.ImpalaConnectionConfig{
				Host:     "localhost",
				Port:     21050,
				AuthType: "noauth",
			},
			wantType: "*connector.ImpalaConnector",
		},
		{
			name:       "hive",
			sourceType: model.DataSourceTypeHive,
			config: model.HiveConnectionConfig{
				Host:          "localhost",
				Port:          10000,
				AuthType:      "noauth",
				TransportMode: "binary",
			},
			wantType: "*connector.HiveConnector",
		},
		{
			name:       "hdfs",
			sourceType: model.DataSourceTypeHDFS,
			config: model.HDFSConnectionConfig{
				NameNodes: []string{"namenode:8020"},
			},
			wantType: "*connector.HDFSConnector",
		},
		{
			name:       "spark",
			sourceType: model.DataSourceTypeSpark,
			config: model.SparkConnectionConfig{
				Thrift: &model.SparkThriftConfig{
					Host:     "localhost",
					Port:     10001,
					AuthType: "noauth",
				},
				REST: model.SparkRESTConfig{
					MasterURL: "http://localhost:8080",
				},
			},
			wantType: "*connector.SparkConnector",
		},
		{
			name:       "dagster",
			sourceType: model.DataSourceTypeDagster,
			config: model.DagsterConnectionConfig{
				GraphQLURL: "http://localhost:3000/graphql",
			},
			wantType: "*connector.DagsterConnector",
		},
		{
			name:       "dolt",
			sourceType: model.DataSourceTypeDolt,
			config: model.DoltConnectionConfig{
				Host:     "localhost",
				Port:     3306,
				Database: "app",
				Username: "root",
				Password: "secret",
			},
			wantType: "*connector.DoltConnector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.config)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			conn, err := registry.Create(tt.sourceType, raw)
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if got := fmt.Sprintf("%T", conn); got != tt.wantType {
				t.Fatalf("Create() type = %q, want %q", got, tt.wantType)
			}
			_ = conn.Close()
		})
	}
}

func TestRegistry_UnknownType(t *testing.T) {
	registry := newTestRegistry()
	if _, err := registry.Create(model.DataSourceType("oracle"), json.RawMessage(`{}`)); err == nil {
		t.Fatal("Create() expected unsupported source type error")
	}
}
