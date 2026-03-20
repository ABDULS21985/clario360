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
			tenant_id, name, description, category, version, status,
			trigger_config, variables, steps, created_by, updated_by, published_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		def.TenantID,
		def.Name,
		def.Description,
		def.Category,
		def.Version,
		def.Status,
		triggerJSON,
		variablesJSON,
		stepsJSON,
		def.CreatedBy,
		nilIfEmpty(def.UpdatedBy),
		def.PublishedAt,
	).Scan(&def.ID, &def.CreatedAt, &def.UpdatedAt)
}

// GetByID retrieves a definition by ID and tenant. Returns a wrapped
// model.ErrNotFound if the row does not exist or has been soft-deleted.
func (r *DefinitionRepository) GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, category, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, published_at, deleted_at,
		       (SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id) AS instance_count
		FROM workflow_definitions
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	return r.scanDefinition(ctx, r.pool.QueryRow(ctx, query, id, tenantID))
}

// GetActiveByID retrieves a definition only when its status is 'active' and it
// has not been soft-deleted.
func (r *DefinitionRepository) GetActiveByID(ctx context.Context, tenantID, id string) (*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, category, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, published_at, deleted_at,
		       (SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id) AS instance_count
		FROM workflow_definitions
		WHERE id = $1 AND tenant_id = $2 AND status = 'active' AND deleted_at IS NULL`

	return r.scanDefinition(ctx, r.pool.QueryRow(ctx, query, id, tenantID))
}

// GetByIDAndVersion retrieves a specific version of a definition.
func (r *DefinitionRepository) GetByIDAndVersion(ctx context.Context, tenantID, id string, version int) (*model.WorkflowDefinition, error) {
	query := `
		SELECT id, tenant_id, name, description, category, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, published_at, deleted_at,
		       (SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id) AS instance_count
		FROM workflow_definitions
		WHERE id = $1 AND tenant_id = $2 AND version = $3 AND deleted_at IS NULL`

	return r.scanDefinition(ctx, r.pool.QueryRow(ctx, query, id, tenantID, version))
}

// validSortColumns defines the allowed sort columns to prevent SQL injection.
var validSortColumns = map[string]string{
	"name":       "name",
	"category":   "category",
	"status":     "status",
	"version":    "version",
	"created_at":     "created_at",
	"updated_at":     "updated_at",
	"published_at":   "published_at",
	"instance_count": "(SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id)",
}

// List returns paginated definitions filtered by tenant, optional status
// (comma-separated for multiple values), optional category, and optional
// name search (case-insensitive ILIKE). Supports sort column and direction.
// The second return value is the total count for pagination metadata.
func (r *DefinitionRepository) List(ctx context.Context, tenantID, status, nameFilter, category, sortBy, sortOrder string, limit, offset int) ([]*model.WorkflowDefinition, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "deleted_at IS NULL")

	if status != "" {
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, strings.TrimSpace(statuses[0]))
			argIdx++
		} else {
			placeholders := make([]string, len(statuses))
			for i, s := range statuses {
				placeholders[i] = fmt.Sprintf("$%d", argIdx)
				args = append(args, strings.TrimSpace(s))
				argIdx++
			}
			conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
		}
	}
	if category != "" {
		categories := strings.Split(category, ",")
		if len(categories) == 1 {
			conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
			args = append(args, strings.TrimSpace(categories[0]))
			argIdx++
		} else {
			placeholders := make([]string, len(categories))
			for i, c := range categories {
				placeholders[i] = fmt.Sprintf("$%d", argIdx)
				args = append(args, strings.TrimSpace(c))
				argIdx++
			}
			conditions = append(conditions, fmt.Sprintf("category IN (%s)", strings.Join(placeholders, ",")))
		}
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

	// Resolve sort column with whitelist; default to updated_at DESC.
	orderCol := "updated_at"
	if col, ok := validSortColumns[sortBy]; ok {
		orderCol = col
	}
	orderDir := "DESC"
	if strings.EqualFold(sortOrder, "asc") {
		orderDir = "ASC"
	}

	// Paginated data query.
	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, category, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, published_at, deleted_at,
		       (SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id) AS instance_count
		FROM workflow_definitions
		WHERE %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, where, orderCol, orderDir, argIdx, argIdx+1)
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
		SELECT id, tenant_id, name, description, category, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, published_at, deleted_at,
		       (SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id) AS instance_count
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
		    category = $4,
		    status = $5,
		    trigger_config = $6,
		    variables = $7,
		    steps = $8,
		    updated_by = $9,
		    published_at = $10,
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	ct, err := r.pool.Exec(ctx, query,
		def.ID,
		def.TenantID,
		def.Description,
		def.Category,
		def.Status,
		triggerJSON,
		variablesJSON,
		stepsJSON,
		nilIfEmpty(def.UpdatedBy),
		def.PublishedAt,
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
		SELECT id, tenant_id, name, description, category, version, status,
		       trigger_config, variables, steps,
		       created_by, updated_by, created_at, updated_at, published_at, deleted_at,
		       (SELECT COUNT(*) FROM workflow_instances WHERE definition_id = workflow_definitions.id) AS instance_count
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
		&def.Category,
		&def.Version,
		&def.Status,
		&triggerJSON,
		&variablesJSON,
		&stepsJSON,
		&def.CreatedBy,
		&updatedByNullable,
		&def.CreatedAt,
		&def.UpdatedAt,
		&def.PublishedAt,
		&def.DeletedAt,
		&def.InstanceCount,
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
			&def.Category,
			&def.Version,
			&def.Status,
			&triggerJSON,
			&variablesJSON,
			&stepsJSON,
			&def.CreatedBy,
			&updatedByNullable,
			&def.CreatedAt,
			&def.UpdatedAt,
			&def.PublishedAt,
			&def.DeletedAt,
			&def.InstanceCount,
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
