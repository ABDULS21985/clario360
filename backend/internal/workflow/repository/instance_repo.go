package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/workflow/model"
)

// InstanceRepository handles all database operations for workflow instances
// and their associated step executions.
type InstanceRepository struct {
	pool *pgxpool.Pool
}

// NewInstanceRepository creates a new InstanceRepository backed by the
// provided connection pool.
func NewInstanceRepository(pool *pgxpool.Pool) *InstanceRepository {
	return &InstanceRepository{pool: pool}
}

// Create inserts a new workflow instance. JSONB fields (variables,
// step_outputs, trigger_data) are marshaled before insertion. The generated
// ID and timestamps are scanned back into the struct.
func (r *InstanceRepository) Create(ctx context.Context, inst *model.WorkflowInstance) error {
	variablesJSON, err := json.Marshal(inst.Variables)
	if err != nil {
		return fmt.Errorf("marshaling instance variables: %w", err)
	}
	stepOutputsJSON, err := json.Marshal(inst.StepOutputs)
	if err != nil {
		return fmt.Errorf("marshaling step_outputs: %w", err)
	}

	// trigger_data is already json.RawMessage; use nil for empty.
	var triggerData []byte
	if len(inst.TriggerData) > 0 {
		triggerData = []byte(inst.TriggerData)
	}

	query := `
		INSERT INTO workflow_instances (
			tenant_id, definition_id, definition_ver, status,
			current_step_id, variables, step_outputs, trigger_data,
			error_message, started_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, started_at, updated_at, lock_version`

	return r.pool.QueryRow(ctx, query,
		inst.TenantID,
		inst.DefinitionID,
		inst.DefinitionVer,
		inst.Status,
		inst.CurrentStepID,
		variablesJSON,
		stepOutputsJSON,
		triggerData,
		inst.ErrorMessage,
		inst.StartedBy,
	).Scan(&inst.ID, &inst.StartedAt, &inst.UpdatedAt, &inst.LockVersion)
}

// GetByID retrieves an instance by ID and tenant. Returns model.ErrNotFound
// when the row does not exist.
func (r *InstanceRepository) GetByID(ctx context.Context, tenantID, id string) (*model.WorkflowInstance, error) {
	query := `
		SELECT id, tenant_id, definition_id, definition_ver, status,
		       current_step_id, variables, step_outputs, trigger_data,
		       error_message, started_by, started_at, completed_at,
		       updated_at, lock_version
		FROM workflow_instances
		WHERE id = $1 AND tenant_id = $2`

	return r.scanInstance(r.pool.QueryRow(ctx, query, id, tenantID))
}

// GetByIDForUpdate retrieves an instance using a row-level lock inside the
// provided transaction (SELECT ... FOR UPDATE). This must be called within a
// transaction obtained from pool.Begin() so the lock is held until commit.
func (r *InstanceRepository) GetByIDForUpdate(ctx context.Context, tx pgx.Tx, id string) (*model.WorkflowInstance, error) {
	query := `
		SELECT id, tenant_id, definition_id, definition_ver, status,
		       current_step_id, variables, step_outputs, trigger_data,
		       error_message, started_by, started_at, completed_at,
		       updated_at, lock_version
		FROM workflow_instances
		WHERE id = $1
		FOR UPDATE`

	return r.scanInstance(tx.QueryRow(ctx, query, id))
}

