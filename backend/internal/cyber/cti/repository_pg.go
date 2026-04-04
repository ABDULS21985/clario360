package cti

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	apperrors "github.com/clario360/platform/internal/errors"

	"github.com/clario360/platform/internal/database"
)

type dbtx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PgRepository implements Repository backed by PostgreSQL.
type PgRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewPgRepository(db *pgxpool.Pool, logger zerolog.Logger) *PgRepository {
	return &PgRepository{db: db, logger: logger}
}

// ---------------------------------------------------------------------------
// Tenant-scoped transactions (matches existing cyber repo pattern)
// ---------------------------------------------------------------------------

func (r *PgRepository) withTenantRead(ctx context.Context, tenantID uuid.UUID, fn func(dbtx) error) error {
	return r.withTenantTx(ctx, tenantID, pgx.TxOptions{AccessMode: pgx.ReadOnly}, fn)
}

func (r *PgRepository) withTenantWrite(ctx context.Context, tenantID uuid.UUID, fn func(dbtx) error) error {
	return r.withTenantTx(ctx, tenantID, pgx.TxOptions{}, fn)
}

func (r *PgRepository) withTenantTx(ctx context.Context, tenantID uuid.UUID, opts pgx.TxOptions, fn func(dbtx) error) error {
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin tenant transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Reference data
// ---------------------------------------------------------------------------

func (r *PgRepository) ListSeverityLevels(ctx context.Context, tenantID uuid.UUID) ([]ThreatSeverityLevel, error) {
	var items []ThreatSeverityLevel
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT id,tenant_id,code,label,color_hex,sort_order,created_at
			FROM cti_threat_severity_levels WHERE tenant_id=$1 ORDER BY sort_order`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var s ThreatSeverityLevel
			if err := rows.Scan(&s.ID, &s.TenantID, &s.Code, &s.Label, &s.ColorHex, &s.SortOrder, &s.CreatedAt); err != nil {
				return err
			}
			items = append(items, s)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) ListCategories(ctx context.Context, tenantID uuid.UUID) ([]ThreatCategory, error) {
	var items []ThreatCategory
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT id,tenant_id,code,label,description,mitre_tactic_ids,created_at
			FROM cti_threat_categories WHERE tenant_id=$1 ORDER BY code`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c ThreatCategory
			if err := rows.Scan(&c.ID, &c.TenantID, &c.Code, &c.Label, &c.Description, &c.MitreTacticIDs, &c.CreatedAt); err != nil {
				return err
			}
			items = append(items, c)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) ListRegions(ctx context.Context, tenantID uuid.UUID, parentID *uuid.UUID) ([]GeographicRegion, error) {
	var items []GeographicRegion
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		q := `SELECT id,tenant_id,code,label,parent_region_id,latitude,longitude,iso_country_code,created_at
			FROM cti_geographic_regions WHERE tenant_id=$1`
		args := []any{tenantID}
		if parentID != nil {
			q += ` AND parent_region_id=$2`
			args = append(args, *parentID)
		}
		q += ` ORDER BY label`
		rows, err := db.Query(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var g GeographicRegion
			if err := rows.Scan(&g.ID, &g.TenantID, &g.Code, &g.Label, &g.ParentRegionID, &g.Latitude, &g.Longitude, &g.ISOCountryCode, &g.CreatedAt); err != nil {
				return err
			}
			items = append(items, g)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) ListSectors(ctx context.Context, tenantID uuid.UUID) ([]IndustrySector, error) {
	var items []IndustrySector
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT id,tenant_id,code,label,description,naics_code,created_at
			FROM cti_industry_sectors WHERE tenant_id=$1 ORDER BY label`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var s IndustrySector
			if err := rows.Scan(&s.ID, &s.TenantID, &s.Code, &s.Label, &s.Description, &s.NAICSCode, &s.CreatedAt); err != nil {
				return err
			}
			items = append(items, s)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) ListDataSources(ctx context.Context, tenantID uuid.UUID) ([]DataSource, error) {
	var items []DataSource
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT id,tenant_id,name,source_type,url,api_endpoint,api_key_vault_path,
			reliability_score,is_active,last_polled_at,poll_interval_seconds,created_at,updated_at
			FROM cti_data_sources WHERE tenant_id=$1 ORDER BY name`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d DataSource
			if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.SourceType, &d.URL, &d.APIEndpoint, &d.APIKeyVaultPath,
				&d.ReliabilityScore, &d.IsActive, &d.LastPolledAt, &d.PollIntervalSecs, &d.CreatedAt, &d.UpdatedAt); err != nil {
				return err
			}
			items = append(items, d)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) GetSeverityByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ThreatSeverityLevel, error) {
	var s ThreatSeverityLevel
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT id,tenant_id,code,label,color_hex,sort_order,created_at
			FROM cti_threat_severity_levels WHERE tenant_id=$1 AND code=$2`, tenantID, code).
			Scan(&s.ID, &s.TenantID, &s.Code, &s.Label, &s.ColorHex, &s.SortOrder, &s.CreatedAt)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PgRepository) GetCategoryByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ThreatCategory, error) {
	var c ThreatCategory
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT id,tenant_id,code,label,description,mitre_tactic_ids,created_at
			FROM cti_threat_categories WHERE tenant_id=$1 AND code=$2`, tenantID, code).
			Scan(&c.ID, &c.TenantID, &c.Code, &c.Label, &c.Description, &c.MitreTacticIDs, &c.CreatedAt)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *PgRepository) GetSectorByCode(ctx context.Context, tenantID uuid.UUID, code string) (*IndustrySector, error) {
	var s IndustrySector
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT id,tenant_id,code,label,description,naics_code,created_at
			FROM cti_industry_sectors WHERE tenant_id=$1 AND code=$2`, tenantID, code).
			Scan(&s.ID, &s.TenantID, &s.Code, &s.Label, &s.Description, &s.NAICSCode, &s.CreatedAt)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *PgRepository) GetSourceByName(ctx context.Context, tenantID uuid.UUID, name string) (*DataSource, error) {
	var d DataSource
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT id,tenant_id,name,source_type,url,api_endpoint,api_key_vault_path,
			reliability_score,is_active,last_polled_at,poll_interval_seconds,created_at,updated_at
			FROM cti_data_sources WHERE tenant_id=$1 AND name=$2`, tenantID, name).
			Scan(&d.ID, &d.TenantID, &d.Name, &d.SourceType, &d.URL, &d.APIEndpoint, &d.APIKeyVaultPath,
				&d.ReliabilityScore, &d.IsActive, &d.LastPolledAt, &d.PollIntervalSecs, &d.CreatedAt, &d.UpdatedAt)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

