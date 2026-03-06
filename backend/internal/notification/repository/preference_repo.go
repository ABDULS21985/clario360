package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/notification/model"
)

// PreferenceRepository handles notification preference CRUD.
type PreferenceRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPreferenceRepository creates a new PreferenceRepository.
func NewPreferenceRepository(db *pgxpool.Pool, logger zerolog.Logger) *PreferenceRepository {
	return &PreferenceRepository{db: db, logger: logger.With().Str("component", "preference_repo").Logger()}
}

// Get returns the preference record for a user in a tenant, or nil if none exists.
func (r *PreferenceRepository) Get(ctx context.Context, userID, tenantID string) (*model.NotificationPreference, error) {
	query := `
		SELECT user_id, tenant_id, global_prefs, per_type_prefs, quiet_hours, digest_config, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND tenant_id = $2`

	var pref model.NotificationPreference
	var globalBytes, perTypeBytes, quietBytes, digestBytes []byte

	err := r.db.QueryRow(ctx, query, userID, tenantID).Scan(
		&pref.UserID, &pref.TenantID,
		&globalBytes, &perTypeBytes, &quietBytes, &digestBytes,
		&pref.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	if err := json.Unmarshal(globalBytes, &pref.GlobalPrefs); err != nil {
		return nil, fmt.Errorf("unmarshal global_prefs: %w", err)
	}
	if len(perTypeBytes) > 0 {
		if err := json.Unmarshal(perTypeBytes, &pref.PerTypePrefs); err != nil {
			return nil, fmt.Errorf("unmarshal per_type_prefs: %w", err)
		}
	}
	if len(quietBytes) > 0 && string(quietBytes) != "null" {
		var qh model.QuietHours
		if err := json.Unmarshal(quietBytes, &qh); err != nil {
			return nil, fmt.Errorf("unmarshal quiet_hours: %w", err)
		}
		pref.QuietHours = &qh
	}
	if len(digestBytes) > 0 {
		if err := json.Unmarshal(digestBytes, &pref.DigestConfig); err != nil {
			return nil, fmt.Errorf("unmarshal digest_config: %w", err)
		}
	}

	return &pref, nil
}

// Upsert creates or updates preference record for a user.
func (r *PreferenceRepository) Upsert(ctx context.Context, pref *model.NotificationPreference) error {
	globalBytes, err := json.Marshal(pref.GlobalPrefs)
	if err != nil {
		return fmt.Errorf("marshal global_prefs: %w", err)
	}

	perTypeBytes, err := json.Marshal(pref.PerTypePrefs)
	if err != nil {
		return fmt.Errorf("marshal per_type_prefs: %w", err)
	}

	var quietBytes []byte
	if pref.QuietHours != nil {
		quietBytes, err = json.Marshal(pref.QuietHours)
		if err != nil {
			return fmt.Errorf("marshal quiet_hours: %w", err)
		}
	}

	digestBytes, err := json.Marshal(pref.DigestConfig)
	if err != nil {
		return fmt.Errorf("marshal digest_config: %w", err)
	}

	query := `
		INSERT INTO notification_preferences (user_id, tenant_id, global_prefs, per_type_prefs, quiet_hours, digest_config, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		ON CONFLICT (user_id, tenant_id) DO UPDATE SET
			global_prefs = EXCLUDED.global_prefs,
			per_type_prefs = EXCLUDED.per_type_prefs,
			quiet_hours = EXCLUDED.quiet_hours,
			digest_config = EXCLUDED.digest_config,
			updated_at = now()`

	_, err = r.db.Exec(ctx, query,
		pref.UserID, pref.TenantID,
		globalBytes, perTypeBytes, quietBytes, digestBytes,
	)
	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}
	return nil
}

// GetDigestSubscribers returns user IDs that have the specified digest type enabled.
func (r *PreferenceRepository) GetDigestSubscribers(ctx context.Context, tenantID, digestType string) ([]string, error) {
	var jsonPath string
	switch digestType {
	case "daily":
		jsonPath = "$.daily"
	case "weekly":
		jsonPath = "$.weekly"
	default:
		return nil, fmt.Errorf("invalid digest type: %s", digestType)
	}

	query := fmt.Sprintf(`
		SELECT user_id FROM notification_preferences
		WHERE tenant_id = $1 AND digest_config @@ '%s == true'`, jsonPath)

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get digest subscribers: %w", err)
	}
	defer rows.Close()

	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, fmt.Errorf("scan user_id: %w", err)
		}
		userIDs = append(userIDs, uid)
	}
	return userIDs, rows.Err()
}

// ListAllTenants returns distinct tenant IDs that have preference records.
func (r *PreferenceRepository) ListAllTenants(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, "SELECT DISTINCT tenant_id FROM notification_preferences")
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}