// UpdateWithLock performs an optimistic-locking update. The caller must supply
// the current lock_version read during the preceding SELECT. If the version in
// the database no longer matches, no rows are affected and
// model.ErrConcurrencyConfl is returned.
func (r *InstanceRepository) UpdateWithLock(ctx context.Context, inst *model.WorkflowInstance) error {
	variablesJSON, err := json.Marshal(inst.Variables)
	if err != nil {
		return fmt.Errorf("marshaling instance variables: %w", err)
	}
	stepOutputsJSON, err := json.Marshal(inst.StepOutputs)
	if err != nil {
		return fmt.Errorf("marshaling step_outputs: %w", err)
	}

	var triggerData []byte
	if len(inst.TriggerData) > 0 {
		triggerData = []byte(inst.TriggerData)
	}

	query := `
		UPDATE workflow_instances
		SET status = $2,
		    current_step_id = $3,
		    variables = $4,
		    step_outputs = $5,
		    trigger_data = $6,
		    error_message = $7,
		    completed_at = $8,
		    lock_version = lock_version + 1,
		    updated_at = now()
		WHERE id = $1 AND lock_version = $9`

	ct, err := r.pool.Exec(ctx, query,
		inst.ID,
		inst.Status,
		inst.CurrentStepID,
		variablesJSON,
		stepOutputsJSON,
		triggerData,
		inst.ErrorMessage,
		inst.CompletedAt,
		inst.LockVersion,
	)
	if err != nil {
		return fmt.Errorf("updating workflow instance: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("instance %s: %w", inst.ID, model.ErrConcurrencyConfl)
	}

	// Reflect the new lock version in the caller's struct.
	inst.LockVersion++
	return nil
}

// List returns paginated instances filtered by tenant and optional criteria:
// status, definition_id, started_by, and date range. The second return value
// is the total count matching the filters.
func (r *InstanceRepository) List(ctx context.Context, tenantID, status, definitionID, startedBy string, dateFrom, dateTo *time.Time, limit, offset int) ([]*model.WorkflowInstance, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if definitionID != "" {
		conditions = append(conditions, fmt.Sprintf("definition_id = $%d", argIdx))
		args = append(args, definitionID)
		argIdx++
	}
	if startedBy != "" {
		conditions = append(conditions, fmt.Sprintf("started_by = $%d", argIdx))
		args = append(args, startedBy)
		argIdx++
	}
	if dateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("started_at >= $%d", argIdx))
		args = append(args, *dateFrom)
		argIdx++
	}
	if dateTo != nil {
		conditions = append(conditions, fmt.Sprintf("started_at <= $%d", argIdx))
		args = append(args, *dateTo)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Total count.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM workflow_instances WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting workflow instances: %w", err)
	}

	if total == 0 {
		return []*model.WorkflowInstance{}, 0, nil
	}

	// Paginated data.
	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, definition_id, definition_ver, status,
		       current_step_id, variables, step_outputs, trigger_data,
		       error_message, started_by, started_at, completed_at,
		       updated_at, lock_version
		FROM workflow_instances
		WHERE %s
		ORDER BY started_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing workflow instances: %w", err)
	}
	defer rows.Close()

	instances, err := r.scanInstances(rows)
	if err != nil {
		return nil, 0, err
	}

	return instances, total, nil
}

// ListRunning returns all instances with status 'running', paginated. This is
// used during engine recovery to resume in-flight workflows.
func (r *InstanceRepository) ListRunning(ctx context.Context, limit, offset int) ([]*model.WorkflowInstance, error) {
	query := `
		SELECT id, tenant_id, definition_id, definition_ver, status,
		       current_step_id, variables, step_outputs, trigger_data,
		       error_message, started_by, started_at, completed_at,
		       updated_at, lock_version
		FROM workflow_instances
		WHERE status = 'running'
		ORDER BY started_at ASC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("listing running instances: %w", err)
	}
	defer rows.Close()

	return r.scanInstances(rows)
}

// ---------------------------------------------------------------------------
// Step Execution operations
// ---------------------------------------------------------------------------

// CreateStepExecution inserts a step execution record. The generated ID and
// created_at are scanned back into the struct.
func (r *InstanceRepository) CreateStepExecution(ctx context.Context, exec *model.StepExecution) error {
	var inputData []byte
	if len(exec.InputData) > 0 {
		inputData = []byte(exec.InputData)
	}
	var outputData []byte
	if len(exec.OutputData) > 0 {
		outputData = []byte(exec.OutputData)
	}

	query := `
		INSERT INTO workflow_step_executions (
			instance_id, step_id, step_type, status,
			input_data, output_data, error_message, attempt,
			started_at, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		exec.InstanceID,
		exec.StepID,
		exec.StepType,
		exec.Status,
		inputData,
		outputData,
		exec.ErrorMessage,
		exec.Attempt,
		exec.StartedAt,
		exec.CompletedAt,
	).Scan(&exec.ID, &exec.CreatedAt)
}

