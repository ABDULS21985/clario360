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

	cyberrepo "github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/ueba/model"
	"github.com/clario360/platform/internal/database"
)

type AlertRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewAlertRepository(db *pgxpool.Pool, logger zerolog.Logger) *AlertRepository {
	return &AlertRepository{
		db:     db,
		logger: logger.With().Str("component", "ueba-alert-repo").Logger(),
	}
}

func (r *AlertRepository) Create(ctx context.Context, alert *model.UEBAAlert) (*model.UEBAAlert, error) {
	if alert == nil {
		return nil, fmt.Errorf("alert is required")
	}
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}
	if alert.Status == "" {
		alert.Status = "new"
	}
	now := time.Now().UTC()
	alert.CreatedAt = now
	alert.UpdatedAt = now
	err := database.RunWithTenant(ctx, r.db, alert.TenantID, func(tx pgx.Tx) error {
		signalsJSON, baselineJSON, err := marshalAlertJSON(alert)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO ueba_alerts (
				id, tenant_id, cyber_alert_id, entity_type, entity_id, entity_name, alert_type, severity, confidence,
				risk_score_before, risk_score_after, risk_score_delta, title, description, triggering_signals,
				triggering_event_ids, baseline_comparison, correlated_signal_count, correlation_window_start,
				correlation_window_end, mitre_technique_ids, mitre_tactic, status, resolved_at, resolved_by,
				resolution_notes, created_at, updated_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,
				$10,$11,$12,$13,$14,$15,
				$16,$17,$18,$19,
				$20,$21,$22,$23,$24,$25,
				$26,$27,$28
			)`,
			alert.ID, alert.TenantID, alert.CyberAlertID, alert.EntityType, alert.EntityID, nullString(alert.EntityName), alert.AlertType, alert.Severity, alert.Confidence,
			alert.RiskScoreBefore, alert.RiskScoreAfter, alert.RiskScoreDelta, alert.Title, alert.Description, signalsJSON,
			alert.TriggeringEventIDs, baselineJSON, alert.CorrelatedSignalCount, alert.CorrelationWindowStart,
			alert.CorrelationWindowEnd, alert.MITRETechniqueIDs, nullString(alert.MITRETactic), alert.Status, alert.ResolvedAt, alert.ResolvedBy,
			nullString(alert.ResolutionNotes), alert.CreatedAt, alert.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("create ueba alert: %w", err)
	}
	return r.GetByID(ctx, alert.TenantID, alert.ID)
}

func (r *AlertRepository) GetByID(ctx context.Context, tenantID, alertID uuid.UUID) (*model.UEBAAlert, error) {
	var item *model.UEBAAlert
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
				id, tenant_id, cyber_alert_id, entity_type, entity_id, COALESCE(entity_name, ''), alert_type, severity,
				confidence::double precision, risk_score_before::double precision, risk_score_after::double precision,
				risk_score_delta::double precision, title, description, triggering_signals, triggering_event_ids,
				baseline_comparison, correlated_signal_count, correlation_window_start, correlation_window_end,
				COALESCE(mitre_technique_ids, ARRAY[]::text[]), COALESCE(mitre_tactic, ''), status,
				resolved_at, resolved_by, COALESCE(resolution_notes, ''), created_at, updated_at
			FROM ueba_alerts
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, alertID,
		)
		alert, err := scanAlert(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return cyberrepo.ErrNotFound
			}
			return err
		}
		item = alert
		return nil
	})
	return item, err
}

func (r *AlertRepository) FindRecentOpenByType(ctx context.Context, tenantID uuid.UUID, entityID string, alertType model.AlertType, since time.Time) (*model.UEBAAlert, error) {
	var item *model.UEBAAlert
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT
				id, tenant_id, cyber_alert_id, entity_type, entity_id, COALESCE(entity_name, ''), alert_type, severity,
				confidence::double precision, risk_score_before::double precision, risk_score_after::double precision,
				risk_score_delta::double precision, title, description, triggering_signals, triggering_event_ids,
				baseline_comparison, correlated_signal_count, correlation_window_start, correlation_window_end,
				COALESCE(mitre_technique_ids, ARRAY[]::text[]), COALESCE(mitre_tactic, ''), status,
				resolved_at, resolved_by, COALESCE(resolution_notes, ''), created_at, updated_at
			FROM ueba_alerts
			WHERE tenant_id = $1 AND entity_id = $2 AND alert_type = $3 AND status IN ('new','acknowledged','investigating') AND created_at >= $4
			ORDER BY created_at DESC
			LIMIT 1`,
			tenantID, entityID, alertType, since,
		)
		alert, err := scanAlert(row)
		if err != nil {
			if err == pgx.ErrNoRows {
				return cyberrepo.ErrNotFound
			}
			return err
		}
		item = alert
		return nil
	})
	return item, err
}

