package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/clario360/platform/internal/observability/metrics"
	oteltracing "github.com/clario360/platform/internal/observability/tracing"
)

const maxSQLLogLen = 200

// InstrumentedDB wraps pgxpool.Pool with automatic metrics and tracing for every query.
type InstrumentedDB struct {
	pool    *pgxpool.Pool
	metrics *metrics.DBMetrics
	tracer  trace.Tracer
	logger  zerolog.Logger
	service string
}

// NewInstrumentedDB creates an instrumented database wrapper.
func NewInstrumentedDB(pool *pgxpool.Pool, dbMetrics *metrics.DBMetrics, tracer trace.Tracer, logger zerolog.Logger) *InstrumentedDB {
	return &InstrumentedDB{
		pool:    pool,
		metrics: dbMetrics,
		tracer:  tracer,
		logger:  logger,
		service: "database",
	}
}

// Query executes a query that returns rows, with tracing and metrics.
func (db *InstrumentedDB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	op := extractOperation(sql)
	ctx, span := db.startSpan(ctx, op, sql)
	defer span.End()

	start := time.Now()
	rows, err := db.pool.Query(ctx, sql, args...)
	db.recordMetrics(op, start, err)

	if err != nil {
		db.recordError(span, op, sql, err)
		return nil, err
	}

	return rows, nil
}

// QueryRow executes a query that returns at most one row.
func (db *InstrumentedDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	op := extractOperation(sql)
	ctx, span := db.startSpan(ctx, op, sql)

	start := time.Now()
	row := db.pool.QueryRow(ctx, sql, args...)

	// QueryRow doesn't return an error directly; it's deferred to Scan.
	// We record the span end and a success metric optimistically.
	// Errors will be caught when the caller calls row.Scan().
	duration := time.Since(start).Seconds()
	db.metrics.QueryDuration.WithLabelValues(op, db.service).Observe(duration)
	db.metrics.QueriesTotal.WithLabelValues(op, "ok", db.service).Inc()
	span.End()

	return row
}

// Exec executes a SQL statement that doesn't return rows.
func (db *InstrumentedDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	op := extractOperation(sql)
	ctx, span := db.startSpan(ctx, op, sql)
	defer span.End()

	start := time.Now()
	tag, err := db.pool.Exec(ctx, sql, args...)
	db.recordMetrics(op, start, err)

	if err != nil {
		db.recordError(span, op, sql, err)
		return tag, err
	}

	return tag, nil
}

// CopyFrom bulk-inserts rows using the COPY protocol.
func (db *InstrumentedDB) CopyFrom(ctx context.Context, tableName pgx.Identifier, columns []string, rowSrc pgx.CopyFromSource) (int64, error) {
	ctx, span := db.startSpan(ctx, "copy", fmt.Sprintf("COPY %s", tableName.Sanitize()))
	defer span.End()

	start := time.Now()
	n, err := db.pool.CopyFrom(ctx, tableName, columns, rowSrc)
	db.recordMetrics("copy", start, err)

	if err != nil {
		db.recordError(span, "copy", fmt.Sprintf("COPY %s", tableName.Sanitize()), err)
		return n, err
	}

	span.SetAttributes(attribute.Int64("db.rows_affected", n))
	return n, nil
}

// RunInTx executes fn within a database transaction.
// If fn returns an error, the transaction is rolled back.
// If fn returns nil, the transaction is committed.
func (db *InstrumentedDB) RunInTx(ctx context.Context, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	ctx, span := db.tracer.Start(ctx, "db.transaction",
		trace.WithAttributes(
			oteltracing.AttrDBSystem.String("postgresql"),
			oteltracing.AttrDBOperation.String("transaction"),
		),
	)
	defer span.End()

	start := time.Now()

	tx, err := db.pool.BeginTx(ctx, opts)
	if err != nil {
		oteltracing.RecordError(span, err)
		db.metrics.QueriesTotal.WithLabelValues("transaction", "error", db.service).Inc()
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			db.logger.Warn().Err(rbErr).Msg("transaction rollback failed")
		}
		oteltracing.RecordError(span, err)
		db.metrics.QueriesTotal.WithLabelValues("transaction", "error", db.service).Inc()
		db.metrics.QueryDuration.WithLabelValues("transaction", db.service).Observe(time.Since(start).Seconds())
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		oteltracing.RecordError(span, err)
		db.metrics.QueriesTotal.WithLabelValues("transaction", "error", db.service).Inc()
		db.metrics.QueryDuration.WithLabelValues("transaction", db.service).Observe(time.Since(start).Seconds())
		return fmt.Errorf("commit transaction: %w", err)
	}

	db.metrics.QueriesTotal.WithLabelValues("transaction", "ok", db.service).Inc()
	db.metrics.QueryDuration.WithLabelValues("transaction", db.service).Observe(time.Since(start).Seconds())
	return nil
}

