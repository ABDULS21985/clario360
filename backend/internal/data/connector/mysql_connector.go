package connector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

type MySQLConnector struct {
	config model.MySQLConnectionConfig
	db     *sql.DB
	logger zerolog.Logger
	limits ConnectorLimits
}

func NewMySQLConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.MySQLConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("decode mysql config: %w", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 3306
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("validate mysql config: %w", err)
	}
	return &MySQLConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", "mysql").Logger(),
		limits: options.Limits,
	}, nil
}

func (c *MySQLConnector) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	db, err := c.openDB()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, c.limits.ConnectTimeout)
	defer cancel()

	start := time.Now()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql source: %w", err)
	}

	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		return nil, fmt.Errorf("query mysql version: %w", err)
	}

	warnings := make([]string, 0, 2)
	if strings.EqualFold(c.config.TLSMode, "false") && !isLocalhost(c.config.Host) {
		warnings = append(warnings, "Connection is not encrypted.")
	}
	if time.Since(start) > 5*time.Second {
		warnings = append(warnings, "Slow connection.")
	}

	grantsRows, err := db.QueryContext(ctx, "SHOW GRANTS FOR CURRENT_USER()")
	if err != nil {
		return nil, fmt.Errorf("show grants: %w", err)
	}
	defer grantsRows.Close()
	permissions := make([]string, 0)
	for grantsRows.Next() {
		var grant string
		if err := grantsRows.Scan(&grant); err == nil {
			permissions = append(permissions, grant)
		}
	}

	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     version,
		Message:     "MySQL connection established successfully.",
		Permissions: permissions,
		Warnings:    warnings,
	}, nil
}

