package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DataSourceType string

const (
	DataSourceTypePostgreSQL DataSourceType = "postgresql"
	DataSourceTypeMySQL      DataSourceType = "mysql"
	DataSourceTypeMSSQL      DataSourceType = "mssql"
	DataSourceTypeAPI        DataSourceType = "api"
	DataSourceTypeCSV        DataSourceType = "csv"
	DataSourceTypeS3         DataSourceType = "s3"
	DataSourceTypeClickHouse DataSourceType = "clickhouse"
	DataSourceTypeImpala     DataSourceType = "impala"
	DataSourceTypeHive       DataSourceType = "hive"
	DataSourceTypeHDFS       DataSourceType = "hdfs"
	DataSourceTypeSpark      DataSourceType = "spark"
	DataSourceTypeDagster    DataSourceType = "dagster"
	DataSourceTypeDolt       DataSourceType = "dolt"
	DataSourceTypeStream     DataSourceType = "stream"
)

func (t DataSourceType) IsValid() bool {
	switch t {
	case DataSourceTypePostgreSQL,
		DataSourceTypeMySQL,
		DataSourceTypeMSSQL,
		DataSourceTypeAPI,
		DataSourceTypeCSV,
		DataSourceTypeS3,
		DataSourceTypeClickHouse,
		DataSourceTypeImpala,
		DataSourceTypeHive,
		DataSourceTypeHDFS,
		DataSourceTypeSpark,
		DataSourceTypeDagster,
		DataSourceTypeDolt,
		DataSourceTypeStream:
		return true
	default:
		return false
	}
}

type DataSourceStatus string

const (
	DataSourceStatusPendingTest DataSourceStatus = "pending_test"
	DataSourceStatusActive      DataSourceStatus = "active"
	DataSourceStatusInactive    DataSourceStatus = "inactive"
	DataSourceStatusError       DataSourceStatus = "error"
	DataSourceStatusSyncing     DataSourceStatus = "syncing"
)

func (s DataSourceStatus) IsValid() bool {
	switch s {
	case DataSourceStatusPendingTest, DataSourceStatusActive, DataSourceStatusInactive, DataSourceStatusError, DataSourceStatusSyncing:
		return true
	default:
		return false
	}
}

type DataClassification string

const (
	DataClassificationPublic       DataClassification = "public"
	DataClassificationInternal     DataClassification = "internal"
	DataClassificationConfidential DataClassification = "confidential"
	DataClassificationRestricted   DataClassification = "restricted"
)

func (c DataClassification) IsValid() bool {
	switch c {
	case DataClassificationPublic, DataClassificationInternal, DataClassificationConfidential, DataClassificationRestricted:
		return true
	default:
		return false
	}
}

