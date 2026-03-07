package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) InsertComplianceChecks(ctx context.Context, q DBTX, checks []model.ComplianceCheck) error {
	for _, check := range checks {
		evidence, err := marshalJSON(check.Evidence)
		if err != nil {
			return fmt.Errorf("marshal compliance evidence: %w", err)
		}
		_, err = q.Exec(ctx, `
			INSERT INTO compliance_checks (
				id, tenant_id, committee_id, check_type, check_name, status,
				severity, description, finding, recommendation, evidence,
				period_start, period_end, checked_at, checked_by, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11,
				$12, $13, $14, $15, $16
			)`,
			check.ID,
			check.TenantID,
			nullableUUID(check.CommitteeID),
			check.CheckType,
			check.CheckName,
			check.Status,
			check.Severity,
			check.Description,
			check.Finding,
			check.Recommendation,
			evidence,
			check.PeriodStart,
			check.PeriodEnd,
			check.CheckedAt,
			check.CheckedBy,
			check.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert compliance check: %w", err)
		}
	}
	return nil
}

func (s *Store) ListComplianceChecks(ctx context.Context, tenantID uuid.UUID, filters model.ComplianceFilters) ([]model.ComplianceCheck, int, error) {
	offset := (filters.Page - 1) * filters.PerPage
	where := []string{"tenant_id = $1"}
	args := []any{tenantID}
	argPos := 2
	if filters.CommitteeID != nil {
		where = append(where, fmt.Sprintf("committee_id = $%d", argPos))
		args = append(args, *filters.CommitteeID)
		argPos++
	}
	if filters.CheckType != nil {
		where = append(where, fmt.Sprintf("check_type = $%d", argPos))
		args = append(args, string(*filters.CheckType))
		argPos++
	}
	if len(filters.Statuses) > 0 {
		statuses := make([]string, 0, len(filters.Statuses))
		for _, status := range filters.Statuses {
			statuses = append(statuses, string(status))
		}
		where = append(where, fmt.Sprintf("status = ANY($%d)", argPos))
		args = append(args, statuses)
		argPos++
	}
	if filters.DateFrom != nil {
		where = append(where, fmt.Sprintf("checked_at >= $%d", argPos))
		args = append(args, *filters.DateFrom)
		argPos++
	}
	if filters.DateTo != nil {
		where = append(where, fmt.Sprintf("checked_at <= $%d", argPos))
		args = append(args, *filters.DateTo)
		argPos++
	}
	whereClause := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM compliance_checks WHERE "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count compliance checks: %w", err)
	}
	query := fmt.Sprintf(`
		SELECT id, tenant_id, committee_id, check_type, check_name, status,
		       severity, description, finding, recommendation, evidence,
		       period_start, period_end, checked_at, checked_by, created_at
		FROM compliance_checks
		WHERE %s
		ORDER BY checked_at DESC, created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1,
	)
	args = append(args, filters.PerPage, offset)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list compliance checks: %w", err)
	}
	defer rows.Close()
	out := make([]model.ComplianceCheck, 0, filters.PerPage)
	for rows.Next() {
		item, err := scanComplianceCheck(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *item)
	}
	return out, total, rows.Err()
}

func (s *Store) LatestComplianceChecksByCommittee(ctx context.Context, tenantID uuid.UUID) ([]model.ComplianceCheck, error) {
	rows, err := s.db.Query(ctx, `
		SELECT DISTINCT ON (committee_id, check_type)
		       id, tenant_id, committee_id, check_type, check_name, status,
		       severity, description, finding, recommendation, evidence,
		       period_start, period_end, checked_at, checked_by, created_at
		FROM compliance_checks
		WHERE tenant_id = $1 AND committee_id IS NOT NULL
		ORDER BY committee_id, check_type, checked_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("latest compliance checks by committee: %w", err)
	}
	defer rows.Close()
	out := make([]model.ComplianceCheck, 0)
	for rows.Next() {
		item, err := scanComplianceCheck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func scanComplianceCheck(scanner rowScanner) (*model.ComplianceCheck, error) {
	var (
		item           model.ComplianceCheck
		committeeID    *uuid.UUID
		finding        *string
		recommendation *string
		evidenceRaw    []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&committeeID,
		&item.CheckType,
		&item.CheckName,
		&item.Status,
		&item.Severity,
		&item.Description,
		&finding,
		&recommendation,
		&evidenceRaw,
		&item.PeriodStart,
		&item.PeriodEnd,
		&item.CheckedAt,
		&item.CheckedBy,
		&item.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan compliance check: %w", err)
	}
	item.CommitteeID = committeeID
	item.Finding = finding
	item.Recommendation = recommendation
	evidence, err := decodeJSONMap(evidenceRaw)
	if err != nil {
		return nil, fmt.Errorf("decode compliance evidence: %w", err)
	}
	item.Evidence = evidence
	return &item, nil
}
