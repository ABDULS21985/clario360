package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// DBQuerier is the minimal interface for database query operations.
// Satisfied by *pgxpool.Pool and pgxmock.PgxPoolIface for testing.
type DBQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

// allowedTables is a whitelist of tables that can be queried for ownership verification.
var allowedTables = map[string]bool{
	"assets": true, "vulnerabilities": true, "alerts": true,
	"threats": true, "threat_indicators": true, "detection_rules": true,
	"ctem_assessments": true, "ctem_findings": true, "ctem_remediation_groups": true,
	"remediation_actions": true, "dspm_data_assets": true, "dspm_scans": true,
	"vciso_briefings": true, "asset_relationships": true, "security_events": true,
	"users": true, "roles": true, "audit_logs": true,
	"workflow_definitions": true, "workflow_instances": true, "human_tasks": true,
	"notifications": true,
}

// forbiddenFields are fields that must never be accepted from user input.
var forbiddenFields = map[string]bool{
	"id": true, "tenant_id": true,
	"created_at": true, "updated_at": true, "deleted_at": true,
	"created_by": true, "updated_by": true,
	"password_hash": true, "mfa_secret": true, "mfa_recovery_codes": true,
	"refresh_token_hash": true, "api_key_hash": true,
	"is_superadmin": true, "is_system": true,
}

// APISecurityChecker provides BOLA, BFLA, and mass assignment prevention.
type APISecurityChecker struct {
	pool    DBQuerier
	metrics *Metrics
	logger  zerolog.Logger
}

// NewAPISecurityChecker creates a new API security checker using a pgxpool.Pool.
func NewAPISecurityChecker(pool *pgxpool.Pool, metrics *Metrics, logger zerolog.Logger) *APISecurityChecker {
	return &APISecurityChecker{
		pool:    pool,
		metrics: metrics,
		logger:  logger.With().Str("component", "api_security").Logger(),
	}
}

// NewAPISecurityCheckerWithPool creates a new API security checker using the DBQuerier
// interface. This allows pgxmock or other database mock implementations for testing.
func NewAPISecurityCheckerWithPool(pool DBQuerier, metrics *Metrics, logger zerolog.Logger) *APISecurityChecker {
	return &APISecurityChecker{
		pool:    pool,
		metrics: metrics,
		logger:  logger.With().Str("component", "api_security").Logger(),
	}
}

// VerifyOwnership verifies that the given resource belongs to the current tenant.
// Returns 404 (not 403) to prevent information disclosure about other tenants' resources.
func (c *APISecurityChecker) VerifyOwnership(ctx context.Context, tableName string, resourceID uuid.UUID) error {
	if !allowedTables[tableName] {
		c.logger.Warn().
			Str("table", tableName).
			Msg("ownership check on non-whitelisted table")
		if c.metrics != nil {
			c.metrics.BOLAAttempts.Inc()
		}
		return ErrInvalidTable
	}

	tenantID := auth.TenantFromContext(ctx)
	if tenantID == "" {
		return ErrForbidden
	}

	parsedTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		return ErrForbidden
	}

	if c.pool == nil {
		c.logger.Error().Msg("database pool is nil — cannot verify ownership")
		return ErrNotFound
	}

	// Use parameterized query with table name from whitelist only
	query := fmt.Sprintf(
		`SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL)`,
		sanitizeTableName(tableName),
	)

	var exists bool
	err = c.pool.QueryRow(ctx, query, resourceID, parsedTenantID).Scan(&exists)
	if err != nil {
		c.logger.Error().Err(err).
			Str("table", tableName).
			Str("resource_id", resourceID.String()).
			Msg("ownership verification query failed")
		return ErrNotFound // Don't expose internal errors
	}

	if !exists {
		if c.metrics != nil {
			c.metrics.BOLAAttempts.Inc()
		}
		return ErrNotFound // Return 404, NOT 403
	}

	return nil
}

// PreventMassAssignment validates that request body only contains allowed fields.
func PreventMassAssignment(allowedFields []string, requestBody map[string]interface{}, metrics *Metrics) error {
	allowed := make(map[string]bool, len(allowedFields))
	for _, f := range allowedFields {
		allowed[strings.ToLower(f)] = true
	}

	for key := range requestBody {
		normalized := strings.ToLower(strings.TrimSpace(key))

		if forbiddenFields[normalized] {
			if metrics != nil {
				metrics.MassAssignment.Inc()
			}
			return fmt.Errorf("%w: %q", ErrForbiddenField, key)
		}

		if !allowed[normalized] {
			if metrics != nil {
				metrics.MassAssignment.Inc()
			}
			return fmt.Errorf("%w: %q", ErrUnknownField, key)
		}
	}

	return nil
}

// EnforceRole verifies the authenticated user has at least one of the required roles.
func EnforceRole(ctx context.Context, requiredRoles ...string) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return ErrForbidden
	}

	for _, required := range requiredRoles {
		for _, role := range user.Roles {
			if role == required || role == "super_admin" {
				return nil
			}
		}
	}

	return ErrForbidden
}

// EnforceResourceOwner verifies the current user is the resource creator or an admin.
func EnforceResourceOwner(ctx context.Context, resourceCreatedBy string) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return ErrForbidden
	}

	// Admins can always access
	for _, role := range user.Roles {
		if role == "super_admin" || role == "tenant_admin" {
			return nil
		}
	}

	if user.ID == resourceCreatedBy {
		return nil
	}

	return ErrForbidden
}

// EnforceApprovalAuthority verifies the current user can approve a remediation action.
// The approver cannot be the same person who submitted the action (segregation of duties).
func EnforceApprovalAuthority(ctx context.Context, submittedBy string, requiredRole string) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return ErrForbidden
	}

	// Segregation of duties: submitter cannot approve
	if user.ID == submittedBy {
		return fmt.Errorf("%w: submitter cannot approve own action", ErrForbidden)
	}

	// Check role
	if err := EnforceRole(ctx, requiredRole, "super_admin"); err != nil {
		return err
	}

	return nil
}

// ValidateUUID validates that a string is a valid UUID v4.
func ValidateUUID(input string) error {
	if _, err := uuid.Parse(input); err != nil {
		return ErrInvalidUUID
	}
	return nil
}

// sanitizeTableName ensures the table name is safe for SQL interpolation.
// Only called with whitelisted names, but added as defence-in-depth.
func sanitizeTableName(name string) string {
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