// ---------------------------------------------------------------------------
// Threat events
// ---------------------------------------------------------------------------

const eventDetailSelect = `SELECT e.id,e.tenant_id,e.event_type,e.title,e.description,
	e.severity_id,e.category_id,e.source_id,e.source_reference,e.confidence_score,
	e.origin_latitude,e.origin_longitude,e.origin_country_code,e.origin_city,e.origin_region_id,
	e.target_sector_id,e.target_org_name,e.target_country_code,e.ioc_type,e.ioc_value,
	e.mitre_technique_ids,e.raw_payload,e.is_false_positive,e.resolved_at,e.resolved_by,
	e.first_seen_at,e.last_seen_at,e.created_at,e.updated_at,e.created_by,e.updated_by,e.deleted_at,
	COALESCE(sl.code,'') AS severity_code, COALESCE(sl.label,'') AS severity_label,
	COALESCE(tc.code,'') AS category_code, COALESCE(tc.label,'') AS category_label,
	COALESCE(ds.name,'') AS source_name,
	COALESCE(sec.label,'') AS sector_label
FROM cti_threat_events e
LEFT JOIN cti_threat_severity_levels sl ON e.severity_id = sl.id
LEFT JOIN cti_threat_categories tc ON e.category_id = tc.id
LEFT JOIN cti_data_sources ds ON e.source_id = ds.id
LEFT JOIN cti_industry_sectors sec ON e.target_sector_id = sec.id`

func scanEventDetail(row pgx.Row) (*ThreatEventDetail, error) {
	var d ThreatEventDetail
	err := row.Scan(
		&d.ID, &d.TenantID, &d.EventType, &d.Title, &d.Description,
		&d.SeverityID, &d.CategoryID, &d.SourceID, &d.SourceReference, &d.ConfidenceScore,
		&d.OriginLatitude, &d.OriginLongitude, &d.OriginCountryCode, &d.OriginCity, &d.OriginRegionID,
		&d.TargetSectorID, &d.TargetOrgName, &d.TargetCountryCode, &d.IOCType, &d.IOCValue,
		&d.MitreTechniqueIDs, &d.RawPayload, &d.IsFalsePositive, &d.ResolvedAt, &d.ResolvedBy,
		&d.FirstSeenAt, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy, &d.DeletedAt,
		&d.SeverityCode, &d.SeverityLabel,
		&d.CategoryCode, &d.CategoryLabel,
		&d.SourceName,
		&d.SectorLabel,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func scanEventDetailRows(rows pgx.Rows) (*ThreatEventDetail, error) {
	var d ThreatEventDetail
	err := rows.Scan(
		&d.ID, &d.TenantID, &d.EventType, &d.Title, &d.Description,
		&d.SeverityID, &d.CategoryID, &d.SourceID, &d.SourceReference, &d.ConfidenceScore,
		&d.OriginLatitude, &d.OriginLongitude, &d.OriginCountryCode, &d.OriginCity, &d.OriginRegionID,
		&d.TargetSectorID, &d.TargetOrgName, &d.TargetCountryCode, &d.IOCType, &d.IOCValue,
		&d.MitreTechniqueIDs, &d.RawPayload, &d.IsFalsePositive, &d.ResolvedAt, &d.ResolvedBy,
		&d.FirstSeenAt, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy, &d.DeletedAt,
		&d.SeverityCode, &d.SeverityLabel,
		&d.CategoryCode, &d.CategoryLabel,
		&d.SourceName,
		&d.SectorLabel,
	)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *PgRepository) CreateThreatEvent(ctx context.Context, tenantID uuid.UUID, event *ThreatEvent) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `INSERT INTO cti_threat_events
			(id,tenant_id,event_type,title,description,severity_id,category_id,source_id,source_reference,
			confidence_score,origin_latitude,origin_longitude,origin_country_code,origin_city,origin_region_id,
			target_sector_id,target_org_name,target_country_code,ioc_type,ioc_value,mitre_technique_ids,
			raw_payload,first_seen_at,last_seen_at,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)`,
			event.ID, tenantID, event.EventType, event.Title, event.Description,
			event.SeverityID, event.CategoryID, event.SourceID, event.SourceReference,
			event.ConfidenceScore, event.OriginLatitude, event.OriginLongitude,
			event.OriginCountryCode, event.OriginCity, event.OriginRegionID,
			event.TargetSectorID, event.TargetOrgName, event.TargetCountryCode,
			event.IOCType, event.IOCValue, event.MitreTechniqueIDs,
			event.RawPayload, event.FirstSeenAt, event.LastSeenAt, event.CreatedBy)
		return err
	})
}

func (r *PgRepository) GetThreatEvent(ctx context.Context, tenantID, eventID uuid.UUID) (*ThreatEventDetail, error) {
	var item *ThreatEventDetail
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		row := db.QueryRow(ctx, eventDetailSelect+` WHERE e.tenant_id=$1 AND e.id=$2 AND e.deleted_at IS NULL`, tenantID, eventID)
		d, err := scanEventDetail(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return apperrors.ErrNotFound
			}
			return fmt.Errorf("get threat event: %w", err)
		}
		item = d
		return nil
	})
	return item, err
}