func (r *AlertRepository) UpdateCorrelation(ctx context.Context, tenantID uuid.UUID, alert *model.UEBAAlert) (*model.UEBAAlert, error) {
	if alert == nil {
		return nil, fmt.Errorf("alert is required")
	}
	alert.UpdatedAt = time.Now().UTC()
	err := database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		signalsJSON, baselineJSON, err := marshalAlertJSON(alert)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE ueba_alerts
			SET severity = $3,
				confidence = $4,
				description = $5,
				triggering_signals = $6,
				triggering_event_ids = $7,
				baseline_comparison = $8,
				correlated_signal_count = $9,
				correlation_window_start = $10,
				correlation_window_end = $11,
				mitre_technique_ids = $12,
				mitre_tactic = $13,
				cyber_alert_id = $14,
				updated_at = $15
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, alert.ID, alert.Severity, alert.Confidence, alert.Description, signalsJSON, alert.TriggeringEventIDs,
			baselineJSON, alert.CorrelatedSignalCount, alert.CorrelationWindowStart, alert.CorrelationWindowEnd,
			alert.MITRETechniqueIDs, nullString(alert.MITRETactic), alert.CyberAlertID, alert.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alert.ID)
}

func (r *AlertRepository) ListByEntitySince(ctx context.Context, tenantID uuid.UUID, entityID string, since time.Time) ([]*model.UEBAAlert, error) {
	items := make([]*model.UEBAAlert, 0)
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT
				id, tenant_id, cyber_alert_id, entity_type, entity_id, COALESCE(entity_name, ''), alert_type, severity,
				confidence::double precision, risk_score_before::double precision, risk_score_after::double precision,
				risk_score_delta::double precision, title, description, triggering_signals, triggering_event_ids,
				baseline_comparison, correlated_signal_count, correlation_window_start, correlation_window_end,
				COALESCE(mitre_technique_ids, ARRAY[]::text[]), COALESCE(mitre_tactic, ''), status,
				resolved_at, resolved_by, COALESCE(resolution_notes, ''), created_at, updated_at
			FROM ueba_alerts
			WHERE tenant_id = $1 AND entity_id = $2 AND created_at >= $3
			ORDER BY created_at DESC`,
			tenantID, entityID, since,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, scanErr := scanAlert(rows)
			if scanErr != nil {
				return scanErr
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *AlertRepository) UpdateRiskImpact(ctx context.Context, tenantID uuid.UUID, alertID uuid.UUID, before, after float64) error {
	return database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE ueba_alerts
			SET risk_score_before = $3,
				risk_score_after = $4,
				risk_score_delta = $4 - $3,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, alertID, before, after,
		)
		return err
	})
}

func (r *AlertRepository) UpdateStatus(ctx context.Context, tenantID uuid.UUID, alertID uuid.UUID, status string, resolvedBy *uuid.UUID, notes string) (*model.UEBAAlert, error) {
	err := database.RunWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			UPDATE ueba_alerts
			SET status = $3,
				resolved_by = $4,
				resolution_notes = $5,
				resolved_at = CASE WHEN $3 IN ('resolved', 'false_positive') THEN now() ELSE resolved_at END,
				updated_at = now()
			WHERE tenant_id = $1 AND id = $2`,
			tenantID, alertID, status, resolvedBy, nullString(notes),
		)
		return err
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, tenantID, alertID)
}

