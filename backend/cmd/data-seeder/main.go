package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	datamodel "github.com/clario360/platform/internal/data/model"
	datasvc "github.com/clario360/platform/internal/data/service"
	"github.com/clario360/platform/internal/observability"
)

type sourceSeed struct {
	ID              uuid.UUID
	Name            string
	Description     string
	Type            datamodel.DataSourceType
	Config          any
	Status          datamodel.DataSourceStatus
	Schema          datamodel.DiscoveredSchema
	TableCount      int
	TotalRowCount   int64
	TotalSizeBytes  int64
	SyncFrequency   *string
	LastSyncedAt    *time.Time
	LastSyncStatus  *string
	LastSyncError   *string
	LastSyncMs      *int64
	Tags            []string
	Metadata        json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type modelSeed struct {
	ID                 uuid.UUID
	Name               string
	DisplayName        string
	Description        string
	Status             datamodel.DataModelStatus
	SourceName         string
	SourceTable        string
	SchemaDefinition   []datamodel.ModelField
	QualityRules       []datamodel.ValidationRule
	DataClassification datamodel.DataClassification
	ContainsPII        bool
	PIIColumns         []string
	Tags               []string
	Metadata           json.RawMessage
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func main() {
	var (
		dbURL       = flag.String("db-url", os.Getenv("DATA_DB_URL"), "Data suite PostgreSQL connection string")
		keyBase64   = flag.String("encryption-key", os.Getenv("DATA_ENCRYPTION_KEY"), "Base64-encoded AES-256 key")
		tenantIDRaw = flag.String("tenant-id", os.Getenv("DATA_SEED_TENANT_ID"), "Tenant UUID to seed")
	)
	flag.Parse()

	if strings.TrimSpace(*dbURL) == "" {
		fmt.Fprintln(os.Stderr, "--db-url or DATA_DB_URL is required")
		os.Exit(1)
	}
	if strings.TrimSpace(*keyBase64) == "" {
		fmt.Fprintln(os.Stderr, "--encryption-key or DATA_ENCRYPTION_KEY is required")
		os.Exit(1)
	}

	tenantID := uuid.New()
	if raw := strings.TrimSpace(*tenantIDRaw); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid tenant id: %v\n", err)
			os.Exit(1)
		}
		tenantID = parsed
	}

	logger := observability.NewLogger("info", "console", "data-seeder")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create database pool")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to data database")
	}

	encryptor, err := datasvc.NewConfigEncryptor(*keyBase64)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize config encryptor")
	}

	now := time.Now().UTC().Truncate(time.Second)
	createdBy := uuid.New()
	sources := buildSourceSeeds(now)
	models := buildModelSeeds(now, sources)

	if err := seedDataSuite(ctx, pool, encryptor, tenantID, createdBy, sources, models); err != nil {
		logger.Fatal().Err(err).Str("tenant_id", tenantID.String()).Msg("failed to seed data suite")
	}

	logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("sources", len(sources)).
		Int("models", len(models)).
		Msg("data suite seed completed")
}