func (r *PgRepository) ListThreatEvents(ctx context.Context, tenantID uuid.UUID, f ThreatEventFilters) ([]ThreatEventDetail, int, error) {
	var items []ThreatEventDetail
	var total int

	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		qb := database.NewQueryBuilder(eventDetailSelect)
		qb.Where("e.tenant_id = ?", tenantID)
		qb.Where("e.deleted_at IS NULL")

		if f.Search != nil {
			s := "%" + *f.Search + "%"
			qb.Where("(e.title ILIKE ? OR e.ioc_value ILIKE ?)", s, s)
		}
		if len(f.Severities) > 0 {
			qb.WhereIn("sl.code", f.Severities)
		}
		if len(f.Categories) > 0 {
			qb.WhereIn("tc.code", f.Categories)
		}
		if len(f.EventTypes) > 0 {
			qb.WhereIn("e.event_type", f.EventTypes)
		}
		if len(f.OriginCountries) > 0 {
			qb.WhereIn("e.origin_country_code", f.OriginCountries)
		}
		if len(f.TargetCountries) > 0 {
			qb.WhereIn("e.target_country_code", f.TargetCountries)
		}
		if f.IOCType != nil {
			qb.Where("e.ioc_type = ?", *f.IOCType)
		}
		if f.IsFalsePositive != nil {
			qb.Where("e.is_false_positive = ?", *f.IsFalsePositive)
		}
		if len(f.TargetSectors) > 0 {
			qb.WhereIn("sec.code", f.TargetSectors)
		}
		if f.MinConfidence != nil {
			qb.Where("e.confidence_score >= ?", *f.MinConfidence)
		}
		if f.MaxConfidence != nil {
			qb.Where("e.confidence_score <= ?", *f.MaxConfidence)
		}
		if f.FirstSeenFrom != nil {
			qb.Where("e.first_seen_at >= ?", *f.FirstSeenFrom)
		}
		if f.FirstSeenTo != nil {
			qb.Where("e.first_seen_at <= ?", *f.FirstSeenTo)
		}
		if f.SourceID != nil {
			qb.Where("e.source_id = ?", *f.SourceID)
		}

		sortAllowlist := []string{"first_seen_at", "last_seen_at", "created_at", "confidence_score", "title"}
		qb.OrderBy(f.Sort, f.Order, sortAllowlist)
		qb.Paginate(f.Page, f.PerPage)

		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("count events: %w", err)
		}

		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("list events: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanEventDetailRows(rows)
			if err != nil {
				return err
			}
			items = append(items, *d)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *PgRepository) UpdateThreatEvent(ctx context.Context, tenantID, eventID uuid.UUID, updates map[string]interface{}) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		sets, args := buildUpdateSets(updates, tenantID, eventID)
		if len(sets) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`UPDATE cti_threat_events SET %s, updated_at=NOW() WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, strings.Join(sets, ","))
		tag, err := db.Exec(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("update event: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) DeleteThreatEvent(ctx context.Context, tenantID, eventID, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `UPDATE cti_threat_events SET deleted_at=NOW(), updated_by=$3
			WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, eventID, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) MarkFalsePositive(ctx context.Context, tenantID, eventID, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `UPDATE cti_threat_events SET is_false_positive=true, updated_by=$3, updated_at=NOW()
			WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, eventID, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) ResolveThreatEvent(ctx context.Context, tenantID, eventID, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `UPDATE cti_threat_events SET resolved_at=NOW(), resolved_by=$3, updated_by=$3, updated_at=NOW()
			WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, eventID, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Event tags
// ---------------------------------------------------------------------------

func (r *PgRepository) AddEventTags(ctx context.Context, tenantID, eventID uuid.UUID, tags []string) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		for _, tag := range tags {
			_, err := db.Exec(ctx, `INSERT INTO cti_threat_event_tags (tenant_id, event_id, tag)
				VALUES ($1,$2,$3) ON CONFLICT (tenant_id, event_id, tag) DO NOTHING`, tenantID, eventID, tag)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PgRepository) RemoveEventTag(ctx context.Context, tenantID, eventID uuid.UUID, tag string) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `DELETE FROM cti_threat_event_tags
			WHERE tenant_id=$1 AND event_id=$2 AND tag=$3`, tenantID, eventID, tag)
		return err
	})
}

func (r *PgRepository) GetEventTags(ctx context.Context, tenantID, eventID uuid.UUID) ([]string, error) {
	var tags []string
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT tag FROM cti_threat_event_tags
			WHERE tenant_id=$1 AND event_id=$2 ORDER BY tag`, tenantID, eventID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				return err
			}
			tags = append(tags, t)
		}
		return rows.Err()
	})
	return tags, err
}

// ---------------------------------------------------------------------------
// Threat actors
// ---------------------------------------------------------------------------

func (r *PgRepository) CreateThreatActor(ctx context.Context, tenantID uuid.UUID, actor *ThreatActor) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		refs := json.RawMessage("{}")
		if actor.ExternalReferences != nil {
			refs = actor.ExternalReferences
		}
		_, err := db.Exec(ctx, `INSERT INTO cti_threat_actors
			(id,tenant_id,name,aliases,actor_type,origin_country_code,origin_region_id,sophistication_level,
			primary_motivation,description,mitre_group_id,external_references,is_active,risk_score,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			actor.ID, tenantID, actor.Name, actor.Aliases, actor.ActorType,
			actor.OriginCountryCode, actor.OriginRegionID, actor.SophisticationLevel,
			actor.PrimaryMotivation, actor.Description, actor.MitreGroupID,
			refs, actor.IsActive, actor.RiskScore, actor.CreatedBy)
		return err
	})
}

func (r *PgRepository) GetThreatActor(ctx context.Context, tenantID, actorID uuid.UUID) (*ThreatActor, error) {
	var a ThreatActor
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT id,tenant_id,name,aliases,actor_type,origin_country_code,origin_region_id,
			sophistication_level,primary_motivation,description,first_observed_at,last_activity_at,
			mitre_group_id,external_references,is_active,risk_score,
			created_at,updated_at,created_by,updated_by,deleted_at
			FROM cti_threat_actors WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, actorID).
			Scan(&a.ID, &a.TenantID, &a.Name, &a.Aliases, &a.ActorType,
				&a.OriginCountryCode, &a.OriginRegionID, &a.SophisticationLevel,
				&a.PrimaryMotivation, &a.Description, &a.FirstObservedAt, &a.LastActivityAt,
				&a.MitreGroupID, &a.ExternalReferences, &a.IsActive, &a.RiskScore,
				&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy, &a.DeletedAt)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (r *PgRepository) ListThreatActors(ctx context.Context, tenantID uuid.UUID, f ThreatActorFilters) ([]ThreatActor, int, error) {
	var items []ThreatActor
	var total int
	baseSQL := `SELECT id,tenant_id,name,aliases,actor_type,origin_country_code,origin_region_id,
		sophistication_level,primary_motivation,description,first_observed_at,last_activity_at,
		mitre_group_id,external_references,is_active,risk_score,
		created_at,updated_at,created_by,updated_by,deleted_at
		FROM cti_threat_actors`

	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		qb := database.NewQueryBuilder(baseSQL)
		qb.Where("tenant_id = ?", tenantID)
		qb.Where("deleted_at IS NULL")
		if f.Search != nil {
			s := "%" + *f.Search + "%"
			qb.Where("(name ILIKE ? OR ? = ANY(aliases))", s, *f.Search)
		}
		if len(f.ActorTypes) > 0 {
			qb.WhereIn("actor_type", f.ActorTypes)
		}
		if f.IsActive != nil {
			qb.Where("is_active = ?", *f.IsActive)
		}
		qb.OrderBy(f.Sort, f.Order, []string{"risk_score", "name", "last_activity_at", "created_at"})
		qb.Paginate(f.Page, f.PerPage)

		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return err
		}
		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a ThreatActor
			if err := rows.Scan(&a.ID, &a.TenantID, &a.Name, &a.Aliases, &a.ActorType,
				&a.OriginCountryCode, &a.OriginRegionID, &a.SophisticationLevel,
				&a.PrimaryMotivation, &a.Description, &a.FirstObservedAt, &a.LastActivityAt,
				&a.MitreGroupID, &a.ExternalReferences, &a.IsActive, &a.RiskScore,
				&a.CreatedAt, &a.UpdatedAt, &a.CreatedBy, &a.UpdatedBy, &a.DeletedAt); err != nil {
				return err
			}
			items = append(items, a)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *PgRepository) UpdateThreatActor(ctx context.Context, tenantID, actorID uuid.UUID, updates map[string]interface{}) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		sets, args := buildUpdateSets(updates, tenantID, actorID)
		if len(sets) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`UPDATE cti_threat_actors SET %s, updated_at=NOW() WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, strings.Join(sets, ","))
		tag, err := db.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) DeleteThreatActor(ctx context.Context, tenantID, actorID, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `UPDATE cti_threat_actors SET deleted_at=NOW(), updated_by=$3
			WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, actorID, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

