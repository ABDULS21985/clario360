package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"crypto/tls"

	"github.com/beltran/gohive"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

const impalaConnectorType = "impala"

type ImpalaConnector struct {
	config   model.ImpalaConnectionConfig
	conn     *gohive.Connection
	sourceID uuid.UUID
	tenantID uuid.UUID
	logger   zerolog.Logger
	limits   ConnectorLimits
}

func NewImpalaConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.ImpalaConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(impalaConnectorType, "connect", ErrorCodeConfigurationError, "decode impala config", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 21050
	}
	if cfg.AuthType == "" {
		cfg.AuthType = "noauth"
	}
	if cfg.QueryTimeoutSeconds == 0 {
		cfg.QueryTimeoutSeconds = 60
	}
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = minInt(options.Limits.MaxPoolSize, 5)
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = max(1, cfg.MaxOpenConns/2)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(impalaConnectorType, "connect", ErrorCodeConfigurationError, "validate impala config", err)
	}
	connector := &ImpalaConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", impalaConnectorType).Logger(),
		limits: options.Limits,
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(impalaConnectorType).Inc()
	return connector, nil
}

func (c *ImpalaConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *ImpalaConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "test", start, err) }()

	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	_, rows, err := gohiveQueryRows(ctx, db, "SHOW DATABASES", 1000)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "test", ErrorCodeQueryFailed, "show impala databases", err)
	}
	databases := make([]string, 0)
	for _, row := range rows {
		databases = append(databases, firstRowValue(row))
	}
	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     "Impala",
		Message:     fmt.Sprintf("Connected to Impala. %d databases accessible.", len(databases)),
		Permissions: databases,
	}, nil
}

func (c *ImpalaConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "discover", start, err) }()
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	databaseName := defaultString(c.config.Database, "default")
	_, rows, err := gohiveQueryRows(ctx, db, fmt.Sprintf("SHOW TABLES IN %s", backtickQuote(databaseName)), opts.MaxTables)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "show impala tables", err)
	}
	tables := make([]model.DiscoveredTable, 0)
	totalColumns := 0
	containsPII := false
	highest := model.DataClassificationPublic
	for _, row := range rows {
		tableName := firstRowValue(row)
		table, tableErr := c.discoverTable(ctx, db, databaseName, tableName, opts)
		if tableErr != nil {
			return nil, tableErr
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highest = discovery.MaxClassification(highest, table.InferredClass)
		if opts.MaxTables > 0 && len(tables) >= opts.MaxTables {
			break
		}
	}
	observeSchemaMetrics(impalaConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *ImpalaConnector) FetchData(ctx context.Context, table string, params FetchParams) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "fetch", start, err) }()
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
	query := fmt.Sprintf("SELECT %s FROM %s", columns, quoteDotBacktickIdentifier(table))
	if len(params.Filters) > 0 {
		conditions := make([]string, 0, len(params.Filters))
		for column, value := range params.Filters {
			conditions = append(conditions, fmt.Sprintf("%s = %s", backtickQuote(column), sqlLiteral(value)))
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if params.OrderBy != "" {
		query += " ORDER BY " + backtickQuote(params.OrderBy)
	}
	if params.BatchSize <= 0 {
		params.BatchSize = 5000
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", params.BatchSize, params.Offset)
	description, rows, err := gohiveQueryRows(ctx, db, query, params.BatchSize)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "fetch", ErrorCodeQueryFailed, "fetch impala rows", err)
	}
	columnNames := descriptionNames(description)
	observeFetchMetrics(impalaConnectorType, len(rows), int64(len(mustJSON(rows))))
	return &DataBatch{
		Columns:  columnNames,
		Rows:     rows,
		RowCount: len(rows),
		HasMore:  len(rows) == params.BatchSize,
	}, nil
}

func (c *ImpalaConnector) ReadQuery(ctx context.Context, query string, args []any) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "read_query", start, err) }()
	if !isReadOnlyQuery(query) {
		return nil, newConnectorError(impalaConnectorType, "read_query", ErrorCodeUnsupportedOperation, "only read-only queries are allowed", ErrCapabilityUnsupported)
	}
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	description, rows, err := gohiveQueryRows(ctx, db, query, 0)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "read_query", ErrorCodeQueryFailed, "execute impala query", err)
	}
	return &DataBatch{Columns: descriptionNames(description), Rows: rows, RowCount: len(rows)}, nil
}