type DataSource struct {
	ID                 uuid.UUID         `json:"id"`
	TenantID           uuid.UUID         `json:"tenant_id"`
	Name               string            `json:"name"`
	Description        string            `json:"description"`
	Type               DataSourceType    `json:"type"`
	ConnectionConfig   json.RawMessage   `json:"connection_config,omitempty"`
	EncryptionKeyID    string            `json:"encryption_key_id,omitempty"`
	Status             DataSourceStatus  `json:"status"`
	LastError          *string           `json:"last_error,omitempty"`
	SchemaMetadata     *DiscoveredSchema `json:"schema_metadata,omitempty"`
	SchemaDiscoveredAt *time.Time        `json:"schema_discovered_at,omitempty"`
	LastSyncedAt       *time.Time        `json:"last_synced_at,omitempty"`
	LastSyncStatus     *string           `json:"last_sync_status,omitempty"`
	LastSyncError      *string           `json:"last_sync_error,omitempty"`
	LastSyncDurationMs *int64            `json:"last_sync_duration_ms,omitempty"`
	SyncFrequency      *string           `json:"sync_frequency,omitempty"`
	NextSyncAt         *time.Time        `json:"next_sync_at,omitempty"`
	TableCount         *int              `json:"table_count,omitempty"`
	TotalRowCount      *int64            `json:"total_row_count,omitempty"`
	TotalSizeBytes     *int64            `json:"total_size_bytes,omitempty"`
	Tags               []string          `json:"tags"`
	Metadata           json.RawMessage   `json:"metadata"`
	CreatedBy          uuid.UUID         `json:"created_by"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
	DeletedAt          *time.Time        `json:"deleted_at,omitempty"`
}

type PostgresConnectionConfig struct {
	Host               string `json:"host" validate:"required,hostname|ip"`
	Port               int    `json:"port" validate:"required,gte=1,lte=65535"`
	Database           string `json:"database" validate:"required"`
	Schema             string `json:"schema"`
	Username           string `json:"username" validate:"required"`
	Password           string `json:"password" validate:"required"`
	SSLMode            string `json:"ssl_mode" validate:"required,oneof=disable allow prefer require verify-ca verify-full"`
	StatementTimeoutMs int    `json:"statement_timeout_ms"`
}

type MySQLConnectionConfig struct {
	Host         string `json:"host" validate:"required,hostname|ip"`
	Port         int    `json:"port" validate:"required,gte=1,lte=65535"`
	Database     string `json:"database" validate:"required"`
	Username     string `json:"username" validate:"required"`
	Password     string `json:"password" validate:"required"`
	TLSMode      string `json:"tls_mode" validate:"omitempty,oneof=false true skip-verify preferred"`
	ReadTimeout  string `json:"read_timeout"`
	WriteTimeout string `json:"write_timeout"`
}

type APIAuthType string

const (
	APIAuthNone   APIAuthType = "none"
	APIAuthBasic  APIAuthType = "basic"
	APIAuthBearer APIAuthType = "bearer"
	APIAuthAPIKey APIAuthType = "api_key"
	APIAuthOAuth2 APIAuthType = "oauth2"
)

type APIPaginationType string

const (
	APIPaginationOffset     APIPaginationType = "offset"
	APIPaginationCursor     APIPaginationType = "cursor"
	APIPaginationPage       APIPaginationType = "page"
	APIPaginationLinkHeader APIPaginationType = "link_header"
)

type APIConnectionConfig struct {
	BaseURL               string            `json:"base_url" validate:"required,url"`
	HealthURL             string            `json:"health_url,omitempty"`
	DataPath              string            `json:"data_path,omitempty"`
	AuthType              APIAuthType       `json:"auth_type" validate:"required,oneof=none basic bearer api_key oauth2"`
	AuthConfig            map[string]any    `json:"auth_config,omitempty"`
	Headers               map[string]string `json:"headers,omitempty"`
	AllowHTTP             bool              `json:"allow_http"`
	AllowPrivateAddresses bool              `json:"allow_private_addresses"`
	AllowlistedHosts      []string          `json:"allowlisted_hosts,omitempty"`
	RateLimit             int               `json:"rate_limit,omitempty"`
	PaginationType        APIPaginationType `json:"pagination_type" validate:"required,oneof=offset cursor page link_header"`
	PaginationConfig      map[string]any    `json:"pagination_config,omitempty"`
	QueryParams           map[string]string `json:"query_params,omitempty"`
}

type CSVConnectionConfig struct {
	MinioEndpoint string `json:"minio_endpoint" validate:"required"`
	Bucket        string `json:"bucket" validate:"required"`
	FilePath      string `json:"file_path" validate:"required"`
	Delimiter     string `json:"delimiter,omitempty"`
	HasHeader     bool   `json:"has_header"`
	Encoding      string `json:"encoding,omitempty"`
	QuoteChar     string `json:"quote_char,omitempty"`
	AccessKey     string `json:"access_key" validate:"required"`
	SecretKey     string `json:"secret_key" validate:"required"`
	UseSSL        bool   `json:"use_ssl"`
}

type S3ConnectionConfig struct {
	Endpoint        string   `json:"endpoint" validate:"required"`
	Bucket          string   `json:"bucket" validate:"required"`
	Prefix          string   `json:"prefix,omitempty"`
	Region          string   `json:"region,omitempty"`
	AccessKey       string   `json:"access_key" validate:"required"`
	SecretKey       string   `json:"secret_key" validate:"required"`
	UseSSL          bool     `json:"use_ssl"`
	AllowedFormats  []string `json:"allowed_formats,omitempty"`
	MaxObjects      int      `json:"max_objects,omitempty"`
	SchemaFromFirst bool     `json:"schema_from_first"`
}

type KerberosConfig struct {
	Realm     string `json:"realm" validate:"required"`
	KDC       string `json:"kdc" validate:"required"`
	Principal string `json:"principal" validate:"required"`
	Keytab    string `json:"keytab,omitempty"`
}

type ClickHouseConnectionConfig struct {
	Host                string `json:"host" validate:"required,hostname|ip"`
	Port                int    `json:"port" validate:"required,gte=1,lte=65535"`
	Database            string `json:"database" validate:"required"`
	Username            string `json:"username" validate:"required"`
	Password            string `json:"password" validate:"required"`
	Secure              bool   `json:"secure"`
	Protocol            string `json:"protocol" validate:"required,oneof=native http"`
	Cluster             string `json:"cluster,omitempty"`
	MaxOpenConns        int    `json:"max_open_conns,omitempty"`
	MaxIdleConns        int    `json:"max_idle_conns,omitempty"`
	DialTimeoutSeconds  int    `json:"dial_timeout_seconds,omitempty"`
	ReadTimeoutSeconds  int    `json:"read_timeout_seconds,omitempty"`
	Compression         bool   `json:"compression"`
	ConnMaxLifetimeMins int    `json:"conn_max_lifetime_mins,omitempty"`
	ConnMaxIdleTimeMins int    `json:"conn_max_idle_time_mins,omitempty"`
}

type ImpalaConnectionConfig struct {
	Host                string `json:"host" validate:"required,hostname|ip"`
	Port                int    `json:"port" validate:"required,gte=1,lte=65535"`
	Database            string `json:"database,omitempty"`
	Username            string `json:"username,omitempty"`
	Password            string `json:"password,omitempty"`
	AuthType            string `json:"auth_type" validate:"required,oneof=noauth ldap kerberos"`
	UseTLS              bool   `json:"use_tls"`
	QueryTimeoutSeconds int    `json:"query_timeout_seconds,omitempty"`
	MaxOpenConns        int    `json:"max_open_conns,omitempty"`
	MaxIdleConns        int    `json:"max_idle_conns,omitempty"`
	ConnMaxLifetimeMins int    `json:"conn_max_lifetime_mins,omitempty"`
	AuditLogTable       string `json:"audit_log_table,omitempty"`
	CACertPath          string `json:"ca_cert_path,omitempty"`
}

type HiveConnectionConfig struct {
	Host                string          `json:"host" validate:"required,hostname|ip"`
	Port                int             `json:"port" validate:"required,gte=1,lte=65535"`
	Database            string          `json:"database,omitempty"`
	Username            string          `json:"username,omitempty"`
	Password            string          `json:"password,omitempty"`
	AuthType            string          `json:"auth_type" validate:"required,oneof=noauth plain kerberos"`
	TransportMode       string          `json:"transport_mode" validate:"required,oneof=binary http"`
	HTTPPath            string          `json:"http_path,omitempty"`
	KerberosConfig      *KerberosConfig `json:"kerberos,omitempty"`
	UseTLS              bool            `json:"use_tls"`
	QueryTimeoutSeconds int             `json:"query_timeout_seconds,omitempty"`
	FetchSize           int             `json:"fetch_size,omitempty"`
}

type HDFSConnectionConfig struct {
	NameNodes      []string        `json:"name_nodes" validate:"required,min=1,dive,required"`
	User           string          `json:"user,omitempty"`
	KerberosConfig *KerberosConfig `json:"kerberos,omitempty"`
	BasePaths      []string        `json:"base_paths,omitempty"`
	MaxFileSizeMB  int64           `json:"max_file_size_mb,omitempty"`
	AuditLogPath   string          `json:"audit_log_path,omitempty"`
}

type SparkThriftConfig struct {
	Host     string `json:"host" validate:"required,hostname|ip"`
	Port     int    `json:"port" validate:"required,gte=1,lte=65535"`
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	AuthType string `json:"auth_type" validate:"required,oneof=noauth plain kerberos"`
}

type SparkRESTConfig struct {
	MasterURL  string `json:"master_url" validate:"required,url"`
	HistoryURL string `json:"history_url,omitempty" validate:"omitempty,url"`
}

type SparkConnectionConfig struct {
	Thrift              *SparkThriftConfig `json:"thrift,omitempty"`
	REST                SparkRESTConfig    `json:"rest" validate:"required"`
	QueryTimeoutSeconds int                `json:"query_timeout_seconds,omitempty"`
	MaxOpenConns        int                `json:"max_open_conns,omitempty"`
	MaxIdleConns        int                `json:"max_idle_conns,omitempty"`
	ConnMaxLifetimeMins int                `json:"conn_max_lifetime_mins,omitempty"`
}

type DagsterConnectionConfig struct {
	GraphQLURL     string `json:"graphql_url" validate:"required,url"`
	APIToken       string `json:"api_token,omitempty"`
	Workspace      string `json:"workspace,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type DoltConnectionConfig struct {
	Host                string `json:"host" validate:"required,hostname|ip"`
	Port                int    `json:"port" validate:"required,gte=1,lte=65535"`
	Database            string `json:"database" validate:"required"`
	Username            string `json:"username" validate:"required"`
	Password            string `json:"password" validate:"required"`
	Branch              string `json:"branch,omitempty"`
	UseTLS              bool   `json:"use_tls"`
	MaxOpenConns        int    `json:"max_open_conns,omitempty"`
	MaxIdleConns        int    `json:"max_idle_conns,omitempty"`
	ConnMaxLifetimeMins int    `json:"conn_max_lifetime_mins,omitempty"`
}
