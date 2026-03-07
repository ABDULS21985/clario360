package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/clario360/platform/internal/cyber/model"
)

type scanner interface {
	Scan(dest ...any) error
}

type dbtx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func queryable(pool *pgxpool.Pool) dbtx {
	return pool
}

func marshalJSON(value interface{}) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func ensureRawMessage(value json.RawMessage, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return value
}

func scanAlert(row scanner) (*model.Alert, error) {
	var (
		alert          model.Alert
		explanationRaw []byte
		metadataRaw    []byte
	)
	if err := row.Scan(
		&alert.ID,
		&alert.TenantID,
		&alert.Title,
		&alert.Description,
		&alert.Severity,
		&alert.Status,
		&alert.Source,
		&alert.RuleID,
		&alert.AssetID,
		&alert.AssetIDs,
		&alert.AssignedTo,
		&alert.AssignedAt,
		&alert.EscalatedTo,
		&alert.EscalatedAt,
		&explanationRaw,
		&alert.ConfidenceScore,
		&alert.MITRETacticID,
		&alert.MITRETacticName,
		&alert.MITRETechniqueID,
		&alert.MITRETechniqueName,
		&alert.EventCount,
		&alert.FirstEventAt,
		&alert.LastEventAt,
		&alert.ResolvedAt,
		&alert.ResolutionNotes,
		&alert.FalsePositiveReason,
		&alert.Tags,
		&metadataRaw,
		&alert.CreatedAt,
		&alert.UpdatedAt,
		&alert.DeletedAt,
	); err != nil {
		return nil, err
	}
	if len(explanationRaw) > 0 {
		if err := json.Unmarshal(explanationRaw, &alert.Explanation); err != nil {
			return nil, fmt.Errorf("decode alert explanation: %w", err)
		}
	}
	alert.Metadata = ensureRawMessage(metadataRaw, "{}")
	if alert.Tags == nil {
		alert.Tags = []string{}
	}
	if alert.AssetIDs == nil {
		alert.AssetIDs = []uuid.UUID{}
	}
	return &alert, nil
}

func scanAlertComment(row scanner) (*model.AlertComment, error) {
	var comment model.AlertComment
	if err := row.Scan(
		&comment.ID,
		&comment.TenantID,
		&comment.AlertID,
		&comment.UserID,
		&comment.UserName,
		&comment.UserEmail,
		&comment.Content,
		&comment.IsSystem,
		&comment.Metadata,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	); err != nil {
		return nil, err
	}
	comment.Metadata = ensureRawMessage(comment.Metadata, "{}")
	return &comment, nil
}

func scanAlertTimeline(row scanner) (*model.AlertTimelineEntry, error) {
	var item model.AlertTimelineEntry
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.AlertID,
		&item.Action,
		&item.ActorID,
		&item.ActorName,
		&item.OldValue,
		&item.NewValue,
		&item.Description,
		&item.Metadata,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = ensureRawMessage(item.Metadata, "{}")
	return &item, nil
}

func scanRule(row scanner) (*model.DetectionRule, error) {
	var item model.DetectionRule
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.RuleType,
		&item.Severity,
		&item.Enabled,
		&item.RuleContent,
		&item.MITRETacticIDs,
		&item.MITRETechniqueIDs,
		&item.BaseConfidence,
		&item.FalsePositiveCount,
		&item.TruePositiveCount,
		&item.LastTriggeredAt,
		&item.TriggerCount,
		&item.Tags,
		&item.IsTemplate,
		&item.TemplateID,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	if item.RuleContent == nil {
		item.RuleContent = json.RawMessage("{}")
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.MITRETacticIDs == nil {
		item.MITRETacticIDs = []string{}
	}
	if item.MITRETechniqueIDs == nil {
		item.MITRETechniqueIDs = []string{}
	}
	return &item, nil
}

func scanThreat(row scanner) (*model.Threat, error) {
	var item model.Threat
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Description,
		&item.Type,
		&item.Severity,
		&item.Status,
		&item.ThreatActor,
		&item.Campaign,
		&item.MITRETacticIDs,
		&item.MITRETechniqueIDs,
		&item.AffectedAssetCount,
		&item.AlertCount,
		&item.FirstSeenAt,
		&item.LastSeenAt,
		&item.ContainedAt,
		&item.Tags,
		&item.Metadata,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.DeletedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = ensureRawMessage(item.Metadata, "{}")
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.MITRETacticIDs == nil {
		item.MITRETacticIDs = []string{}
	}
	if item.MITRETechniqueIDs == nil {
		item.MITRETechniqueIDs = []string{}
	}
	return &item, nil
}

func scanIndicator(row scanner) (*model.ThreatIndicator, error) {
	var item model.ThreatIndicator
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.ThreatID,
		&item.Type,
		&item.Value,
		&item.Description,
		&item.Severity,
		&item.Source,
		&item.Confidence,
		&item.Active,
		&item.FirstSeenAt,
		&item.LastSeenAt,
		&item.ExpiresAt,
		&item.Tags,
		&item.Metadata,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	item.Metadata = ensureRawMessage(item.Metadata, "{}")
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return &item, nil
}

func scanSecurityEvent(row scanner) (*model.SecurityEvent, error) {
	var item model.SecurityEvent
	var sourceIP, destIP *string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.Timestamp,
		&item.Source,
		&item.Type,
		&item.Severity,
		&sourceIP,
		&destIP,
		&item.DestPort,
		&item.Protocol,
		&item.Username,
		&item.Process,
		&item.ParentProcess,
		&item.CommandLine,
		&item.FilePath,
		&item.FileHash,
		&item.AssetID,
		&item.RawEvent,
		&item.MatchedRules,
		&item.ProcessedAt,
	); err != nil {
		return nil, err
	}
	item.SourceIP = sourceIP
	item.DestIP = destIP
	item.RawEvent = ensureRawMessage(item.RawEvent, "{}")
	if item.MatchedRules == nil {
		item.MatchedRules = []uuid.UUID{}
	}
	return &item, nil
}
