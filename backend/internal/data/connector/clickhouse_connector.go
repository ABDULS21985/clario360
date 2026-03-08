package connector

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

const clickHouseConnectorType = "clickhouse"

type ClickHouseConnector struct {
	config   model.ClickHouseConnectionConfig
	db       *sql.DB
	sourceID uuid.UUID
	tenantID uuid.UUID
	logger   zerolog.Logger
	limits   ConnectorLimits
}

func NewClickHouseConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.ClickHouseConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "connect", ErrorCodeConfigurationError, "decode clickhouse config", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 9000
	}
	if cfg.Protocol == "" {
		cfg.Protocol = "native"
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = options.Limits.MaxPoolSize
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = max(1, cfg.MaxOpenConns/2)
	}
	if cfg.DialTimeoutSeconds == 0 {
		cfg.DialTimeoutSeconds = int(options.Limits.ConnectTimeout.Seconds())
		if cfg.DialTimeoutSeconds <= 0 {
			cfg.DialTimeoutSeconds = 10
		}
	}
	if cfg.ReadTimeoutSeconds == 0 {
		cfg.ReadTimeoutSeconds = int(options.Limits.StatementTimeout.Seconds())
		if cfg.ReadTimeoutSeconds <= 0 {
			cfg.ReadTimeoutSeconds = 30
		}
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "connect", ErrorCodeConfigurationError, "validate clickhouse config", err)
	}
	connector := &ClickHouseConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", clickHouseConnectorType).Logger(),
		limits: options.Limits,
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(clickHouseConnectorType).Inc()
	return connector, nil
}

func (c *ClickHouseConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *ClickHouseConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "test", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	var version, user string
	if err = db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "test", ErrorCodeQueryFailed, "query clickhouse version", err)
	}
	if err = db.QueryRowContext(ctx, "SELECT currentUser()").Scan(&user); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "test", ErrorCodeQueryFailed, "query clickhouse current user", err)
	}
	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "test", ErrorCodeQueryFailed, "list clickhouse databases", err)
	}
	defer rows.Close()
	databases := make([]string, 0)
	for rows.Next() {
		var name string
		if err = rows.Scan(&name); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "test", ErrorCodeDriverError, "scan clickhouse database", err)
		}
		databases = append(databases, name)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "test", ErrorCodeDriverError, "iterate clickhouse databases", err)
	}
	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     "ClickHouse " + version,
		Message:     fmt.Sprintf("Connected as %s. %d databases accessible.", user, len(databases)),
		Permissions: databases,
	}, nil
}

