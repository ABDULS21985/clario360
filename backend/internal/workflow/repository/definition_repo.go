package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/workflow/model"
)

// DefinitionRepository handles all database operations for workflow definitions.
type DefinitionRepository struct {
	pool *pgxpool.Pool
}

// NewDefinitionRepository creates a new DefinitionRepository backed by the
// provided connection pool.
func NewDefinitionRepository(pool *pgxpool.Pool) *DefinitionRepository {
	return &DefinitionRepository{pool: pool}
}

// Create inserts a new workflow definition. JSONB fields (trigger_config,
// variables, steps) are marshaled to JSON before insertion. The generated ID
// and timestamps are scanned back into the struct.
func (r *DefinitionRepository) Create(ctx context.Context, def *model.WorkflowDefinition) error {
	triggerJSON, err := json.Marshal(def.TriggerConfig)
	if err != nil {
		return fmt.Errorf("marshaling trigger_config: %w", err)
	}
	variablesJSON, err := json.Marshal(def.Variables)
	if err != nil {
		return fmt.Errorf("marshaling variables: %w", err)
	}
	stepsJSON, err := json.Marshal(def.Steps)
	if err != nil {
		return fmt.Errorf("marshaling steps: %w", err)
	}

	query := `
		INSERT INTO workflow_definitions (
			tenant_id, name, description, version, status,
			trigger_config, variables, steps, created_by, updated_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		def.TenantID,
		def.Name,
		def.Description,
		def.Version,
		def.Status,
		triggerJSON,
		variablesJSON,
		stepsJSON,
		def.CreatedBy,
		nilIfEmpty(def.UpdatedBy),
	).Scan(&def.ID, &def.CreatedAt, &def.UpdatedAt)
}

// GetByID retrieves a definition by ID and tenant. Returns a wrapped
// model.ErrNotFound if the row does not exist or has been soft-deleted.
func (r *DefinitionRepository) GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, deleted_at
		FROM workflow_definitions
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	return r.scanDefinition(ctx, r.pool.QueryRow(ctx, query, id, tenantID))
}

// GetActiveByID retrieves a definition only when its status is 'active' and it
// has not been soft-deleted.
func (r *DefinitionRepository) GetActiveByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, deleted_at
		FROM workflow_definitions
		WHERE id = $1 AND tenant_id = $2 AND status = 'active' AND deleted_at IS NULL`

	return r.scanDefinition(ctx, r.pool.QueryRow(ctx, query, id, tenantID))
}

