package connector

import (
	"encoding/json"
	"testing"

	"github.com/clario360/platform/internal/data/model"
)

func TestSanitizeConnectionConfig(t *testing.T) {
	tests := []struct {
		name     string
		sourceType model.DataSourceType
		input    string
		blocked  []string
		required map[string]any
	}{
		{
			name:       "postgres",
			sourceType: model.DataSourceTypePostgreSQL,
			input:      `{"host":"db.example.com","port":5432,"database":"prod","username":"admin","password":"secret","ssl_mode":"require"}`,
			blocked:    []string{"password"},
			required:   map[string]any{"host": "db.example.com", "database": "prod", "username": "admin", "ssl_mode": "require"},
		},
		{
			name:       "mysql",
			sourceType: model.DataSourceTypeMySQL,
			input:      `{"host":"mysql.example.com","port":3306,"database":"core","username":"app","password":"secret"}`,
			blocked:    []string{"password"},
			required:   map[string]any{"host": "mysql.example.com", "database": "core", "username": "app"},
		},
		{
			name:       "api",
			sourceType: model.DataSourceTypeAPI,
			input:      `{"base_url":"https://api.example.com","auth_type":"bearer","auth_config":{"token":"secret"}}`,
			blocked:    []string{"auth_config"},
			required:   map[string]any{"base_url": "https://api.example.com", "auth_type": "bearer"},
		},
		{
			name:       "s3",
			sourceType: model.DataSourceTypeS3,
			input:      `{"endpoint":"minio.internal:9000","bucket":"lake","access_key":"abc","secret_key":"secret"}`,
			blocked:    []string{"secret_key", "access_key"},
			required:   map[string]any{"endpoint": "minio.internal:9000", "bucket": "lake"},
		},
		{
			name:       "csv",
			sourceType: model.DataSourceTypeCSV,
			input:      `{"bucket":"files","file_path":"reports.csv","access_key":"abc","secret_key":"secret"}`,
			blocked:    []string{"secret_key", "access_key"},
			required:   map[string]any{"bucket": "files", "file_path": "reports.csv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sanitized := SanitizeConnectionConfig(tt.sourceType, json.RawMessage(tt.input))
			var payload map[string]any
			if err := json.Unmarshal(sanitized, &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			for _, key := range tt.blocked {
				if _, exists := payload[key]; exists {
					t.Fatalf("blocked key %q still present", key)
				}
			}
			for key, want := range tt.required {
				if got := payload[key]; got != want {
					t.Fatalf("payload[%q] = %#v, want %#v", key, got, want)
				}
			}
		})
	}
}