func (c *MySQLConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error) {
	db, err := c.openDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT TABLE_NAME, TABLE_TYPE, TABLE_COMMENT, COALESCE(TABLE_ROWS, 0), COALESCE(DATA_LENGTH, 0)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?
		  AND (TABLE_TYPE = 'BASE TABLE' OR (? AND TABLE_TYPE = 'VIEW'))
		ORDER BY TABLE_NAME
		LIMIT ?`

	rows, err := db.QueryContext(ctx, query, c.config.Database, opts.IncludeViews, opts.MaxTables)
	if err != nil {
		return nil, fmt.Errorf("list mysql tables: %w", err)
	}
	defer rows.Close()

	tables := make([]model.DiscoveredTable, 0)
	totalColumns := 0
	highest := model.DataClassificationPublic
	containsPII := false

	for rows.Next() {
		var tableName, tableType, comment string
		var tableRows, dataLength int64
		if err := rows.Scan(&tableName, &tableType, &comment, &tableRows, &dataLength); err != nil {
			return nil, fmt.Errorf("scan mysql table: %w", err)
		}
		table, err := c.discoverTable(ctx, db, tableName, tableType, comment, tableRows, dataLength, opts)
		if err != nil {
			return nil, err
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		highest = discovery.MaxClassification(highest, table.InferredClass)
		containsPII = containsPII || table.ContainsPII
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mysql tables: %w", err)
	}

	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *MySQLConnector) FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error) {
	db, err := c.openDB()
	if err != nil {
		return nil, err
	}

	columns := "*"
	if len(params.Columns) > 0 {
		quoted := make([]string, 0, len(params.Columns))
		for _, column := range params.Columns {
			quoted = append(quoted, backtickQuote(column))
		}
		columns = strings.Join(quoted, ", ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", columns, backtickQuote(table))
	args := make([]any, 0, len(params.Filters)+2)
	conditions := make([]string, 0, len(params.Filters))
	for column, value := range params.Filters {
		conditions = append(conditions, fmt.Sprintf("%s = ?", backtickQuote(column)))
		args = append(args, value)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if params.OrderBy != "" {
		query += " ORDER BY " + backtickQuote(params.OrderBy)
	}
	if params.BatchSize <= 0 {
		params.BatchSize = 100
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.BatchSize, params.Offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fetch mysql data: %w", err)
	}
	defer rows.Close()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("mysql rows columns: %w", err)
	}
	batchRows := make([]map[string]any, 0, params.BatchSize)
	for rows.Next() {
		values := make([]any, len(columnNames))
		scanTargets := make([]any, len(columnNames))
		for i := range values {
			scanTargets[i] = &values[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, fmt.Errorf("scan mysql row: %w", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = values[i]
		}
		batchRows = append(batchRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mysql rows: %w", err)
	}

	return &DataBatch{
		Columns:  columnNames,
		Rows:     batchRows,
		RowCount: len(batchRows),
		HasMore:  len(batchRows) == params.BatchSize,
	}, nil
}

func (c *MySQLConnector) EstimateSize(ctx context.Context) (*SizeEstimate, error) {
	db, err := c.openDB()
	if err != nil {
		return nil, err
	}

	query := `
		SELECT COUNT(*), COALESCE(SUM(TABLE_ROWS), 0), COALESCE(SUM(DATA_LENGTH), 0)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ?`

	var estimate SizeEstimate
	if err := db.QueryRowContext(ctx, query, c.config.Database).Scan(&estimate.TableCount, &estimate.TotalRows, &estimate.TotalBytes); err != nil {
		return nil, fmt.Errorf("estimate mysql size: %w", err)
	}
	return &estimate, nil
}

func (c *MySQLConnector) Close() error {
	if c.db != nil {
		err := c.db.Close()
		c.db = nil
		return err
	}
	return nil
}

func (c *MySQLConnector) openDB() (*sql.DB, error) {
	if c.db != nil {
		return c.db, nil
	}

	cfg := mysqlDriver.NewConfig()
	cfg.User = c.config.Username
	cfg.Passwd = c.config.Password
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	cfg.DBName = c.config.Database
	if c.config.TLSMode != "" {
		cfg.TLSConfig = c.config.TLSMode
	}
	cfg.Timeout = c.limits.ConnectTimeout
	cfg.ReadTimeout = c.limits.StatementTimeout
	cfg.WriteTimeout = c.limits.StatementTimeout
	cfg.ParseTime = true

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql source: %w", err)
	}
	db.SetMaxOpenConns(c.limits.MaxPoolSize)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)
	c.db = db
	return db, nil
}

func (c *MySQLConnector) discoverTable(ctx context.Context, db *sql.DB, tableName, tableType, comment string, tableRows, dataLength int64, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	query := `
		SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, CHARACTER_MAXIMUM_LENGTH, IS_NULLABLE,
		       COLUMN_DEFAULT, COLUMN_COMMENT, COLUMN_KEY
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	rows, err := db.QueryContext(ctx, query, c.config.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("discover mysql columns for %s: %w", tableName, err)
	}
	defer rows.Close()

	columns := make([]model.DiscoveredColumn, 0)
	primaryKeys := make([]string, 0)
	for rows.Next() {
		var columnName, dataType, columnType, nullable, columnComment, columnKey string
		var maxLength *int
		var defaultValue *string
		if err := rows.Scan(&columnName, &dataType, &columnType, &maxLength, &nullable, &defaultValue, &columnComment, &columnKey); err != nil {
			return nil, fmt.Errorf("scan mysql column for %s: %w", tableName, err)
		}
		mapped := discovery.MapNativeType(columnType)
		column := model.DiscoveredColumn{
			Name:         columnName,
			DataType:     dataType,
			NativeType:   columnType,
			MappedType:   mapped.Type,
			Subtype:      mapped.Subtype,
			MaxLength:    maxLength,
			Nullable:     nullable == "YES",
			DefaultValue: defaultValue,
			Comment:      columnComment,
			IsPrimaryKey: columnKey == "PRI",
		}
		if columnKey == "PRI" {
			primaryKeys = append(primaryKeys, columnName)
		}
		if opts.SampleValues {
			samples, err := c.sampleColumnValues(ctx, db, tableName, columnName, opts.MaxSamples)
			if err == nil {
				column.SampleValues = samples
				column.SampleStats = discovery.AnalyzeSamples(samples)
				if inferred := discovery.InferSampleType(samples); column.MappedType == "string" && inferred != "string" {
					column.MappedType = inferred
				}
			}
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mysql columns for %s: %w", tableName, err)
	}

	foreignKeys, err := c.loadForeignKeys(ctx, db, tableName)
	if err != nil {
		return nil, err
	}
	fkSet := make(map[string]model.ForeignKeyRef, len(foreignKeys))
	for _, fk := range foreignKeys {
		fkSet[fk.Column] = fk.ReferencedRef
	}

	columns = discovery.DetectPII(columns)
	nullableCount := 0
	piiCount := 0
	for i := range columns {
		if columns[i].Nullable {
			nullableCount++
		}
		if ref, ok := fkSet[columns[i].Name]; ok {
			columns[i].IsForeignKey = true
			columns[i].ForeignKeyRef = &ref
		}
		if columns[i].InferredPII {
			piiCount++
		}
	}

	return &model.DiscoveredTable{
		Name:            tableName,
		Type:            strings.ToLower(tableType),
		Comment:         comment,
		Columns:         columns,
		PrimaryKeys:     primaryKeys,
		ForeignKeys:     foreignKeys,
		EstimatedRows:   tableRows,
		SizeBytes:       dataLength,
		InferredClass:   discovery.TableClassification(columns),
		ContainsPII:     piiCount > 0,
		PIIColumnCount:  piiCount,
		NullableCount:   nullableCount,
		SampledRowCount: opts.MaxSamples,
	}, nil
}

func (c *MySQLConnector) loadForeignKeys(ctx context.Context, db *sql.DB, tableName string) ([]model.ForeignKey, error) {
	query := `
		SELECT COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
		FROM information_schema.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND REFERENCED_TABLE_NAME IS NOT NULL`

	rows, err := db.QueryContext(ctx, query, c.config.Database, tableName)
	if err != nil {
		return nil, fmt.Errorf("load mysql foreign keys for %s: %w", tableName, err)
	}
	defer rows.Close()

	values := make([]model.ForeignKey, 0)
	for rows.Next() {
		var column, refTable, refColumn string
		if err := rows.Scan(&column, &refTable, &refColumn); err != nil {
			return nil, fmt.Errorf("scan mysql foreign key for %s: %w", tableName, err)
		}
		values = append(values, model.ForeignKey{
			Column: column,
			ReferencedRef: model.ForeignKeyRef{
				Table:  refTable,
				Column: refColumn,
			},
		})
	}
	return values, rows.Err()
}

func (c *MySQLConnector) sampleColumnValues(ctx context.Context, db *sql.DB, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	query := fmt.Sprintf(
		"SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL LIMIT %d",
		backtickQuote(columnName),
		backtickQuote(tableName),
		backtickQuote(columnName),
		maxSamples,
	)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sample mysql column %s.%s: %w", tableName, columnName, err)
	}
	defer rows.Close()

	values := make([]string, 0, maxSamples)
	for rows.Next() {
		var value any
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan mysql sample %s.%s: %w", tableName, columnName, err)
		}
		values = append(values, fmt.Sprint(value))
	}
	return values, rows.Err()
}

func backtickQuote(identifier string) string {
	escaped := strings.ReplaceAll(identifier, "`", "``")
	return "`" + escaped + "`"
}
