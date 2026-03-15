package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/model"
)

// AuditRepository handles all database operations for audit log entries.
type AuditRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewAuditRepository creates a new AuditRepository.
func NewAuditRepository(db *pgxpool.Pool, logger zerolog.Logger) *AuditRepository {
	return &AuditRepository{db: db, logger: logger}
}

// BatchInsert inserts multiple audit entries using a multi-row INSERT with ON CONFLICT DO NOTHING
// for deduplication by event_id.
func (r *AuditRepository) BatchInsert(ctx context.Context, entries []model.AuditEntry) (int64, error) {
	if len(entries) == 0 {
		return 0, nil
	}

	// Build multi-row INSERT
	var b strings.Builder
	b.WriteString(`INSERT INTO audit_logs (
		id, tenant_id, user_id, user_email, service, action, severity,
		resource_type, resource_id, old_value, new_value, ip_address,
		user_agent, metadata, event_id, correlation_id, previous_hash,
		entry_hash, created_at
	) VALUES `)

	args := make([]interface{}, 0, len(entries)*19)
	for i, e := range entries {
		if i > 0 {
			b.WriteString(", ")
		}
		offset := i * 19
		b.WriteString(fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			offset+1, offset+2, offset+3, offset+4, offset+5,
			offset+6, offset+7, offset+8, offset+9, offset+10,
			offset+11, offset+12, offset+13, offset+14, offset+15,
			offset+16, offset+17, offset+18, offset+19,
		))

		metadataJSON := e.Metadata
		if len(metadataJSON) == 0 {
			metadataJSON = json.RawMessage(`{}`)
		}

		args = append(args,
			e.ID, e.TenantID, e.UserID, e.UserEmail, e.Service,
			e.Action, e.Severity, e.ResourceType, e.ResourceID,
			nullableJSON(e.OldValue), nullableJSON(e.NewValue),
			e.IPAddress, e.UserAgent, metadataJSON, e.EventID,
			e.CorrelationID, e.PreviousHash, e.EntryHash, e.CreatedAt,
		)
	}

	b.WriteString(" ON CONFLICT (event_id, created_at) DO NOTHING")

	tag, err := r.db.Exec(ctx, b.String(), args...)
	if err != nil {
		return 0, fmt.Errorf("batch insert audit entries: %w", err)
	}

	return tag.RowsAffected(), nil
}

// FindByID retrieves a single audit entry by ID and tenant.
func (r *AuditRepository) FindByID(ctx context.Context, tenantID, id string) (*model.AuditEntry, error) {
	query := `SELECT id, tenant_id, user_id, user_email, service, action, severity,
		resource_type, resource_id, old_value, new_value, ip_address,
		user_agent, metadata, event_id, correlation_id, previous_hash,
		entry_hash, created_at
		FROM audit_logs WHERE id = $1 AND tenant_id = $2`

	var e model.AuditEntry
	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&e.ID, &e.TenantID, &e.UserID, &e.UserEmail, &e.Service,
		&e.Action, &e.Severity, &e.ResourceType, &e.ResourceID,
		&e.OldValue, &e.NewValue, &e.IPAddress, &e.UserAgent,
		&e.Metadata, &e.EventID, &e.CorrelationID, &e.PreviousHash,
		&e.EntryHash, &e.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("find audit entry by id: %w", err)
	}
	return &e, nil
}

// QueryFilter holds parameterized query filter fields.
type QueryFilter struct {
	TenantID     string
	UserID       string
	Service      string
	Action       string
	ResourceType string
	ResourceID   string
	DateFrom     time.Time
	DateTo       time.Time
	Search       string
	Severity     string
	Sort         string
	Order        string
	Limit        int
	Offset       int
}

