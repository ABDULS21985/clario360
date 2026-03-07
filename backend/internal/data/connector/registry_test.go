package connector

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
)

func TestRegistry_CreateKnownConnectors(t *testing.T) {
	registry := NewConnectorRegistry(ConnectorLimits{
		MaxPoolSize:      3,
		StatementTimeout: 30 * time.Second,
		ConnectTimeout:   10 * time.Second,
		MaxSampleRows:    100,
		MaxTables:        500,
		APIRateLimit:     10,
	}, zerolog.Nop())

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
	registry := NewConnectorRegistry(ConnectorLimits{
		MaxPoolSize:      3,
		StatementTimeout: 30 * time.Second,
		ConnectTimeout:   10 * time.Second,
		MaxSampleRows:    100,
		MaxTables:        500,
		APIRateLimit:     10,
	}, zerolog.Nop())

	if _, err := registry.Create(model.DataSourceType("oracle"), json.RawMessage(`{}`)); err == nil || !strings.Contains(err.Error(), "unsupported source type") {
		t.Fatalf("Create() error = %v, want unsupported source type", err)
	}
}
