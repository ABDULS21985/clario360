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

	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/ueba/model"
	"github.com/clario360/platform/internal/database"
)

type ProfileRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewProfileRepository(db *pgxpool.Pool, logger zerolog.Logger) *ProfileRepository {
	return &ProfileRepository{
		db:     db,
		logger: logger.With().Str("component", "ueba-profile-repo").Logger(),
	}
}

func (r *ProfileRepository) GetOrCreate(ctx context.Context, tenantID uuid.UUID, entityType model.EntityType, entityID, entityName, entityEmail string) (*model.UEBAProfile, error) {
	profile, err := r.GetByEntityTypeAndID(ctx, tenantID, entityType, entityID)
	if err == nil {
		return profile, nil
	}
	if err != repository.ErrNotFound {
		return nil, err
	}

	now := time.Now().UTC()
	profile = &model.UEBAProfile{
		ID:              uuid.New(),
		TenantID:        tenantID,
		EntityType:      entityType,
		EntityID:        entityID,
		EntityName:      entityName,
		EntityEmail:     entityEmail,
		FirstSeenAt:     now,
		LastSeenAt:      now,
		ProfileMaturity: model.ProfileMaturityLearning,
		Status:          model.ProfileStatusActive,
		RiskLevel:       model.RiskLevelLow,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	profile.EnsureDefaults()
	if err := database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		baselineJSON, factorsJSON, err := marshalProfileJSON(profile)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO ueba_profiles (
				id, tenant_id, entity_type, entity_id, entity_name, entity_email,
				baseline, observation_count, profile_maturity, first_seen_at, last_seen_at,
				days_active, risk_score, risk_level, risk_factors, status, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18
			)
			ON CONFLICT (tenant_id, entity_type, entity_id) DO NOTHING`,
			profile.ID, profile.TenantID, profile.EntityType, profile.EntityID, profile.EntityName, nullString(profile.EntityEmail),
			baselineJSON, profile.ObservationCount, profile.ProfileMaturity, profile.FirstSeenAt, profile.LastSeenAt,
			profile.DaysActive, profile.RiskScore, profile.RiskLevel, factorsJSON, profile.Status, profile.CreatedAt, profile.UpdatedAt,
		)
		return err
	}); err != nil {
		return nil, fmt.Errorf("create ueba profile: %w", err)
	}
	return r.GetByEntityTypeAndID(ctx, tenantID, entityType, entityID)
}

func (r *ProfileRepository) GetByEntity(ctx context.Context, tenantID uuid.UUID, entityID string) (*model.UEBAProfile, error) {
	var profile *model.UEBAProfile
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
				id, tenant_id, entity_type, entity_id, COALESCE(entity_name, ''), COALESCE(entity_email, ''),
				baseline, observation_count, profile_maturity, first_seen_at, last_seen_at, days_active,
				risk_score::double precision, risk_level, risk_factors,
				risk_last_updated, risk_last_decayed, alert_count_7d, alert_count_30d, last_alert_at,
				status, suppressed_until, COALESCE(suppressed_reason, ''), created_at, updated_at
			FROM ueba_profiles
			WHERE tenant_id = $1 AND entity_id = $2`,
			tenantID, entityID,
		)
		item, scanErr := scanProfile(row)
		if scanErr != nil {
			if scanErr == pgx.ErrNoRows {
				return repository.ErrNotFound
			}
			return scanErr
		}
		profile = item
		return nil
	})
	return profile, err
}

func (r *ProfileRepository) GetByEntityTypeAndID(ctx context.Context, tenantID uuid.UUID, entityType model.EntityType, entityID string) (*model.UEBAProfile, error) {
	var profile *model.UEBAProfile
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
				id, tenant_id, entity_type, entity_id, COALESCE(entity_name, ''), COALESCE(entity_email, ''),
				baseline, observation_count, profile_maturity, first_seen_at, last_seen_at, days_active,
				risk_score::double precision, risk_level, risk_factors,
				risk_last_updated, risk_last_decayed, alert_count_7d, alert_count_30d, last_alert_at,
				status, suppressed_until, COALESCE(suppressed_reason, ''), created_at, updated_at
			FROM ueba_profiles
			WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3`,
			tenantID, entityType, entityID,
		)
		item, scanErr := scanProfile(row)
		if scanErr != nil {
			if scanErr == pgx.ErrNoRows {
				return repository.ErrNotFound
			}
			return scanErr
		}
		profile = item
		return nil
	})
	return profile, err
}

func (r *ProfileRepository) Update(ctx context.Context, profile *model.UEBAProfile) error {
	if profile == nil {
		return fmt.Errorf("profile is required")
	}
	profile.UpdatedAt = time.Now().UTC()
	return database.RunWithTenant(ctx, r.db, profile.TenantID, func(tx pgx.Tx) error {
		baselineJSON, factorsJSON, err := marshalProfileJSON(profile)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE ueba_profiles
			SET entity_name = $3,
				entity_email = $4,
				baseline = $5,
				observation_count = $6,
				profile_maturity = $7,
				first_seen_at = $8,
				last_seen_at = $9,
				days_active = $10,
				risk_score = $11,
				risk_level = $12,
				risk_factors = $13,
				risk_last_updated = $14,
				risk_last_decayed = $15,
				alert_count_7d = $16,
				alert_count_30d = $17,
				last_alert_at = $18,
				status = $19,
				suppressed_until = $20,
				suppressed_reason = $21,
				updated_at = $22
			WHERE tenant_id = $1 AND id = $2`,
			profile.TenantID, profile.ID, profile.EntityName, nullString(profile.EntityEmail),
			baselineJSON, profile.ObservationCount, profile.ProfileMaturity, profile.FirstSeenAt, profile.LastSeenAt,
			profile.DaysActive, profile.RiskScore, profile.RiskLevel, factorsJSON, profile.RiskLastUpdated,
			profile.RiskLastDecayed, profile.AlertCount7D, profile.AlertCount30D, profile.LastAlertAt,
			profile.Status, profile.SuppressedUntil, nullString(profile.SuppressedReason), profile.UpdatedAt,
		)
		return err
	})
}

