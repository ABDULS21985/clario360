package connector

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/beltran/gohive"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

const hiveConnectorType = "hive"

type HiveConnector struct {
	config   model.HiveConnectionConfig
	conn     *gohive.Connection
	sourceID uuid.UUID
	tenantID uuid.UUID
	logger   zerolog.Logger
	limits   ConnectorLimits
}

func NewHiveConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.HiveConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, newConnectorError(hiveConnectorType, "connect", ErrorCodeConfigurationError, "decode hive config", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 10000
	}
	if cfg.TransportMode == "" {
		cfg.TransportMode = "binary"
	}
	if cfg.AuthType == "" {
		cfg.AuthType = "noauth"
	}
	if cfg.HTTPPath == "" {
		cfg.HTTPPath = "cliservice"
	}
	if cfg.QueryTimeoutSeconds == 0 {
		cfg.QueryTimeoutSeconds = 120
	}
	if cfg.FetchSize == 0 {
		cfg.FetchSize = 1000
	}
	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, newConnectorError(hiveConnectorType, "connect", ErrorCodeConfigurationError, "validate hive config", err)
	}
	connector := &HiveConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", hiveConnectorType).Logger(),
		limits: options.Limits,
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(hiveConnectorType).Inc()
	return connector, nil
}

func (c *HiveConnector) SetSourceContext(sourceID, tenantID uuid.UUID) {
	c.sourceID = sourceID
	c.tenantID = tenantID
}

func (c *HiveConnector) TestConnection(ctx context.Context) (_ *ConnectionTestResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "test", start, err) }()

	conn, err := c.openConnection(ctx)
	if err != nil {
		return nil, err
	}
	_, rows, err := gohiveQueryRows(ctx, conn, "SHOW DATABASES", 1000)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "test", ErrorCodeQueryFailed, "show hive databases", err)
	}
	databases := make([]string, 0, len(rows))
	for _, row := range rows {
		databases = append(databases, firstRowValue(row))
	}
	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   time.Since(start).Milliseconds(),
		Version:     "HiveServer2",
		Message:     fmt.Sprintf("Connected to Hive. %d databases accessible.", len(databases)),
		Permissions: databases,
	}, nil
}

func (c *HiveConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (_ *model.DiscoveredSchema, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "discover", start, err) }()

	conn, err := c.openConnection(ctx)
	if err != nil {
		return nil, err
	}
	databaseName := c.config.Database
	if databaseName == "" {
		databaseName = "default"
	}
	_, rows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("SHOW TABLES IN %s", backtickQuote(databaseName)), opts.MaxTables)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "show hive tables", err)
	}

	tables := make([]model.DiscoveredTable, 0)
	totalColumns := 0
	containsPII := false
	highest := model.DataClassificationPublic
	for _, row := range rows {
		tableName := firstRowValue(row)
		if tableName == "" {
			continue
		}
		table, tableErr := c.discoverTable(ctx, conn, databaseName, tableName, opts)
		if tableErr != nil {
			return nil, tableErr
		}
		tables = append(tables, *table)
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highest = discovery.MaxClassification(highest, table.InferredClass)
	}
	observeSchemaMetrics(hiveConnectorType, tables)
	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highest,
	}, nil
}

func (c *HiveConnector) FetchData(ctx context.Context, table string, params FetchParams) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "fetch", start, err) }()

	conn, err := c.openConnection(ctx)
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
	target := quoteDotBacktickIdentifier(table)
	if !strings.Contains(table, ".") {
		target = quoteDotBacktickIdentifier(defaultString(c.config.Database, "default") + "." + table)
	}
	query := fmt.Sprintf("SELECT %s FROM %s", columns, target)
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
		params.BatchSize = 1000
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", params.BatchSize, params.Offset)
	description, rows, err := gohiveQueryRows(ctx, conn, query, params.BatchSize)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "fetch", ErrorCodeQueryFailed, "fetch hive rows", err)
	}
	columnNames := descriptionNames(description)
	observeFetchMetrics(hiveConnectorType, len(rows), int64(len(mustJSON(rows))))
	return &DataBatch{
		Columns:  columnNames,
		Rows:     rows,
		RowCount: len(rows),
		HasMore:  len(rows) == params.BatchSize,
	}, nil
}

