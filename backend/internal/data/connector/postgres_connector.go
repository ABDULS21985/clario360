package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

type PostgresConnector struct {
	config model.PostgresConnectionConfig
	pool   *pgxpool.Pool
	logger zerolog.Logger
	limits ConnectorLimits
}

func NewPostgresConnector(configJSON json.RawMessage, options FactoryOptions) (Connector, error) {
	var cfg model.PostgresConnectionConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("decode postgres config: %w", err)
	}
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.Schema == "" {
		cfg.Schema = "public"
	}
	if cfg.StatementTimeoutMs == 0 {
		cfg.StatementTimeoutMs = int(options.Limits.StatementTimeout.Milliseconds())
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("validate postgres config: %w", err)
	}

	return &PostgresConnector{
		config: cfg,
		logger: options.Logger.With().Str("connector", "postgresql").Logger(),
		limits: options.Limits,
	}, nil
}

func (c *PostgresConnector) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	ctx, cancel := context.WithTimeout(ctx, c.limits.ConnectTimeout)
	defer cancel()

	start := time.Now()
	pool, err := c.openPool(ctx)
	if err != nil {
		return nil, err
	}

	var version string
	if err := pool.QueryRow(ctx, "SELECT version()").Scan(&version); err != nil {
		return nil, fmt.Errorf("query postgres version: %w", err)
	}

	permissions := make([]string, 0, 3)
	var canConnect bool
	if err := pool.QueryRow(ctx, "SELECT has_database_privilege(current_user, current_database(), 'CONNECT')").Scan(&canConnect); err == nil && canConnect {
		permissions = append(permissions, "connect")
	}
	var canUseSchema bool
	if err := pool.QueryRow(ctx, "SELECT has_schema_privilege(current_user, $1, 'USAGE')", c.config.Schema).Scan(&canUseSchema); err == nil && canUseSchema {
		permissions = append(permissions, "schema_usage")
	}
	var canReadSchema bool
	if err := pool.QueryRow(ctx, "SELECT has_table_privilege(current_user, 'information_schema.tables', 'SELECT')").Scan(&canReadSchema); err == nil && canReadSchema {
		permissions = append(permissions, "read_schema")
	}

	duration := time.Since(start)
	warnings := make([]string, 0, 2)
	if strings.EqualFold(c.config.SSLMode, "disable") && !isLocalhost(c.config.Host) {
		warnings = append(warnings, "Connection is not encrypted.")
	}
	if duration > 5*time.Second {
		warnings = append(warnings, "Slow connection.")
	}

	return &ConnectionTestResult{
		Success:     true,
		LatencyMs:   duration.Milliseconds(),
		Version:     version,
		Message:     "PostgreSQL connection established successfully.",
		Permissions: permissions,
		Warnings:    warnings,
		Duration:    duration,
	}, nil
}

