package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// ExceptionRepository handles persistence for DSPM risk exceptions.
type ExceptionRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewExceptionRepository creates a new ExceptionRepository.
func NewExceptionRepository(db *pgxpool.Pool, logger zerolog.Logger) *ExceptionRepository {
	return &ExceptionRepository{db: db, logger: logger}
}

// exceptionColumns is the shared column list for SELECT queries on dspm_risk_exceptions.
const exceptionColumns = `id, tenant_id, exception_type, remediation_id, data_asset_id, policy_id,
	justification, business_reason, compensating_controls, risk_score, risk_level,
	requested_by, approved_by, approval_status, approved_at, rejection_reason,
	expires_at, review_interval_days, next_review_at, last_reviewed_at, review_count,
	status, created_at, updated_at`

// Create inserts a new risk exception and returns it with server-generated fields.
func (r *ExceptionRepository) Create(ctx context.Context, exception *model.RiskException) (*model.RiskException, error) {
	if exception.ID == uuid.Nil {
		exception.ID = uuid.New()
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO dspm_risk_exceptions (
			id, tenant_id, exception_type, remediation_id, data_asset_id, policy_id,
			justification, business_reason, compensating_controls, risk_score, risk_level,
			requested_by, approved_by, approval_status, approved_at, rejection_reason,
			expires_at, review_interval_days, next_review_at, last_reviewed_at, review_count,
			status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21,
			$22, now(), now()
		)
		RETURNING `+exceptionColumns,
		exception.ID, exception.TenantID, exception.ExceptionType,
		exception.RemediationID, exception.DataAssetID, exception.PolicyID,
		exception.Justification, exception.BusinessReason, exception.CompensatingControls,
		exception.RiskScore, exception.RiskLevel,
		exception.RequestedBy, exception.ApprovedBy, exception.ApprovalStatus,
		exception.ApprovedAt, exception.RejectionReason,
		exception.ExpiresAt, exception.ReviewIntervalDays, exception.NextReviewAt,
		exception.LastReviewedAt, exception.ReviewCount,
		exception.Status,
	)

	result, err := scanException(row)
	if err != nil {
		return nil, fmt.Errorf("create risk exception: %w", err)
	}
	return result, nil
}

// GetByID fetches a single risk exception by ID with tenant isolation.
func (r *ExceptionRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*model.RiskException, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+exceptionColumns+` FROM dspm_risk_exceptions WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)

	result, err := scanException(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("risk exception not found")
		}
		return nil, fmt.Errorf("get risk exception: %w", err)
	}
	return result, nil
}

// Update updates all mutable fields of a risk exception.
func (r *ExceptionRepository) Update(ctx context.Context, exception *model.RiskException) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE dspm_risk_exceptions
		SET
			exception_type = $3,
			remediation_id = $4,
			data_asset_id = $5,
			policy_id = $6,
			justification = $7,
			business_reason = $8,
			compensating_controls = $9,
			risk_score = $10,
			risk_level = $11,
			requested_by = $12,
			approved_by = $13,
			approval_status = $14,
			approved_at = $15,
			rejection_reason = $16,
			expires_at = $17,
			review_interval_days = $18,
			next_review_at = $19,
			last_reviewed_at = $20,
			review_count = $21,
			status = $22,
			updated_at = now()
		WHERE tenant_id = $1 AND id = $2`,
		exception.TenantID, exception.ID,
		exception.ExceptionType, exception.RemediationID,
		exception.DataAssetID, exception.PolicyID,
		exception.Justification, exception.BusinessReason,
		exception.CompensatingControls, exception.RiskScore,
		exception.RiskLevel, exception.RequestedBy,
		exception.ApprovedBy, exception.ApprovalStatus,
		exception.ApprovedAt, exception.RejectionReason,
		exception.ExpiresAt, exception.ReviewIntervalDays,
		exception.NextReviewAt, exception.LastReviewedAt,
		exception.ReviewCount, exception.Status,
	)
	if err != nil {
		return fmt.Errorf("update risk exception: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("risk exception not found")
	}
	return nil
}

// List returns paginated risk exceptions with filtering.
func (r *ExceptionRepository) List(ctx context.Context, tenantID uuid.UUID, params *dto.ExceptionListParams) ([]model.RiskException, int, error) {
	params.SetDefaults()

	var (
		conditions []string
		args       []interface{}
		argIdx     int
	)

	nextArg := func(val interface{}) string {
		argIdx++
		args = append(args, val)
		return fmt.Sprintf("$%d", argIdx)
	}

	conditions = append(conditions, "tenant_id = "+nextArg(tenantID))

	if len(params.Status) > 0 {
		placeholders := make([]string, len(params.Status))
		for i, v := range params.Status {
			placeholders[i] = nextArg(v)
		}
		conditions = append(conditions, "status IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(params.ApprovalStatus) > 0 {
		placeholders := make([]string, len(params.ApprovalStatus))
		for i, v := range params.ApprovalStatus {
			placeholders[i] = nextArg(v)
		}
		conditions = append(conditions, "approval_status IN ("+strings.Join(placeholders, ", ")+")")
	}

	if len(params.ExceptionType) > 0 {
		placeholders := make([]string, len(params.ExceptionType))
		for i, v := range params.ExceptionType {
			placeholders[i] = nextArg(v)
		}
		conditions = append(conditions, "exception_type IN ("+strings.Join(placeholders, ", ")+")")
	}

	if params.AssetID != nil {
		conditions = append(conditions, "data_asset_id = "+nextArg(*params.AssetID))
	}

	if strings.TrimSpace(params.Search) != "" {
		search := "%" + strings.TrimSpace(params.Search) + "%"
		conditions = append(conditions, "(justification ILIKE "+nextArg(search)+" OR business_reason ILIKE "+nextArg(search)+")")
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query.
	var total int
	countSQL := "SELECT COUNT(*) FROM dspm_risk_exceptions WHERE " + whereClause
	if err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count risk exceptions: %w", err)
	}

	// Data query with pagination.
	offset := (params.Page - 1) * params.PerPage
	dataSQL := `SELECT ` + exceptionColumns + ` FROM dspm_risk_exceptions WHERE ` + whereClause +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT %s OFFSET %s", nextArg(params.PerPage), nextArg(offset))

	rows, err := r.db.Query(ctx, dataSQL, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list risk exceptions: %w", err)
	}
	defer rows.Close()

	items := make([]model.RiskException, 0)
	for rows.Next() {
		item, err := scanException(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan exception row: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate risk exceptions: %w", err)
	}

	return items, total, nil
}

// ListByTenant returns all risk exceptions for a tenant (used by the exception checker).
func (r *ExceptionRepository) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+exceptionColumns+` FROM dspm_risk_exceptions WHERE tenant_id = $1 ORDER BY created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list all risk exceptions: %w", err)
	}
	defer rows.Close()

	items := make([]model.RiskException, 0)
	for rows.Next() {
		item, err := scanException(rows)
		if err != nil {
			return nil, fmt.Errorf("scan exception: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all risk exceptions: %w", err)
	}
	return items, nil
}