func (c *HiveConnector) ReadQuery(ctx context.Context, query string, args []any) (_ *DataBatch, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "read_query", start, err) }()
	_ = args
	if !isReadOnlyQuery(query) {
		return nil, newConnectorError(hiveConnectorType, "read_query", ErrorCodeUnsupportedOperation, "only read-only queries are allowed", ErrCapabilityUnsupported)
	}
	conn, err := c.openConnection(ctx)
	if err != nil {
		return nil, err
	}
	description, rows, err := gohiveQueryRows(ctx, conn, query, 0)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "read_query", ErrorCodeQueryFailed, "execute hive query", err)
	}
	return &DataBatch{Columns: descriptionNames(description), Rows: rows, RowCount: len(rows)}, nil
}

func (c *HiveConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (_ *WriteResult, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "write", start, err) }()
	if params.Strategy == "merge" {
		return nil, newConnectorError(hiveConnectorType, "write", ErrorCodeUnsupportedOperation, "Hive connector does not support merge writes", ErrCapabilityUnsupported)
	}
	if len(rows) == 0 {
		return &WriteResult{}, nil
	}
	conn, err := c.openConnection(ctx)
	if err != nil {
		return nil, err
	}
	columns := writeColumns(rows)
	if params.Replace {
		if err = gohiveExec(ctx, conn, "TRUNCATE TABLE "+quoteDotBacktickIdentifier(table)); err != nil {
			return nil, newConnectorError(hiveConnectorType, "write", ErrorCodeQueryFailed, "truncate hive table", err)
		}
	}
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
	query := fmt.Sprintf("INSERT INTO TABLE %s (%s) VALUES %s", quoteDotBacktickIdentifier(table), strings.Join(quotedColumns, ", "), strings.Join(valueGroups, ", "))
	if err = gohiveExec(ctx, conn, query); err != nil {
		return nil, newConnectorError(hiveConnectorType, "write", ErrorCodeQueryFailed, "insert hive rows", err)
	}
	return &WriteResult{RowsWritten: int64(len(rows)), BytesWritten: int64(len(mustJSON(rows)))}, nil
}

func (c *HiveConnector) EstimateSize(ctx context.Context) (_ *SizeEstimate, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "estimate", start, err) }()
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

func (c *HiveConnector) QueryAccessLogs(ctx context.Context, since time.Time) ([]DataAccessEvent, error) {
	_ = ctx
	_ = since
	return []DataAccessEvent{}, nil
}

func (c *HiveConnector) ListDataLocations(ctx context.Context) (_ []DataLocation, err error) {
	start := time.Now()
	defer func() { observeConnectorOperation(hiveConnectorType, "locations", start, err) }()
	conn, err := c.openConnection(ctx)
	if err != nil {
		return nil, err
	}
	databaseName := defaultString(c.config.Database, "default")
	_, rows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("SHOW TABLES IN %s", backtickQuote(databaseName)), c.limits.MaxTables)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "locations", ErrorCodeQueryFailed, "show hive tables for locations", err)
	}
	locations := make([]DataLocation, 0, len(rows))
	for _, row := range rows {
		tableName := firstRowValue(row)
		if tableName == "" {
			continue
		}
		_, formattedRows, queryErr := gohiveQueryRows(ctx, conn, fmt.Sprintf("DESCRIBE FORMATTED %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), 500)
		if queryErr != nil {
			return nil, newConnectorError(hiveConnectorType, "locations", ErrorCodeQueryFailed, "describe formatted hive table", queryErr)
		}
		meta := parseHiveDescribeFormatted(formattedRows)
		location := DataLocation{
			SourceID:     c.sourceID,
			SourceType:   hiveConnectorType,
			Table:        tableName,
			Database:     databaseName,
			Location:     meta.Location,
			Format:       inputFormatToFormat(meta.InputFormat),
			SizeBytes:    meta.RawDataSize,
			Partitioned:  len(meta.PartitionColumns) > 0,
			Partitions:   len(meta.PartitionColumns),
			LastModified: time.Now().UTC(),
		}
		if meta.LastDDLTime != nil {
			location.LastModified = *meta.LastDDLTime
		}
		locations = append(locations, location)
	}
	return locations, nil
}

func (c *HiveConnector) Close() error {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		getConnectorMetrics().ActiveConnections.WithLabelValues(hiveConnectorType).Dec()
		return nil
	}
	getConnectorMetrics().ActiveConnections.WithLabelValues(hiveConnectorType).Dec()
	return nil
}