func (c *PostgresConnector) DiscoverSchema(ctx context.Context, opts DiscoveryOptions) (*model.DiscoveredSchema, error) {
	pool, err := c.openPool(ctx)
	if err != nil {
		return nil, err
	}

	tableQuery := `
		SELECT t.table_schema, t.table_name, t.table_type,
		       pg_catalog.obj_description(pc.oid, 'pg_class') AS comment
		FROM information_schema.tables t
		LEFT JOIN pg_catalog.pg_class pc ON pc.relname = t.table_name
		LEFT JOIN pg_catalog.pg_namespace pn ON pn.oid = pc.relnamespace AND pn.nspname = t.table_schema
		WHERE t.table_schema NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		  AND ($1 = '' OR t.table_schema = $1)
		  AND (t.table_type = 'BASE TABLE' OR ($2 AND t.table_type = 'VIEW'))
		ORDER BY t.table_schema, t.table_name
		LIMIT $3`

	rows, err := pool.Query(ctx, tableQuery, opts.SchemaFilter, opts.IncludeViews, opts.MaxTables)
	if err != nil {
		return nil, fmt.Errorf("list postgres tables: %w", err)
	}
	defer rows.Close()

	var tables []model.DiscoveredTable
	totalColumns := 0
	containsPII := false
	highestClass := model.DataClassificationPublic

	for rows.Next() {
		var schemaName, tableName, tableType string
		var comment *string
		if err := rows.Scan(&schemaName, &tableName, &tableType, &comment); err != nil {
			return nil, fmt.Errorf("scan postgres table: %w", err)
		}

		table, err := c.discoverTable(ctx, pool, schemaName, tableName, tableType, stringValue(comment), opts)
		if err != nil {
			return nil, err
		}
		totalColumns += len(table.Columns)
		containsPII = containsPII || table.ContainsPII
		highestClass = discovery.MaxClassification(highestClass, table.InferredClass)
		tables = append(tables, *table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres tables: %w", err)
	}

	return &model.DiscoveredSchema{
		Tables:       tables,
		TableCount:   len(tables),
		ColumnCount:  totalColumns,
		ContainsPII:  containsPII,
		HighestClass: highestClass,
	}, nil
}

func (c *PostgresConnector) FetchData(ctx context.Context, table string, params FetchParams) (*DataBatch, error) {
	pool, err := c.openPool(ctx)
	if err != nil {
		return nil, err
	}

	schemaName, tableName := splitQualifiedName(table, c.config.Schema)
	tableIdent := pgx.Identifier{schemaName, tableName}.Sanitize()

	columns := "*"
	if len(params.Columns) > 0 {
		quoted := make([]string, 0, len(params.Columns))
		for _, column := range params.Columns {
			quoted = append(quoted, pgx.Identifier{column}.Sanitize())
		}
		columns = strings.Join(quoted, ", ")
	}

	query := fmt.Sprintf("SELECT %s FROM %s", columns, tableIdent)
	args := make([]any, 0, len(params.Filters)+2)
	conditions := make([]string, 0, len(params.Filters))
	index := 1
	for column, value := range params.Filters {
		conditions = append(conditions, fmt.Sprintf("%s = $%d", pgx.Identifier{column}.Sanitize(), index))
		args = append(args, value)
		index++
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if params.OrderBy != "" {
		query += " ORDER BY " + pgx.Identifier{params.OrderBy}.Sanitize()
	}
	if params.BatchSize <= 0 {
		params.BatchSize = 100
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", index, index+1)
	args = append(args, params.BatchSize, params.Offset)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("fetch postgres data: %w", err)
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, 0, len(fieldDescriptions))
	for _, field := range fieldDescriptions {
		columnNames = append(columnNames, field.Name)
	}

	batchRows := make([]map[string]any, 0, params.BatchSize)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("read postgres row values: %w", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = values[i]
		}
		batchRows = append(batchRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres rows: %w", err)
	}

	return &DataBatch{
		Columns:  columnNames,
		Rows:     batchRows,
		RowCount: len(batchRows),
		HasMore:  len(batchRows) == params.BatchSize,
	}, nil
}

func (c *PostgresConnector) ReadQuery(ctx context.Context, query string, args []any) (*DataBatch, error) {
	if !isReadOnlyQuery(query) {
		return nil, fmt.Errorf("%w: postgres connector only allows read-only queries", ErrCapabilityUnsupported)
	}

	pool, err := c.openPool(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin postgres read-only transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("execute postgres query: %w", err)
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columnNames := make([]string, 0, len(fieldDescriptions))
	for _, field := range fieldDescriptions {
		columnNames = append(columnNames, field.Name)
	}

	resultRows := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("read postgres query row: %w", err)
		}
		row := make(map[string]any, len(columnNames))
		for i, column := range columnNames {
			row[column] = normalizeSQLValue(values[i])
		}
		resultRows = append(resultRows, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate postgres query rows: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit postgres read-only transaction: %w", err)
	}

	return &DataBatch{
		Columns:  columnNames,
		Rows:     resultRows,
		RowCount: len(resultRows),
	}, nil
}

func (c *PostgresConnector) WriteData(ctx context.Context, table string, rows []map[string]any, params WriteParams) (*WriteResult, error) {
	if len(rows) == 0 {
		return &WriteResult{}, nil
	}

	pool, err := c.openPool(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin postgres write transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	schemaName, tableName := splitQualifiedName(table, c.config.Schema)
	tableIdent := pgx.Identifier{schemaName, tableName}.Sanitize()
	if params.Replace {
		if _, err := tx.Exec(ctx, "TRUNCATE TABLE "+tableIdent); err != nil {
			return nil, fmt.Errorf("truncate postgres target %s: %w", tableIdent, err)
		}
	}

	columns := writeColumns(rows)
	quotedColumns := make([]string, 0, len(columns))
	for _, column := range columns {
		quotedColumns = append(quotedColumns, pgx.Identifier{column}.Sanitize())
	}

	args := make([]any, 0, len(rows)*len(columns))
	valueGroups := make([]string, 0, len(rows))
	argIndex := 1
	for _, row := range rows {
		values := rowValues(row, columns)
		placeholders := make([]string, 0, len(values))
		for _, value := range values {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			args = append(args, value)
			argIndex++
		}
		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ", ")+")")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		tableIdent,
		strings.Join(quotedColumns, ", "),
		strings.Join(valueGroups, ", "),
	)

	if len(params.MergeKeys) > 0 {
		mergeColumns := make([]string, 0, len(params.MergeKeys))
		mergeSet := make(map[string]struct{}, len(params.MergeKeys))
		for _, key := range params.MergeKeys {
			mergeColumns = append(mergeColumns, pgx.Identifier{key}.Sanitize())
			mergeSet[key] = struct{}{}
		}
		switch params.Strategy {
		case "incremental":
			query += " ON CONFLICT (" + strings.Join(mergeColumns, ", ") + ") DO NOTHING"
		case "merge":
			assignments := make([]string, 0, len(columns))
			for _, column := range columns {
				if _, skip := mergeSet[column]; skip {
					continue
				}
				quoted := pgx.Identifier{column}.Sanitize()
				assignments = append(assignments, quoted+" = EXCLUDED."+quoted)
			}
			if len(assignments) == 0 {
				query += " ON CONFLICT (" + strings.Join(mergeColumns, ", ") + ") DO NOTHING"
			} else {
				query += " ON CONFLICT (" + strings.Join(mergeColumns, ", ") + ") DO UPDATE SET " + strings.Join(assignments, ", ")
			}
		}
	}

	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("insert postgres target data: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit postgres write transaction: %w", err)
	}

	return &WriteResult{
		RowsWritten:  result.RowsAffected(),
		BytesWritten: int64(len(mustJSON(rows))),
	}, nil
}