func (r *AlertRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int, entityID string, status string) ([]*model.UEBAAlert, int, error) {
	if limit <= 0 {
		limit = 25
	}
	var (
		items []*model.UEBAAlert
		total int
	)
	err := database.RunReadWithTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		query := `FROM ueba_alerts WHERE tenant_id = $1`
		args := []any{tenantID}
		if entityID != "" {
			query += fmt.Sprintf(" AND entity_id = $%d", len(args)+1)
			args = append(args, entityID)
		}
		if status != "" {
			query += fmt.Sprintf(" AND status = $%d", len(args)+1)
			args = append(args, status)
		}
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) `+query, args...).Scan(&total); err != nil {
			return err
		}
		args = append(args, limit, offset)
		rows, err := tx.Query(ctx, `
			SELECT
				id, tenant_id, cyber_alert_id, entity_type, entity_id, COALESCE(entity_name, ''), alert_type, severity,
				confidence::double precision, risk_score_before::double precision, risk_score_after::double precision,
				risk_score_delta::double precision, title, description, triggering_signals, triggering_event_ids,
				baseline_comparison, correlated_signal_count, correlation_window_start, correlation_window_end,
				COALESCE(mitre_technique_ids, ARRAY[]::text[]), COALESCE(mitre_tactic, ''), status,
				resolved_at, resolved_by, COALESCE(resolution_notes, ''), created_at, updated_at `+query+`
			ORDER BY created_at DESC
			LIMIT $`+fmt.Sprint(len(args)-1)+` OFFSET $`+fmt.Sprint(len(args)),
			args...,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			item, scanErr := scanAlert(rows)
			if scanErr != nil {
				return scanErr
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, total, err
}

type alertScanner interface {
	Scan(dest ...any) error
}

func scanAlert(row alertScanner) (*model.UEBAAlert, error) {
	var (
		item           model.UEBAAlert
		signalsJSON    []byte
		comparisonJSON []byte
	)
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.CyberAlertID, &item.EntityType, &item.EntityID, &item.EntityName, &item.AlertType, &item.Severity,
		&item.Confidence, &item.RiskScoreBefore, &item.RiskScoreAfter, &item.RiskScoreDelta, &item.Title, &item.Description,
		&signalsJSON, &item.TriggeringEventIDs, &comparisonJSON, &item.CorrelatedSignalCount, &item.CorrelationWindowStart, &item.CorrelationWindowEnd,
		&item.MITRETechniqueIDs, &item.MITRETactic, &item.Status, &item.ResolvedAt, &item.ResolvedBy, &item.ResolutionNotes, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if len(signalsJSON) > 0 {
		if err := json.Unmarshal(signalsJSON, &item.TriggeringSignals); err != nil {
			return nil, fmt.Errorf("decode ueba alert signals: %w", err)
		}
	}
	if len(comparisonJSON) > 0 {
		if err := json.Unmarshal(comparisonJSON, &item.BaselineComparison); err != nil {
			return nil, fmt.Errorf("decode ueba alert comparison: %w", err)
		}
	}
	if item.MITRETechniqueIDs == nil {
		item.MITRETechniqueIDs = []string{}
	}
	return &item, nil
}

func marshalAlertJSON(alert *model.UEBAAlert) ([]byte, []byte, error) {
	signalsJSON, err := json.Marshal(alert.TriggeringSignals)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ueba alert signals: %w", err)
	}
	comparisonJSON, err := json.Marshal(alert.BaselineComparison)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal ueba alert comparison: %w", err)
	}
	return signalsJSON, comparisonJSON, nil
}
