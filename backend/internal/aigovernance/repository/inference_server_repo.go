package repository

import (
	"context"
	"fmt"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type InferenceServerRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewInferenceServerRepository(db *pgxpool.Pool, logger zerolog.Logger) *InferenceServerRepository {
	return &InferenceServerRepository{db: db, logger: loggerWithRepo(logger, "ai_inference_server")}
}

const serverColumns = `id, tenant_id, name, backend_type, base_url, health_endpoint,
	model_name, api_key, quantization, status, cpu_cores, memory_mb,
	gpu_type, gpu_count, max_concurrent, stream_capable, metadata, created_at, updated_at`

func (r *InferenceServerRepository) Create(ctx context.Context, item *aigovmodel.InferenceServer) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO ai_inference_servers (
			id, tenant_id, name, backend_type, base_url, health_endpoint,
			model_name, api_key, quantization, status, cpu_cores, memory_mb,
			gpu_type, gpu_count, max_concurrent, stream_capable, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		)`,
		item.ID, item.TenantID, item.Name, item.BackendType, item.BaseURL, item.HealthEndpoint,
		item.ModelName, item.APIKey, item.Quantization, item.Status, item.CPUCores, item.MemoryMB,
		item.GPUType, item.GPUCount, item.MaxConcurrent, item.StreamCapable, item.Metadata, item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ai inference server: %w", err)
	}
	return nil
}

func (r *InferenceServerRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*aigovmodel.InferenceServer, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+serverColumns+`
		FROM ai_inference_servers
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	item, err := scanInferenceServer(row)
	if err != nil {
		return nil, rowNotFound(err)
	}
	return item, nil
}

type ListServersParams struct {
	BackendType string
	Status      string
	Page        int
	PerPage     int
}

func (r *InferenceServerRepository) List(ctx context.Context, tenantID uuid.UUID, params ListServersParams) ([]aigovmodel.InferenceServer, int, error) {
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PerPage <= 0 {
		params.PerPage = 25
	}
	where := "WHERE tenant_id = $1"
	args := []any{tenantID}
	idx := 2
	if params.BackendType != "" {
		where += fmt.Sprintf(" AND backend_type = $%d", idx)
		args = append(args, params.BackendType)
		idx++
	}
	if params.Status != "" {
		where += fmt.Sprintf(" AND status = $%d", idx)
		args = append(args, params.Status)
		idx++
	}

	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ai_inference_servers "+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count ai inference servers: %w", err)
	}

	offset := (params.Page - 1) * params.PerPage
	args = append(args, params.PerPage, offset)
	query := fmt.Sprintf(`
		SELECT %s
		FROM ai_inference_servers %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, serverColumns, where, idx, idx+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ai inference servers: %w", err)
	}
	defer rows.Close()
	items := make([]aigovmodel.InferenceServer, 0)
	for rows.Next() {
		item, scanErr := scanInferenceServer(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *InferenceServerRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status aigovmodel.InferenceServerStatus) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE ai_inference_servers SET status = $3
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id, status,
	)
	if err != nil {
		return fmt.Errorf("update ai inference server status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InferenceServerRepository) Update(ctx context.Context, item *aigovmodel.InferenceServer) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE ai_inference_servers SET
			name = $3, base_url = $4, health_endpoint = $5,
			model_name = $6, api_key = $7, quantization = $8, status = $9,
			cpu_cores = $10, memory_mb = $11, gpu_type = $12, gpu_count = $13,
			max_concurrent = $14, stream_capable = $15, metadata = $16, updated_at = $17
		WHERE tenant_id = $1 AND id = $2`,
		item.TenantID, item.ID,
		item.Name, item.BaseURL, item.HealthEndpoint,
		item.ModelName, item.APIKey, item.Quantization, item.Status,
		item.CPUCores, item.MemoryMB, item.GPUType, item.GPUCount,
		item.MaxConcurrent, item.StreamCapable, item.Metadata, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update ai inference server: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InferenceServerRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE ai_inference_servers SET status = 'decommissioned'
		WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	)
	if err != nil {
		return fmt.Errorf("decommission ai inference server: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type serverScannable interface {
	Scan(dest ...any) error
}

func scanInferenceServer(row serverScannable) (*aigovmodel.InferenceServer, error) {
	item := &aigovmodel.InferenceServer{}
	var metadata []byte
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.BackendType, &item.BaseURL,
		&item.HealthEndpoint, &item.ModelName, &item.APIKey, &item.Quantization, &item.Status,
		&item.CPUCores, &item.MemoryMB, &item.GPUType, &item.GPUCount,
		&item.MaxConcurrent, &item.StreamCapable, &metadata, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = nullJSON(metadata, "{}")
	return item, nil
}