func (c *ClickHouseConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "discover", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT database, name, engine, total_rows, total_bytes, metadata_modification_time
		FROM system.tables
		WHERE database = ?
		  AND engine NOT IN ('MaterializedView', 'View')
		  AND name NOT LIKE '.inner.%'
		ORDER BY name
		LIMIT ?`, c.config.Database, opts.MaxTables)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "list clickhouse tables", err)
	}
	defer rows.Close()

	tables := make([]model.DiscoveredTable, 0)
	totalColumns := 0
	containsPII := false
	highest := model.DataClassificationPublic
	for rows.Next() {
		var databaseName, tableName, engine string
		var totalRows, totalBytes uint64
		var modifiedAt time.Time
		if err = rows.Scan(&databaseName, &tableName, &engine, &totalRows, &totalBytes, &modifiedAt); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "discover", ErrorCodeDriverError, "scan clickhouse table", err)
		}
		table, tableErr := c.discoverTable(ctx, db, databaseName, tableName, engine, totalRows, totalBytes, modifiedAt, opts)
		if tableErr != nil {
			return nil, tableErr
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highest = discovery.MaxClassification(highest, table.InferredClass)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "discover", ErrorCodeDriverError, "iterate clickhouse tables", err)
	}
	observeSchemaMetrics(clickHouseConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *ClickHouseConnector) FetchData(ctx context.Context, table string, params FetchParams) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "fetch", start, err) }()

	db, err := c.openDB(ctx)
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
	tableIdent := quoteDotBacktickIdentifier(table)
	if !strings.Contains(table, ".") {
		tableIdent = quoteDotBacktickIdentifier(c.config.Database + "." + table)
	}
	query := fmt.Sprintf("SELECT %s FROM %s", columns, tableIdent)
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
		params.BatchSize = 10000
	}
	query += " LIMIT ? OFFSET ?"
	args = append(args, params.BatchSize, params.Offset)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "fetch", ErrorCodeQueryFailed, "fetch clickhouse rows", err)
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "fetch", ErrorCodeDriverError, "read clickhouse columns", err)
	}
	resultRows := make([]map[string]any, 0, params.BatchSize)
	for rows.Next() {
		values := make([]any, len(columnNames))
		targets := make([]any, len(columnNames))
		for i := range values {
			targets[i] = &values[i]
		}
		if err = rows.Scan(targets...); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "fetch", ErrorCodeDriverError, "scan clickhouse row", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = normalizeSQLValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "fetch", ErrorCodeDriverError, "iterate clickhouse rows", err)
	}
	observeFetchMetrics(clickHouseConnectorType, len(resultRows), int64(len(mustJSON(resultRows))))
	return &DataBatch{
		Columns:  columnNames,
		Rows:     resultRows,
		RowCount: len(resultRows),
		HasMore:  len(resultRows) == params.BatchSize,
	}, nil
}

func (c *ClickHouseConnector) ReadQuery(ctx context.Context, query string, args []any) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "read_query", start, err) }()
	if !isReadOnlyQuery(query) {
		return nil, newConnectorError(clickHouseConnectorType, "read_query", ErrorCodeUnsupportedOperation, "only read-only queries are allowed", ErrCapabilityUnsupported)
	}
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "read_query", ErrorCodeQueryFailed, "execute clickhouse query", err)
	}
	defer rows.Close()
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "read_query", ErrorCodeDriverError, "read clickhouse query columns", err)
	}
	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columnNames))
		targets := make([]any, len(columnNames))
		for i := range values {
			targets[i] = &values[i]
		}
		if err = rows.Scan(targets...); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "read_query", ErrorCodeDriverError, "scan clickhouse query row", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = normalizeSQLValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	return &DataBatch{Columns: columnNames, Rows: resultRows, RowCount: len(resultRows)}, rows.Err()
}

func (c *ClickHouseConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (_ *WriteResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "write", start, err) }()
	if params.Strategy == "merge" {
		return nil, newConnectorError(clickHouseConnectorType, "write", ErrorCodeUnsupportedOperation, "ClickHouse connector does not support merge writes", ErrCapabilityUnsupported)
	}
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &WriteResult{}, nil
	}
	if params.Replace {
		target := quoteDotBacktickIdentifier(table)
		if !strings.Contains(table, ".") {
			target = quoteDotBacktickIdentifier(c.config.Database + "." + table)
		}
		if _, err = db.ExecContext(ctx, "TRUNCATE TABLE "+target); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "write", ErrorCodeQueryFailed, "truncate clickhouse table", err)
		}
	}
	columns := writeColumns(rows)
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, backtickQuote(column))
	}
	valueGroups := make([]string, 0, len(rows))
	args := make([]any, 0, len(rows)*len(columns))
	for _, row := range rows {
		placeholders := make([]string, 0, len(columns))
		for range columns {
			placeholders = append(placeholders, "?")
		}
		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ", ")+")")
		args = append(args, rowValues(row, columns)...)
	}
	target := quoteDotBacktickIdentifier(table)
	if !strings.Contains(table, ".") {
		target = quoteDotBacktickIdentifier(c.config.Database + "." + table)
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", target, strings.Join(quotedColumns, ", "), strings.Join(valueGroups, ", "))
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "write", ErrorCodeQueryFailed, "insert clickhouse rows", err)
	}
	affected, _ := result.RowsAffected()
	return &WriteResult{RowsWritten: affected, BytesWritten: int64(len(mustJSON(rows)))}, nil
}

func (c *ClickHouseConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "estimate", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT name, total_rows, total_bytes
		FROM system.tables
		WHERE database = ?
		  AND engine NOT IN ('MaterializedView', 'View')`, c.config.Database)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "estimate", ErrorCodeQueryFailed, "estimate clickhouse size", err)
	}
	defer rows.Close()
	estimate := &SizeEstimate{}
	for rows.Next() {
		var name string
		var totalRows, totalBytes uint64
		if err = rows.Scan(&name, &totalRows, &totalBytes); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "estimate", ErrorCodeDriverError, "scan clickhouse estimate row", err)
		}
		estimate.TableCount++
		estimate.TotalRows += int64(totalRows)
		estimate.TotalBytes += int64(totalBytes)
	}
	return estimate, rows.Err()
}

