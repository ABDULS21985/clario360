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

// TimerRecord represents a row from the workflow_timers table.
type TimerRecord struct {
	ID         string
	InstanceID string
	StepID     string
	FireAt     time.Time
}

// TaskRepository handles all database operations for human tasks, SLA
// management, and workflow timers.
type TaskRepository struct {
	pool *pgxpool.Pool
}

// NewTaskRepository creates a new TaskRepository backed by the provided
// connection pool.
func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

// Create inserts a new human task. JSONB fields (form_schema, form_data,
// metadata) are marshaled before insertion. The generated ID and timestamps
// are scanned back into the struct.
func (r *TaskRepository) Create(ctx context.Context, task *model.HumanTask) error {
	formSchemaJSON, err := json.Marshal(task.FormSchema)
	if err != nil {
		return fmt.Errorf("marshaling form_schema: %w", err)
	}

	var formDataJSON []byte
	if task.FormData != nil {
		formDataJSON, err = json.Marshal(task.FormData)
		if err != nil {
			return fmt.Errorf("marshaling form_data: %w", err)
		}
	}

	metadataJSON, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling task metadata: %w", err)
	}

	query := `
		INSERT INTO workflow_tasks (
			tenant_id, instance_id, step_id, step_exec_id,
			name, description, status,
			assignee_id, assignee_role,
			form_schema, form_data,
			sla_deadline, priority, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		task.TenantID,
		task.InstanceID,
		task.StepID,
		task.StepExecID,
		task.Name,
		task.Description,
		task.Status,
		task.AssigneeID,
		task.AssigneeRole,
		formSchemaJSON,
		formDataJSON,
		task.SLADeadline,
		task.Priority,
		metadataJSON,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

// GetByID retrieves a task by ID and tenant. Returns model.ErrNotFound when
// the row does not exist.
func (r *TaskRepository) GetByID(ctx context.Context, tenantID, id string) (*model.HumanTask, error) {
	query := `
		SELECT id, tenant_id, instance_id, step_id, step_exec_id,
		       name, description, status,
		       assignee_id, assignee_role, claimed_by, claimed_at,
		       form_schema, form_data,
		       sla_deadline, sla_breached,
		       escalated_to, escalation_role,
		       delegated_by, delegated_at,
		       priority, metadata,
		       completed_at, created_at, updated_at
		FROM workflow_tasks
		WHERE id = $1 AND tenant_id = $2`

	return r.scanTask(r.pool.QueryRow(ctx, query, id, tenantID))
}

// ListForUser returns tasks that are visible to a user: either directly
// assigned (assignee_id = userID), or claimable by the user's roles
// (assignee_role IN roles AND status = 'pending'). An optional status filter
// can further restrict results. Returns the matching tasks and the total count.
func (r *TaskRepository) ListForUser(ctx context.Context, tenantID, userID string, roles []string, status string, limit, offset int) ([]*model.HumanTask, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	// Build the visibility predicate: assigned to user OR role-claimable.
	visibilityParts := []string{
		fmt.Sprintf("assignee_id = $%d", argIdx),
		fmt.Sprintf("claimed_by = $%d", argIdx),
	}
	args = append(args, userID)
	argIdx++

	if len(roles) > 0 {
		placeholders := make([]string, len(roles))
		for i, role := range roles {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, role)
			argIdx++
		}
		visibilityParts = append(visibilityParts,
			fmt.Sprintf("(assignee_role IN (%s) AND status = 'pending')", strings.Join(placeholders, ", ")),
		)
	}

	conditions = append(conditions, "("+strings.Join(visibilityParts, " OR ")+")")

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Total count.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM workflow_tasks WHERE %s", where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting tasks for user: %w", err)
	}

	if total == 0 {
		return []*model.HumanTask{}, 0, nil
	}

	// Paginated data.
	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, instance_id, step_id, step_exec_id,
		       name, description, status,
		       assignee_id, assignee_role, claimed_by, claimed_at,
		       form_schema, form_data,
		       sla_deadline, sla_breached,
		       escalated_to, escalation_role,
		       delegated_by, delegated_at,
		       priority, metadata,
		       completed_at, created_at, updated_at
		FROM workflow_tasks
		WHERE %s
		ORDER BY priority DESC, created_at ASC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing tasks for user: %w", err)
	}
	defer rows.Close()

	tasks, err := r.scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// ClaimTask atomically claims a pending task using SELECT FOR UPDATE SKIP
// LOCKED inside a transaction. If the task is not in 'pending' status or has
// already been locked by another concurrent claim, a conflict error is returned.
func (r *TaskRepository) ClaimTask(ctx context.Context, tenantID, taskID, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning claim transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Attempt to lock the row. SKIP LOCKED means if another transaction holds
	// the lock we get zero rows instead of blocking.
	var lockedID string
	err = tx.QueryRow(ctx,
		`SELECT id FROM workflow_tasks
		 WHERE id = $1 AND tenant_id = $2 AND status = 'pending'
		 FOR UPDATE SKIP LOCKED`,
		taskID, tenantID,
	).Scan(&lockedID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("task %s: %w", taskID, model.ErrTaskNotClaimable)
		}
		return fmt.Errorf("locking task for claim: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE workflow_tasks
		 SET status = 'claimed', claimed_by = $2, claimed_at = now(), updated_at = now()
		 WHERE id = $1`,
		taskID, userID,
	)
	if err != nil {
		return fmt.Errorf("updating task claim: %w", err)
	}

	return tx.Commit(ctx)
}