func (c *ImpalaConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (_ *WriteResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "write", start, err) }()
	if params.Strategy == "merge" {
		return nil, newConnectorError(impalaConnectorType, "write", ErrorCodeUnsupportedOperation, "Impala connector does not support merge writes", ErrCapabilityUnsupported)
	}
	if len(rows) == 0 {
		return &WriteResult{}, nil
	}
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	if params.Replace {
		if err = gohiveExec(ctx, db, "TRUNCATE TABLE "+quoteDotBacktickIdentifier(table)); err != nil {
			return nil, newConnectorError(impalaConnectorType, "write", ErrorCodeQueryFailed, "truncate impala table", err)
		}
	}
	columns := writeColumns(rows)
	valueGroups := make([]string, 0, len(rows))
	for _, row := range rows {
		literals := make([]string, 0, len(columns))
		for _, column := range columns {
			literals = append(literals, sqlLiteral(row[column]))
		}
		valueGroups = append(valueGroups, "("+strings.Join(literals, ", ")+")")
	}
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, backtickQuote(column))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", quoteDotBacktickIdentifier(table), strings.Join(quotedColumns, ", "), strings.Join(valueGroups, ", "))
	if err = gohiveExec(ctx, db, query); err != nil {
		return nil, newConnectorError(impalaConnectorType, "write", ErrorCodeQueryFailed, "insert impala rows", err)
	}
	return &WriteResult{RowsWritten: int64(len(rows)), BytesWritten: int64(len(mustJSON(rows)))}, nil
}

func (c *ImpalaConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "estimate", start, err) }()
	schema, err := c.DiscoverSchema(ctx, DiscoveryOptions{MaxTables: c.limits.MaxTables})
	if err != nil {
		return nil, err
	}
	estimate := &SizeEstimate{TableCount: schema.TableCount}
	for _, table := range schema.Tables {
		estimate.TotalRows += table.EstimatedRows
		estimate.TotalBytes += table.SizeBytes
	}
	return estimate, nil
}

func (c *ImpalaConnector) QueryAccessLogs(ctx context.Context, since time.Time) (_ []DataAccessEvent, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "access_logs", start, err) }()
	if strings.TrimSpace(c.config.AuditLogTable) == "" {
		return []DataAccessEvent{}, nil
	}
	db, err := c.openDB(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf(`
		SELECT user_name, source_ip, action, database_name, table_name, statement,
		       event_time, rows_read, rows_written, bytes_read, bytes_written, duration_ms, success, error_message
		FROM %s
		WHERE event_time > ?
		ORDER BY event_time DESC
		LIMIT 10000`, quoteDotBacktickIdentifier(c.config.AuditLogTable))
	_, rows, err := gohiveQueryRows(ctx, db, strings.ReplaceAll(query, "?", sqlLiteral(since.UTC().Format("2006-01-02 15:04:05"))), 10000)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "access_logs", ErrorCodeQueryFailed, "query impala audit logs", err)
	}
	events := make([]DataAccessEvent, 0)
	for _, row := range rows {
		queryText := fmt.Sprint(row["statement"])
		event := DataAccessEvent{
			User:         fmt.Sprint(row["user_name"]),
			SourceIP:     fmt.Sprint(row["source_ip"]),
			Action:       fmt.Sprint(row["action"]),
			Database:     fmt.Sprint(row["database_name"]),
			Table:        fmt.Sprint(row["table_name"]),
			Timestamp:    parseSparkTime(fmt.Sprint(row["event_time"])),
			RowsRead:     int64Number(row["rows_read"]),
			RowsWritten:  int64Number(row["rows_written"]),
			BytesRead:    int64Number(row["bytes_read"]),
			BytesWritten: int64Number(row["bytes_written"]),
			DurationMs:   int64Number(row["duration_ms"]),
			Success:      strings.EqualFold(fmt.Sprint(row["success"]), "true"),
			ErrorMsg:     fmt.Sprint(row["error_message"]),
		}
		event.QueryHash = sha256Hex(queryText)
		event.QueryPreview = truncateString(queryText, 500)
		event.SourceType = impalaConnectorType
		event.SourceID = c.sourceID
		event.TenantID = c.tenantID
		events = append(events, event)
	}
	getConnectorMetrics().AccessEventsTotal.WithLabelValues(impalaConnectorType).Add(float64(len(events)))
	return events, nil
}

func (c *ImpalaConnector) ListDataLocations(ctx context.Context) (_ []DataLocation, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(impalaConnectorType, "locations", start, err) }()
	schema, err := c.DiscoverSchema(ctx, DiscoveryOptions{MaxTables: c.limits.MaxTables})
	if err != nil {
		return nil, err
	}
	locations := make([]DataLocation, 0, len(schema.Tables))
	for _, table := range schema.Tables {
		locations = append(locations, DataLocation{
			SourceID:     c.sourceID,
			SourceType:   impalaConnectorType,
			Table:        table.Name,
			Database:     table.SchemaName,
			Location:     table.Comment,
			Format:       table.Type,
			SizeBytes:    table.SizeBytes,
			LastModified: time.Now().UTC(),
			Partitioned:  false,
		})
	}
	return locations, nil
}