const campaignDetailSelect = `SELECT c.id,c.tenant_id,c.campaign_code,c.name,c.description,c.status,
	c.severity_id,c.primary_actor_id,c.target_sectors,c.target_regions,c.target_description,
	c.mitre_technique_ids,c.ttps_summary,c.ioc_count,c.event_count,
	c.first_seen_at,c.last_seen_at,c.resolved_at,c.resolved_by,c.external_references,
	c.created_at,c.updated_at,c.created_by,c.updated_by,c.deleted_at,
	COALESCE(a.name,'') AS actor_name,
	COALESCE(sl.code,'') AS severity_code, COALESCE(sl.label,'') AS severity_label
FROM cti_campaigns c
LEFT JOIN cti_threat_actors a ON c.primary_actor_id = a.id
LEFT JOIN cti_threat_severity_levels sl ON c.severity_id = sl.id`

func scanCampaignDetail(rows pgx.Rows) (*CampaignDetail, error) {
	var d CampaignDetail
	err := rows.Scan(
		&d.ID, &d.TenantID, &d.CampaignCode, &d.Name, &d.Description, &d.Status,
		&d.SeverityID, &d.PrimaryActorID, &d.TargetSectors, &d.TargetRegions, &d.TargetDescription,
		&d.MitreTechniqueIDs, &d.TTPsSummary, &d.IOCCount, &d.EventCount,
		&d.FirstSeenAt, &d.LastSeenAt, &d.ResolvedAt, &d.ResolvedBy, &d.ExternalRefs,
		&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy, &d.DeletedAt,
		&d.ActorName, &d.SeverityCode, &d.SeverityLabel,
	)
	return &d, err
}