// Query executes a parameterized query against the audit_logs table.
func (r *AuditRepository) Query(ctx context.Context, f QueryFilter) ([]model.AuditEntry, int, error) {
	whereClause, args := r.buildWhereClause(f)

	// Count query
	countQuery := "SELECT COUNT(*) FROM audit_logs " + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit entries: %w", err)
	}

	if total == 0 {
		return []model.AuditEntry{}, 0, nil
	}

	// Data query
	orderClause := fmt.Sprintf(" ORDER BY %s %s", f.Sort, f.Order)
	limitClause := fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)

	dataQuery := `SELECT id, tenant_id, user_id, user_email, service, action, severity,
		resource_type, resource_id, old_value, new_value, ip_address,
		user_agent, metadata, event_id, correlation_id, previous_hash,
		entry_hash, created_at
		FROM audit_logs ` + whereClause + orderClause + limitClause

	rows, err := r.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit entries: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.UserID, &e.UserEmail, &e.Service,
			&e.Action, &e.Severity, &e.ResourceType, &e.ResourceID,
			&e.OldValue, &e.NewValue, &e.IPAddress, &e.UserAgent,
			&e.Metadata, &e.EventID, &e.CorrelationID, &e.PreviousHash,
			&e.EntryHash, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating audit entries: %w", err)
	}

	return entries, total, nil
}