func (c *ClickHouseConnector) QueryAccessLogs(ctx context.Context, since time.Time) (_ []DataAccessEvent, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "access_logs", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT
			user,
			client_hostname,
			query_kind,
			query,
			event_time,
			query_duration_ms,
			read_rows,
			read_bytes,
			written_rows,
			written_bytes,
			exception_code,
			exception
		FROM system.query_log
		WHERE event_time > ?
		  AND type IN ('QueryFinish', 'ExceptionBeforeStart', 'ExceptionWhileProcessing')
		  AND query_kind != 'System'
		ORDER BY event_time DESC
		LIMIT 10000`, since)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "access_logs", ErrorCodeQueryFailed, "query clickhouse access logs", err)
	}
	defer rows.Close()
	events := make([]DataAccessEvent, 0)
	for rows.Next() {
		var user, host, kind, queryText, exception string
		var eventTime time.Time
		var durationMs int64
		var readRows, readBytes, writtenRows, writtenBytes uint64
		var exceptionCode int32
		if err = rows.Scan(&user, &host, &kind, &queryText, &eventTime, &durationMs, &readRows, &readBytes, &writtenRows, &writtenBytes, &exceptionCode, &exception); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "access_logs", ErrorCodeDriverError, "scan clickhouse access log row", err)
		}
		action := "query"
		switch strings.ToLower(kind) {
		case "insert":
			action = "insert"
		case "create":
			action = "create"
		case "drop":
			action = "drop"
		case "alter":
			action = "alter"
		case "delete":
			action = "delete"
		case "update":
			action = "update"
		}
		events = append(events, DataAccessEvent{
			Timestamp:    eventTime.UTC(),
			User:         user,
			SourceIP:     host,
			Action:       action,
			Database:     c.config.Database,
			Table:        extractTableFromQuery(queryText),
			QueryHash:    sha256Hex(queryText),
			QueryPreview: truncateString(queryText, 500),
			RowsRead:     int64(readRows),
			RowsWritten:  int64(writtenRows),
			BytesRead:    int64(readBytes),
			BytesWritten: int64(writtenBytes),
			DurationMs:   durationMs,
			Success:      exceptionCode == 0,
			ErrorMsg:     truncateString(exception, 200),
			SourceType:   clickHouseConnectorType,
			SourceID:     c.sourceID,
			TenantID:     c.tenantID,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "access_logs", ErrorCodeDriverError, "iterate clickhouse access logs", err)
	}
	getConnectorMetrics().AccessEventsTotal.WithLabelValues(clickHouseConnectorType).Add(float64(len(events)))
	return events, nil
}

func (c *ClickHouseConnector) ListDataLocations(ctx context.Context) (_ []DataLocation, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(clickHouseConnectorType, "locations", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT name, total_bytes, metadata_modification_time, engine
		FROM system.tables
		WHERE database = ?`, c.config.Database)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "locations", ErrorCodeQueryFailed, "query clickhouse locations", err)
	}
	defer rows.Close()
	locations := make([]DataLocation, 0)
	for rows.Next() {
		var tableName, engine string
		var totalBytes uint64
		var modifiedAt time.Time
		if err = rows.Scan(&tableName, &totalBytes, &modifiedAt, &engine); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "locations", ErrorCodeDriverError, "scan clickhouse location row", err)
		}
		locations = append(locations, DataLocation{
			SourceID:     c.sourceID,
			SourceType:   clickHouseConnectorType,
			Table:        tableName,
			Database:     c.config.Database,
			Location:     fmt.Sprintf("clickhouse://%s:%d/%s/%s", c.config.Host, c.config.Port, c.config.Database, tableName),
			Format:       "managed",
			SizeBytes:    int64(totalBytes),
			LastModified: modifiedAt.UTC(),
			Partitioned:  strings.Contains(strings.ToLower(engine), "merge"),
		})
	}
	return locations, rows.Err()
}

func (c *ClickHouseConnector) Close() error {
	if c.db != nil {
		err := c.db.Close()
		c.db = nil
		getConnectorMetrics().ActiveConnections.WithLabelValues(clickHouseConnectorType).Dec()
		return err
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(clickHouseConnectorType).Dec()
	return nil
}

func (c *ClickHouseConnector) openDB(ctx context.Context) (*sql.DB, error) {
	if c.db != nil {
		return c.db, nil
	}
	protocol := clickhouse.Native
	if strings.EqualFold(c.config.Protocol, "http") {
		protocol = clickhouse.HTTP
	}
	opts := &clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)},
		Auth: clickhouse.Auth{
			Database: c.config.Database,
			Username: c.config.Username,
			Password: c.config.Password,
		},
		Protocol:    protocol,
		DialTimeout: time.Duration(c.config.DialTimeoutSeconds) * time.Second,
		ReadTimeout: time.Duration(c.config.ReadTimeoutSeconds) * time.Second,
		MaxOpenConns: c.config.MaxOpenConns,
		MaxIdleConns: c.config.MaxIdleConns,
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
	}
	if c.config.Secure {
		opts.TLS = &tls.Config{ServerName: c.config.Host, MinVersion: tls.VersionTLS12}
	}
	if c.config.Compression {
		opts.Compression = &clickhouse.Compression{Method: clickhouse.CompressionLZ4}
	}
	db := clickhouse.OpenDB(opts)
	lifetime := time.Duration(c.config.ConnMaxLifetimeMins) * time.Minute
	if lifetime <= 0 {
		lifetime = time.Hour
	}
	idleLifetime := time.Duration(c.config.ConnMaxIdleTimeMins) * time.Minute
	if idleLifetime <= 0 {
		idleLifetime = 10 * time.Minute
	}
	db.SetConnMaxLifetime(lifetime)
	db.SetConnMaxIdleTime(idleLifetime)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, newConnectorError(clickHouseConnectorType, "connect", ErrorCodeConnectionFailed, "ping clickhouse source", err)
	}
	c.db = db
	return db, nil
}

