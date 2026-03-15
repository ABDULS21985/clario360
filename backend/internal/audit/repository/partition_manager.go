package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/model"
)

// PartitionManager handles creating and listing table partitions.
type PartitionManager struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPartitionManager creates a new PartitionManager.
func NewPartitionManager(db *pgxpool.Pool, logger zerolog.Logger) *PartitionManager {
	return &PartitionManager{db: db, logger: logger}
}

// EnsurePartitions creates partitions for the current month and next 2 months.
// This is idempotent — it skips partitions that already exist.
func (pm *PartitionManager) EnsurePartitions(ctx context.Context) ([]string, error) {
	now := time.Now().UTC()
	var created []string

	for offset := 0; offset < 3; offset++ {
		target := now.AddDate(0, offset, 0)
		year, month, _ := target.Date()

		partName := fmt.Sprintf("audit_logs_%d_%02d", year, month)
		rangeStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		rangeEnd := rangeStart.AddDate(0, 1, 0)

		exists, err := pm.partitionExists(ctx, partName)
		if err != nil {
			return created, fmt.Errorf("checking partition %s: %w", partName, err)
		}
		if exists {
			continue
		}

		ddl := fmt.Sprintf(
			`CREATE TABLE IF NOT EXISTS %s PARTITION OF audit_logs FOR VALUES FROM ('%s') TO ('%s')`,
			partName,
			rangeStart.Format("2006-01-02"),
			rangeEnd.Format("2006-01-02"),
		)

		_, err = pm.db.Exec(ctx, ddl)
		if err != nil {
			// Ignore "already exists" errors from concurrent instances
			if isAlreadyExistsError(err) {
				pm.logger.Info().Str("partition", partName).Msg("partition already exists (concurrent create)")
				continue
			}
			return created, fmt.Errorf("creating partition %s: %w", partName, err)
		}

		pm.logger.Info().
			Str("partition", partName).
			Time("range_start", rangeStart).
			Time("range_end", rangeEnd).
			Msg("partition created")
		created = append(created, partName)
	}

	return created, nil
}

// ListPartitions returns metadata about all existing partitions with derived status fields.
// The returned PartitionInfo structs are aligned with the frontend AuditPartition interface.
func (pm *PartitionManager) ListPartitions(ctx context.Context) ([]model.PartitionInfo, error) {
	query := `
		SELECT
			c.relname AS name,
			pg_get_expr(c.relpartbound, c.oid) AS bound_expr,
			pg_total_relation_size(c.oid) AS size_bytes
		FROM pg_catalog.pg_inherits i
		JOIN pg_catalog.pg_class c ON c.oid = i.inhrelid
		JOIN pg_catalog.pg_class p ON p.oid = i.inhparent
		WHERE p.relname = 'audit_logs'
		ORDER BY c.relname`

	rows, err := pm.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list partitions: %w", err)
	}
	defer rows.Close()

	now := time.Now().UTC()
	var partitions []model.PartitionInfo
	for rows.Next() {
		var name, boundExpr string
		var sizeBytes int64
		if err := rows.Scan(&name, &boundExpr, &sizeBytes); err != nil {
			return nil, fmt.Errorf("scan partition info: %w", err)
		}

		pi := model.PartitionInfo{
			ID:        name, // use table name as stable ID
			Name:      name,
			SizeBytes: sizeBytes,
		}

		// Parse range bounds from the partition expression
		rangeStart, rangeEnd, parseErr := parsePartitionBounds(boundExpr)
		if parseErr == nil {
			pi.DateRangeStart = rangeStart
			pi.DateRangeEnd = rangeEnd
			pi.CreatedAt = rangeStart // approximate: partition was created at range start
			pi.Status = derivePartitionStatus(rangeStart, rangeEnd, now)
		} else {
			pi.Status = "active"
			pi.CreatedAt = now
		}

		// Get accurate record count
		var count int64
		if countErr := pm.db.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", name)).Scan(&count); countErr == nil {
			pi.RecordCount = count
		}

		partitions = append(partitions, pi)
	}

	if partitions == nil {
		partitions = []model.PartitionInfo{}
	}

	return partitions, rows.Err()
}

// ArchivePartition detaches the named partition from the parent audit_logs table.
// The partition table still exists and can be queried directly, but is excluded
// from normal audit_logs queries. Returns the archived partition's current info.
func (pm *PartitionManager) ArchivePartition(ctx context.Context, partitionName string) error {
	// Validate the partition name to prevent SQL injection
	if !isValidPartitionName(partitionName) {
		return fmt.Errorf("invalid partition name: %q", partitionName)
	}

	exists, err := pm.partitionExists(ctx, partitionName)
	if err != nil {
		return fmt.Errorf("checking partition %s: %w", partitionName, err)
	}
	if !exists {
		return fmt.Errorf("partition %s not found", partitionName)
	}

	ddl := fmt.Sprintf("ALTER TABLE audit_logs DETACH PARTITION %s", partitionName)
	if _, err := pm.db.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("detaching partition %s: %w", partitionName, err)
	}

	pm.logger.Info().Str("partition", partitionName).Msg("partition archived (detached)")
	return nil
}

// DeletePartition drops the named partition table entirely.
// The partition must already be detached (archived) before it can be deleted.
func (pm *PartitionManager) DeletePartition(ctx context.Context, partitionName string) error {
	if !isValidPartitionName(partitionName) {
		return fmt.Errorf("invalid partition name: %q", partitionName)
	}

	ddl := fmt.Sprintf("DROP TABLE IF EXISTS %s", partitionName)
	if _, err := pm.db.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("dropping partition %s: %w", partitionName, err)
	}

	pm.logger.Info().Str("partition", partitionName).Msg("partition deleted (dropped)")
	return nil
}

// partitionExists checks if a partition table exists (either attached or detached).
func (pm *PartitionManager) partitionExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := pm.db.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM pg_catalog.pg_class
			WHERE relname = $1 AND relkind = 'r'
		)`, name,
	).Scan(&exists)
	return exists, err
}

// derivePartitionStatus determines the status of a partition based on its date range.
// A partition is "pending" if it starts in the future, "active" if it covers the current
// period, and "archived" if its end date has passed.
func derivePartitionStatus(rangeStart, rangeEnd, now time.Time) string {
	if rangeStart.After(now) {
		return "pending"
	}
	if rangeEnd.Before(now) || rangeEnd.Equal(now) {
		return "archived"
	}
	return "active"
}

// isValidPartitionName checks that a partition name matches the expected pattern
// audit_logs_YYYY_MM to prevent SQL injection.
func isValidPartitionName(name string) bool {
	if len(name) < 15 || len(name) > 20 {
		return false
	}
	if name[:11] != "audit_logs_" {
		return false
	}
	rest := name[11:]
	if len(rest) != 7 {
		return false
	}
	for i, c := range rest {
		if i == 4 {
			if c != '_' {
				return false
			}
		} else {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// parsePartitionBounds extracts start and end times from a partition bound expression.
func parsePartitionBounds(expr string) (time.Time, time.Time, error) {
	// Format: FOR VALUES FROM ('2026-03-01') TO ('2026-04-01')
	var startStr, endStr string
	_, err := fmt.Sscanf(expr, "FOR VALUES FROM ('%10s') TO ('%10s')", &startStr, &endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing partition bounds: %w", err)
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("parsing end date: %w", err)
	}
	return start, end, nil
}

// isAlreadyExistsError checks if a PostgreSQL error is a "relation already exists" error.
func isAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "already exists") || contains(errStr, "42P07")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