// CompleteTask marks a claimed task as completed and stores the submitted form
// data. Only tasks in 'claimed' status can be completed.
func (r *TaskRepository) CompleteTask(ctx context.Context, tenantID, taskID string, formData map[string]interface{}) error {
	formDataJSON, err := json.Marshal(formData)
	if err != nil {
		return fmt.Errorf("marshaling form_data: %w", err)
	}

	query := `
		UPDATE workflow_tasks
		SET status = 'completed',
		    form_data = $3,
		    completed_at = now(),
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND status = 'claimed'`

	ct, err := r.pool.Exec(ctx, query, taskID, tenantID, formDataJSON)
	if err != nil {
		return fmt.Errorf("completing task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task %s: %w", taskID, model.ErrNotFound)
	}
	return nil
}

// DelegateTask transfers a claimed task from one user to another. The current
// assignee becomes delegated_by, and the task goes back to 'pending' with the
// new assignee.
func (r *TaskRepository) DelegateTask(ctx context.Context, tenantID, taskID, fromUserID, toUserID string) error {
	query := `
		UPDATE workflow_tasks
		SET assignee_id = $3,
		    status = 'pending',
		    claimed_by = NULL,
		    claimed_at = NULL,
		    delegated_by = $4,
		    delegated_at = now(),
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND status = 'claimed' AND claimed_by = $4`

	ct, err := r.pool.Exec(ctx, query, taskID, tenantID, toUserID, fromUserID)
	if err != nil {
		return fmt.Errorf("delegating task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task %s: %w", taskID, model.ErrTaskNotOwned)
	}
	return nil
}

// RejectTask rejects a claimed task, returning it to 'pending' status and
// clearing the claim fields. The reason is stored in the metadata for audit
// purposes.
func (r *TaskRepository) RejectTask(ctx context.Context, tenantID, taskID, userID, reason string) error {
	query := `
		UPDATE workflow_tasks
		SET status = 'rejected',
		    metadata = metadata || jsonb_build_object('rejection_reason', $4::text, 'rejected_by', $3::text),
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND status = 'claimed' AND claimed_by = $3`

	ct, err := r.pool.Exec(ctx, query, taskID, tenantID, userID, reason)
	if err != nil {
		return fmt.Errorf("rejecting task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task %s: %w", taskID, model.ErrTaskNotOwned)
	}
	return nil
}

