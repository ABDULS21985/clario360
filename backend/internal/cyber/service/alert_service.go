package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// AlertService manages alert lifecycle operations and detection-engine upserts.
type AlertService struct {
	alertRepo   *repository.AlertRepository
	commentRepo *repository.CommentRepository
	db          *pgxpool.Pool
	producer    *events.Producer
	logger      zerolog.Logger
}

// NewAlertService creates a new AlertService.
func NewAlertService(
	alertRepo *repository.AlertRepository,
	commentRepo *repository.CommentRepository,
	db *pgxpool.Pool,
	producer *events.Producer,
	logger zerolog.Logger,
) *AlertService {
	return &AlertService{
		alertRepo:   alertRepo,
		commentRepo: commentRepo,
		db:          db,
		producer:    producer,
		logger:      logger,
	}
}

// ListAlerts returns a paginated list of alerts.
func (s *AlertService) ListAlerts(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *Actor) (*dto.AlertListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	alerts, total, err := s.alertRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if alerts == nil {
		alerts = []*model.Alert{}
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.listed", tenantID, actor, map[string]interface{}{
		"filters": params,
		"count":   len(alerts),
	})
	return &dto.AlertListResponse{
		Data: alerts,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

// GetAlert returns a single alert.
func (s *AlertService) GetAlert(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor) (*model.Alert, error) {
	alert, err := s.alertRepo.GetByID(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.viewed", tenantID, actor, map[string]interface{}{
		"id": alertID.String(),
	})
	return alert, nil
}

// UpdateStatus changes an alert status and writes a timeline entry.
func (s *AlertService) UpdateStatus(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor, req *dto.AlertStatusUpdateRequest) (*model.Alert, error) {
	before, err := s.alertRepo.GetByID(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	after, err := s.alertRepo.UpdateStatus(ctx, tenantID, alertID, req.Status, req.Notes, req.Reason)
	if err != nil {
		return nil, err
	}
	oldStatus := string(before.Status)
	newStatus := string(after.Status)
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    tenantID,
		AlertID:     alertID,
		Action:      "status_changed",
		ActorID:     actorUUID(actor),
		ActorName:   actorName(actor),
		OldValue:    stringPtr(oldStatus),
		NewValue:    stringPtr(newStatus),
		Description: fmt.Sprintf("Status changed from %s to %s", oldStatus, newStatus),
		Metadata:    mustJSON(map[string]interface{}{"notes": req.Notes, "reason": req.Reason}),
	})
	if req.Notes != nil && strings.TrimSpace(*req.Notes) != "" {
		_, _ = s.commentRepo.Create(ctx, &model.AlertComment{
			TenantID:  tenantID,
			AlertID:   alertID,
			UserID:    actor.UserID,
			UserName:  safeActorName(actor),
			UserEmail: actor.UserEmail,
			Content:   *req.Notes,
			Metadata:  mustJSON(map[string]interface{}{"type": "status_note"}),
		})
	}
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.status_changed", tenantID, actor, map[string]interface{}{
		"id":         alertID.String(),
		"old_status": oldStatus,
		"new_status": newStatus,
		"changed_by": actor.UserID.String(),
	})
	if req.Status == model.AlertStatusResolved {
		_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.resolved", tenantID, actor, map[string]interface{}{
			"id":               alertID.String(),
			"resolution_notes": req.Notes,
			"resolved_by":      actor.UserID.String(),
		})
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.status_changed", tenantID, actor, map[string]interface{}{
		"id":     alertID.String(),
		"before": before.Status,
		"after":  after.Status,
	})
	return after, nil
}

// Assign assigns or reassigns an alert.
func (s *AlertService) Assign(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor, assignedTo uuid.UUID) (*model.Alert, error) {
	before, err := s.alertRepo.GetByID(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	after, err := s.alertRepo.Assign(ctx, tenantID, alertID, assignedTo)
	if err != nil {
		return nil, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    tenantID,
		AlertID:     alertID,
		Action:      "assigned",
		ActorID:     actorUUID(actor),
		ActorName:   actorName(actor),
		OldValue:    uuidPtrString(before.AssignedTo),
		NewValue:    stringPtr(assignedTo.String()),
		Description: fmt.Sprintf("Alert assigned to %s", assignedTo.String()),
		Metadata:    mustJSON(map[string]interface{}{"assigned_to": assignedTo.String()}),
	})
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.assigned", tenantID, actor, map[string]interface{}{
		"id":          alertID.String(),
		"assigned_to": assignedTo.String(),
		"assigned_by": actor.UserID.String(),
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.assigned", tenantID, actor, map[string]interface{}{
		"id":          alertID.String(),
		"assigned_to": assignedTo.String(),
	})
	return after, nil
}