func seedDataSuite(ctx context.Context, pool *pgxpool.Pool, encryptor *datasvc.ConfigEncryptor, tenantID, createdBy uuid.UUID, sources []sourceSeed, models []modelSeed) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin data seed transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `DELETE FROM sync_history WHERE tenant_id = $1`, tenantID); err != nil {
		return fmt.Errorf("clear sync_history: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM data_models WHERE tenant_id = $1`, tenantID); err != nil {
		return fmt.Errorf("clear data_models: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM data_sources WHERE tenant_id = $1`, tenantID); err != nil {
		return fmt.Errorf("clear data_sources: %w", err)
	}

	sourceIDs := make(map[string]uuid.UUID, len(sources))
	for _, source := range sources {
		rawConfig, err := json.Marshal(source.Config)
		if err != nil {
			return fmt.Errorf("marshal config for %s: %w", source.Name, err)
		}
		encryptedConfig, keyID, err := encryptor.Encrypt(append([]byte(nil), rawConfig...))
		if err != nil {
			return fmt.Errorf("encrypt config for %s: %w", source.Name, err)
		}
		schemaJSON, err := json.Marshal(source.Schema)
		if err != nil {
			return fmt.Errorf("marshal schema for %s: %w", source.Name, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO data_sources (
				id, tenant_id, name, description, type, connection_config, encryption_key_id, status,
				schema_metadata, schema_discovered_at, last_synced_at, last_sync_status, last_sync_error,
				last_sync_duration_ms, sync_frequency, next_sync_at, table_count, total_row_count,
				total_size_bytes, tags, metadata, created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, $12, $13,
				$14, $15, $16, $17, $18,
				$19, $20, $21, $22, $23, $24
			)`,
			source.ID, tenantID, source.Name, source.Description, source.Type, encryptedConfig, keyID, source.Status,
			schemaJSON, source.UpdatedAt, source.LastSyncedAt, source.LastSyncStatus, source.LastSyncError,
			source.LastSyncMs, source.SyncFrequency, nextSyncTime(source.SyncFrequency, source.UpdatedAt), source.TableCount, source.TotalRowCount,
			source.TotalSizeBytes, source.Tags, coalesceJSON(source.Metadata), createdBy, source.CreatedAt, source.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert data source %s: %w", source.Name, err)
		}
		sourceIDs[source.Name] = source.ID

		if source.LastSyncedAt != nil && source.LastSyncStatus != nil {
			duration := int64(4500)
			if source.LastSyncMs != nil {
				duration = *source.LastSyncMs
			}
			startedAt := source.LastSyncedAt.Add(-time.Duration(duration) * time.Millisecond)
			errorsJSON := json.RawMessage(`[]`)
			if source.LastSyncError != nil && strings.TrimSpace(*source.LastSyncError) != "" {
				payload, _ := json.Marshal([]string{*source.LastSyncError})
				errorsJSON = payload
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO sync_history (
					id, tenant_id, source_id, status, sync_type, tables_synced, rows_read, rows_written,
					bytes_transferred, errors, error_count, started_at, completed_at, duration_ms,
					triggered_by, triggered_by_user, created_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8,
					$9, $10, $11, $12, $13, $14,
					$15, $16, $17
				)`,
				uuid.New(), tenantID, source.ID, *source.LastSyncStatus, datamodel.SyncTypeFull, source.TableCount, source.TotalRowCount, source.TotalRowCount,
				source.TotalSizeBytes, errorsJSON, countErrors(source.LastSyncError), startedAt, source.LastSyncedAt, duration,
				datamodel.SyncTriggerSchedule, createdBy, source.LastSyncedAt,
			)
			if err != nil {
				return fmt.Errorf("insert sync history for %s: %w", source.Name, err)
			}
		}
	}

	for _, item := range models {
		sourceID, ok := sourceIDs[item.SourceName]
		if !ok {
			return fmt.Errorf("unknown source reference %s for model %s", item.SourceName, item.Name)
		}
		schemaJSON, err := json.Marshal(item.SchemaDefinition)
		if err != nil {
			return fmt.Errorf("marshal schema definition for %s: %w", item.Name, err)
		}
		rulesJSON, err := json.Marshal(item.QualityRules)
		if err != nil {
			return fmt.Errorf("marshal quality rules for %s: %w", item.Name, err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO data_models (
				id, tenant_id, name, display_name, description, status, schema_definition, source_id,
				source_table, quality_rules, data_classification, contains_pii, pii_columns, field_count,
				version, previous_version_id, tags, metadata, created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, $11, $12, $13, $14,
				$15, $16, $17, $18, $19, $20, $21
			)`,
			item.ID, tenantID, item.Name, item.DisplayName, item.Description, item.Status, schemaJSON, sourceID,
			item.SourceTable, rulesJSON, item.DataClassification, item.ContainsPII, item.PIIColumns, len(item.SchemaDefinition),
			1, nil, item.Tags, coalesceJSON(item.Metadata), createdBy, item.CreatedAt, item.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert data model %s: %w", item.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit data seed transaction: %w", err)
	}
	return nil
}

func buildSourceSeeds(now time.Time) []sourceSeed {
	hourly := "0 * * * *"
	daily := "0 2 * * *"
	lastSync := now.Add(-6 * time.Hour)
	success := "success"
	partial := "partial"
	partialError := "one non-critical table timed out during incremental load"
	lastSyncMs := int64(6200)
	lastSyncSlow := int64(11800)

	customerTables := []datamodel.DiscoveredTable{
		makeTable("public", "customers", "base table", datamodel.DataClassificationConfidential, 120000, 64<<20,
			column("customer_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("first_name", "varchar", "varchar(120)", false, false, "name", datamodel.DataClassificationConfidential, nil, []string{"Ada", "Maya"}),
			column("last_name", "varchar", "varchar(120)", false, false, "name", datamodel.DataClassificationConfidential, nil, []string{"Stone", "Bello"}),
			column("email", "varchar", "varchar(255)", false, false, "email", datamodel.DataClassificationConfidential, nil, []string{"ada@example.com", "maya@example.com"}),
			column("phone_number", "varchar", "varchar(40)", true, false, "phone", datamodel.DataClassificationConfidential, nil, []string{"+1 555 0100", "+1 555 0101"}),
			column("street_address", "varchar", "varchar(255)", true, false, "address", datamodel.DataClassificationConfidential, nil, []string{"1 Main St", "42 Ridge Ave"}),
			column("created_at", "timestamptz", "timestamptz", false, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
		makeTable("public", "transactions", "base table", datamodel.DataClassificationRestricted, 840000, 96<<20,
			column("transaction_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("customer_id", "uuid", "uuid", false, false, "", datamodel.DataClassificationInternal, &datamodel.ForeignKeyRef{Table: "customers", Column: "customer_id"}, nil),
			column("card_number", "varchar", "varchar(24)", false, false, "credit_card", datamodel.DataClassificationRestricted, nil, []string{"4111111111111111"}),
			column("billing_amount", "numeric", "numeric(12,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"19.99", "120.50"}),
			column("currency", "varchar", "varchar(3)", false, false, "", datamodel.DataClassificationPublic, nil, []string{"USD", "EUR"}),
			column("posted_at", "timestamptz", "timestamptz", false, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
		makeTable("public", "activity", "base table", datamodel.DataClassificationInternal, 6200000, 180<<20,
			column("activity_id", "bigint", "bigint", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("customer_id", "uuid", "uuid", false, false, "", datamodel.DataClassificationInternal, &datamodel.ForeignKeyRef{Table: "customers", Column: "customer_id"}, nil),
			column("ip_address", "inet", "inet", true, false, "technical_id", datamodel.DataClassificationInternal, nil, []string{"203.0.113.15"}),
			column("user_agent", "text", "text", true, false, "technical_id", datamodel.DataClassificationInternal, nil, []string{"Mozilla/5.0"}),
			column("event_type", "varchar", "varchar(50)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"login", "purchase"}),
			column("occurred_at", "timestamptz", "timestamptz", false, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
	}

	warehouseTables := []datamodel.DiscoveredTable{
		makeTable("finance", "gl_entries", "base table", datamodel.DataClassificationConfidential, 2800000, 220<<20,
			column("entry_id", "bigint", "bigint", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("ledger_account", "varchar", "varchar(60)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"4000", "5000"}),
			column("amount", "numeric", "numeric(18,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"1250.00"}),
			column("cost_center", "varchar", "varchar(50)", true, false, "", datamodel.DataClassificationInternal, nil, []string{"ENG", "OPS"}),
			column("posted_at", "date", "date", false, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
		makeTable("analytics", "customer_360", "view", datamodel.DataClassificationConfidential, 120000, 48<<20,
			column("customer_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("full_name", "varchar", "varchar(255)", false, false, "name", datamodel.DataClassificationConfidential, nil, []string{"Ada Stone"}),
			column("email", "varchar", "varchar(255)", false, false, "email", datamodel.DataClassificationConfidential, nil, []string{"ada@example.com"}),
			column("lifetime_value", "numeric", "numeric(12,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"899.50"}),
			column("churn_risk_band", "varchar", "varchar(20)", true, false, "", datamodel.DataClassificationInternal, nil, []string{"low", "medium", "high"}),
		),
		makeTable("analytics", "revenue_summary", "view", datamodel.DataClassificationInternal, 365, 2<<20,
			column("summary_date", "date", "date", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("revenue_amount", "numeric", "numeric(18,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"150000.00"}),
			column("region", "varchar", "varchar(30)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"NA", "EMEA"}),
		),
	}

	legacyCRMTables := []datamodel.DiscoveredTable{
		makeTable("crm", "vendors", "base table", datamodel.DataClassificationConfidential, 4500, 6<<20,
			column("vendor_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("vendor_name", "varchar", "varchar(255)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"Acme Supplies"}),
			column("contact_name", "varchar", "varchar(255)", true, false, "name", datamodel.DataClassificationConfidential, nil, []string{"Jon Snow"}),
			column("contact_email", "varchar", "varchar(255)", true, false, "email", datamodel.DataClassificationConfidential, nil, []string{"jon@vendor.com"}),
			column("contact_phone", "varchar", "varchar(40)", true, false, "phone", datamodel.DataClassificationConfidential, nil, []string{"+1 555 0199"}),
		),
		makeTable("crm", "opportunities", "base table", datamodel.DataClassificationInternal, 38000, 14<<20,
			column("opportunity_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("account_name", "varchar", "varchar(255)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"Globex"}),
			column("stage", "varchar", "varchar(40)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"qualified", "proposal", "closed"}),
			column("amount", "numeric", "numeric(14,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"42000.00"}),
		),
	}

	hrTables := []datamodel.DiscoveredTable{
		makeTable("hr", "employees", "base table", datamodel.DataClassificationRestricted, 9200, 18<<20,
			column("employee_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("full_name", "varchar", "varchar(255)", false, false, "name", datamodel.DataClassificationConfidential, nil, []string{"Amaka Obi"}),
			column("work_email", "varchar", "varchar(255)", false, false, "email", datamodel.DataClassificationConfidential, nil, []string{"amaka@company.com"}),
			column("ssn", "varchar", "varchar(20)", false, false, "national_id", datamodel.DataClassificationRestricted, nil, []string{"123-45-6789"}),
			column("date_of_birth", "date", "date", true, false, "dob", datamodel.DataClassificationConfidential, nil, nil),
			column("base_salary", "numeric", "numeric(12,2)", false, false, "financial", datamodel.DataClassificationRestricted, nil, []string{"85000.00"}),
		),
		makeTable("hr", "payroll", "base table", datamodel.DataClassificationRestricted, 9200, 12<<20,
			column("payroll_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("employee_id", "uuid", "uuid", false, false, "", datamodel.DataClassificationInternal, &datamodel.ForeignKeyRef{Table: "employees", Column: "employee_id"}, nil),
			column("bank_account", "varchar", "varchar(40)", false, false, "bank_account", datamodel.DataClassificationRestricted, nil, []string{"DE75512108001245126199"}),
			column("net_pay", "numeric", "numeric(12,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"6100.00"}),
		),
	}

	ecommerceTables := []datamodel.DiscoveredTable{
		makeTable("commerce", "products", "base table", datamodel.DataClassificationPublic, 32000, 10<<20,
			column("product_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationPublic, nil, nil),
			column("sku", "varchar", "varchar(64)", false, false, "", datamodel.DataClassificationPublic, nil, []string{"SKU-1000"}),
			column("product_name", "varchar", "varchar(255)", false, false, "", datamodel.DataClassificationPublic, nil, []string{"Wireless Headset"}),
			column("category", "varchar", "varchar(80)", true, false, "", datamodel.DataClassificationPublic, nil, []string{"audio", "mobile"}),
			column("list_price", "numeric", "numeric(10,2)", false, false, "", datamodel.DataClassificationPublic, nil, []string{"59.99"}),
		),
		makeTable("commerce", "orders", "base table", datamodel.DataClassificationConfidential, 410000, 76<<20,
			column("order_id", "uuid", "uuid", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("customer_email", "varchar", "varchar(255)", false, false, "email", datamodel.DataClassificationConfidential, nil, []string{"buyer@example.com"}),
			column("shipping_address", "varchar", "varchar(255)", true, false, "address", datamodel.DataClassificationConfidential, nil, []string{"20 Harbor Rd"}),
			column("total_amount", "numeric", "numeric(12,2)", false, false, "", datamodel.DataClassificationInternal, nil, []string{"149.99"}),
			column("ordered_at", "timestamptz", "timestamptz", false, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
	}

	apiTables := []datamodel.DiscoveredTable{
		makeTable("", "api_response", "api", datamodel.DataClassificationPublic, 10000, 1<<20,
			column("city", "string", "string", false, false, "", datamodel.DataClassificationPublic, nil, []string{"Lagos", "Berlin"}),
			column("temperature_c", "float", "float", false, false, "", datamodel.DataClassificationPublic, nil, []string{"28.4", "14.2"}),
			column("observed_at", "datetime", "datetime", false, false, "", datamodel.DataClassificationPublic, nil, nil),
		),
	}

	crmAPITables := []datamodel.DiscoveredTable{
		makeTable("", "leads", "api", datamodel.DataClassificationConfidential, 14000, 2<<20,
			column("lead_id", "string", "string", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("company_name", "string", "string", false, false, "", datamodel.DataClassificationInternal, nil, []string{"Globex"}),
			column("contact_email", "string", "string", true, false, "email", datamodel.DataClassificationConfidential, nil, []string{"lead@globex.com"}),
			column("contact_phone", "string", "string", true, false, "phone", datamodel.DataClassificationConfidential, nil, []string{"+1 555 0104"}),
		),
		makeTable("", "accounts", "api", datamodel.DataClassificationConfidential, 3200, 1<<20,
			column("account_id", "string", "string", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("account_name", "string", "string", false, false, "", datamodel.DataClassificationInternal, nil, []string{"Initech"}),
			column("billing_address", "string", "string", true, false, "address", datamodel.DataClassificationConfidential, nil, []string{"4 Sunset Blvd"}),
		),
	}

	csvReportTables := []datamodel.DiscoveredTable{
		makeTable("", "quarterly_reports", "file", datamodel.DataClassificationInternal, 48, 128<<10,
			column("report_period", "string", "string", false, true, "", datamodel.DataClassificationInternal, nil, []string{"2025-Q1"}),
			column("business_unit", "string", "string", false, false, "", datamodel.DataClassificationInternal, nil, []string{"Retail"}),
			column("revenue", "float", "float", false, false, "", datamodel.DataClassificationInternal, nil, []string{"1250000.00"}),
			column("operating_margin", "float", "float", true, false, "", datamodel.DataClassificationInternal, nil, []string{"0.19"}),
		),
	}

	csvDirectoryTables := []datamodel.DiscoveredTable{
		makeTable("", "directory", "file", datamodel.DataClassificationConfidential, 9200, 512<<10,
			column("employee_id", "string", "string", false, true, "", datamodel.DataClassificationInternal, nil, []string{"EMP-1001"}),
			column("full_name", "string", "string", false, false, "name", datamodel.DataClassificationConfidential, nil, []string{"Amaka Obi"}),
			column("email_address", "string", "string", false, false, "email", datamodel.DataClassificationConfidential, nil, []string{"amaka@company.com"}),
			column("department", "string", "string", true, false, "", datamodel.DataClassificationInternal, nil, []string{"Engineering"}),
		),
	}

	s3Tables := []datamodel.DiscoveredTable{
		makeTable("lake", "raw_events", "object", datamodel.DataClassificationInternal, 12000000, 480<<20,
			column("event_id", "string", "string", false, true, "", datamodel.DataClassificationInternal, nil, nil),
			column("source_system", "string", "string", false, false, "", datamodel.DataClassificationInternal, nil, []string{"web", "mobile"}),
			column("payload", "json", "json", false, false, "", datamodel.DataClassificationInternal, nil, nil),
			column("ingested_at", "datetime", "datetime", false, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
		makeTable("lake", "device_inventory", "object", datamodel.DataClassificationInternal, 850000, 90<<20,
			column("device_id", "string", "string", false, true, "technical_id", datamodel.DataClassificationInternal, nil, []string{"dev-001"}),
			column("ip_address", "string", "string", true, false, "technical_id", datamodel.DataClassificationInternal, nil, []string{"10.1.0.5"}),
			column("hostname", "string", "string", true, false, "", datamodel.DataClassificationInternal, nil, []string{"db-prod-01"}),
			column("last_seen_at", "datetime", "datetime", true, false, "", datamodel.DataClassificationInternal, nil, nil),
		),
	}

	return []sourceSeed{
		newSourceSeed("customer_db", "Primary production customer system", datamodel.DataSourceTypePostgreSQL, datamodel.PostgresConnectionConfig{Host: "customer-db.internal", Port: 5432, Database: "customer", Schema: "public", Username: "svc_data", Password: "replace-me", SSLMode: "require", StatementTimeoutMs: 30000}, customerTables, 260<<20, 7160000, &hourly, &lastSync, &success, nil, &lastSyncMs, []string{"production", "customer"}),
		newSourceSeed("analytics_warehouse", "Reporting warehouse for finance and customer analytics", datamodel.DataSourceTypePostgreSQL, datamodel.PostgresConnectionConfig{Host: "warehouse.internal", Port: 5432, Database: "analytics", Schema: "analytics", Username: "svc_data", Password: "replace-me", SSLMode: "require", StatementTimeoutMs: 30000}, warehouseTables, 270<<20, 2920365, &daily, &lastSync, &success, nil, &lastSyncSlow, []string{"warehouse", "reporting"}),
		newSourceSeed("legacy_crm", "Legacy CRM pending migration", datamodel.DataSourceTypePostgreSQL, datamodel.PostgresConnectionConfig{Host: "legacy-crm.internal", Port: 5432, Database: "legacy_crm", Schema: "crm", Username: "svc_data", Password: "replace-me", SSLMode: "require", StatementTimeoutMs: 30000}, legacyCRMTables, 20<<20, 42500, &daily, &lastSync, &partial, &partialError, &lastSyncSlow, []string{"legacy", "crm"}),
		newSourceSeed("hr_system", "Human resources and payroll system", datamodel.DataSourceTypeMySQL, datamodel.MySQLConnectionConfig{Host: "hr-mysql.internal", Port: 3306, Database: "hr", Username: "svc_data", Password: "replace-me", TLSMode: "preferred"}, hrTables, 30<<20, 18400, &daily, &lastSync, &success, nil, &lastSyncMs, []string{"hr", "restricted"}),
		newSourceSeed("ecommerce_db", "E-commerce transactional system", datamodel.DataSourceTypeMySQL, datamodel.MySQLConnectionConfig{Host: "ecommerce-mysql.internal", Port: 3306, Database: "commerce", Username: "svc_data", Password: "replace-me", TLSMode: "preferred"}, ecommerceTables, 86<<20, 442000, &hourly, &lastSync, &success, nil, &lastSyncMs, []string{"commerce", "orders"}),
		newSourceSeed("weather_api", "Public weather enrichment feed", datamodel.DataSourceTypeAPI, datamodel.APIConnectionConfig{BaseURL: "https://api.weather.example/v1/observations", AuthType: datamodel.APIAuthAPIKey, AuthConfig: map[string]any{"key_name": "X-API-Key", "key_value": "replace-me"}, PaginationType: datamodel.APIPaginationOffset, DataPath: "$.data", RateLimit: 5}, apiTables, 1<<20, 10000, &hourly, &lastSync, &success, nil, &lastSyncMs, []string{"public", "enrichment"}),
		newSourceSeed("crm_api", "External CRM platform API", datamodel.DataSourceTypeAPI, datamodel.APIConnectionConfig{BaseURL: "https://crm-api.example/v2/leads", AuthType: datamodel.APIAuthBearer, AuthConfig: map[string]any{"token": "replace-me"}, PaginationType: datamodel.APIPaginationCursor, PaginationConfig: map[string]any{"pagination_cursor_field": "meta.next_cursor"}, DataPath: "$.data", RateLimit: 4}, crmAPITables, 3<<20, 17200, &hourly, &lastSync, &success, nil, &lastSyncMs, []string{"crm", "external"}),
		newSourceSeed("quarterly_reports.csv", "Quarterly board reporting extracts", datamodel.DataSourceTypeCSV, datamodel.CSVConnectionConfig{MinioEndpoint: "minio:9000", Bucket: "clario-data", FilePath: "reports/quarterly_reports.csv", Delimiter: ",", HasHeader: true, Encoding: "utf-8", QuoteChar: "\"", AccessKey: "minio", SecretKey: "replace-me", UseSSL: false}, csvReportTables, 128<<10, 48, nil, &lastSync, &success, nil, &lastSyncMs, []string{"csv", "finance"}),
		newSourceSeed("employee_directory.csv", "Employee directory export from HR", datamodel.DataSourceTypeCSV, datamodel.CSVConnectionConfig{MinioEndpoint: "minio:9000", Bucket: "clario-data", FilePath: "exports/employee_directory.csv", Delimiter: ",", HasHeader: true, Encoding: "utf-8", QuoteChar: "\"", AccessKey: "minio", SecretKey: "replace-me", UseSSL: false}, csvDirectoryTables, 512<<10, 9200, nil, &lastSync, &success, nil, &lastSyncMs, []string{"csv", "directory"}),
		newSourceSeed("data-lake-raw", "Raw object storage landing zone", datamodel.DataSourceTypeS3, datamodel.S3ConnectionConfig{Endpoint: "minio:9000", Bucket: "clario-lake", Prefix: "raw/", Region: "local", AccessKey: "minio", SecretKey: "replace-me", UseSSL: false, AllowedFormats: []string{"json", "jsonl", "csv"}, MaxObjects: 100, SchemaFromFirst: true}, s3Tables, 570<<20, 12850000, &daily, &lastSync, &success, nil, &lastSyncSlow, []string{"lake", "raw"}),
	}
}

func buildModelSeeds(now time.Time, sources []sourceSeed) []modelSeed {
	specs := []struct {
		name        string
		displayName string
		description string
		status      datamodel.DataModelStatus
		source      string
		table       string
		tags        []string
	}{
		{"customer_master", "Customer Master", "Canonical customer profile for downstream analytics and AI.", datamodel.DataModelStatusActive, "customer_db", "customers", []string{"golden-record", "pii"}},
		{"transaction_log", "Transaction Log", "Normalized customer payment transactions.", datamodel.DataModelStatusActive, "customer_db", "transactions", []string{"finance", "payments"}},
		{"employee_record", "Employee Record", "Restricted HR employee master record.", datamodel.DataModelStatusActive, "hr_system", "employees", []string{"hr", "restricted"}},
		{"product_catalog", "Product Catalog", "Public product metadata for storefront and search.", datamodel.DataModelStatusActive, "ecommerce_db", "products", []string{"catalog", "public"}},
		{"financial_ledger", "Financial Ledger", "Finance ledger entries for audit and reporting.", datamodel.DataModelStatusActive, "analytics_warehouse", "gl_entries", []string{"finance", "ledger"}},
		{"user_activity_log", "User Activity Log", "Behavioral events for retention and anomaly detection.", datamodel.DataModelStatusActive, "customer_db", "activity", []string{"behavioral", "events"}},
		{"vendor_contacts", "Vendor Contacts", "Third-party vendor contact management model.", datamodel.DataModelStatusDraft, "legacy_crm", "vendors", []string{"vendors", "crm"}},
		{"sales_pipeline", "Sales Pipeline", "Opportunity pipeline extracted from legacy CRM.", datamodel.DataModelStatusDraft, "legacy_crm", "opportunities", []string{"sales"}},
		{"lead_intake", "Lead Intake", "Inbound CRM leads from external platform.", datamodel.DataModelStatusActive, "crm_api", "leads", []string{"crm", "leads"}},
		{"account_master", "Account Master", "CRM customer account record.", datamodel.DataModelStatusActive, "crm_api", "accounts", []string{"crm", "accounts"}},
		{"quarterly_financial_report", "Quarterly Financial Report", "Board-level quarterly performance dataset.", datamodel.DataModelStatusDraft, "quarterly_reports.csv", "quarterly_reports", []string{"finance", "board"}},
		{"employee_directory", "Employee Directory", "Operational people directory model.", datamodel.DataModelStatusActive, "employee_directory.csv", "directory", []string{"directory"}},
		{"raw_event_stream", "Raw Event Stream", "Landing-zone raw events for downstream processing.", datamodel.DataModelStatusActive, "data-lake-raw", "raw_events", []string{"lake", "raw"}},
		{"device_inventory_model", "Device Inventory", "Asset inventory extracted from the data lake.", datamodel.DataModelStatusActive, "data-lake-raw", "device_inventory", []string{"inventory"}},
		{"customer_analytics", "Customer Analytics", "Curated customer 360 model for BI and segmentation.", datamodel.DataModelStatusActive, "analytics_warehouse", "customer_360", []string{"analytics", "customer"}},
	}

	sourceByName := make(map[string]sourceSeed, len(sources))
	for _, source := range sources {
		sourceByName[source.Name] = source
	}

	models := make([]modelSeed, 0, len(specs))
	for _, spec := range specs {
		source := sourceByName[spec.source]
		table := findTable(source.Schema, spec.table)
		fields := deriveFields(table)
		rules := deriveQualityRules(fields)
		classification, piiColumns, containsPII := deriveClassification(fields)
		models = append(models, modelSeed{
			ID:                 uuid.New(),
			Name:               spec.name,
			DisplayName:        spec.displayName,
			Description:        spec.description,
			Status:             spec.status,
			SourceName:         spec.source,
			SourceTable:        spec.table,
			SchemaDefinition:   fields,
			QualityRules:       rules,
			DataClassification: classification,
			ContainsPII:        containsPII,
			PIIColumns:         piiColumns,
			Tags:               spec.tags,
			Metadata:           json.RawMessage(`{"seeded":true}`),
			CreatedAt:          now,
			UpdatedAt:          now,
		})
	}
	return models
}

func newSourceSeed(name, description string, sourceType datamodel.DataSourceType, config any, tables []datamodel.DiscoveredTable, totalBytes, totalRows int64, syncFrequency *string, lastSyncedAt *time.Time, lastSyncStatus *string, lastSyncError *string, lastSyncMs *int64, tags []string) sourceSeed {
	schema := datamodel.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  countColumns(tables),
		ContainsPII:  containsPIITable(tables),
		HighestClass: highestClassification(tables),
	}
	now := time.Now().UTC().Truncate(time.Second)
	return sourceSeed{
		ID:             uuid.New(),
		Name:           name,
		Description:    description,
		Type:           sourceType,
		Config:         config,
		Status:         datamodel.DataSourceStatusActive,
		Schema:         schema,
		TableCount:     len(tables),
		TotalRowCount:  totalRows,
		TotalSizeBytes: totalBytes,
		SyncFrequency:  syncFrequency,
		LastSyncedAt:   lastSyncedAt,
		LastSyncStatus: lastSyncStatus,
		LastSyncError:  lastSyncError,
		LastSyncMs:     lastSyncMs,
		Tags:           tags,
		Metadata:       json.RawMessage(`{"seeded":true}`),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func makeTable(schema, name, tableType string, classification datamodel.DataClassification, estimatedRows, sizeBytes int64, columns ...datamodel.DiscoveredColumn) datamodel.DiscoveredTable {
	piiCount := 0
	for _, column := range columns {
		if column.InferredPII {
			piiCount++
		}
	}
	return datamodel.DiscoveredTable{
		SchemaName:      schema,
		Name:            name,
		Type:            tableType,
		Columns:         columns,
		EstimatedRows:   estimatedRows,
		SizeBytes:       sizeBytes,
		InferredClass:   classification,
		ContainsPII:     piiCount > 0,
		PIIColumnCount:  piiCount,
		SampledRowCount: 10,
	}
}

func column(name, dataType, nativeType string, nullable, primary bool, piiType string, classification datamodel.DataClassification, fk *datamodel.ForeignKeyRef, samples []string) datamodel.DiscoveredColumn {
	col := datamodel.DiscoveredColumn{
		Name:         name,
		DataType:     dataType,
		NativeType:   nativeType,
		MappedType:   mappedType(dataType),
		Nullable:     nullable,
		IsPrimaryKey: primary,
		IsForeignKey: fk != nil,
		ForeignKeyRef: fk,
		InferredPII:  piiType != "",
		InferredPIIType: piiType,
		InferredClass: classification,
		SampleValues: samples,
	}
	return col
}

func mappedType(value string) string {
	switch strings.ToLower(value) {
	case "uuid", "varchar", "text", "string":
		return "string"
	case "bigint":
		return "integer"
	case "numeric", "float":
		return "float"
	case "date", "timestamptz", "datetime":
		return "datetime"
	case "json":
		return "json"
	default:
		return "string"
	}
}

func deriveFields(table datamodel.DiscoveredTable) []datamodel.ModelField {
	fields := make([]datamodel.ModelField, 0, len(table.Columns))
	for _, column := range table.Columns {
		fields = append(fields, datamodel.ModelField{
			Name:            column.Name,
			DisplayName:     displayName(column.Name),
			DataType:        column.MappedType,
			NativeType:      column.NativeType,
			Nullable:        column.Nullable,
			IsPrimaryKey:    column.IsPrimaryKey,
			IsForeignKey:    column.IsForeignKey,
			ForeignKeyRef:   column.ForeignKeyRef,
			Description:     fmt.Sprintf("Derived from %s.%s", table.Name, column.Name),
			PIIType:         column.InferredPIIType,
			Classification:  column.InferredClass,
			SampleValues:    append([]string(nil), column.SampleValues...),
			ValidationRules: []datamodel.ValidationRule{},
		})
	}
	return fields
}

func deriveQualityRules(fields []datamodel.ModelField) []datamodel.ValidationRule {
	rules := make([]datamodel.ValidationRule, 0)
	for _, field := range fields {
		if !field.Nullable {
			rules = append(rules, datamodel.ValidationRule{Type: "not_null", Field: field.Name})
		}
		if field.IsPrimaryKey {
			rules = append(rules, datamodel.ValidationRule{Type: "unique", Field: field.Name})
		}
		if field.DataType == "string" && len(field.SampleValues) > 0 {
			maxLen := 0
			enumSet := make(map[string]struct{})
			for _, sample := range field.SampleValues {
				if len(sample) > maxLen {
					maxLen = len(sample)
				}
				enumSet[sample] = struct{}{}
			}
			rules = append(rules, datamodel.ValidationRule{Type: "max_length", Field: field.Name, Params: map[string]any{"max": maxLen}})
			if len(enumSet) > 0 && len(enumSet) < 20 {
				values := make([]string, 0, len(enumSet))
				for value := range enumSet {
					values = append(values, value)
				}
				rules = append(rules, datamodel.ValidationRule{Type: "enum", Field: field.Name, Params: map[string]any{"values": values}})
			}
		}
		if field.PIIType == "email" {
			rules = append(rules, datamodel.ValidationRule{Type: "format", Field: field.Name, Params: map[string]any{"pattern": "email"}})
		}
		if field.DataType == "datetime" {
			rules = append(rules, datamodel.ValidationRule{Type: "not_future", Field: field.Name})
		}
	}
	return rules
}

func deriveClassification(fields []datamodel.ModelField) (datamodel.DataClassification, []string, bool) {
	classification := datamodel.DataClassificationPublic
	piiColumns := make([]string, 0)
	for _, field := range fields {
		switch field.Classification {
		case datamodel.DataClassificationRestricted:
			classification = datamodel.DataClassificationRestricted
		case datamodel.DataClassificationConfidential:
			if classification != datamodel.DataClassificationRestricted {
				classification = datamodel.DataClassificationConfidential
			}
		case datamodel.DataClassificationInternal:
			if classification == datamodel.DataClassificationPublic {
				classification = datamodel.DataClassificationInternal
			}
		}
		if field.PIIType != "" {
			piiColumns = append(piiColumns, field.Name)
		}
	}
	return classification, piiColumns, len(piiColumns) > 0
}

func findTable(schema datamodel.DiscoveredSchema, tableName string) datamodel.DiscoveredTable {
	for _, table := range schema.Tables {
		if strings.EqualFold(table.Name, tableName) {
			return table
		}
	}
	return datamodel.DiscoveredTable{Name: tableName}
}

func displayName(value string) string {
	parts := strings.Fields(strings.ReplaceAll(value, "_", " "))
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func countColumns(tables []datamodel.DiscoveredTable) int {
	total := 0
	for _, table := range tables {
		total += len(table.Columns)
	}
	return total
}

func containsPIITable(tables []datamodel.DiscoveredTable) bool {
	for _, table := range tables {
		if table.ContainsPII {
			return true
		}
	}
	return false
}

func highestClassification(tables []datamodel.DiscoveredTable) datamodel.DataClassification {
	highest := datamodel.DataClassificationPublic
	for _, table := range tables {
		switch table.InferredClass {
		case datamodel.DataClassificationRestricted:
			return table.InferredClass
		case datamodel.DataClassificationConfidential:
			if highest != datamodel.DataClassificationRestricted {
				highest = table.InferredClass
			}
		case datamodel.DataClassificationInternal:
			if highest == datamodel.DataClassificationPublic {
				highest = table.InferredClass
			}
		}
	}
	return highest
}

func nextSyncTime(schedule *string, base time.Time) *time.Time {
	if schedule == nil || strings.TrimSpace(*schedule) == "" {
		return nil
	}
	next := base.Add(24 * time.Hour)
	return &next
}

func countErrors(message *string) int {
	if message == nil || strings.TrimSpace(*message) == "" {
		return 0
	}
	return 1
}

func coalesceJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}