// Pool returns the underlying pgxpool.Pool for cases that need direct access
// (e.g., LISTEN/NOTIFY, migrations).
func (db *InstrumentedDB) Pool() *pgxpool.Pool {
	return db.pool
}

// RunInTxWithTenant executes fn within a transaction where app.current_tenant_id
// is set via SET LOCAL. This is the InstrumentedDB equivalent of RunWithTenant.
// Combines tracing/metrics from RunInTx with RLS tenant context setting.
//
// The tenant context is scoped to this transaction and cannot leak to other connections
// in the pool. RLS policies on all tenant-scoped tables will automatically filter rows
// to those belonging to tenantID.
func (db *InstrumentedDB) RunInTxWithTenant(ctx context.Context, tenantID uuid.UUID, opts pgx.TxOptions, fn func(pgx.Tx) error) error {
	return db.RunInTx(ctx, opts, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String()); err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}
		return fn(tx)
	})
}

func (db *InstrumentedDB) startSpan(ctx context.Context, op, sql string) (context.Context, trace.Span) {
	return db.tracer.Start(ctx, "db."+op,
		trace.WithAttributes(
			oteltracing.AttrDBSystem.String("postgresql"),
			oteltracing.AttrDBOperation.String(op),
			oteltracing.AttrDBStatement.String(truncateSQL(sql)),
			oteltracing.AttrDBTable.String(extractTable(sql)),
		),
	)
}

func (db *InstrumentedDB) recordMetrics(op string, start time.Time, err error) {
	duration := time.Since(start).Seconds()
	db.metrics.QueryDuration.WithLabelValues(op, db.service).Observe(duration)

	status := "ok"
	if err != nil {
		status = "error"
	}
	db.metrics.QueriesTotal.WithLabelValues(op, status, db.service).Inc()
}

func (db *InstrumentedDB) recordError(span trace.Span, op, sql string, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())

	db.logger.Warn().
		Err(err).
		Str("operation", op).
		Str("sql", truncateSQL(sql)).
		Msg("database query error")
}

// extractOperation extracts the SQL operation from the first keyword.
func extractOperation(sql string) string {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return "other"
	}

	// Find the first word.
	end := strings.IndexAny(trimmed, " \t\n\r")
	if end == -1 {
		end = len(trimmed)
	}

	switch strings.ToLower(trimmed[:end]) {
	case "select":
		return "select"
	case "insert":
		return "insert"
	case "update":
		return "update"
	case "delete":
		return "delete"
	default:
		return "other"
	}
}

// extractTable attempts to extract the table name from SQL.
func extractTable(sql string) string {
	upper := strings.ToUpper(strings.TrimSpace(sql))

	// Look for FROM, INTO, or UPDATE table patterns.
	for _, keyword := range []string{"FROM ", "INTO ", "UPDATE "} {
		idx := strings.Index(upper, keyword)
		if idx == -1 {
			continue
		}
		rest := strings.TrimSpace(sql[idx+len(keyword):])
		// Take the next word as the table name.
		end := strings.IndexAny(rest, " \t\n\r(,;")
		if end == -1 {
			end = len(rest)
		}
		table := rest[:end]
		// Strip schema prefix for display.
		if dot := strings.LastIndex(table, "."); dot != -1 {
			table = table[dot+1:]
		}
		return strings.ToLower(table)
	}

	return ""
}

// truncateSQL truncates SQL to maxSQLLogLen characters.
// CRITICAL: Never log full queries with data — they may contain PII.
func truncateSQL(sql string) string {
	if len(sql) <= maxSQLLogLen {
		return sql
	}
	return sql[:maxSQLLogLen] + "..."
}