// Escalate escalates an alert to another analyst or manager.
func (s *AlertService) Escalate(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor, escalatedTo uuid.UUID, reason string) (*model.Alert, error) {
	after, err := s.alertRepo.Escalate(ctx, tenantID, alertID, escalatedTo)
	if err != nil {
		return nil, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    tenantID,
		AlertID:     alertID,
		Action:      "escalated",
		ActorID:     actorUUID(actor),
		ActorName:   actorName(actor),
		NewValue:    stringPtr(escalatedTo.String()),
		Description: fmt.Sprintf("Alert escalated to %s", escalatedTo.String()),
		Metadata:    mustJSON(map[string]interface{}{"reason": reason}),
	})
	_, _ = s.commentRepo.Create(ctx, &model.AlertComment{
		TenantID:  tenantID,
		AlertID:   alertID,
		UserID:    actor.UserID,
		UserName:  safeActorName(actor),
		UserEmail: actor.UserEmail,
		Content:   reason,
		IsSystem:  false,
		Metadata:  mustJSON(map[string]interface{}{"type": "escalation_reason"}),
	})
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.escalated", tenantID, actor, map[string]interface{}{
		"id":           alertID.String(),
		"escalated_to": escalatedTo.String(),
		"escalated_by": actor.UserID.String(),
		"reason":       reason,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.escalated", tenantID, actor, map[string]interface{}{
		"id":           alertID.String(),
		"escalated_to": escalatedTo.String(),
		"reason":       reason,
	})
	return after, nil
}

// AddComment adds an analyst comment to an alert.
func (s *AlertService) AddComment(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor, req *dto.AlertCommentRequest) (*model.AlertComment, error) {
	comment, err := s.commentRepo.Create(ctx, &model.AlertComment{
		TenantID:  tenantID,
		AlertID:   alertID,
		UserID:    actor.UserID,
		UserName:  safeActorName(actor),
		UserEmail: actor.UserEmail,
		Content:   req.Content,
		Metadata:  req.Metadata,
	})
	if err != nil {
		return nil, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    tenantID,
		AlertID:     alertID,
		Action:      "commented",
		ActorID:     actorUUID(actor),
		ActorName:   actorName(actor),
		Description: "Comment added to alert",
		Metadata:    mustJSON(map[string]interface{}{"comment_id": comment.ID.String()}),
	})
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.commented", tenantID, actor, map[string]interface{}{
		"id":         alertID.String(),
		"comment_id": comment.ID.String(),
		"user_id":    actor.UserID.String(),
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.commented", tenantID, actor, map[string]interface{}{
		"id":         alertID.String(),
		"comment_id": comment.ID.String(),
	})
	return comment, nil
}

// ListComments returns all alert comments.
func (s *AlertService) ListComments(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor) ([]*model.AlertComment, error) {
	items, err := s.commentRepo.ListByAlert(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.comments_listed", tenantID, actor, map[string]interface{}{
		"id":    alertID.String(),
		"count": len(items),
	})
	return items, nil
}

// ListTimeline returns all timeline entries for an alert.
func (s *AlertService) ListTimeline(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor) ([]*model.AlertTimelineEntry, error) {
	items, err := s.alertRepo.ListTimeline(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.timeline_listed", tenantID, actor, map[string]interface{}{
		"id":    alertID.String(),
		"count": len(items),
	})
	return items, nil
}