func (r *PgRepository) CreateCampaign(ctx context.Context, tenantID uuid.UUID, c *Campaign) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		refs := c.ExternalRefs
		if refs == nil {
			refs = json.RawMessage("{}")
		}
		_, err := db.Exec(ctx, `INSERT INTO cti_campaigns
			(id,tenant_id,campaign_code,name,description,status,severity_id,primary_actor_id,
			target_sectors,target_regions,target_description,mitre_technique_ids,ttps_summary,
			first_seen_at,last_seen_at,external_references,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
			c.ID, tenantID, c.CampaignCode, c.Name, c.Description, c.Status,
			c.SeverityID, c.PrimaryActorID, c.TargetSectors, c.TargetRegions,
			c.TargetDescription, c.MitreTechniqueIDs, c.TTPsSummary,
			c.FirstSeenAt, c.LastSeenAt, refs, c.CreatedBy)
		if err != nil {
			if isUniqueViolation(err) {
				return apperrors.ErrConflict
			}
			return err
		}
		return nil
	})
}

func (r *PgRepository) GetCampaign(ctx context.Context, tenantID, campaignID uuid.UUID) (*CampaignDetail, error) {
	var item *CampaignDetail
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, campaignDetailSelect+` WHERE c.tenant_id=$1 AND c.id=$2 AND c.deleted_at IS NULL`, tenantID, campaignID)
		if err != nil {
			return err
		}
		defer rows.Close()
		if !rows.Next() {
			return apperrors.ErrNotFound
		}
		item, err = scanCampaignDetail(rows)
		return err
	})
	return item, err
}

func (r *PgRepository) ListCampaigns(ctx context.Context, tenantID uuid.UUID, f CampaignFilters) ([]CampaignDetail, int, error) {
	var items []CampaignDetail
	var total int
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		qb := database.NewQueryBuilder(campaignDetailSelect)
		qb.Where("c.tenant_id = ?", tenantID)
		qb.Where("c.deleted_at IS NULL")
		if f.Search != nil {
			s := "%" + *f.Search + "%"
			qb.Where("(c.name ILIKE ? OR c.campaign_code ILIKE ?)", s, s)
		}
		if len(f.Statuses) > 0 {
			qb.WhereIn("c.status", f.Statuses)
		}
		if len(f.Severities) > 0 {
			qb.WhereIn("sl.code", f.Severities)
		}
		if f.ActorID != nil {
			qb.Where("c.primary_actor_id = ?", *f.ActorID)
		}
		if f.FirstSeenFrom != nil {
			qb.Where("c.first_seen_at >= ?", *f.FirstSeenFrom)
		}
		if f.FirstSeenTo != nil {
			qb.Where("c.first_seen_at <= ?", *f.FirstSeenTo)
		}
		qb.OrderBy(f.Sort, f.Order, []string{"first_seen_at", "last_seen_at", "name", "status", "ioc_count", "event_count", "created_at"})
		qb.Paginate(f.Page, f.PerPage)

		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return err
		}
		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanCampaignDetail(rows)
			if err != nil {
				return err
			}
			items = append(items, *d)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *PgRepository) UpdateCampaign(ctx context.Context, tenantID, campaignID uuid.UUID, updates map[string]interface{}) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		sets, args := buildUpdateSets(updates, tenantID, campaignID)
		if len(sets) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`UPDATE cti_campaigns SET %s, updated_at=NOW() WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, strings.Join(sets, ","))
		tag, err := db.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) DeleteCampaign(ctx context.Context, tenantID, campaignID, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `UPDATE cti_campaigns SET deleted_at=NOW(), updated_by=$3
			WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, tenantID, campaignID, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) UpdateCampaignStatus(ctx context.Context, tenantID, campaignID uuid.UUID, status string, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		q := `UPDATE cti_campaigns SET status=$3, updated_by=$4, updated_at=NOW()`
		if status == "resolved" {
			q += `, resolved_at=NOW(), resolved_by=$4`
		}
		q += ` WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`
		tag, err := db.Exec(ctx, q, tenantID, campaignID, status, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Campaign events
// ---------------------------------------------------------------------------

func (r *PgRepository) LinkEventToCampaign(ctx context.Context, tenantID, campaignID, eventID uuid.UUID, userID *uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `INSERT INTO cti_campaign_events (tenant_id, campaign_id, event_id, linked_by)
			VALUES ($1,$2,$3,$4) ON CONFLICT (tenant_id, campaign_id, event_id) DO NOTHING`,
			tenantID, campaignID, eventID, userID)
		if err != nil {
			return err
		}
		// Update event_count
		_, err = db.Exec(ctx, `UPDATE cti_campaigns SET event_count = (
			SELECT count(*) FROM cti_campaign_events WHERE campaign_id=$2 AND tenant_id=$1
		) WHERE tenant_id=$1 AND id=$2`, tenantID, campaignID)
		return err
	})
}

func (r *PgRepository) UnlinkEventFromCampaign(ctx context.Context, tenantID, campaignID, eventID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `DELETE FROM cti_campaign_events
			WHERE tenant_id=$1 AND campaign_id=$2 AND event_id=$3`, tenantID, campaignID, eventID)
		if err != nil {
			return err
		}
		_, err = db.Exec(ctx, `UPDATE cti_campaigns SET event_count = (
			SELECT count(*) FROM cti_campaign_events WHERE campaign_id=$2 AND tenant_id=$1
		) WHERE tenant_id=$1 AND id=$2`, tenantID, campaignID)
		return err
	})
}

func (r *PgRepository) ListCampaignEvents(ctx context.Context, tenantID, campaignID uuid.UUID, p ListParams) ([]ThreatEventDetail, int, error) {
	var items []ThreatEventDetail
	var total int
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		if err := db.QueryRow(ctx, `SELECT count(*) FROM cti_campaign_events ce
			JOIN cti_threat_events e ON ce.event_id=e.id
			WHERE ce.tenant_id=$1 AND ce.campaign_id=$2 AND e.deleted_at IS NULL`, tenantID, campaignID).Scan(&total); err != nil {
			return err
		}
		offset := (p.Page - 1) * p.PerPage
		rows, err := db.Query(ctx, eventDetailSelect+`
			JOIN cti_campaign_events ce ON e.id = ce.event_id
			WHERE ce.tenant_id=$1 AND ce.campaign_id=$2 AND e.deleted_at IS NULL
			ORDER BY e.first_seen_at DESC LIMIT $3 OFFSET $4`, tenantID, campaignID, p.PerPage, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanEventDetailRows(rows)
			if err != nil {
				return err
			}
			items = append(items, *d)
		}
		return rows.Err()
	})
	return items, total, err
}

// ---------------------------------------------------------------------------
// Campaign IOCs
// ---------------------------------------------------------------------------

func (r *PgRepository) CreateCampaignIOC(ctx context.Context, tenantID uuid.UUID, ioc *CampaignIOC) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `INSERT INTO cti_campaign_iocs
			(id,tenant_id,campaign_id,ioc_type,ioc_value,confidence_score,first_seen_at,last_seen_at,is_active,source_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			ioc.ID, tenantID, ioc.CampaignID, ioc.IOCType, ioc.IOCValue,
			ioc.ConfidenceScore, ioc.FirstSeenAt, ioc.LastSeenAt, ioc.IsActive, ioc.SourceID)
		if err != nil {
			return err
		}
		// Update ioc_count
		_, err = db.Exec(ctx, `UPDATE cti_campaigns SET ioc_count = (
			SELECT count(*) FROM cti_campaign_iocs WHERE campaign_id=$2 AND tenant_id=$1
		) WHERE tenant_id=$1 AND id=$2`, tenantID, ioc.CampaignID)
		return err
	})
}

func (r *PgRepository) ListCampaignIOCs(ctx context.Context, tenantID, campaignID uuid.UUID, p ListParams) ([]CampaignIOC, int, error) {
	var items []CampaignIOC
	var total int
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		if err := db.QueryRow(ctx, `SELECT count(*) FROM cti_campaign_iocs WHERE tenant_id=$1 AND campaign_id=$2`,
			tenantID, campaignID).Scan(&total); err != nil {
			return err
		}
		offset := (p.Page - 1) * p.PerPage
		rows, err := db.Query(ctx, `SELECT id,tenant_id,campaign_id,ioc_type,ioc_value,confidence_score,
			first_seen_at,last_seen_at,is_active,source_id,created_at,updated_at
			FROM cti_campaign_iocs WHERE tenant_id=$1 AND campaign_id=$2
			ORDER BY created_at DESC LIMIT $3 OFFSET $4`, tenantID, campaignID, p.PerPage, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var i CampaignIOC
			if err := rows.Scan(&i.ID, &i.TenantID, &i.CampaignID, &i.IOCType, &i.IOCValue,
				&i.ConfidenceScore, &i.FirstSeenAt, &i.LastSeenAt, &i.IsActive, &i.SourceID,
				&i.CreatedAt, &i.UpdatedAt); err != nil {
				return err
			}
			items = append(items, i)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *PgRepository) DeleteCampaignIOC(ctx context.Context, tenantID, iocID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		// Get campaign_id first
		var campaignID uuid.UUID
		err := db.QueryRow(ctx, `SELECT campaign_id FROM cti_campaign_iocs WHERE tenant_id=$1 AND id=$2`,
			tenantID, iocID).Scan(&campaignID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return apperrors.ErrNotFound
			}
			return err
		}
		if _, err := db.Exec(ctx, `DELETE FROM cti_campaign_iocs WHERE tenant_id=$1 AND id=$2`, tenantID, iocID); err != nil {
			return err
		}
		_, err = db.Exec(ctx, `UPDATE cti_campaigns SET ioc_count = (
			SELECT count(*) FROM cti_campaign_iocs WHERE campaign_id=$2 AND tenant_id=$1
		) WHERE tenant_id=$1 AND id=$2`, tenantID, campaignID)
		return err
	})
}

// ---------------------------------------------------------------------------
// Brand abuse
// ---------------------------------------------------------------------------