func (r *ProfileRepository) UpdateRisk(ctx context.Context, profile *model.UEBAProfile) error {
	return r.Update(ctx, profile)
}

func (r *ProfileRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, entityID string, entityType model.EntityType, status model.ProfileStatus, until *time.Time, reason string) (*model.UEBAProfile, error) {
	err := database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(ctx, `
			UPDATE ueba_profiles
			SET status = $4,
				suppressed_until = $5,
				suppressed_reason = $6,
				updated_at = now()
			WHERE tenant_id = $1 AND entity_type = $2 AND entity_id = $3`,
			tenantID, entityType, entityID, status, until, nullString(reason),
		)
		return execErr
	})
	if err != nil {
		return nil, err
	}
	return r.GetByEntityTypeAndID(ctx, tenantID, entityType, entityID)
}

func (r *ProfileRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int, status model.ProfileStatus) ([]*model.UEBAProfile, int, error) {
	if limit <= 0 {
		limit = 25
	}
	var (
		items []*model.UEBAProfile
		total int
	)
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		statusFilter := ""
		args := []any{tenantID}
		if status != "" {
			statusFilter = " AND status = $2"
			args = append(args, status)
		}
		countQuery := `SELECT COUNT(*) FROM ueba_profiles WHERE tenant_id = $1` + statusFilter
		if err := tx.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}
		args = append(args, limit, offset)
		rows, err := tx.Query(ctx, `
			SELECT
				id, tenant_id, entity_type, entity_id, COALESCE(entity_name, ''), COALESCE(entity_email, ''),
				baseline, observation_count, profile_maturity, first_seen_at, last_seen_at, days_active,
				risk_score::double precision, risk_level, risk_factors,
				risk_last_updated, risk_last_decayed, alert_count_7d, alert_count_30d, last_alert_at,
				status, suppressed_until, COALESCE(suppressed_reason, ''), created_at, updated_at
			FROM ueba_profiles
			WHERE tenant_id = $1`+statusFilter+`
			ORDER BY risk_score DESC, last_seen_at DESC
			LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
			args...,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, scanErr := scanProfile(rows)
			if scanErr != nil {
				return scanErr
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *ProfileRepository) ListRiskRanking(ctx context.Context, tenantID uuid.UUID, limit int) ([]*model.UEBAProfile, error) {
	items, _, err := r.List(ctx, tenantID, limit, 0, model.ProfileStatusActive)
	return items, err
}

func (r *ProfileRepository) ListTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT tenant_id FROM ueba_profiles`)
	if err != nil {
		return nil, fmt.Errorf("list ueba tenant ids: %w", err)
	}
	defer rows.Close()
	var items []uuid.UUID
	for rows.Next() {
		var tenantID uuid.UUID
		if err := rows.Scan(&tenantID); err != nil {
			return nil, err
		}
		items = append(items, tenantID)
	}
	return items, rows.Err()
}

func (r *ProfileRepository) DecayRiskScores(ctx context.Context, tenantID uuid.UUID, rate float64, now time.Time) (int64, error) {
	var affected int64
	err := database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE ueba_profiles
			SET risk_score = GREATEST(0, risk_score * (1 - $2)),
				risk_level = CASE
					WHEN risk_score * (1 - $2) < 25 THEN 'low'
					WHEN risk_score * (1 - $2) < 50 THEN 'medium'
					WHEN risk_score * (1 - $2) < 75 THEN 'high'
					ELSE 'critical'
				END,
				risk_last_decayed = $3,
				updated_at = $3
			WHERE tenant_id = $1 AND risk_score > 0 AND status = 'active'`,
			tenantID, rate, now,
		)
		if err != nil {
			return err
		}
		affected = tag.RowsAffected()
		return nil
	})
	return affected, err
}

type profileScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row profileScanner) (*model.UEBAProfile, error) {
	var (
		item         model.UEBAProfile
		baselineJSON []byte
		factorsJSON  []byte
	)
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.EntityType, &item.EntityID, &item.EntityName, &item.EntityEmail,
		&baselineJSON, &item.ObservationCount, &item.ProfileMaturity, &item.FirstSeenAt, &item.LastSeenAt, &item.DaysActive,
		&item.RiskScore, &item.RiskLevel, &factorsJSON,
		&item.RiskLastUpdated, &item.RiskLastDecayed, &item.AlertCount7D, &item.AlertCount30D, &item.LastAlertAt,
		&item.Status, &item.SuppressedUntil, &item.SuppressedReason, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(baselineJSON) > 0 {
		if err := json.Unmarshal(baselineJSON, &item.Baseline); err != nil {
			return nil, fmt.Errorf("decode ueba profile baseline: %w", err)
		}
	}
	if len(factorsJSON) > 0 {
		if err := json.Unmarshal(factorsJSON, &item.RiskFactors); err != nil {
			return nil, fmt.Errorf("decode ueba profile risk factors: %w", err)
		}
	}
	item.EnsureDefaults()
	return &item, nil
}

func marshalProfileJSON(profile *model.UEBAProfile) ([]byte, []byte, error) {
	baselineJSON, err := json.Marshal(profile.Baseline)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ueba baseline: %w", err)
	}
	factorsJSON, err := json.Marshal(profile.RiskFactors)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ueba risk factors: %w", err)
	}
	return baselineJSON, factorsJSON, nil
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