// Merge merges secondary alerts into the primary alert.
func (s *AlertService) Merge(ctx context.Context, tenantID, primaryAlertID uuid.UUID, mergeIDs []uuid.UUID, actor *Actor) (*model.Alert, error) {
	primary, err := s.alertRepo.GetByID(ctx, tenantID, primaryAlertID)
	if err != nil {
		return nil, err
	}
	secondaryAlerts, err := s.alertRepo.GetByIDs(ctx, tenantID, mergeIDs)
	if err != nil {
		return nil, err
	}
	if len(secondaryAlerts) != len(mergeIDs) {
		return nil, repository.ErrNotFound
	}

	assetIDs := uniqueUUIDs(primary.AssetIDs)
	if primary.AssetID != nil {
		assetIDs = append(assetIDs, *primary.AssetID)
	}
	mergedTitles := make([]string, 0, len(secondaryAlerts))
	totalEvents := primary.EventCount
	for _, secondary := range secondaryAlerts {
		if secondary.ID == primaryAlertID {
			return nil, repository.ErrInvalidInput
		}
		totalEvents += secondary.EventCount
		if secondary.AssetID != nil {
			assetIDs = append(assetIDs, *secondary.AssetID)
		}
		assetIDs = append(assetIDs, secondary.AssetIDs...)
		mergedTitles = append(mergedTitles, secondary.Title)
		_ = s.commentRepo.ReassignAlert(ctx, tenantID, secondary.ID, primaryAlertID)
		_ = s.alertRepo.CloneTimeline(ctx, tenantID, secondary.ID, primaryAlertID)
		_ = s.alertRepo.MarkMerged(ctx, tenantID, secondary.ID, primaryAlertID)
		_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
			TenantID:    tenantID,
			AlertID:     secondary.ID,
			Action:      "merged",
			ActorID:     actorUUID(actor),
			ActorName:   actorName(actor),
			NewValue:    stringPtr(primaryAlertID.String()),
			Description: fmt.Sprintf("Merged with alert %s", primary.Title),
			Metadata:    mustJSON(map[string]interface{}{"primary_alert_id": primaryAlertID.String()}),
		})
	}
	primaryExplanation := primary.Explanation
	if primaryExplanation.Details == nil {
		primaryExplanation.Details = make(map[string]interface{})
	}
	primaryExplanation.Details["merged_alert_ids"] = uuidStrings(mergeIDs)
	primaryExplanation.Details["merged_titles"] = mergedTitles
	updated, err := s.alertRepo.UpdateAfterMerge(ctx, tenantID, primaryAlertID, totalEvents, primary.AssetID, uniqueUUIDs(assetIDs), &primaryExplanation)
	if err != nil {
		return nil, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    tenantID,
		AlertID:     primaryAlertID,
		Action:      "merged",
		ActorID:     actorUUID(actor),
		ActorName:   actorName(actor),
		Description: fmt.Sprintf("Merged %d related alerts: %s", len(secondaryAlerts), strings.Join(mergedTitles, ", ")),
		Metadata:    mustJSON(map[string]interface{}{"merged_ids": uuidStrings(mergeIDs)}),
	})
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.merged", tenantID, actor, map[string]interface{}{
		"primary_id": primaryAlertID.String(),
		"merged_ids": uuidStrings(mergeIDs),
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.merged", tenantID, actor, map[string]interface{}{
		"primary_id": primaryAlertID.String(),
		"merged_ids": uuidStrings(mergeIDs),
	})
	return updated, nil
}

// Related returns alerts related to the target alert.
func (s *AlertService) Related(ctx context.Context, tenantID, alertID uuid.UUID, actor *Actor) ([]*model.Alert, error) {
	items, err := s.alertRepo.FindRelated(ctx, tenantID, alertID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.related_listed", tenantID, actor, map[string]interface{}{
		"id":    alertID.String(),
		"count": len(items),
	})
	return items, nil
}

// Stats returns aggregated alert stats.
func (s *AlertService) Stats(ctx context.Context, tenantID uuid.UUID, actor *Actor) (*model.AlertStats, error) {
	stats, err := s.alertRepo.Stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.stats_viewed", tenantID, actor, map[string]interface{}{})
	return stats, nil
}

// Count returns a simple filtered alert count.
func (s *AlertService) Count(ctx context.Context, tenantID uuid.UUID, params *dto.AlertListParams, actor *Actor) (int, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return 0, err
	}
	count, err := s.alertRepo.Count(ctx, tenantID, params)
	if err != nil {
		return 0, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.counted", tenantID, actor, map[string]interface{}{"count": count})
	return count, nil
}