func (c *ClickHouseConnector) discoverTable(ctx context.Context, db *sql.DB, databaseName, tableName, engine string, totalRows, totalBytes uint64, modifiedAt time.Time, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT name, type, default_kind, default_expression, comment, is_in_primary_key, is_in_partition_key
		FROM system.columns
		WHERE database = ? AND table = ?
		ORDER BY position`, databaseName, tableName)
	if err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "query clickhouse columns", err)
	}
	defer rows.Close()
	columns := make([]model.DiscoveredColumn, 0)
	primaryKeys := make([]string, 0)
	nullableCount := 0
	piiCount := 0
	for rows.Next() {
		var columnName, nativeType, defaultKind, defaultExpr, comment string
		var inPrimaryKey, inPartitionKey uint8
		if err = rows.Scan(&columnName, &nativeType, &defaultKind, &defaultExpr, &comment, &inPrimaryKey, &inPartitionKey); err != nil {
			return nil, newConnectorError(clickHouseConnectorType, "discover", ErrorCodeDriverError, "scan clickhouse column", err)
		}
		mappedType, subtype, nullable := clickHouseTypeMapping(nativeType)
		column := model.DiscoveredColumn{
			Name:         columnName,
			DataType:     nativeType,
			NativeType:   nativeType,
			MappedType:   mappedType,
			Subtype:      subtype,
			Nullable:     nullable,
			Comment:      comment,
			IsPrimaryKey: inPrimaryKey == 1,
		}
		if defaultKind != "" && defaultExpr != "" {
			value := defaultExpr
			column.DefaultValue = &value
		}
		if opts.SampleValues {
			samples, sampleErr := c.sampleColumnValues(ctx, db, tableName, columnName, opts.MaxSamples)
			if sampleErr == nil {
				column.SampleValues = samples
				column.SampleStats = discovery.AnalyzeSamples(samples)
			}
		}
		if inPrimaryKey == 1 {
			primaryKeys = append(primaryKeys, columnName)
		}
		columns = append(columns, column)
	}
	if err = rows.Err(); err != nil {
		return nil, newConnectorError(clickHouseConnectorType, "discover", ErrorCodeDriverError, "iterate clickhouse columns", err)
	}
	columns = discovery.DetectPII(columns)
	for _, column := range columns {
		if column.Nullable {
			nullableCount++
		}
		if column.InferredPII {
			piiCount++
		}
	}
	comment := fmt.Sprintf("engine=%s modified=%s", engine, modifiedAt.UTC().Format(time.RFC3339))
	return &model.DiscoveredTable{
		SchemaName:     databaseName,
		Name:           tableName,
		Type:           strings.ToLower(engine),
		Comment:        comment,
		Columns:        columns,
		PrimaryKeys:    primaryKeys,
		EstimatedRows:  int64(totalRows),
		SizeBytes:      int64(totalBytes),
		InferredClass:  discovery.TableClassification(columns),
		ContainsPII:    piiCount > 0,
		PIIColumnCount: piiCount,
		NullableCount:  nullableCount,
	}, nil
}

func (c *ClickHouseConnector) sampleColumnValues(ctx context.Context, db *sql.DB, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	query := fmt.Sprintf(
		"SELECT DISTINCT %s FROM %s LIMIT %d",
		backtickQuote(columnName),
		quoteDotBacktickIdentifier(c.config.Database+"."+tableName),
		maxSamples,
	)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	samples := make([]string, 0, maxSamples)
	for rows.Next() {
		var value any
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		samples = append(samples, fmt.Sprint(normalizeSQLValue(value)))
	}
	return samples, rows.Err()
}