// UpdateStepExecution updates a step execution's mutable fields: status,
// output_data, error_message, started_at, and completed_at.
func (r *InstanceRepository) UpdateStepExecution(ctx context.Context, exec *model.StepExecution) error {
	var outputData []byte
	if len(exec.OutputData) > 0 {
		outputData = []byte(exec.OutputData)
	}

	query := `
		UPDATE workflow_step_executions
		SET status = $2,
		    output_data = $3,
		    error_message = $4,
		    started_at = $5,
		    completed_at = $6
		WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query,
		exec.ID,
		exec.Status,
		outputData,
		exec.ErrorMessage,
		exec.StartedAt,
		exec.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("updating step execution: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("step execution %s: %w", exec.ID, model.ErrNotFound)
	}
	return nil
}

// GetStepExecutions returns all step executions for an instance, ordered by
// created_at ascending so they appear in execution order.
func (r *InstanceRepository) GetStepExecutions(ctx context.Context, instanceID string) ([]*model.StepExecution, error) {
	query := `
		SELECT id, instance_id, step_id, step_type, status,
		       input_data, output_data, error_message, attempt,
		       started_at, completed_at, created_at
		FROM workflow_step_executions
		WHERE instance_id = $1
		ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("getting step executions: %w", err)
	}
	defer rows.Close()

	return r.scanStepExecutions(rows)
}

// GetLastFailedStep returns the most recent failed step execution for an
// instance, or nil if none exist.
func (r *InstanceRepository) GetLastFailedStep(ctx context.Context, instanceID string) (*model.StepExecution, error) {
	query := `
		SELECT id, instance_id, step_id, step_type, status,
		       input_data, output_data, error_message, attempt,
		       started_at, completed_at, created_at
		FROM workflow_step_executions
		WHERE instance_id = $1 AND status = 'failed'
		ORDER BY created_at DESC
		LIMIT 1`

	exec, err := r.scanStepExecution(r.pool.QueryRow(ctx, query, instanceID))
	if err != nil {
		// No failed step is a valid state, not an error.
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return exec, nil
}

// CheckTriggerDedup checks whether an instance already exists for the given
// definition with matching trigger_data->>'event_id'. Returns true if a
// duplicate exists.
func (r *InstanceRepository) CheckTriggerDedup(ctx context.Context, definitionID, eventID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM workflow_instances
			WHERE definition_id = $1
			  AND trigger_data->>'event_id' = $2
		)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, definitionID, eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking trigger dedup: %w", err)
	}
	return exists, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// scanInstance scans a single pgx.Row into a WorkflowInstance, handling JSONB
