package rca

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// TimelineBuilder collects events from multiple sources into a sorted timeline.
type TimelineBuilder struct {
	cyberDB *pgxpool.Pool
	dataDB  *pgxpool.Pool
	logger  zerolog.Logger
}

// NewTimelineBuilder creates a timeline builder.
func NewTimelineBuilder(cyberDB, dataDB *pgxpool.Pool, logger zerolog.Logger) *TimelineBuilder {
	return &TimelineBuilder{
		cyberDB: cyberDB,
		dataDB:  dataDB,
		logger:  logger.With().Str("component", "rca-timeline").Logger(),
	}
}

// BuildForAlert builds a timeline around a security alert.
func (tb *TimelineBuilder) BuildForAlert(ctx context.Context, tenantID uuid.UUID, alertID uuid.UUID, windowBefore, windowAfter time.Duration) ([]TimelineEvent, error) {
	// Get the alert's creation time to define the window
	var alertTime time.Time
	err := tb.cyberDB.QueryRow(ctx, `
		SELECT created_at FROM alerts WHERE tenant_id = $1 AND id = $2
	`, tenantID, alertID).Scan(&alertTime)
	if err != nil {
		return nil, err
	}

	start := alertTime.Add(-windowBefore)
	end := alertTime.Add(windowAfter)

	var events []TimelineEvent

	// 1. Cyber alerts (same assets, same IP, same technique)
	alertEvents, err := tb.collectAlertEvents(ctx, tenantID, alertID, start, end)
	if err != nil {
		tb.logger.Warn().Err(err).Msg("collect alert events")
	} else {
		events = append(events, alertEvents...)
	}

	// 2. UEBA access events (same entity)
	uebaEvents, err := tb.collectUEBAEvents(ctx, tenantID, alertID, start, end)
	if err != nil {
		tb.logger.Warn().Err(err).Msg("collect UEBA events")
	} else {
		events = append(events, uebaEvents...)
	}

	// 3. Audit log events
	auditEvents, err := tb.collectAuditEvents(ctx, tenantID, alertID, start, end)
	if err != nil {
		tb.logger.Warn().Err(err).Msg("collect audit events")
	} else {
		events = append(events, auditEvents...)
	}

	// Sort chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

// BuildForPipeline builds a timeline around a pipeline failure.
func (tb *TimelineBuilder) BuildForPipeline(ctx context.Context, tenantID uuid.UUID, runID uuid.UUID, windowBefore time.Duration) ([]TimelineEvent, error) {
	if tb.dataDB == nil {
		return nil, nil
	}

	var failTime time.Time
	err := tb.dataDB.QueryRow(ctx, `
		SELECT COALESCE(completed_at, created_at) FROM pipeline_runs
		WHERE tenant_id = $1 AND id = $2
	`, tenantID, runID).Scan(&failTime)
	if err != nil {
		return nil, err
	}

	start := failTime.Add(-windowBefore)
	end := failTime

	var events []TimelineEvent

	// Pipeline run events
	pipelineEvents, err := tb.collectPipelineEvents(ctx, tenantID, runID, start, end)
	if err != nil {
		tb.logger.Warn().Err(err).Msg("collect pipeline events")
	} else {
		events = append(events, pipelineEvents...)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

func (tb *TimelineBuilder) collectAlertEvents(ctx context.Context, tenantID, alertID uuid.UUID, start, end time.Time) ([]TimelineEvent, error) {
	// Get the primary alert's assets and MITRE info for correlation
	rows, err := tb.cyberDB.Query(ctx, `
		SELECT a.id, a.title, a.severity::text, a.created_at, a.source,
		       a.mitre_technique_id, a.mitre_tactic_id, a.metadata
		FROM alerts a
		WHERE a.tenant_id = $1
		  AND a.created_at BETWEEN $2 AND $3
		  AND a.deleted_at IS NULL
		  AND (
		    a.id = $4
		    OR a.asset_id IN (SELECT asset_id FROM alerts WHERE id = $4 AND asset_id IS NOT NULL)
		    OR a.mitre_technique_id IN (SELECT mitre_technique_id FROM alerts WHERE id = $4 AND mitre_technique_id IS NOT NULL)
		  )
		ORDER BY a.created_at
	`, tenantID, start, end, alertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var (
			id, title, severity, source string
			timestamp                   time.Time
			mitreT, mitreTA             *string
			metadata                    []byte
		)
		if err := rows.Scan(&id, &title, &severity, &timestamp, &source, &mitreT, &mitreTA, &metadata); err != nil {
			continue
		}
		events = append(events, TimelineEvent{
			ID:               id,
			Timestamp:        timestamp,
			Source:           "alert",
			Type:             "cyber_alert",
			Summary:          title,
			Severity:         severity,
			MITRETechID: ptrToString(mitreT),
		})
	}
	return events, rows.Err()
}

func (tb *TimelineBuilder) collectUEBAEvents(ctx context.Context, tenantID, alertID uuid.UUID, start, end time.Time) ([]TimelineEvent, error) {
	rows, err := tb.cyberDB.Query(ctx, `
		SELECT ua.id, ua.title, ua.severity::text, ua.created_at, ua.entity_id
		FROM ueba_alerts ua
		WHERE ua.tenant_id = $1
		  AND ua.created_at BETWEEN $2 AND $3
		  AND (
		    ua.cyber_alert_id = $4
		    OR ua.entity_id IN (
		      SELECT metadata->>'user_id' FROM alerts WHERE id = $4 AND metadata->>'user_id' IS NOT NULL
		    )
		  )
		ORDER BY ua.created_at
	`, tenantID, start, end, alertID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var id, title, severity string
		var timestamp time.Time
		var entityID *string
		if err := rows.Scan(&id, &title, &severity, &timestamp, &entityID); err != nil {
			continue
		}
		events = append(events, TimelineEvent{
			ID:        id,
			Timestamp: timestamp,
			Source:    "ueba",
			Type:      "ueba_alert",
			Summary:   title,
			Severity:  severity,
			UserID:    ptrToString(entityID),
		})
	}
	return events, rows.Err()
}

func (tb *TimelineBuilder) collectAuditEvents(ctx context.Context, tenantID, alertID uuid.UUID, start, end time.Time) ([]TimelineEvent, error) {
	rows, err := tb.cyberDB.Query(ctx, `
		SELECT t.id, t.action, t.description, t.created_at,
		       t.actor_id::text, t.actor_name
		FROM alert_timeline t
		WHERE t.tenant_id = $1
		  AND t.alert_id = $2
		  AND t.created_at BETWEEN $3 AND $4
		ORDER BY t.created_at
	`, tenantID, alertID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var id, action, description string
		var timestamp time.Time
		var actorID, actorName *string
		if err := rows.Scan(&id, &action, &description, &timestamp, &actorID, &actorName); err != nil {
			continue
		}
		events = append(events, TimelineEvent{
			ID:        id,
			Timestamp: timestamp,
			Source:    "audit",
			Type:      action,
			Summary:   description,
			UserID:    ptrToString(actorID),
		})
	}
	return events, rows.Err()
}

func (tb *TimelineBuilder) collectPipelineEvents(ctx context.Context, tenantID, runID uuid.UUID, start, end time.Time) ([]TimelineEvent, error) {
	if tb.dataDB == nil {
		return nil, nil
	}

	var events []TimelineEvent
	pipelineRows, pErr := tb.dataDB.Query(ctx, `
		SELECT r.id, r.status::text, r.current_phase, r.error_message,
		       r.created_at, p.name
		FROM pipeline_runs r
		JOIN pipelines p ON p.id = r.pipeline_id AND p.tenant_id = r.tenant_id
		WHERE r.tenant_id = $1
		  AND r.pipeline_id = (SELECT pipeline_id FROM pipeline_runs WHERE id = $2)
		  AND r.created_at BETWEEN $3 AND $4
		ORDER BY r.created_at
	`, tenantID, runID, start, end)
	if pErr != nil {
		return nil, pErr
	}
	defer pipelineRows.Close()

	for pipelineRows.Next() {
		var id, status, phase, pipelineName string
		var errorMsg *string
		var createdAt time.Time
		if err := pipelineRows.Scan(&id, &status, &phase, &errorMsg, &createdAt, &pipelineName); err != nil {
			continue
		}
		summary := fmt.Sprintf("Pipeline %s: %s (phase: %s)", pipelineName, status, phase)
		if errorMsg != nil && *errorMsg != "" {
			summary += " — " + *errorMsg
		}
		events = append(events, TimelineEvent{
			ID:        id,
			Timestamp: createdAt,
			Source:    "pipeline",
			Type:      "pipeline_run",
			Summary:   summary,
		})
	}

	return events, pipelineRows.Err()
}