func (r *PgRepository) CreateMonitoredBrand(ctx context.Context, tenantID uuid.UUID, brand *MonitoredBrand) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `INSERT INTO cti_monitored_brands (id,tenant_id,brand_name,domain_pattern,keywords,is_active,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			brand.ID, tenantID, brand.BrandName, brand.DomainPattern, brand.Keywords, brand.IsActive, brand.CreatedBy)
		if err != nil {
			if isUniqueViolation(err) {
				return apperrors.ErrConflict
			}
			return err
		}
		return nil
	})
}

func (r *PgRepository) ListMonitoredBrands(ctx context.Context, tenantID uuid.UUID) ([]MonitoredBrand, error) {
	var items []MonitoredBrand
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT id,tenant_id,brand_name,domain_pattern,logo_file_id,keywords,is_active,
			created_at,updated_at,created_by,updated_by
			FROM cti_monitored_brands WHERE tenant_id=$1 ORDER BY brand_name`, tenantID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var b MonitoredBrand
			if err := rows.Scan(&b.ID, &b.TenantID, &b.BrandName, &b.DomainPattern, &b.LogoFileID,
				&b.Keywords, &b.IsActive, &b.CreatedAt, &b.UpdatedAt, &b.CreatedBy, &b.UpdatedBy); err != nil {
				return err
			}
			items = append(items, b)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) UpdateMonitoredBrand(ctx context.Context, tenantID, brandID uuid.UUID, updates map[string]interface{}) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		sets, args := buildUpdateSets(updates, tenantID, brandID)
		if len(sets) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`UPDATE cti_monitored_brands SET %s, updated_at=NOW() WHERE tenant_id=$1 AND id=$2`, strings.Join(sets, ","))
		tag, err := db.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) DeleteMonitoredBrand(ctx context.Context, tenantID, brandID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		tag, err := db.Exec(ctx, `DELETE FROM cti_monitored_brands WHERE tenant_id=$1 AND id=$2`, tenantID, brandID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) CreateBrandAbuseIncident(ctx context.Context, tenantID uuid.UUID, inc *BrandAbuseIncident) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `INSERT INTO cti_brand_abuse_incidents
			(id,tenant_id,brand_id,malicious_domain,abuse_type,risk_level,region_id,source_id,
			whois_registrant,ssl_issuer,hosting_ip,hosting_asn,takedown_status,first_detected_at,last_detected_at,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::inet,$12,$13,$14,$15,$16)`,
			inc.ID, tenantID, inc.BrandID, inc.MaliciousDomain, inc.AbuseType, inc.RiskLevel,
			inc.RegionID, inc.SourceID, inc.WhoisRegistrant, inc.SSLIssuer,
			inc.HostingIP, inc.HostingASN, inc.TakedownStatus,
			inc.FirstDetectedAt, inc.LastDetectedAt, inc.CreatedBy)
		return err
	})
}

