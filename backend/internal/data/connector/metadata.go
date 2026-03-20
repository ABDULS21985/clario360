package connector

import "github.com/clario360/platform/internal/data/model"

type ConnectorTypeMetadata struct {
	Type             model.DataSourceType `json:"type"`
	DisplayName      string               `json:"display_name"`
	Description      string               `json:"description"`
	Icon             string               `json:"icon"`
	Category         string               `json:"category"`
	SupportsSchema   bool                 `json:"supports_schema"`
	SupportsData     bool                 `json:"supports_data"`
	SupportsSecurity bool                 `json:"supports_security"`
	SupportsDSPM     bool                 `json:"supports_dspm"`
	ConfigFields     []ConfigField        `json:"config_fields"`
}

type ConfigField struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Placeholder string      `json:"placeholder,omitempty"`
	Options     []string    `json:"options,omitempty"`
	HelpText    string      `json:"help_text,omitempty"`
	Group       string      `json:"group,omitempty"`
}

func defaultConnectorMetadata() map[model.DataSourceType]ConnectorTypeMetadata {
	return map[model.DataSourceType]ConnectorTypeMetadata{
		model.DataSourceTypePostgreSQL: {
			Type:             model.DataSourceTypePostgreSQL,
			DisplayName:      "PostgreSQL",
			Description:      "Relational database",
			Icon:             "Database",
			Category:         "database",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "host", Label: "Host", Type: "text", Required: true, Placeholder: "db.example.com", Group: "connection"},
				{Name: "port", Label: "Port", Type: "number", Required: true, Default: 5432, Group: "connection"},
				{Name: "database", Label: "Database", Type: "text", Required: true, Group: "connection"},
				{Name: "schema", Label: "Schema", Type: "text", Required: false, Default: "public", Group: "connection"},
				{Name: "username", Label: "Username", Type: "text", Required: true, Group: "credentials"},
				{Name: "password", Label: "Password", Type: "password", Required: true, Group: "credentials"},
				{Name: "ssl_mode", Label: "SSL mode", Type: "select", Required: true, Default: "require", Options: []string{"disable", "require", "verify-ca", "verify-full"}, Group: "security"},
			},
		},
		model.DataSourceTypeMySQL: {
			Type:             model.DataSourceTypeMySQL,
			DisplayName:      "MySQL",
			Description:      "Relational database",
			Icon:             "Database",
			Category:         "database",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "host", Label: "Host", Type: "text", Required: true, Placeholder: "mysql.example.com", Group: "connection"},
				{Name: "port", Label: "Port", Type: "number", Required: true, Default: 3306, Group: "connection"},
				{Name: "database", Label: "Database", Type: "text", Required: true, Group: "connection"},
				{Name: "username", Label: "Username", Type: "text", Required: true, Group: "credentials"},
				{Name: "password", Label: "Password", Type: "password", Required: true, Group: "credentials"},
				{Name: "tls_mode", Label: "TLS mode", Type: "select", Required: false, Default: "true", Options: []string{"true", "false", "skip-verify", "preferred"}, Group: "security"},
			},
		},
		model.DataSourceTypeAPI: {
			Type:           model.DataSourceTypeAPI,
			DisplayName:    "REST API",
			Description:    "HTTP API endpoint",
			Icon:           "Globe",
			Category:       "api",
			SupportsSchema: true,
			SupportsData:   true,
			ConfigFields: []ConfigField{
				{Name: "base_url", Label: "Base URL", Type: "text", Required: true, Placeholder: "https://api.example.com", Group: "connection"},
				{Name: "auth_type", Label: "Authentication", Type: "select", Required: true, Default: "none", Options: []string{"none", "basic", "bearer", "api_key", "oauth2"}, Group: "security"},
			},
		},
		model.DataSourceTypeCSV: {
			Type:           model.DataSourceTypeCSV,
			DisplayName:    "CSV / File",
			Description:    "Delimited file in object storage",
			Icon:           "FileText",
			Category:       "file",
			SupportsSchema: true,
			SupportsData:   true,
			ConfigFields: []ConfigField{
				{Name: "bucket", Label: "Bucket", Type: "text", Required: true, Group: "connection"},
				{Name: "file_path", Label: "File path", Type: "text", Required: true, Group: "connection"},
			},
		},
		model.DataSourceTypeS3: {
			Type:           model.DataSourceTypeS3,
			DisplayName:    "S3 / MinIO",
			Description:    "Object storage",
			Icon:           "Cloud",
			Category:       "file",
			SupportsSchema: true,
			SupportsData:   true,
			ConfigFields: []ConfigField{
				{Name: "endpoint", Label: "Endpoint", Type: "text", Required: true, Group: "connection"},
				{Name: "bucket", Label: "Bucket", Type: "text", Required: true, Group: "connection"},
				{Name: "access_key", Label: "Access key", Type: "password", Required: true, Group: "credentials"},
				{Name: "secret_key", Label: "Secret key", Type: "password", Required: true, Group: "credentials"},
			},
		},
		model.DataSourceTypeClickHouse: {
			Type:             model.DataSourceTypeClickHouse,
			DisplayName:      "ClickHouse",
			Description:      "High-performance columnar analytics database",
			Icon:             "BarChart3",
			Category:         "database",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "host", Label: "Host", Type: "text", Required: true, Placeholder: "clickhouse.example.com", Group: "connection"},
				{Name: "port", Label: "Port", Type: "number", Required: true, Default: 9000, HelpText: "9000 for native protocol, 8123 for HTTP.", Group: "connection"},
				{Name: "database", Label: "Database", Type: "text", Required: true, Default: "default", Group: "connection"},
				{Name: "protocol", Label: "Protocol", Type: "select", Required: true, Default: "native", Options: []string{"native", "http"}, Group: "connection"},
				{Name: "username", Label: "Username", Type: "text", Required: true, Default: "default", Group: "credentials"},
				{Name: "password", Label: "Password", Type: "password", Required: true, Group: "credentials"},
				{Name: "secure", Label: "TLS", Type: "toggle", Required: false, Default: false, Group: "security"},
				{Name: "compression", Label: "Compression", Type: "toggle", Required: false, Default: true, Group: "performance"},
			},
		},
		model.DataSourceTypeImpala: {
			Type:             model.DataSourceTypeImpala,
			DisplayName:      "Apache Impala",
			Description:      "Interactive SQL analytics for Cloudera/Hadoop",
			Icon:             "Zap",
			Category:         "hadoop",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "host", Label: "Host", Type: "text", Required: true, Placeholder: "impala-coordinator.example.com", Group: "connection"},
				{Name: "port", Label: "Port", Type: "number", Required: true, Default: 21050, Group: "connection"},
				{Name: "database", Label: "Database", Type: "text", Required: false, Default: "default", Group: "connection"},
				{Name: "auth_type", Label: "Authentication", Type: "select", Required: true, Default: "noauth", Options: []string{"noauth", "ldap", "kerberos"}, Group: "security"},
				{Name: "username", Label: "Username", Type: "text", Required: false, Group: "credentials"},
				{Name: "password", Label: "Password", Type: "password", Required: false, Group: "credentials"},
				{Name: "use_tls", Label: "TLS", Type: "toggle", Required: false, Default: false, Group: "security"},
				{Name: "audit_log_table", Label: "Audit Log Table", Type: "text", Required: false, Placeholder: "sys.impala_audit", Group: "security"},
			},
		},
		model.DataSourceTypeHive: {
			Type:             model.DataSourceTypeHive,
			DisplayName:      "Apache Hive",
			Description:      "HiveServer2 data warehouse connector",
			Icon:             "Warehouse",
			Category:         "hadoop",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			SupportsDSPM:     true,
			ConfigFields: []ConfigField{
				{Name: "host", Label: "Host", Type: "text", Required: true, Placeholder: "hiveserver2.example.com", Group: "connection"},
				{Name: "port", Label: "Port", Type: "number", Required: true, Default: 10000, Group: "connection"},
				{Name: "database", Label: "Database", Type: "text", Required: false, Default: "default", Group: "connection"},
				{Name: "transport_mode", Label: "Transport mode", Type: "select", Required: true, Default: "binary", Options: []string{"binary", "http"}, Group: "connection"},
				{Name: "http_path", Label: "HTTP path", Type: "text", Required: false, Default: "cliservice", Group: "connection"},
				{Name: "auth_type", Label: "Authentication", Type: "select", Required: true, Default: "noauth", Options: []string{"noauth", "plain", "kerberos"}, Group: "security"},
				{Name: "username", Label: "Username", Type: "text", Required: false, Group: "credentials"},
				{Name: "password", Label: "Password", Type: "password", Required: false, Group: "credentials"},
				{Name: "use_tls", Label: "TLS", Type: "toggle", Required: false, Default: false, Group: "security"},
			},
		},
		model.DataSourceTypeHDFS: {
			Type:             model.DataSourceTypeHDFS,
			DisplayName:      "HDFS",
			Description:      "Hadoop Distributed File System",
			Icon:             "HardDrive",
			Category:         "hadoop",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			SupportsDSPM:     true,
			ConfigFields: []ConfigField{
				{Name: "name_nodes", Label: "Name nodes", Type: "multi-text", Required: true, Placeholder: "namenode:8020", Group: "connection"},
				{Name: "user", Label: "User", Type: "text", Required: false, Placeholder: "hdfs", Group: "credentials"},
				{Name: "base_paths", Label: "Base paths", Type: "multi-text", Required: false, Default: []string{"/user/hive/warehouse"}, Group: "discovery"},
				{Name: "max_file_size_mb", Label: "Max file size (MB)", Type: "number", Required: false, Default: 100, Group: "discovery"},
				{Name: "audit_log_path", Label: "Audit log path", Type: "text", Required: false, Group: "security"},
			},
		},
		model.DataSourceTypeSpark: {
			Type:             model.DataSourceTypeSpark,
			DisplayName:      "Apache Spark",
			Description:      "Spark SQL and job monitoring",
			Icon:             "Flame",
			Category:         "hadoop",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "thrift.host", Label: "Thrift host", Type: "text", Required: false, Group: "sql"},
				{Name: "thrift.port", Label: "Thrift port", Type: "number", Required: false, Default: 10001, Group: "sql"},
				{Name: "thrift.database", Label: "Database", Type: "text", Required: false, Default: "default", Group: "sql"},
				{Name: "thrift.auth_type", Label: "Authentication", Type: "select", Required: false, Default: "noauth", Options: []string{"noauth", "plain", "kerberos"}, Group: "sql"},
				{Name: "rest.master_url", Label: "Master URL", Type: "text", Required: true, Placeholder: "http://spark-master:8080", Group: "monitoring"},
				{Name: "rest.history_url", Label: "History URL", Type: "text", Required: false, Placeholder: "http://spark-history:18080", Group: "monitoring"},
			},
		},
		model.DataSourceTypeDagster: {
			Type:             model.DataSourceTypeDagster,
			DisplayName:      "Dagster",
			Description:      "Data pipeline orchestration via GraphQL",
			Icon:             "GitBranch",
			Category:         "orchestration",
			SupportsSchema:   true,
			SupportsData:     false,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "graphql_url", Label: "GraphQL URL", Type: "text", Required: true, Placeholder: "http://dagster-webserver:3000/graphql", Group: "connection"},
				{Name: "api_token", Label: "API token", Type: "password", Required: false, Group: "credentials"},
				{Name: "workspace", Label: "Workspace", Type: "text", Required: false, Group: "connection"},
			},
		},
		model.DataSourceTypeDolt: {
			Type:             model.DataSourceTypeDolt,
			DisplayName:      "Dolt",
			Description:      "Versioned SQL database (MySQL-compatible)",
			Icon:             "GitCommit",
			Category:         "database",
			SupportsSchema:   true,
			SupportsData:     true,
			SupportsSecurity: true,
			ConfigFields: []ConfigField{
				{Name: "host", Label: "Host", Type: "text", Required: true, Placeholder: "dolt-server.example.com", Group: "connection"},
				{Name: "port", Label: "Port", Type: "number", Required: true, Default: 3306, Group: "connection"},
				{Name: "database", Label: "Database", Type: "text", Required: true, Group: "connection"},
				{Name: "username", Label: "Username", Type: "text", Required: true, Group: "credentials"},
				{Name: "password", Label: "Password", Type: "password", Required: true, Group: "credentials"},
				{Name: "branch", Label: "Branch", Type: "text", Required: false, Default: "main", Group: "connection"},
				{Name: "use_tls", Label: "TLS", Type: "toggle", Required: false, Default: false, Group: "security"},
			},
		},
	}
}