// CountByStatus returns task counts grouped by status for tasks visible to the
// given user (by direct assignment or role-based claimability).
func (r *TaskRepository) CountByStatus(ctx context.Context, tenantID, userID string, roles []string) (map[string]int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	visibilityParts := []string{
		fmt.Sprintf("assignee_id = $%d", argIdx),
		fmt.Sprintf("claimed_by = $%d", argIdx),
	}
	args = append(args, userID)
	argIdx++

	if len(roles) > 0 {
		placeholders := make([]string, len(roles))
		for i, role := range roles {
			placeholders[i] = fmt.Sprintf("$%d", argIdx)
			args = append(args, role)
			argIdx++
		}
		visibilityParts = append(visibilityParts,
			fmt.Sprintf("(assignee_role IN (%s) AND status = 'pending')", strings.Join(placeholders, ", ")),
		)
	}

	conditions = append(conditions, "("+strings.Join(visibilityParts, " OR ")+")")
	where := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT status, COUNT(*) FROM workflow_tasks
		WHERE %s
		GROUP BY status`, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("counting tasks by status: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scanning task count row: %w", err)
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task count rows: %w", err)
	}

	return counts, nil
}

// GetOverdueTasks returns tasks that have passed their SLA deadline but have
// not yet been marked as breached. Only pending and claimed tasks are
// considered.
func (r *TaskRepository) GetOverdueTasks(ctx context.Context, limit int) ([]*model.HumanTask, error) {
	query := `
		SELECT id, tenant_id, instance_id, step_id, step_exec_id,
		       name, description, status,
		       assignee_id, assignee_role, claimed_by, claimed_at,
		       form_schema, form_data,
		       sla_deadline, sla_breached,
		       escalated_to, escalation_role,
		       delegated_by, delegated_at,
		       priority, metadata,
		       completed_at, created_at, updated_at
		FROM workflow_tasks
		WHERE sla_deadline IS NOT NULL
		  AND sla_deadline < now()
		  AND sla_breached = false
		  AND status IN ('pending', 'claimed')
		ORDER BY sla_deadline ASC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("getting overdue tasks: %w", err)
	}
	defer rows.Close()

	return r.scanTasks(rows)
}

// MarkSLABreached sets sla_breached = true for a task.
func (r *TaskRepository) MarkSLABreached(ctx context.Context, taskID string) error {
	query := `
		UPDATE workflow_tasks
		SET sla_breached = true, updated_at = now()
		WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, taskID)
	if err != nil {
		return fmt.Errorf("marking SLA breached: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task %s: %w", taskID, model.ErrNotFound)
	}
	return nil
}

// EscalateTask updates a task with escalation information, setting the
// escalation_role and changing status to 'escalated'.
func (r *TaskRepository) EscalateTask(ctx context.Context, taskID, escalationRole string) error {
	query := `
		UPDATE workflow_tasks
		SET status = 'escalated',
		    escalation_role = $2,
		    updated_at = now()
		WHERE id = $1 AND status IN ('pending', 'claimed')`

	ct, err := r.pool.Exec(ctx, query, taskID, escalationRole)
	if err != nil {
		return fmt.Errorf("escalating task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("task %s: %w", taskID, model.ErrNotFound)
	}
	return nil
}

// CancelByInstance cancels all non-terminal tasks (pending and claimed) for a
// given workflow instance. This is called when a workflow is cancelled or
// fails.
func (r *TaskRepository) CancelByInstance(ctx context.Context, instanceID string) error {
	query := `
		UPDATE workflow_tasks
		SET status = 'cancelled', updated_at = now()
		WHERE instance_id = $1 AND status IN ('pending', 'claimed')`

	_, err := r.pool.Exec(ctx, query, instanceID)
	if err != nil {
		return fmt.Errorf("cancelling tasks for instance %s: %w", instanceID, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Timer operations
// ---------------------------------------------------------------------------

// CreateTimer inserts a timer record and returns its generated ID.
func (r *TaskRepository) CreateTimer(ctx context.Context, instanceID, stepID string, fireAt time.Time) (string, error) {
	query := `
		INSERT INTO workflow_timers (instance_id, step_id, fire_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	var id string
	err := r.pool.QueryRow(ctx, query, instanceID, stepID, fireAt).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("creating timer: %w", err)
	}
	return id, nil
}