func (r *PgRepository) GetBrandAbuseIncident(ctx context.Context, tenantID, incidentID uuid.UUID) (*BrandAbuseDetail, error) {
	var d BrandAbuseDetail
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT i.id,i.tenant_id,i.brand_id,i.malicious_domain,i.abuse_type,i.risk_level,
			i.region_id,i.detection_count,i.source_id,i.whois_registrant,i.whois_created_date,
			i.ssl_issuer,i.hosting_ip,i.hosting_asn,i.screenshot_file_id,
			i.takedown_status,i.takedown_requested_at,i.taken_down_at,
			i.first_detected_at,i.last_detected_at,i.created_at,i.updated_at,i.created_by,i.updated_by,i.deleted_at,
			COALESCE(b.brand_name,'') AS brand_name,
			COALESCE(gr.label,'') AS region_label
			FROM cti_brand_abuse_incidents i
			LEFT JOIN cti_monitored_brands b ON i.brand_id = b.id
			LEFT JOIN cti_geographic_regions gr ON i.region_id = gr.id
			WHERE i.tenant_id=$1 AND i.id=$2 AND i.deleted_at IS NULL`, tenantID, incidentID).
			Scan(&d.ID, &d.TenantID, &d.BrandID, &d.MaliciousDomain, &d.AbuseType, &d.RiskLevel,
				&d.RegionID, &d.DetectionCount, &d.SourceID, &d.WhoisRegistrant, &d.WhoisCreatedDate,
				&d.SSLIssuer, &d.HostingIP, &d.HostingASN, &d.ScreenshotFileID,
				&d.TakedownStatus, &d.TakedownRequestedAt, &d.TakenDownAt,
				&d.FirstDetectedAt, &d.LastDetectedAt, &d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy, &d.DeletedAt,
				&d.BrandName, &d.RegionLabel)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

func (r *PgRepository) ListBrandAbuseIncidents(ctx context.Context, tenantID uuid.UUID, f BrandAbuseFilters) ([]BrandAbuseDetail, int, error) {
	var items []BrandAbuseDetail
	var total int
	baseSQL := `SELECT i.id,i.tenant_id,i.brand_id,i.malicious_domain,i.abuse_type,i.risk_level,
		i.region_id,i.detection_count,i.source_id,i.whois_registrant,i.whois_created_date,
		i.ssl_issuer,i.hosting_ip,i.hosting_asn,i.screenshot_file_id,
		i.takedown_status,i.takedown_requested_at,i.taken_down_at,
		i.first_detected_at,i.last_detected_at,i.created_at,i.updated_at,i.created_by,i.updated_by,i.deleted_at,
		COALESCE(b.brand_name,'') AS brand_name,
		COALESCE(gr.label,'') AS region_label
		FROM cti_brand_abuse_incidents i
		LEFT JOIN cti_monitored_brands b ON i.brand_id = b.id
		LEFT JOIN cti_geographic_regions gr ON i.region_id = gr.id`

	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		qb := database.NewQueryBuilder(baseSQL)
		qb.Where("i.tenant_id = ?", tenantID)
		qb.Where("i.deleted_at IS NULL")
		if f.BrandID != nil {
			qb.Where("i.brand_id = ?", *f.BrandID)
		}
		if len(f.RiskLevels) > 0 {
			qb.WhereIn("i.risk_level", f.RiskLevels)
		}
		if len(f.AbuseTypes) > 0 {
			qb.WhereIn("i.abuse_type", f.AbuseTypes)
		}
		if len(f.TakedownStatuses) > 0 {
			qb.WhereIn("i.takedown_status", f.TakedownStatuses)
		}
		qb.OrderBy(f.Sort, f.Order, []string{"first_detected_at", "last_detected_at", "risk_level", "created_at"})
		qb.Paginate(f.Page, f.PerPage)

		countSQL, countArgs := qb.BuildCount()
		if err := db.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
			return err
		}
		sql, args := qb.Build()
		rows, err := db.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var d BrandAbuseDetail
			if err := rows.Scan(&d.ID, &d.TenantID, &d.BrandID, &d.MaliciousDomain, &d.AbuseType, &d.RiskLevel,
				&d.RegionID, &d.DetectionCount, &d.SourceID, &d.WhoisRegistrant, &d.WhoisCreatedDate,
				&d.SSLIssuer, &d.HostingIP, &d.HostingASN, &d.ScreenshotFileID,
				&d.TakedownStatus, &d.TakedownRequestedAt, &d.TakenDownAt,
				&d.FirstDetectedAt, &d.LastDetectedAt, &d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy, &d.DeletedAt,
				&d.BrandName, &d.RegionLabel); err != nil {
				return err
			}
			items = append(items, d)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *PgRepository) UpdateBrandAbuseIncident(ctx context.Context, tenantID, incidentID uuid.UUID, updates map[string]interface{}) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		sets, args := buildUpdateSets(updates, tenantID, incidentID)
		if len(sets) == 0 {
			return nil
		}
		sql := fmt.Sprintf(`UPDATE cti_brand_abuse_incidents SET %s, updated_at=NOW() WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`, strings.Join(sets, ","))
		tag, err := db.Exec(ctx, sql, args...)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

func (r *PgRepository) UpdateTakedownStatus(ctx context.Context, tenantID, incidentID uuid.UUID, status string, userID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		q := `UPDATE cti_brand_abuse_incidents SET takedown_status=$3, updated_by=$4, updated_at=NOW()`
		if status == "takedown_requested" {
			q += `, takedown_requested_at=NOW()`
		}
		if status == "taken_down" {
			q += `, taken_down_at=NOW()`
		}
		q += ` WHERE tenant_id=$1 AND id=$2 AND deleted_at IS NULL`
		tag, err := db.Exec(ctx, q, tenantID, incidentID, status, userID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return apperrors.ErrNotFound
		}
		return nil
	})
}

// ---------------------------------------------------------------------------
// Dashboard / aggregation queries
// ---------------------------------------------------------------------------

func (r *PgRepository) GetGeoThreatMap(ctx context.Context, tenantID uuid.UUID, period string) ([]GeoThreatSummary, error) {
	interval := periodToInterval(period)
	var items []GeoThreatSummary
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT id,tenant_id,country_code,city,latitude,longitude,region_id,
			severity_critical_count,severity_high_count,severity_medium_count,severity_low_count,
			total_count,top_category_id,top_threat_type,period_start,period_end,computed_at
			FROM cti_geo_threat_summary
			WHERE tenant_id=$1 AND period_end >= NOW() - $2::interval
			ORDER BY total_count DESC`, tenantID, interval)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var g GeoThreatSummary
			if err := rows.Scan(&g.ID, &g.TenantID, &g.CountryCode, &g.City, &g.Latitude, &g.Longitude, &g.RegionID,
				&g.SeverityCriticalCount, &g.SeverityHighCount, &g.SeverityMediumCount, &g.SeverityLowCount,
				&g.TotalCount, &g.TopCategoryID, &g.TopThreatType, &g.PeriodStart, &g.PeriodEnd, &g.ComputedAt); err != nil {
				return err
			}
			items = append(items, g)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) GetSectorThreatSummary(ctx context.Context, tenantID uuid.UUID, period string) ([]SectorThreatSummary, error) {
	interval := periodToInterval(period)
	var items []SectorThreatSummary
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		rows, err := db.Query(ctx, `SELECT s.id,s.tenant_id,s.sector_id,
			s.severity_critical_count,s.severity_high_count,s.severity_medium_count,s.severity_low_count,
			s.total_count,s.period_start,s.period_end,s.computed_at,
			COALESCE(sec.code,'') AS sector_code, COALESCE(sec.label,'') AS sector_label
			FROM cti_sector_threat_summary s
			LEFT JOIN cti_industry_sectors sec ON s.sector_id = sec.id
			WHERE s.tenant_id=$1 AND s.period_end >= NOW() - $2::interval
			ORDER BY s.total_count DESC`, tenantID, interval)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var st SectorThreatSummary
			if err := rows.Scan(&st.ID, &st.TenantID, &st.SectorID,
				&st.SeverityCriticalCount, &st.SeverityHighCount, &st.SeverityMediumCount, &st.SeverityLowCount,
				&st.TotalCount, &st.PeriodStart, &st.PeriodEnd, &st.ComputedAt,
				&st.SectorCode, &st.SectorLabel); err != nil {
				return err
			}
			items = append(items, st)
		}
		return rows.Err()
	})
	return items, err
}