func (c *ImpalaConnector) Close() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(impalaConnectorType).Dec()
	return nil
}

func (c *ImpalaConnector) openDB(ctx context.Context) (*gohive.Connection, error) {
	if c.conn != nil {
		return c.conn, nil
	}
	cfg := gohive.NewConnectConfiguration()
	cfg.Username = c.config.Username
	cfg.Password = c.config.Password
	cfg.Service = "impala"
	cfg.Database = defaultString(c.config.Database, "default")
	cfg.ConnectTimeout = c.limits.ConnectTimeout
	cfg.SocketTimeout = time.Duration(c.config.QueryTimeoutSeconds) * time.Second
	cfg.HttpTimeout = time.Duration(c.config.QueryTimeoutSeconds) * time.Second
	if c.config.UseTLS {
		cfg.TLSConfig = &tls.Config{ServerName: c.config.Host, MinVersion: tls.VersionTLS12}
	}
	auth := "NOSASL"
	switch c.config.AuthType {
	case "ldap":
		auth = "NONE"
	case "kerberos":
		auth = "KERBEROS"
	}
	conn, err := gohive.Connect(c.config.Host, c.config.Port, auth, cfg)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "connect", ErrorCodeConnectionFailed, "connect to impala", err)
	}
	c.conn = conn
	return conn, nil
}

func (c *ImpalaConnector) discoverTable(ctx context.Context, db *gohive.Connection, databaseName, tableName string, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	_, describeRows, err := gohiveQueryRows(ctx, db, fmt.Sprintf("DESCRIBE %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), max(100, opts.MaxColumns))
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "describe impala table", err)
	}
	columns := make([]model.DiscoveredColumn, 0)
	nullableCount := 0
	for _, row := range describeRows {
		name := strings.TrimSpace(fmt.Sprint(row["col_name"]))
		nativeType := strings.TrimSpace(fmt.Sprint(row["data_type"]))
		comment := strings.TrimSpace(fmt.Sprint(row["comment"]))
		if name == "" || strings.HasPrefix(name, "#") || nativeType == "" {
			continue
		}
		mappedType, subtype := hiveLikeTypeMapping(nativeType)
		column := model.DiscoveredColumn{
			Name:       name,
			DataType:   nativeType,
			NativeType: nativeType,
			MappedType: mappedType,
			Subtype:    subtype,
			Nullable:   true,
			Comment:    comment,
		}
		if opts.SampleValues {
			samples, sampleErr := c.sampleColumnValues(ctx, db, databaseName, tableName, column.Name, opts.MaxSamples)
			if sampleErr == nil {
				column.SampleValues = samples
				column.SampleStats = discovery.AnalyzeSamples(samples)
			}
		}
		columns = append(columns, column)
		nullableCount++
	}
	formattedRows, err := c.describeFormatted(ctx, db, databaseName, tableName)
	if err != nil {
		return nil, err
	}
	meta := parseHiveDescribeFormatted(formattedRows)
	columns = discovery.DetectPII(columns)
	piiCount := 0
	for _, column := range columns {
		if column.InferredPII {
			piiCount++
		}
	}
	return &model.DiscoveredTable{
		SchemaName:     databaseName,
		Name:           tableName,
		Type:           inputFormatToFormat(meta.InputFormat),
		Comment:        meta.Location,
		Columns:        columns,
		EstimatedRows:  meta.NumRows,
		SizeBytes:      meta.RawDataSize,
		InferredClass:  discovery.TableClassification(columns),
		ContainsPII:    piiCount > 0,
		PIIColumnCount: piiCount,
		NullableCount:  nullableCount,
	}, nil
}

func (c *ImpalaConnector) describeFormatted(ctx context.Context, db *gohive.Connection, databaseName, tableName string) ([]map[string]any, error) {
	_, rows, err := gohiveQueryRows(ctx, db, fmt.Sprintf("DESCRIBE FORMATTED %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), 1000)
	if err != nil {
		return nil, newConnectorError(impalaConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "describe formatted impala table", err)
	}
	return rows, nil
}

func (c *ImpalaConnector) sampleColumnValues(ctx context.Context, db *gohive.Connection, databaseName, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	_, rows, err := gohiveQueryRows(ctx, db, fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s IS NOT NULL LIMIT %d",
		backtickQuote(columnName),
		quoteDotBacktickIdentifier(databaseName+"."+tableName),
		backtickQuote(columnName),
		maxSamples,
	), maxSamples)
	if err != nil {
		return nil, err
	}
	samples := make([]string, 0, maxSamples)
	for _, row := range rows {
		samples = append(samples, firstRowValue(row))
	}
	return samples, nil
}