// CreateOrMergeDetectionAlert creates a new alert or aggregates into an open alert for deduplication.
func (s *AlertService) CreateOrMergeDetectionAlert(ctx context.Context, alert *model.Alert) (*model.Alert, bool, error) {
	if alert.RuleID != nil && alert.AssetID != nil {
		existing, err := s.alertRepo.FindOpenByRuleAndAsset(ctx, alert.TenantID, *alert.RuleID, alert.AssetID)
		if err == nil {
			updated, err := s.alertRepo.UpdateAggregatedDetectionAlert(ctx, alert.TenantID, existing.ID, alert.EventCount, alert.LastEventAt, uniqueUUIDs(append(existing.AssetIDs, alert.AssetIDs...)), &alert.Explanation)
			if err != nil {
				return nil, false, err
			}
			_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
				TenantID:    alert.TenantID,
				AlertID:     updated.ID,
				Action:      "correlated",
				Description: fmt.Sprintf("Added %d additional triggering events to existing alert", alert.EventCount),
				Metadata:    mustJSON(map[string]interface{}{"event_count": alert.EventCount}),
			})
			return updated, false, nil
		}
		if err != repository.ErrNotFound {
			return nil, false, err
		}
	}

	created, err := s.alertRepo.Create(ctx, alert)
	if err != nil {
		return nil, false, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    alert.TenantID,
		AlertID:     created.ID,
		Action:      "created",
		Description: fmt.Sprintf("Alert created by detection source %q", alert.Source),
		Metadata:    mustJSON(map[string]interface{}{"rule_id": uuidPtrString(alert.RuleID)}),
	})
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.created", alert.TenantID, nil, map[string]interface{}{
		"id":         created.ID.String(),
		"title":      created.Title,
		"severity":   created.Severity,
		"confidence": created.ConfidenceScore,
		"asset_id":   uuidPtrString(created.AssetID),
		"rule_id":    uuidPtrString(created.RuleID),
		"mitre": map[string]interface{}{
			"tactic_id":    created.MITRETacticID,
			"technique_id": created.MITRETechniqueID,
		},
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.alert.created", alert.TenantID, nil, map[string]interface{}{
		"id":       created.ID.String(),
		"title":    created.Title,
		"source":   created.Source,
		"rule_id":  uuidPtrString(created.RuleID),
		"asset_id": uuidPtrString(created.AssetID),
	})
	return created, true, nil
}

// CreateFromEvent persists a custom cross-suite alert and emits the standard
// cyber alert created event without applying the detection-engine merge rules.
func (s *AlertService) CreateFromEvent(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	if alert == nil {
		return nil, fmt.Errorf("alert is required")
	}
	if alert.Status == "" {
		alert.Status = model.AlertStatusNew
	}
	if alert.EventCount == 0 {
		alert.EventCount = 1
	}
	if alert.FirstEventAt.IsZero() {
		alert.FirstEventAt = time.Now().UTC()
	}
	if alert.LastEventAt.IsZero() {
		alert.LastEventAt = alert.FirstEventAt
	}
	if alert.AssetIDs == nil {
		alert.AssetIDs = []uuid.UUID{}
	}
	if alert.Tags == nil {
		alert.Tags = []string{}
	}

	created, err := s.alertRepo.Create(ctx, alert)
	if err != nil {
		return nil, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    alert.TenantID,
		AlertID:     created.ID,
		Action:      "created",
		Description: fmt.Sprintf("Alert created by event source %q", alert.Source),
		Metadata:    mustJSON(map[string]interface{}{"event_count": alert.EventCount}),
	})
	_ = publishEvent(ctx, s.producer, events.Topics.AlertEvents, "cyber.alert.created", alert.TenantID, nil, map[string]interface{}{
		"id":                   created.ID.String(),
		"title":                created.Title,
		"severity":             created.Severity,
		"confidence_score":     created.ConfidenceScore,
		"affected_asset_count": len(created.AssetIDs),
		"source":               created.Source,
		"mitre_technique_id":   created.MITRETechniqueID,
		"mitre_tactic_id":      created.MITRETacticID,
	})
	return created, nil
}

func (s *AlertService) FindRecentEventAlert(ctx context.Context, tenantID uuid.UUID, source, metadataKey, metadataValue string, window time.Duration) (*model.Alert, error) {
	since := time.Now().UTC().Add(-window)
	return s.alertRepo.FindRecentOpenBySourceAndMetadataValue(ctx, tenantID, source, metadataKey, metadataValue, since)
}

func (s *AlertService) UpdateEventAlert(ctx context.Context, alert *model.Alert) (*model.Alert, error) {
	updated, err := s.alertRepo.UpdateEventAlert(ctx, alert)
	if err != nil {
		return nil, err
	}
	_ = s.alertRepo.InsertTimeline(ctx, &model.AlertTimelineEntry{
		TenantID:    alert.TenantID,
		AlertID:     updated.ID,
		Action:      "correlated",
		Description: fmt.Sprintf("Cross-suite event alert updated to %d correlated events", updated.EventCount),
		Metadata:    mustJSON(map[string]interface{}{"event_count": updated.EventCount}),
	})
	return updated, nil
}

func actorUUID(actor *Actor) *uuid.UUID {
	if actor == nil || actor.UserID == uuid.Nil {
		return nil
	}
	id := actor.UserID
	return &id
}

func actorName(actor *Actor) *string {
	if actor == nil {
		return nil
	}
	value := safeActorName(actor)
	return &value
}

func safeActorName(actor *Actor) string {
	if actor == nil {
		return "system"
	}
	if actor.UserName != "" {
		return actor.UserName
	}
	if actor.UserEmail != "" {
		return actor.UserEmail
	}
	return actor.UserID.String()
}

func stringPtr(value string) *string {
	return &value
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}

func uuidStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