func (c *PostgresConnector) EstimateSize(ctx context.Context) (*SizeEstimate, error) {
	pool, err := c.openPool(ctx)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT COUNT(*),
		       COALESCE(SUM(pc.reltuples::bigint), 0),
		       COALESCE(SUM(pg_total_relation_size(pc.oid)), 0)
		FROM pg_class pc
		JOIN pg_namespace pn ON pn.oid = pc.relnamespace
		WHERE pc.relkind IN ('r', 'v')
		  AND pn.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
		  AND ($1 = '' OR pn.nspname = $1)`

	var estimate SizeEstimate
	if err := pool.QueryRow(ctx, query, c.config.Schema).Scan(&estimate.TableCount, &estimate.TotalRows, &estimate.TotalBytes); err != nil {
		return nil, fmt.Errorf("estimate postgres size: %w", err)
	}
	return &estimate, nil
}

func (c *PostgresConnector) Close() error {
	if c.pool != nil {
		c.pool.Close()
		c.pool = nil
	}
	return nil
}

func (c *PostgresConnector) openPool(ctx context.Context) (*pgxpool.Pool, error) {
	if c.pool != nil {
		return c.pool, nil
	}

	dsn, err := c.connectionString()
	if err != nil {
		return nil, err
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres pool config: %w", err)
	}
	cfg.MaxConns = int32(c.limits.MaxPoolSize)
	cfg.MinConns = 1
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = fmt.Sprintf("%d", c.config.StatementTimeoutMs)
	cfg.ConnConfig.ConnectTimeout = c.limits.ConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres source pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres source: %w", err)
	}
	c.pool = pool
	return pool, nil
}

func (c *PostgresConnector) connectionString() (string, error) {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.config.Username, c.config.Password),
		Host:   fmt.Sprintf("%s:%d", c.config.Host, c.config.Port),
		Path:   c.config.Database,
	}
	query := url.Values{}
	query.Set("sslmode", c.config.SSLMode)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func (c *PostgresConnector) discoverTable(ctx context.Context, pool *pgxpool.Pool, schemaName, tableName, tableType, comment string, opts DiscoveryOptions) (*model.DiscoveredTable, error) {
	columnQuery := `
		SELECT c.column_name,
		       c.data_type,
		       c.udt_name,
		       c.character_maximum_length,
		       c.is_nullable,
		       c.column_default,
		       pgd.description
		FROM information_schema.columns c
		LEFT JOIN pg_catalog.pg_statio_all_tables st ON st.relname = c.table_name AND st.schemaname = c.table_schema
		LEFT JOIN pg_catalog.pg_description pgd ON pgd.objoid = st.relid AND pgd.objsubid = c.ordinal_position
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
		LIMIT $3`

	rows, err := pool.Query(ctx, columnQuery, schemaName, tableName, opts.MaxColumns)
	if err != nil {
		return nil, fmt.Errorf("discover postgres columns for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	columns := make([]model.DiscoveredColumn, 0)
	for rows.Next() {
		var columnName, dataType, udtName, nullable string
		var maxLength *int
		var defaultValue *string
		var columnComment *string
		if err := rows.Scan(&columnName, &dataType, &udtName, &maxLength, &nullable, &defaultValue, &columnComment); err != nil {
			return nil, fmt.Errorf("scan postgres column for %s.%s: %w", schemaName, tableName, err)
		}
		mapped := discovery.MapNativeType(udtName)
		column := model.DiscoveredColumn{
			Name:         columnName,
			DataType:     dataType,
			NativeType:   udtName,
			MappedType:   mapped.Type,
			Subtype:      mapped.Subtype,
			MaxLength:    maxLength,
			Nullable:     nullable == "YES",
			DefaultValue: defaultValue,
			Comment:      stringValue(columnComment),
		}
		if opts.SampleValues {
			samples, err := c.sampleColumnValues(ctx, pool, schemaName, tableName, columnName, opts.MaxSamples)
			if err != nil {
				c.logger.Warn().Err(err).Str("schema", schemaName).Str("table", tableName).Str("column", columnName).Msg("failed to sample postgres column")
			} else {
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
		return nil, fmt.Errorf("iterate postgres columns for %s.%s: %w", schemaName, tableName, err)
	}

	primaryKeys, err := c.loadPrimaryKeys(ctx, pool, schemaName, tableName)
	if err != nil {
		return nil, err
	}
	foreignKeys, err := c.loadForeignKeys(ctx, pool, schemaName, tableName)
	if err != nil {
		return nil, err
	}

	pkSet := make(map[string]struct{}, len(primaryKeys))
	for _, key := range primaryKeys {
		pkSet[key] = struct{}{}
	}
	fkSet := make(map[string]model.ForeignKeyRef, len(foreignKeys))
	for _, fk := range foreignKeys {
		fkSet[fk.Column] = fk.ReferencedRef
	}

	columns = discovery.DetectPII(columns)
	nullableCount := 0
	piiCount := 0
	for i := range columns {
		if _, ok := pkSet[columns[i].Name]; ok {
			columns[i].IsPrimaryKey = true
		}
		if ref, ok := fkSet[columns[i].Name]; ok {
			columns[i].IsForeignKey = true
			columns[i].ForeignKeyRef = &ref
		}
		if columns[i].Nullable {
			nullableCount++
		}
		if columns[i].InferredPII {
			piiCount++
		}
	}

	var estimatedRows, sizeBytes int64
	statsQuery := `
		SELECT COALESCE(pc.reltuples::bigint, 0),
		       COALESCE(pg_total_relation_size(pc.oid), 0)
		FROM pg_class pc
		JOIN pg_namespace pn ON pn.oid = pc.relnamespace
		WHERE pn.nspname = $1 AND pc.relname = $2
		LIMIT 1`
	_ = pool.QueryRow(ctx, statsQuery, schemaName, tableName).Scan(&estimatedRows, &sizeBytes)

	table := &model.DiscoveredTable{
		SchemaName:      schemaName,
		Name:            tableName,
		Type:            strings.ToLower(tableType),
		Comment:         comment,
		Columns:         columns,
		PrimaryKeys:     primaryKeys,
		ForeignKeys:     foreignKeys,
		EstimatedRows:   estimatedRows,
		SizeBytes:       sizeBytes,
		InferredClass:   discovery.TableClassification(columns),
		ContainsPII:     piiCount > 0,
		PIIColumnCount:  piiCount,
		NullableCount:   nullableCount,
		SampledRowCount: opts.MaxSamples,
	}
	return table, nil
}

func (c *PostgresConnector) loadPrimaryKeys(ctx context.Context, pool *pgxpool.Pool, schemaName, tableName string) ([]string, error) {
	query := `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = $1 AND tc.table_name = $2
		ORDER BY kcu.ordinal_position`

	rows, err := pool.Query(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("load postgres primary keys for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	keys := make([]string, 0)
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			return nil, fmt.Errorf("scan postgres primary key for %s.%s: %w", schemaName, tableName, err)
		}
		keys = append(keys, column)
	}
	return keys, rows.Err()
}

func (c *PostgresConnector) loadForeignKeys(ctx context.Context, pool *pgxpool.Pool, schemaName, tableName string) ([]model.ForeignKey, error) {
	query := `
		SELECT kcu.column_name,
		       ccu.table_schema,
		       ccu.table_name,
		       ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = $1 AND tc.table_name = $2`

	rows, err := pool.Query(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("load postgres foreign keys for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	keys := make([]model.ForeignKey, 0)
	for rows.Next() {
		var column, refSchema, refTable, refColumn string
		if err := rows.Scan(&column, &refSchema, &refTable, &refColumn); err != nil {
			return nil, fmt.Errorf("scan postgres foreign key for %s.%s: %w", schemaName, tableName, err)
		}
		keys = append(keys, model.ForeignKey{
			Column: column,
			ReferencedRef: model.ForeignKeyRef{
				Schema: refSchema,
				Table:  refTable,
				Column: refColumn,
			},
		})
	}
	return keys, rows.Err()
}

func (c *PostgresConnector) sampleColumnValues(ctx context.Context, pool *pgxpool.Pool, schemaName, tableName, columnName string, maxSamples int) ([]string, error) {
	if maxSamples <= 0 {
		maxSamples = 5
	}
	columnIdent := pgx.Identifier{columnName}.Sanitize()
	tableIdent := pgx.Identifier{schemaName, tableName}.Sanitize()
	query := fmt.Sprintf("SELECT DISTINCT %s::text FROM %s WHERE %s IS NOT NULL LIMIT %d", columnIdent, tableIdent, columnIdent, maxSamples)

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("sample postgres column %s.%s.%s: %w", schemaName, tableName, columnName, err)
	}
	defer rows.Close()

	samples := make([]string, 0, maxSamples)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan postgres sample %s.%s.%s: %w", schemaName, tableName, columnName, err)
		}
		samples = append(samples, value)
	}
	return samples, rows.Err()
}

func splitQualifiedName(value, fallbackSchema string) (string, string) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	if fallbackSchema == "" {
		fallbackSchema = "public"
	}
	return fallbackSchema, value
}

func isLocalhost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isReadOnlyQuery(query string) bool {
	normalized := strings.TrimSpace(strings.ToLower(query))
	return strings.HasPrefix(normalized, "select") || strings.HasPrefix(normalized, "with")
}