// HasActiveException checks whether an active, approved exception exists for a given asset and policy combination.
// If policyID is uuid.Nil, only the asset-level exception is checked (policy_id IS NULL).
func (r *ExceptionRepository) HasActiveException(ctx context.Context, tenantID, assetID, policyID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM dspm_risk_exceptions
			WHERE tenant_id = $1
				AND status = 'active'
				AND approval_status = 'approved'
				AND data_asset_id = $2
				AND (policy_id = $3 OR policy_id IS NULL)
		)`,
		tenantID, assetID, policyID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active exception: %w", err)
	}
	return exists, nil
}

// FindExpired returns active exceptions whose expiry date has passed.
func (r *ExceptionRepository) FindExpired(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+exceptionColumns+`
		FROM dspm_risk_exceptions
		WHERE tenant_id = $1
			AND expires_at <= now()
			AND status = 'active'`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("find expired exceptions: %w", err)
	}
	defer rows.Close()

	items := make([]model.RiskException, 0)
	for rows.Next() {
		item, err := scanException(rows)
		if err != nil {
			return nil, fmt.Errorf("scan expired exception: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired exceptions: %w", err)
	}
	return items, nil
}

// FindNeedingReview returns active exceptions whose next review date has passed.
func (r *ExceptionRepository) FindNeedingReview(ctx context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	rows, err := r.db.Query(ctx, `
		SELECT `+exceptionColumns+`
		FROM dspm_risk_exceptions
		WHERE tenant_id = $1
			AND next_review_at <= now()
			AND status = 'active'`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("find exceptions needing review: %w", err)
	}
	defer rows.Close()

	items := make([]model.RiskException, 0)
	for rows.Next() {
		item, err := scanException(rows)
		if err != nil {
			return nil, fmt.Errorf("scan review-due exception: %w", err)
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review-due exceptions: %w", err)
	}
	return items, nil
}

// scanException scans a single exception row into a model.RiskException.
func scanException(row scanner) (*model.RiskException, error) {
	var item model.RiskException
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.ExceptionType,
		&item.RemediationID,
		&item.DataAssetID,
		&item.PolicyID,
		&item.Justification,
		&item.BusinessReason,
		&item.CompensatingControls,
		&item.RiskScore,
		&item.RiskLevel,
		&item.RequestedBy,
		&item.ApprovedBy,
		&item.ApprovalStatus,
		&item.ApprovedAt,
		&item.RejectionReason,
		&item.ExpiresAt,
		&item.ReviewIntervalDays,
		&item.NextReviewAt,
		&item.LastReviewedAt,
		&item.ReviewCount,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}