// StreamByTenant streams audit entries for a tenant ordered by created_at ASC
// within the given time range. Used for hash chain verification.
func (r *AuditRepository) StreamByTenant(ctx context.Context, tenantID string, startTime, endTime time.Time, fn func(entry *model.AuditEntry) error) error {
	query := `SELECT id, tenant_id, user_id, user_email, service, action, severity,
		resource_type, resource_id, old_value, new_value, ip_address,
		user_agent, metadata, event_id, correlation_id, previous_hash,
		entry_hash, created_at
		FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, tenantID, startTime, endTime)
	if err != nil {
		return fmt.Errorf("streaming audit entries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.UserID, &e.UserEmail, &e.Service,
			&e.Action, &e.Severity, &e.ResourceType, &e.ResourceID,
			&e.OldValue, &e.NewValue, &e.IPAddress, &e.UserAgent,
			&e.Metadata, &e.EventID, &e.CorrelationID, &e.PreviousHash,
			&e.EntryHash, &e.CreatedAt,
		); err != nil {
			return fmt.Errorf("scan audit entry during stream: %w", err)
		}
		if err := fn(&e); err != nil {
			return err
		}
	}
	return rows.Err()
}

// GetStats returns aggregated statistics for a tenant within a date range.
// The returned AuditStats struct is aligned with the frontend AuditLogStats interface.
func (r *AuditRepository) GetStats(ctx context.Context, tenantID string, dateFrom, dateTo time.Time) (*model.AuditStats, error) {
	stats := &model.AuditStats{
		ByService:    []model.AuditGroupStat{},
		ByAction:     []model.AuditGroupStat{},
		BySeverity:   []model.AuditGroupStat{},
		ByHour:       []model.AuditTimeseriesStat{},
		ByDay:        []model.AuditTimeseriesStat{},
		TopUsers:     []model.AuditUserStat{},
		TopResources: []model.AuditResourceStat{},
	}

	// Total events in range
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz`,
		tenantID, dateFrom, dateTo,
	).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("count total events: %w", err)
	}

	// Events today (UTC calendar day)
	todayStart := time.Now().UTC().Truncate(24 * time.Hour)
	if err = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz`,
		tenantID, todayStart,
	).Scan(&stats.EventsToday); err != nil {
		return nil, fmt.Errorf("count events today: %w", err)
	}

	// Unique users in range
	if err = r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT user_id) FROM audit_logs WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz AND user_id IS NOT NULL`,
		tenantID, dateFrom, dateTo,
	).Scan(&stats.UniqueUsers); err != nil {
		return nil, fmt.Errorf("count unique users: %w", err)
	}

	// Unique services in range
	if err = r.db.QueryRow(ctx,
		`SELECT COUNT(DISTINCT service) FROM audit_logs WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz`,
		tenantID, dateFrom, dateTo,
	).Scan(&stats.UniqueServices); err != nil {
		return nil, fmt.Errorf("count unique services: %w", err)
	}

	// By service (top 20)
	svcRows, err := r.db.Query(ctx,
		`SELECT service, COUNT(*) as cnt FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		GROUP BY service ORDER BY cnt DESC LIMIT 20`,
		tenantID, dateFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query by_service: %w", err)
	}
	defer svcRows.Close()
	for svcRows.Next() {
		var s model.AuditGroupStat
		if err := svcRows.Scan(&s.Key, &s.Count); err != nil {
			return nil, fmt.Errorf("scan by_service: %w", err)
		}
		stats.ByService = append(stats.ByService, s)
	}
	svcRows.Close()
	auditComputePercentages(stats.ByService, stats.TotalEvents)

	// By action (top 20)
	actRows, err := r.db.Query(ctx,
		`SELECT action, COUNT(*) as cnt FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		GROUP BY action ORDER BY cnt DESC LIMIT 20`,
		tenantID, dateFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query by_action: %w", err)
	}
	defer actRows.Close()
	for actRows.Next() {
		var s model.AuditGroupStat
		if err := actRows.Scan(&s.Key, &s.Count); err != nil {
			return nil, fmt.Errorf("scan by_action: %w", err)
		}
		stats.ByAction = append(stats.ByAction, s)
	}
	actRows.Close()
	auditComputePercentages(stats.ByAction, stats.TotalEvents)

	// By severity
	sevRows, err := r.db.Query(ctx,
		`SELECT severity, COUNT(*) as cnt FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		GROUP BY severity ORDER BY cnt DESC`,
		tenantID, dateFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query by_severity: %w", err)
	}
	defer sevRows.Close()
	for sevRows.Next() {
		var s model.AuditGroupStat
		if err := sevRows.Scan(&s.Key, &s.Count); err != nil {
			return nil, fmt.Errorf("scan by_severity: %w", err)
		}
		stats.BySeverity = append(stats.BySeverity, s)
	}
	sevRows.Close()
	auditComputePercentages(stats.BySeverity, stats.TotalEvents)

	// By hour (last 24h from dateTo)
	hourFrom := dateTo.Add(-24 * time.Hour)
	hourRows, err := r.db.Query(ctx,
		`SELECT date_trunc('hour', created_at) as hr, COUNT(*) as cnt FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		GROUP BY hr ORDER BY hr`,
		tenantID, hourFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query by_hour: %w", err)
	}
	defer hourRows.Close()
	for hourRows.Next() {
		var ts time.Time
		var cnt int64
		if err := hourRows.Scan(&ts, &cnt); err != nil {
			return nil, fmt.Errorf("scan by_hour: %w", err)
		}
		stats.ByHour = append(stats.ByHour, model.AuditTimeseriesStat{
			Timestamp: ts.UTC().Format(time.RFC3339),
			Count:     cnt,
		})
	}
	hourRows.Close()

	// By day (full range)
	dayRows, err := r.db.Query(ctx,
		`SELECT DATE(created_at) as day, COUNT(*) as cnt FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		GROUP BY day ORDER BY day`,
		tenantID, dateFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query by_day: %w", err)
	}
	defer dayRows.Close()
	for dayRows.Next() {
		var day time.Time
		var cnt int64
		if err := dayRows.Scan(&day, &cnt); err != nil {
			return nil, fmt.Errorf("scan by_day: %w", err)
		}
		stats.ByDay = append(stats.ByDay, model.AuditTimeseriesStat{
			Timestamp: day.Format("2006-01-02"),
			Count:     cnt,
		})
	}
	dayRows.Close()

	// Top users (top 10 by event count)
	userRows, err := r.db.Query(ctx,
		`SELECT COALESCE(user_id::text, ''), user_email, COUNT(*) as cnt, MAX(created_at) as last_at
		FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz AND user_id IS NOT NULL
		GROUP BY user_id, user_email ORDER BY cnt DESC LIMIT 10`,
		tenantID, dateFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query top_users: %w", err)
	}
	defer userRows.Close()
	for userRows.Next() {
		var uid, email string
		var cnt int64
		var lastAt time.Time
		if err := userRows.Scan(&uid, &email, &cnt, &lastAt); err != nil {
			return nil, fmt.Errorf("scan top_users: %w", err)
		}
		stats.TopUsers = append(stats.TopUsers, model.AuditUserStat{
			UserID:      uid,
			UserName:    auditDeriveUserName(email),
			UserEmail:   email,
			EventCount:  cnt,
			LastEventAt: lastAt.UTC().Format(time.RFC3339),
		})
	}
	userRows.Close()

	// Top resources (top 10)
	resRows, err := r.db.Query(ctx,
		`SELECT resource_type, resource_id, COUNT(*) as cnt FROM audit_logs
		WHERE tenant_id = $1::uuid AND created_at >= $2::timestamptz AND created_at <= $3::timestamptz
		GROUP BY resource_type, resource_id ORDER BY cnt DESC LIMIT 10`,
		tenantID, dateFrom, dateTo,
	)
	if err != nil {
		return nil, fmt.Errorf("query top_resources: %w", err)
	}
	defer resRows.Close()
	for resRows.Next() {
		var rt, rid string
		var cnt int64
		if err := resRows.Scan(&rt, &rid, &cnt); err != nil {
			return nil, fmt.Errorf("scan top_resources: %w", err)
		}
		stats.TopResources = append(stats.TopResources, model.AuditResourceStat{
			ResourceType: rt,
			ResourceID:   rid,
			ResourceName: rid, // no separate name stored; resource_id serves as display value
			EventCount:   cnt,
		})
	}
	resRows.Close()

	return stats, nil
}