func (c *HiveConnector) openConnection(ctx context.Context) (*gohive.Connection, error) {
	if c.conn != nil {
		return c.conn, nil
	}
	cfg := gohive.NewConnectConfiguration()
	cfg.Username = c.config.Username
	cfg.Password = c.config.Password
	cfg.Service = "hive"
	cfg.Database = defaultString(c.config.Database, "default")
	cfg.FetchSize = int64(c.config.FetchSize)
	cfg.TransportMode = c.config.TransportMode
	cfg.HTTPPath = c.config.HTTPPath
	cfg.ConnectTimeout = c.limits.ConnectTimeout
	cfg.SocketTimeout = time.Duration(c.config.QueryTimeoutSeconds) * time.Second
	cfg.HttpTimeout = time.Duration(c.config.QueryTimeoutSeconds) * time.Second
	if c.config.UseTLS {
		cfg.TLSConfig = &tls.Config{ServerName: c.config.Host, MinVersion: tls.VersionTLS12}
	}
	auth := "NOSASL"
	switch c.config.AuthType {
	case "plain":
		auth = "NONE"
	case "kerberos":
		auth = "KERBEROS"
	}
	connection, err := gohive.Connect(c.config.Host, c.config.Port, auth, cfg)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "connect", ErrorCodeConnectionFailed, "connect to hive", err)
	}
	c.conn = connection
	return connection, nil
}

func (c *HiveConnector) discoverTable(ctx context.Context, conn *gohive.Connection, databaseName, tableName string, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	_, describeRows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("DESCRIBE %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), max(100, opts.MaxColumns))
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "describe hive table", err)
	}
	_, formattedRows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf("DESCRIBE FORMATTED %s", quoteDotBacktickIdentifier(databaseName+"."+tableName)), 1000)
	if err != nil {
		return nil, newConnectorError(hiveConnectorType, "discover", ErrorCodeSchemaDiscoveryFailed, "describe formatted hive table", err)
	}
	meta := parseHiveDescribeFormatted(formattedRows)

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
			samples, sampleErr := c.sampleColumnValues(ctx, conn, databaseName, tableName, name, opts.MaxSamples)
			if sampleErr == nil {
				column.SampleValues = samples
				column.SampleStats = discovery.AnalyzeSamples(samples)
			}
		}
		columns = append(columns, column)
		nullableCount++
	}
	columns = discovery.DetectPII(columns)
	piiCount := 0
	for _, column := range columns {
		if column.InferredPII {
			piiCount++
		}
	}
	comment := "Hive table"
	if meta.Location != "" {
		comment = meta.Location
	}
	return &model.DiscoveredTable{
		SchemaName:     databaseName,
		Name:           tableName,
		Type:           inputFormatToFormat(meta.InputFormat),
		Comment:        comment,
		Columns:        columns,
		EstimatedRows:  meta.NumRows,
		SizeBytes:      meta.RawDataSize,
		InferredClass:  discovery.TableClassification(columns),
		ContainsPII:    piiCount > 0,
		PIIColumnCount: piiCount,
		NullableCount:  nullableCount,
	}, nil
}

func (c *HiveConnector) sampleColumnValues(ctx context.Context, conn *gohive.Connection, databaseName, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	_, rows, err := gohiveQueryRows(ctx, conn, fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s IS NOT NULL LIMIT %d",
		backtickQuote(columnName),
		quoteDotBacktickIdentifier(databaseName+"."+tableName),
		backtickQuote(columnName),
		maxSamples,
	), maxSamples)
	if err != nil {
		return nil, err
	}
	samples := make([]string, 0, len(rows))
	for _, row := range rows {
		samples = append(samples, firstRowValue(row))
	}
	return samples, nil
}

func descriptionNames(description [][]string) []string {
	names := make([]string, 0, len(description))
	for _, entry := range description {
		if len(entry) > 0 {
			names = append(names, strings.TrimPrefix(entry[0], "tab_name."))
		}
	}
	return names
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