// GetByIDAndVersion retrieves a specific version of a definition.
func (r *DefinitionRepository) GetByIDAndVersion(ctx context.Context, tenantID, id string, version int) (*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, deleted_at
		FROM workflow_definitions
		WHERE id = $1 AND tenant_id = $2 AND version = $3 AND deleted_at IS NULL`

	return r.scanDefinition(ctx, r.pool.QueryRow(ctx, query, id, tenantID, version))
}

// List returns paginated definitions filtered by tenant, optional status, and
// optional name search (case-insensitive ILIKE). The second return value is
// the total count matching the filters (for pagination metadata).
func (r *DefinitionRepository) List(ctx context.Context, tenantID, status, nameFilter string, limit, offset int) ([]*model.WorkflowDefinition, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if nameFilter != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", argIdx))
		args = append(args, "%"+nameFilter+"%")
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Total count with the same filters.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM workflow_definitions WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting workflow definitions: %w", err)
	}

	if total == 0 {
		return []*model.WorkflowDefinition{}, 0, nil
	}

	// Paginated data query.
	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, deleted_at
		FROM workflow_definitions
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing workflow definitions: %w", err)
	}
	defer rows.Close()

	defs, err := r.scanDefinitions(rows)
	if err != nil {
		return nil, 0, err
	}

	return defs, total, nil
}

// ListVersions returns all versions of definitions that share the same base
// definition ID, ordered by version descending.
func (r *DefinitionRepository) ListVersions(ctx context.Context, tenantID, id string) ([]*model.WorkflowDefinition, error) {
	// First look up the name for this definition ID so we can find all versions.
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM workflow_definitions WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		id, tenantID,
	).Scan(&name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("definition %s: %w", id, model.ErrNotFound)
		}
		return nil, fmt.Errorf("looking up definition name: %w", err)
	}

	query := `
		SELECT id, tenant_id, name, description, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, deleted_at
		FROM workflow_definitions
		WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL
		ORDER BY version DESC`

	rows, err := r.pool.Query(ctx, query, tenantID, name)
	if err != nil {
		return nil, fmt.Errorf("listing definition versions: %w", err)
	}
	defer rows.Close()

	return r.scanDefinitions(rows)
}

// Update updates a definition's mutable fields: description, status, trigger
// config, variables, steps, and updated_by. The ID and tenant_id are used in
// the WHERE clause. Returns model.ErrNotFound when no matching row exists.
func (r *DefinitionRepository) Update(ctx context.Context, def *model.WorkflowDefinition) error {
	triggerJSON, err := json.Marshal(def.TriggerConfig)
	if err != nil {
		return fmt.Errorf("marshaling trigger_config: %w", err)
	}
	variablesJSON, err := json.Marshal(def.Variables)
	if err != nil {
		return fmt.Errorf("marshaling variables: %w", err)
	}
	stepsJSON, err := json.Marshal(def.Steps)
	if err != nil {
		return fmt.Errorf("marshaling steps: %w", err)
	}

	query := `
		UPDATE workflow_definitions
		SET description = $3,
		    status = $4,
		    trigger_config = $5,
		    variables = $6,
		    steps = $7,
		    updated_by = $8,
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	ct, err := r.pool.Exec(ctx, query,
		def.ID,
		def.TenantID,
		def.Description,
		def.Status,
		triggerJSON,
		variablesJSON,
		stepsJSON,
		nilIfEmpty(def.UpdatedBy),
	)
	if err != nil {
		return fmt.Errorf("updating workflow definition: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("definition %s: %w", def.ID, model.ErrNotFound)
	}
	return nil
}

// SoftDelete sets deleted_at on a definition. Returns model.ErrNotFound if the
// definition does not exist or is already deleted.
func (r *DefinitionRepository) SoftDelete(ctx context.Context, tenantID, id string) error {
	query := `
		UPDATE workflow_definitions
		SET deleted_at = now(), updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	ct, err := r.pool.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("soft-deleting workflow definition: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("definition %s: %w", id, model.ErrNotFound)
	}
	return nil
}

// GetMaxVersion returns the highest version number for a definition name
// within a tenant. Returns 0 if no versions exist.
func (r *DefinitionRepository) GetMaxVersion(ctx context.Context, tenantID, name string) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM workflow_definitions
		WHERE tenant_id = $1 AND name = $2 AND deleted_at IS NULL`

	var maxVer int
	err := r.pool.QueryRow(ctx, query, tenantID, name).Scan(&maxVer)
	if err != nil {
		return 0, fmt.Errorf("getting max version for %q: %w", name, err)
	}
	return maxVer, nil
}

// GetActiveByTriggerTopic returns all active, non-deleted definitions whose
// trigger_config contains an event trigger matching the given topic. This
// query uses a JSONB containment operator to match the topic field.
func (r *DefinitionRepository) GetActiveByTriggerTopic(ctx context.Context, topic string) ([]*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, deleted_at
		FROM workflow_definitions
		WHERE status = 'active'
		  AND deleted_at IS NULL
		  AND trigger_config->>'type' = 'event'
		  AND trigger_config->>'topic' = $1
		ORDER BY tenant_id, name`

	rows, err := r.pool.Query(ctx, query, topic)
	if err != nil {
		return nil, fmt.Errorf("querying definitions by trigger topic: %w", err)
	}
	defer rows.Close()

	return r.scanDefinitions(rows)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// scanDefinition scans a single pgx.Row into a WorkflowDefinition, handling
// JSONB deserialization of trigger_config, variables, and steps.
func (r *DefinitionRepository) scanDefinition(_ context.Context, row pgx.Row) (*model.WorkflowDefinition, error) {
	var (
		def             model.WorkflowDefinition
		triggerJSON     []byte
		variablesJSON   []byte
		stepsJSON       []byte
		updatedByNullable *string
	)

	err := row.Scan(
		&def.ID,
		&def.TenantID,
		&def.Name,
		&def.Description,
		&def.Version,
		&def.Status,
		&triggerJSON,
		&variablesJSON,
		&stepsJSON,
		&def.CreatedBy,
		&updatedByNullable,
		&def.CreatedAt,
		&def.UpdatedAt,
		&def.DeletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("definition: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("scanning workflow definition: %w", err)
	}

	if updatedByNullable != nil {
		def.UpdatedBy = *updatedByNullable
	}

	if err := json.Unmarshal(triggerJSON, &def.TriggerConfig); err != nil {
		return nil, fmt.Errorf("unmarshaling trigger_config: %w", err)
	}
	if err := json.Unmarshal(variablesJSON, &def.Variables); err != nil {
		return nil, fmt.Errorf("unmarshaling variables: %w", err)
	}
	if err := json.Unmarshal(stepsJSON, &def.Steps); err != nil {
		return nil, fmt.Errorf("unmarshaling steps: %w", err)
	}

	return &def, nil
}

// scanDefinitions iterates over pgx.Rows and returns a slice of definitions.
func (r *DefinitionRepository) scanDefinitions(rows pgx.Rows) ([]*model.WorkflowDefinition, error) {
	var defs []*model.WorkflowDefinition
	for rows.Next() {
		var (
			def             model.WorkflowDefinition
			triggerJSON     []byte
			variablesJSON   []byte
			stepsJSON       []byte
			updatedByNullable *string
		)

		if err := rows.Scan(
			&def.ID,
			&def.TenantID,
			&def.Name,
			&def.Description,
			&def.Version,
			&def.Status,
			&triggerJSON,
			&variablesJSON,
			&stepsJSON,
			&def.CreatedBy,
			&updatedByNullable,
			&def.CreatedAt,
			&def.UpdatedAt,
			&def.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning workflow definition row: %w", err)
		}

		if updatedByNullable != nil {
			def.UpdatedBy = *updatedByNullable
		}

		if err := json.Unmarshal(triggerJSON, &def.TriggerConfig); err != nil {
			return nil, fmt.Errorf("unmarshaling trigger_config: %w", err)
		}
		if err := json.Unmarshal(variablesJSON, &def.Variables); err != nil {
			return nil, fmt.Errorf("unmarshaling variables: %w", err)
		}
		if err := json.Unmarshal(stepsJSON, &def.Steps); err != nil {
			return nil, fmt.Errorf("unmarshaling steps: %w", err)
		}

		defs = append(defs, &def)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating workflow definition rows: %w", err)
	}
	return defs, nil
}

// nilIfEmpty returns nil when s is empty; otherwise returns a pointer to s.
// Used to store optional UUID fields as NULL rather than empty strings.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