// auditComputePercentages fills the Percentage field on each stat relative to total.
func auditComputePercentages(stats []model.AuditGroupStat, total int64) {
	if total == 0 {
		return
	}
	for i := range stats {
		stats[i].Percentage = float64(stats[i].Count) / float64(total) * 100
	}
}

// auditDeriveUserName extracts a display name from an email address.
// e.g. "john.doe@acme.com" → "John Doe"
func auditDeriveUserName(email string) string {
	if email == "" {
		return ""
	}
	for i, c := range email {
		if c == '@' {
			local := email[:i]
			result := make([]byte, 0, len(local))
			capitalize := true
			for _, b := range []byte(local) {
				if b == '.' || b == '_' || b == '-' {
					result = append(result, ' ')
					capitalize = true
				} else if capitalize {
					if b >= 'a' && b <= 'z' {
						b -= 32
					}
					result = append(result, b)
					capitalize = false
				} else {
					result = append(result, b)
				}
			}
			return string(result)
		}
	}
	return email
}

// GetTimeline returns audit entries for a specific resource ordered by time.
func (r *AuditRepository) GetTimeline(ctx context.Context, tenantID, resourceID string, limit, offset int) ([]model.AuditEntry, int, error) {
	where := "WHERE tenant_id = $1 AND resource_id = $2"
	args := []interface{}{tenantID, resourceID}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count timeline entries: %w", err)
	}

	query := fmt.Sprintf(`SELECT id, tenant_id, user_id, user_email, service, action, severity,
		resource_type, resource_id, old_value, new_value, ip_address,
		user_agent, metadata, event_id, correlation_id, previous_hash,
		entry_hash, created_at
		FROM audit_logs %s ORDER BY created_at DESC LIMIT %d OFFSET %d`, where, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query timeline: %w", err)
	}
	defer rows.Close()

	var entries []model.AuditEntry
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.UserID, &e.UserEmail, &e.Service,
			&e.Action, &e.Severity, &e.ResourceType, &e.ResourceID,
			&e.OldValue, &e.NewValue, &e.IPAddress, &e.UserAgent,
			&e.Metadata, &e.EventID, &e.CorrelationID, &e.PreviousHash,
			&e.EntryHash, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan timeline entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