func (r *PgRepository) GetExecutiveSnapshot(ctx context.Context, tenantID uuid.UUID) (*ExecutiveSnapshot, error) {
	var s ExecutiveSnapshot
	err := r.withTenantRead(ctx, tenantID, func(db dbtx) error {
		return db.QueryRow(ctx, `SELECT id,tenant_id,total_events_24h,total_events_7d,total_events_30d,
			active_campaigns_count,critical_campaigns_count,total_iocs,
			brand_abuse_critical_count,brand_abuse_total_count,
			top_targeted_sector_id,top_threat_origin_country,
			mean_time_to_detect_hours,mean_time_to_respond_hours,
			risk_score_overall,trend_direction,trend_percentage,computed_at
			FROM cti_executive_snapshot WHERE tenant_id=$1`, tenantID).
			Scan(&s.ID, &s.TenantID, &s.TotalEvents24h, &s.TotalEvents7d, &s.TotalEvents30d,
				&s.ActiveCampaignsCount, &s.CriticalCampaignsCount, &s.TotalIOCs,
				&s.BrandAbuseCriticalCount, &s.BrandAbuseTotalCount,
				&s.TopTargetedSectorID, &s.TopThreatOriginCountry,
				&s.MeanTimeToDetectHours, &s.MeanTimeToRespondHours,
				&s.RiskScoreOverall, &s.TrendDirection, &s.TrendPercentage, &s.ComputedAt)
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

// ---------------------------------------------------------------------------
// Aggregation refresh
// ---------------------------------------------------------------------------

func (r *PgRepository) RefreshGeoThreatSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
			INSERT INTO cti_geo_threat_summary
				(tenant_id,country_code,city,latitude,longitude,
				severity_critical_count,severity_high_count,severity_medium_count,severity_low_count,
				total_count,period_start,period_end,computed_at)
			SELECT e.tenant_id, COALESCE(e.origin_country_code,'XX'), COALESCE(e.origin_city,''),
				AVG(e.origin_latitude), AVG(e.origin_longitude),
				COUNT(*) FILTER (WHERE sl.code='critical'),
				COUNT(*) FILTER (WHERE sl.code='high'),
				COUNT(*) FILTER (WHERE sl.code='medium'),
				COUNT(*) FILTER (WHERE sl.code='low'),
				COUNT(*), $2, $3, NOW()
			FROM cti_threat_events e
			LEFT JOIN cti_threat_severity_levels sl ON e.severity_id=sl.id
			WHERE e.tenant_id=$1 AND e.deleted_at IS NULL
			  AND e.first_seen_at >= $2 AND e.first_seen_at < $3
			GROUP BY e.tenant_id, e.origin_country_code, e.origin_city
			ON CONFLICT (tenant_id, country_code, city, period_start, period_end)
			DO UPDATE SET severity_critical_count=EXCLUDED.severity_critical_count,
				severity_high_count=EXCLUDED.severity_high_count,
				severity_medium_count=EXCLUDED.severity_medium_count,
				severity_low_count=EXCLUDED.severity_low_count,
				total_count=EXCLUDED.total_count,
				latitude=EXCLUDED.latitude, longitude=EXCLUDED.longitude,
				computed_at=NOW()`,
			tenantID, start, end)
		return err
	})
}

func (r *PgRepository) RefreshSectorThreatSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
			INSERT INTO cti_sector_threat_summary
				(tenant_id,sector_id,severity_critical_count,severity_high_count,severity_medium_count,severity_low_count,
				total_count,period_start,period_end,computed_at)
			SELECT e.tenant_id, e.target_sector_id,
				COUNT(*) FILTER (WHERE sl.code='critical'),
				COUNT(*) FILTER (WHERE sl.code='high'),
				COUNT(*) FILTER (WHERE sl.code='medium'),
				COUNT(*) FILTER (WHERE sl.code='low'),
				COUNT(*), $2, $3, NOW()
			FROM cti_threat_events e
			LEFT JOIN cti_threat_severity_levels sl ON e.severity_id=sl.id
			WHERE e.tenant_id=$1 AND e.deleted_at IS NULL AND e.target_sector_id IS NOT NULL
			  AND e.first_seen_at >= $2 AND e.first_seen_at < $3
			GROUP BY e.tenant_id, e.target_sector_id
			ON CONFLICT (tenant_id, sector_id, period_start, period_end)
			DO UPDATE SET severity_critical_count=EXCLUDED.severity_critical_count,
				severity_high_count=EXCLUDED.severity_high_count,
				severity_medium_count=EXCLUDED.severity_medium_count,
				severity_low_count=EXCLUDED.severity_low_count,
				total_count=EXCLUDED.total_count, computed_at=NOW()`,
			tenantID, start, end)
		return err
	})
}

func (r *PgRepository) RefreshExecutiveSnapshot(ctx context.Context, tenantID uuid.UUID) error {
	return r.withTenantWrite(ctx, tenantID, func(db dbtx) error {
		_, err := db.Exec(ctx, `
			INSERT INTO cti_executive_snapshot (tenant_id,
				total_events_24h, total_events_7d, total_events_30d,
				active_campaigns_count, critical_campaigns_count, total_iocs,
				brand_abuse_critical_count, brand_abuse_total_count,
				top_threat_origin_country, risk_score_overall, trend_direction, trend_percentage, computed_at)
			SELECT $1,
				(SELECT count(*) FROM cti_threat_events WHERE tenant_id=$1 AND deleted_at IS NULL AND first_seen_at > NOW()-'24h'::interval),
				(SELECT count(*) FROM cti_threat_events WHERE tenant_id=$1 AND deleted_at IS NULL AND first_seen_at > NOW()-'7d'::interval),
				(SELECT count(*) FROM cti_threat_events WHERE tenant_id=$1 AND deleted_at IS NULL AND first_seen_at > NOW()-'30d'::interval),
				(SELECT count(*) FROM cti_campaigns WHERE tenant_id=$1 AND deleted_at IS NULL AND status='active'),
				(SELECT count(*) FROM cti_campaigns c JOIN cti_threat_severity_levels sl ON c.severity_id=sl.id WHERE c.tenant_id=$1 AND c.deleted_at IS NULL AND c.status='active' AND sl.code='critical'),
				(SELECT count(*) FROM cti_campaign_iocs WHERE tenant_id=$1),
				(SELECT count(*) FROM cti_brand_abuse_incidents WHERE tenant_id=$1 AND deleted_at IS NULL AND risk_level='critical'),
				(SELECT count(*) FROM cti_brand_abuse_incidents WHERE tenant_id=$1 AND deleted_at IS NULL),
				(SELECT origin_country_code FROM cti_threat_events WHERE tenant_id=$1 AND deleted_at IS NULL AND origin_country_code IS NOT NULL GROUP BY origin_country_code ORDER BY count(*) DESC LIMIT 1),
				73.50, 'stable', 0, NOW()
			ON CONFLICT (tenant_id) DO UPDATE SET
				total_events_24h=EXCLUDED.total_events_24h, total_events_7d=EXCLUDED.total_events_7d,
				total_events_30d=EXCLUDED.total_events_30d, active_campaigns_count=EXCLUDED.active_campaigns_count,
				critical_campaigns_count=EXCLUDED.critical_campaigns_count, total_iocs=EXCLUDED.total_iocs,
				brand_abuse_critical_count=EXCLUDED.brand_abuse_critical_count,
				brand_abuse_total_count=EXCLUDED.brand_abuse_total_count,
				top_threat_origin_country=EXCLUDED.top_threat_origin_country,
				computed_at=NOW()`, tenantID)
		return err
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildUpdateSets(updates map[string]interface{}, tenantID, entityID uuid.UUID) ([]string, []any) {
	args := []any{tenantID, entityID}
	var sets []string
	i := 3
	for col, val := range updates {
		sets = append(sets, fmt.Sprintf("%s=$%d", col, i))
		args = append(args, val)
		i++
	}
	return sets, args
}

func periodToInterval(period string) string {
	switch period {
	case "24h":
		return "24 hours"
	case "7d":
		return "7 days"
	case "30d":
		return "30 days"
	default:
		return "7 days"
	}
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if ok := errors.As(err, &pgErr); ok {
		return pgErr.Code == "23505"
	}
	return false
}
