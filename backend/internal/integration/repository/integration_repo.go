package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	intdto "github.com/clario360/platform/internal/integration/dto"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

type IntegrationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewIntegrationRepository(db *pgxpool.Pool, logger zerolog.Logger) *IntegrationRepository {
	return &IntegrationRepository{db: db, logger: logger.With().Str("component", "integration_repo").Logger()}
}

func (r *IntegrationRepository) Create(ctx context.Context, integration *intmodel.Integration) (string, error) {
	filters, err := json.Marshal(integration.EventFilters)
	if err != nil {
		return "", fmt.Errorf("marshal event filters: %w", err)
	}

	query := `
		INSERT INTO integrations (
			tenant_id, type, name, description, config_encrypted, config_nonce, config_key_id,
			status, error_message, error_count, last_error_at, event_filters, last_used_at,
			delivery_count, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id`

	var id string
	err = r.db.QueryRow(
		ctx, query,
		integration.TenantID, string(integration.Type), integration.Name, integration.Description,
		integration.ConfigEncrypted, integration.ConfigNonce, integration.ConfigKeyID,
		string(integration.Status), integration.ErrorMessage, integration.ErrorCount, integration.LastErrorAt,
		filters, integration.LastUsedAt, integration.DeliveryCount, integration.CreatedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert integration: %w", err)
	}
	return id, nil
}

func (r *IntegrationRepository) List(ctx context.Context, tenantID string, query *intdto.ListQuery) ([]intmodel.Integration, int, error) {
	where := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	argIdx := 2

	if query.Search != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+query.Search+"%")
		argIdx++
	}
	if query.Type != "" {
		where = append(where, fmt.Sprintf("type = $%d", argIdx))
		args = append(args, query.Type)
		argIdx++
	}
	if query.Status != "" {
		// Support comma-separated multi-status: "active,inactive"
		statusValues := strings.Split(query.Status, ",")
		if len(statusValues) == 1 {
			where = append(where, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, strings.TrimSpace(statusValues[0]))
			argIdx++
		} else {
			placeholders := make([]string, len(statusValues))
			for i, sv := range statusValues {
				placeholders[i] = fmt.Sprintf("$%d", argIdx)
				args = append(args, strings.TrimSpace(sv))
				argIdx++
			}
			where = append(where, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", ")))
		}
	}

	whereSQL := " WHERE " + strings.Join(where, " AND ")

	countQuery := "SELECT COUNT(*) FROM integrations" + whereSQL
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count integrations: %w", err)
	}

	listArgs := append(append([]any{}, args...), query.PerPage, query.Offset())
	sql := fmt.Sprintf(`
		SELECT id, tenant_id, type, name, description, config_encrypted, config_nonce, config_key_id,
		       status, error_message, error_count, last_error_at, event_filters, last_used_at,
		       delivery_count, created_by, created_at, updated_at, deleted_at
		FROM integrations%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`,
		whereSQL, query.Sort, query.Order, argIdx, argIdx+1,
	)

	rows, err := r.db.Query(ctx, sql, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list integrations: %w", err)
	}
	defer rows.Close()

	items := make([]intmodel.Integration, 0, query.PerPage)
	for rows.Next() {
		item, err := scanIntegration(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *IntegrationRepository) GetByID(ctx context.Context, tenantID, id string) (*intmodel.Integration, error) {
	query := `
		SELECT id, tenant_id, type, name, description, config_encrypted, config_nonce, config_key_id,
		       status, error_message, error_count, last_error_at, event_filters, last_used_at,
		       delivery_count, created_by, created_at, updated_at, deleted_at
		FROM integrations
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`

	row := r.db.QueryRow(ctx, query, tenantID, id)
	item, err := scanIntegration(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("get integration: %w", err)
	}
	return item, nil
}

func (r *IntegrationRepository) ListActiveByTenant(ctx context.Context, tenantID string) ([]intmodel.Integration, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, type, name, description, config_encrypted, config_nonce, config_key_id,
		       status, error_message, error_count, last_error_at, event_filters, last_used_at,
		       delivery_count, created_by, created_at, updated_at, deleted_at
		FROM integrations
		WHERE tenant_id = $1 AND status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list active integrations: %w", err)
	}
	defer rows.Close()

	items := make([]intmodel.Integration, 0)
	for rows.Next() {
		item, err := scanIntegration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *IntegrationRepository) ListActiveByType(ctx context.Context, typ intmodel.IntegrationType) ([]intmodel.Integration, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, type, name, description, config_encrypted, config_nonce, config_key_id,
		       status, error_message, error_count, last_error_at, event_filters, last_used_at,
		       delivery_count, created_by, created_at, updated_at, deleted_at
		FROM integrations
		WHERE type = $1 AND status = 'active' AND deleted_at IS NULL
		ORDER BY created_at DESC`, string(typ))
	if err != nil {
		return nil, fmt.Errorf("list integrations by type: %w", err)
	}
	defer rows.Close()

	items := make([]intmodel.Integration, 0)
	for rows.Next() {
		item, err := scanIntegration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *IntegrationRepository) Update(ctx context.Context, integration *intmodel.Integration) error {
	filters, err := json.Marshal(integration.EventFilters)
	if err != nil {
		return fmt.Errorf("marshal event filters: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		UPDATE integrations
		SET name = $3,
		    description = $4,
		    config_encrypted = $5,
		    config_nonce = $6,
		    config_key_id = $7,
		    event_filters = $8,
		    status = $9,
		    error_message = $10,
		    error_count = $11,
		    last_error_at = $12,
		    last_used_at = $13,
		    delivery_count = $14,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		integration.TenantID, integration.ID, integration.Name, integration.Description,
		integration.ConfigEncrypted, integration.ConfigNonce, integration.ConfigKeyID,
		filters, string(integration.Status), integration.ErrorMessage, integration.ErrorCount,
		integration.LastErrorAt, integration.LastUsedAt, integration.DeliveryCount,
	)
	if err != nil {
		return fmt.Errorf("update integration: %w", err)
	}
	return nil
}

func (r *IntegrationRepository) UpdateStatus(ctx context.Context, tenantID, id string, status intmodel.IntegrationStatus, errMsg *string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integrations
		SET status = $3,
		    error_message = $4,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id, string(status), errMsg,
	)
	if err != nil {
		return fmt.Errorf("update integration status: %w", err)
	}
	return nil
}

func (r *IntegrationRepository) SoftDelete(ctx context.Context, tenantID, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integrations
		SET deleted_at = now(), updated_at = now(), status = 'inactive'
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}
	return nil
}

func (r *IntegrationRepository) RecordSuccess(ctx context.Context, integrationID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE integrations
		SET delivery_count = delivery_count + 1,
		    last_used_at = now(),
		    error_count = 0,
		    error_message = NULL,
		    last_error_at = NULL,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		integrationID,
	)
	if err != nil {
		return fmt.Errorf("record integration success: %w", err)
	}
	return nil
}

func (r *IntegrationRepository) RecordFailure(ctx context.Context, integrationID string, message string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		UPDATE integrations
		SET error_count = error_count + 1,
		    error_message = $2,
		    last_error_at = now(),
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING error_count`,
		integrationID, message,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("record integration failure: %w", err)
	}
	return count, nil
}