// StreamForExport streams entries matching the filter to the callback.
func (r *AuditRepository) StreamForExport(ctx context.Context, f QueryFilter, fn func(entry *model.AuditEntry) error) (int64, error) {
	whereClause, args := r.buildWhereClause(f)

	query := `SELECT id, tenant_id, user_id, user_email, service, action, severity,
		resource_type, resource_id, old_value, new_value, ip_address,
		user_agent, metadata, event_id, correlation_id, previous_hash,
		entry_hash, created_at
		FROM audit_logs ` + whereClause + ` ORDER BY created_at ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("stream for export: %w", err)
	}
	defer rows.Close()

	var count int64
	for rows.Next() {
		var e model.AuditEntry
		if err := rows.Scan(
			&e.ID, &e.TenantID, &e.UserID, &e.UserEmail, &e.Service,
			&e.Action, &e.Severity, &e.ResourceType, &e.ResourceID,
			&e.OldValue, &e.NewValue, &e.IPAddress, &e.UserAgent,
			&e.Metadata, &e.EventID, &e.CorrelationID, &e.PreviousHash,
			&e.EntryHash, &e.CreatedAt,
		); err != nil {
			return count, fmt.Errorf("scan export entry: %w", err)
		}
		if err := fn(&e); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}

// CountForExport returns the number of records matching the export filter.
func (r *AuditRepository) CountForExport(ctx context.Context, f QueryFilter) (int64, error) {
	whereClause, args := r.buildWhereClause(f)
	var count int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM audit_logs "+whereClause, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count for export: %w", err)
	}
	return count, nil
}

// GetChainState retrieves the last hash chain state for a tenant.
func (r *AuditRepository) GetChainState(ctx context.Context, tenantID string) (*model.ChainState, error) {
	var cs model.ChainState
	err := r.db.QueryRow(ctx,
		`SELECT tenant_id, last_entry_id, last_hash, last_created_at, updated_at
		FROM audit_chain_state WHERE tenant_id = $1`, tenantID,
	).Scan(&cs.TenantID, &cs.LastEntryID, &cs.LastHash, &cs.LastCreated, &cs.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get chain state: %w", err)
	}
	return &cs, nil
}

// UpsertChainState updates or inserts the chain state for a tenant.
func (r *AuditRepository) UpsertChainState(ctx context.Context, cs *model.ChainState) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO audit_chain_state (tenant_id, last_entry_id, last_hash, last_created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (tenant_id)
		DO UPDATE SET last_entry_id = $2, last_hash = $3, last_created_at = $4, updated_at = NOW()`,
		cs.TenantID, cs.LastEntryID, cs.LastHash, cs.LastCreated,
	)
	if err != nil {
		return fmt.Errorf("upsert chain state: %w", err)
	}
	return nil
}

// GetLastEntryHash gets the hash of the last audit entry for a tenant.
func (r *AuditRepository) GetLastEntryHash(ctx context.Context, tenantID string) (string, string, error) {
	var entryID, entryHash string
	err := r.db.QueryRow(ctx,
		`SELECT id, entry_hash FROM audit_logs
		WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT 1`, tenantID,
	).Scan(&entryID, &entryHash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", nil
		}
		return "", "", fmt.Errorf("get last entry hash: %w", err)
	}
	return entryID, entryHash, nil
}

// buildWhereClause constructs a parameterized WHERE clause from the filter.
func (r *AuditRepository) buildWhereClause(f QueryFilter) (string, []interface{}) {
	conditions := []string{"tenant_id = $1", "created_at >= $2", "created_at <= $3"}
	args := []interface{}{f.TenantID, f.DateFrom, f.DateTo}
	paramIdx := 4

	if f.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", paramIdx))
		args = append(args, f.UserID)
		paramIdx++
	}
	if f.Service != "" {
		conditions = append(conditions, fmt.Sprintf("service = $%d", paramIdx))
		args = append(args, f.Service)
		paramIdx++
	}
	if f.Action != "" {
		if strings.HasSuffix(f.Action, "*") {
			prefix := strings.TrimSuffix(f.Action, "*")
			conditions = append(conditions, fmt.Sprintf("action LIKE $%d", paramIdx))
			args = append(args, prefix+"%")
		} else {
			conditions = append(conditions, fmt.Sprintf("action = $%d", paramIdx))
			args = append(args, f.Action)
		}
		paramIdx++
	}
	if f.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", paramIdx))
		args = append(args, f.ResourceType)
		paramIdx++
	}
	if f.ResourceID != "" {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", paramIdx))
		args = append(args, f.ResourceID)
		paramIdx++
	}
	if f.Severity != "" {
		conditions = append(conditions, fmt.Sprintf("severity = $%d", paramIdx))
		args = append(args, f.Severity)
		paramIdx++
	}
	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"to_tsvector('english', coalesce(action,'') || ' ' || coalesce(resource_type,'') || ' ' || coalesce(user_email,'')) @@ plainto_tsquery('english', $%d)", paramIdx))
		args = append(args, f.Search)
		paramIdx++
	}

	return "WHERE " + strings.Join(conditions, " AND "), args
}

// nullableJSON returns nil for empty JSON messages so pgx stores NULL.
func nullableJSON(data json.RawMessage) interface{} {
	if len(data) == 0 {
		return nil
	}
	return []byte(data)
}