// GetUnfiredTimers returns timers whose fire_at has passed but have not been
// marked as fired. Used during recovery to catch timers that should have
// triggered while the engine was down.
func (r *TaskRepository) GetUnfiredTimers(ctx context.Context) ([]TimerRecord, error) {
	query := `
		SELECT id, instance_id, step_id, fire_at
		FROM workflow_timers
		WHERE fired = false AND fire_at <= now()
		ORDER BY fire_at ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("getting unfired timers: %w", err)
	}
	defer rows.Close()

	var timers []TimerRecord
	for rows.Next() {
		var t TimerRecord
		if err := rows.Scan(&t.ID, &t.InstanceID, &t.StepID, &t.FireAt); err != nil {
			return nil, fmt.Errorf("scanning timer row: %w", err)
		}
		timers = append(timers, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating timer rows: %w", err)
	}
	return timers, nil
}

// MarkTimerFired sets fired = true for a timer.
func (r *TaskRepository) MarkTimerFired(ctx context.Context, timerID string) error {
	query := `UPDATE workflow_timers SET fired = true WHERE id = $1`

	ct, err := r.pool.Exec(ctx, query, timerID)
	if err != nil {
		return fmt.Errorf("marking timer fired: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("timer %s: %w", timerID, model.ErrNotFound)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// scanTask scans a single pgx.Row into a HumanTask, handling JSONB
// deserialization of form_schema, form_data, and metadata.
func (r *TaskRepository) scanTask(row pgx.Row) (*model.HumanTask, error) {
	var (
		task           model.HumanTask
		formSchemaJSON []byte
		formDataJSON   []byte
		metadataJSON   []byte
	)

	err := row.Scan(
		&task.ID,
		&task.TenantID,
		&task.InstanceID,
		&task.StepID,
		&task.StepExecID,
		&task.Name,
		&task.Description,
		&task.Status,
		&task.AssigneeID,
		&task.AssigneeRole,
		&task.ClaimedBy,
		&task.ClaimedAt,
		&formSchemaJSON,
		&formDataJSON,
		&task.SLADeadline,
		&task.SLABreached,
		&task.EscalatedTo,
		&task.EscalationRole,
		&task.DelegatedBy,
		&task.DelegatedAt,
		&task.Priority,
		&metadataJSON,
		&task.CompletedAt,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("task: %w", model.ErrNotFound)
		}
		return nil, fmt.Errorf("scanning task: %w", err)
	}

	if err := json.Unmarshal(formSchemaJSON, &task.FormSchema); err != nil {
		return nil, fmt.Errorf("unmarshaling form_schema: %w", err)
	}
	if formDataJSON != nil {
		if err := json.Unmarshal(formDataJSON, &task.FormData); err != nil {
			return nil, fmt.Errorf("unmarshaling form_data: %w", err)
		}
	}
	if err := json.Unmarshal(metadataJSON, &task.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshaling task metadata: %w", err)
	}

	return &task, nil
}

// scanTasks iterates over pgx.Rows and returns a slice of tasks.
func (r *TaskRepository) scanTasks(rows pgx.Rows) ([]*model.HumanTask, error) {
	var tasks []*model.HumanTask
	for rows.Next() {
		var (
			task           model.HumanTask
			formSchemaJSON []byte
			formDataJSON   []byte
			metadataJSON   []byte
		)

		if err := rows.Scan(
			&task.ID,
			&task.TenantID,
			&task.InstanceID,
			&task.StepID,
			&task.StepExecID,
			&task.Name,
			&task.Description,
			&task.Status,
			&task.AssigneeID,
			&task.AssigneeRole,
			&task.ClaimedBy,
			&task.ClaimedAt,
			&formSchemaJSON,
			&formDataJSON,
			&task.SLADeadline,
			&task.SLABreached,
			&task.EscalatedTo,
			&task.EscalationRole,
			&task.DelegatedBy,
			&task.DelegatedAt,
			&task.Priority,
			&metadataJSON,
			&task.CompletedAt,
			&task.CreatedAt,
			&task.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning task row: %w", err)
		}

		if err := json.Unmarshal(formSchemaJSON, &task.FormSchema); err != nil {
			return nil, fmt.Errorf("unmarshaling form_schema: %w", err)
		}
		if formDataJSON != nil {
			if err := json.Unmarshal(formDataJSON, &task.FormData); err != nil {
				return nil, fmt.Errorf("unmarshaling form_data: %w", err)
			}
		}
		if err := json.Unmarshal(metadataJSON, &task.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling task metadata: %w", err)
		}

		tasks = append(tasks, &task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating task rows: %w", err)
	}
	return tasks, nil
}