func scanIntegration(row interface {
	Scan(dest ...any) error
}) (*intmodel.Integration, error) {
	var (
		item        intmodel.Integration
		typ         string
		status      string
		filterBytes []byte
	)
	err := row.Scan(
		&item.ID,
		&item.TenantID,
		&typ,
		&item.Name,
		&item.Description,
		&item.ConfigEncrypted,
		&item.ConfigNonce,
		&item.ConfigKeyID,
		&status,
		&item.ErrorMessage,
		&item.ErrorCount,
		&item.LastErrorAt,
		&filterBytes,
		&item.LastUsedAt,
		&item.DeliveryCount,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	item.Type = intmodel.IntegrationType(typ)
	item.Status = intmodel.IntegrationStatus(status)
	if len(filterBytes) > 0 {
		if err := json.Unmarshal(filterBytes, &item.EventFilters); err != nil {
			return nil, fmt.Errorf("unmarshal integration filters: %w", err)
		}
	}
	return &item, nil
}

func truncateMessage(message string, max int) string {
	if len(message) <= max {
		return message
	}
	return message[:max]
}

func (r *IntegrationRepository) SetAutoDisabled(ctx context.Context, integrationID string) error {
	msg := truncateMessage("Integration auto-disabled after 10 consecutive delivery failures.", 1000)
	_, err := r.db.Exec(ctx, `
		UPDATE integrations
		SET status = 'error',
		    error_message = $2,
		    updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`,
		integrationID, msg,
	)
	if err != nil {
		return fmt.Errorf("auto disable integration: %w", err)
	}
	return nil
}

func (r *IntegrationRepository) TouchUpdatedAt(ctx context.Context, integrationID string, when time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE integrations SET updated_at = $2 WHERE id = $1`, integrationID, when)
	return err
}
