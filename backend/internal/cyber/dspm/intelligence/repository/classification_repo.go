package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// ClassificationRepository handles persistence for classification history records.
type ClassificationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewClassificationRepository creates a new ClassificationRepository.
func NewClassificationRepository(db *pgxpool.Pool, logger zerolog.Logger) *ClassificationRepository {
	return &ClassificationRepository{db: db, logger: logger}
}

// Insert records a classification change.
func (r *ClassificationRepository) Insert(ctx context.Context, h *model.ClassificationHistory) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	if h.CreatedAt.IsZero() {
		h.CreatedAt = time.Now().UTC()
	}

	evidenceJSON, err := json.Marshal(h.Evidence)
	if err != nil {
		return fmt.Errorf("marshal classification evidence: %w", err)
	}

	if h.OldPIITypes == nil {
		h.OldPIITypes = []string{}
	}
	if h.NewPIITypes == nil {
		h.NewPIITypes = []string{}
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO dspm_classification_history (
			id, tenant_id, data_asset_id,
			old_classification, new_classification,
			old_pii_types, new_pii_types,
			change_type, detected_by, confidence,
			evidence, actor_id, actor_type,
			created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		)`,
		h.ID, h.TenantID, h.DataAssetID,
		h.OldClassification, h.NewClassification,
		h.OldPIITypes, h.NewPIITypes,
		h.ChangeType, h.DetectedBy, h.Confidence,
		evidenceJSON, h.ActorID, h.ActorType,
		h.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert classification history: %w", err)
	}
	return nil
}

// ListByAsset returns classification history for a specific asset, ordered by created_at DESC.
// Returns the history records, total count, and any error.
func (r *ClassificationRepository) ListByAsset(ctx context.Context, tenantID, assetID uuid.UUID, params *dto.ClassificationHistoryParams) ([]model.ClassificationHistory, int, error) {
	if params == nil {
		params = &dto.ClassificationHistoryParams{}
	}
	params.SetDefaults()

	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM dspm_classification_history
		WHERE tenant_id = $1 AND data_asset_id = $2`,
		tenantID, assetID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count classification history: %w", err)
	}

	offset := (params.Page - 1) * params.PerPage
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, data_asset_id,
		       old_classification, new_classification,
		       old_pii_types, new_pii_types,
		       change_type, detected_by, confidence,
		       evidence, actor_id, actor_type,
		       created_at
		FROM dspm_classification_history
		WHERE tenant_id = $1 AND data_asset_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`,
		tenantID, assetID, params.PerPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list classification history: %w", err)
	}
	defer rows.Close()

	items := make([]model.ClassificationHistory, 0)
	for rows.Next() {
		h, err := scanClassificationHistory(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *h)
	}
	return items, total, rows.Err()
}

// scanClassificationHistory scans a single classification history row.
func scanClassificationHistory(row interface{ Scan(...interface{}) error }) (*model.ClassificationHistory, error) {
	var h model.ClassificationHistory
	var evidenceJSON []byte
	var oldPIITypes, newPIITypes []string

	err := row.Scan(
		&h.ID, &h.TenantID, &h.DataAssetID,
		&h.OldClassification, &h.NewClassification,
		&oldPIITypes, &newPIITypes,
		&h.ChangeType, &h.DetectedBy, &h.Confidence,
		&evidenceJSON, &h.ActorID, &h.ActorType,
		&h.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("classification history not found")
		}
		return nil, fmt.Errorf("scan classification history: %w", err)
	}

	h.OldPIITypes = oldPIITypes
	if h.OldPIITypes == nil {
		h.OldPIITypes = []string{}
	}
	h.NewPIITypes = newPIITypes
	if h.NewPIITypes == nil {
		h.NewPIITypes = []string{}
	}

	if len(evidenceJSON) > 0 {
		_ = json.Unmarshal(evidenceJSON, &h.Evidence)
	}

	return &h, nil
}