// deserialization.
func (r *InstanceRepository) scanInstance(row pgx.Row) (*model.WorkflowInstance, error) {
	var (
		inst            model.WorkflowInstance
		variablesJSON   []byte
		stepOutputsJSON []byte
		triggerData     []byte
	)

	err := row.Scan(
		&inst.ID,
		&inst.TenantID,
		&inst.DefinitionID,
		&inst.DefinitionVer,
		&inst.Status,
		&inst.CurrentStepID,
		&variablesJSON,
		&stepOutputsJSON,
		&triggerData,
		&inst.ErrorMessage,
		&inst.StartedBy,
		&inst.StartedAt,
		&inst.CompletedAt,
		&inst.UpdatedAt,
		&inst.LockVersion,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("instance: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("scanning workflow instance: %w", err)
	}

	if err := json.Unmarshal(variablesJSON, &inst.Variables); err != nil {
		return nil, fmt.Errorf("unmarshaling instance variables: %w", err)
	}
	if err := json.Unmarshal(stepOutputsJSON, &inst.StepOutputs); err != nil {
		return nil, fmt.Errorf("unmarshaling step_outputs: %w", err)
	}
	if triggerData != nil {
		inst.TriggerData = json.RawMessage(triggerData)
	}

	return &inst, nil
}

// scanInstances iterates over pgx.Rows and returns a slice of instances.
func (r *InstanceRepository) scanInstances(rows pgx.Rows) ([]*model.WorkflowInstance, error) {
	var instances []*model.WorkflowInstance
	for rows.Next() {
		var (
			inst            model.WorkflowInstance
			variablesJSON   []byte
			stepOutputsJSON []byte
			triggerData     []byte
		)

		if err := rows.Scan(
			&inst.ID,
			&inst.TenantID,
			&inst.DefinitionID,
			&inst.DefinitionVer,
			&inst.Status,
			&inst.CurrentStepID,
			&variablesJSON,
			&stepOutputsJSON,
			&triggerData,
			&inst.ErrorMessage,
			&inst.StartedBy,
			&inst.StartedAt,
			&inst.CompletedAt,
			&inst.UpdatedAt,
			&inst.LockVersion,
		); err != nil {
			return nil, fmt.Errorf("scanning workflow instance row: %w", err)
		}

		if err := json.Unmarshal(variablesJSON, &inst.Variables); err != nil {
			return nil, fmt.Errorf("unmarshaling instance variables: %w", err)
		}
		if err := json.Unmarshal(stepOutputsJSON, &inst.StepOutputs); err != nil {
			return nil, fmt.Errorf("unmarshaling step_outputs: %w", err)
		}
		if triggerData != nil {
			inst.TriggerData = json.RawMessage(triggerData)
		}

		instances = append(instances, &inst)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating workflow instance rows: %w", err)
	}
	return instances, nil
}

// scanStepExecution scans a single pgx.Row into a StepExecution.
func (r *InstanceRepository) scanStepExecution(row pgx.Row) (*model.StepExecution, error) {
	var (
		exec      model.StepExecution
		inputData []byte
		outData   []byte
	)

	err := row.Scan(
		&exec.ID,
		&exec.InstanceID,
		&exec.StepID,
		&exec.StepType,
		&exec.Status,
		&inputData,
		&outData,
		&exec.ErrorMessage,
		&exec.Attempt,
		&exec.StartedAt,
		&exec.CompletedAt,
		&exec.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("step execution: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("scanning step execution: %w", err)
	}

	if inputData != nil {
		exec.InputData = json.RawMessage(inputData)
	}
	if outData != nil {
		exec.OutputData = json.RawMessage(outData)
	}

	return &exec, nil
}

// scanStepExecutions iterates over pgx.Rows and returns a slice of step executions.
func (r *InstanceRepository) scanStepExecutions(rows pgx.Rows) ([]*model.StepExecution, error) {
	var executions []*model.StepExecution
	for rows.Next() {
		var (
			exec      model.StepExecution
			inputData []byte
			outData   []byte
		)

		if err := rows.Scan(
			&exec.ID,
			&exec.InstanceID,
			&exec.StepID,
			&exec.StepType,
			&exec.Status,
			&inputData,
			&outData,
			&exec.ErrorMessage,
			&exec.Attempt,
			&exec.StartedAt,
			&exec.CompletedAt,
			&exec.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning step execution row: %w", err)
		}

		if inputData != nil {
			exec.InputData = json.RawMessage(inputData)
		}
		if outData != nil {
			exec.OutputData = json.RawMessage(outData)
		}

		executions = append(executions, &exec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating step execution rows: %w", err)
	}
	return executions, nil
}

// isNotFound returns true when the error wraps model.ErrNotFound.
func isNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), model.ErrNotFound.Error())
}
