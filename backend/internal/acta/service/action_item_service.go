package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/dto"
	"github.com/clario360/platform/internal/acta/metrics"
	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
)

type ActionItemService struct {
	store     *repository.Store
	publisher Publisher
	metrics   *metrics.Metrics
	logger    zerolog.Logger
}

func NewActionItemService(store *repository.Store, publisher Publisher, metrics *metrics.Metrics, logger zerolog.Logger) *ActionItemService {
	return &ActionItemService{
		store:     store,
		publisher: publisherOrNoop(publisher),
		metrics:   metrics,
		logger:    logger.With().Str("component", "acta_action_item_service").Logger(),
	}
}

func (s *ActionItemService) CreateActionItem(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateActionItemRequest) (*model.ActionItem, error) {
	req.Normalize()
	if req.Title == "" {
		return nil, validationError("title is required", map[string]string{"title": "required"})
	}
	if req.MeetingID == uuid.Nil {
		return nil, validationError("meeting_id is required", map[string]string{"meeting_id": "required"})
	}
	if req.CommitteeID == uuid.Nil {
		return nil, validationError("committee_id is required", map[string]string{"committee_id": "required"})
	}
	if req.AssignedTo == uuid.Nil || req.AssigneeName == "" {
		return nil, validationError("assigned_to and assignee_name are required", map[string]string{"assigned_to": "required", "assignee_name": "required"})
	}
	if req.DueDate.IsZero() {
		return nil, validationError("due_date is required", map[string]string{"due_date": "required"})
	}
	meeting, err := s.store.GetMeeting(ctx, tenantID, req.MeetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.CommitteeID != req.CommitteeID {
		return nil, validationError("committee_id does not match the meeting committee", map[string]string{"committee_id": "mismatch"})
	}
	committee, err := s.store.GetCommittee(ctx, tenantID, req.CommitteeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	now := time.Now().UTC()
	item := &model.ActionItem{
		ID:              uuid.New(),
		TenantID:        tenantID,
		MeetingID:       req.MeetingID,
		AgendaItemID:    req.AgendaItemID,
		CommitteeID:     req.CommitteeID,
		Title:           req.Title,
		Description:     req.Description,
		Priority:        model.ActionItemPriority(req.Priority),
		AssignedTo:      req.AssignedTo,
		AssigneeName:    req.AssigneeName,
		AssignedBy:      userID,
		DueDate:         req.DueDate.UTC(),
		OriginalDueDate: req.DueDate.UTC(),
		Status:          model.ActionItemStatusPending,
		Tags:            req.Tags,
		Metadata:        req.Metadata,
		CreatedBy:       userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if item.Priority == "" {
		item.Priority = model.ActionItemPriorityMedium
	}
	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if err := s.store.CreateActionItem(ctx, tx, item); err != nil {
			return err
		}
		return s.store.UpdateMeetingActionItemCount(ctx, tx, tenantID, req.MeetingID)
	}); err != nil {
		if appErr := unwrapAppError(err); appErr != nil {
			return nil, appErr
		}
		return nil, internalError("failed to create action item", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.action_item.created", tenantID, &userID, map[string]any{
		"id":            item.ID,
		"title":         item.Title,
		"meeting_id":    item.MeetingID,
		"assigned_to":   item.AssignedTo,
		"due_date":      item.DueDate.Format("2006-01-02"),
		"chair_user_id": committee.ChairUserID,
	}, s.logger)
	return s.store.GetActionItem(ctx, tenantID, item.ID)
}

func (s *ActionItemService) CreateFromExtracted(ctx context.Context, tenantID, userID, meetingID, committeeID uuid.UUID, agendaItemID *uuid.UUID, extracted []model.ExtractedAction) ([]model.ActionItem, error) {
	out := make([]model.ActionItem, 0, len(extracted))
	for _, action := range extracted {
		assignedTo := uuid.New()
		dueDate := time.Now().UTC().AddDate(0, 0, 7)
		if action.DueDate != nil {
			dueDate = action.DueDate.UTC()
		}
		item := dto.CreateActionItemRequest{
			MeetingID:    meetingID,
			AgendaItemID: agendaItemID,
			CommitteeID:  committeeID,
			Title:        action.Title,
			Description:  action.Description,
			Priority:     action.Priority,
			AssignedTo:   assignedTo,
			AssigneeName: action.AssignedTo,
			DueDate:      dueDate,
		}
		created, err := s.CreateActionItem(ctx, tenantID, userID, item)
		if err != nil {
			return nil, err
		}
		out = append(out, *created)
	}
	return out, nil
}

func (s *ActionItemService) ListActionItems(ctx context.Context, tenantID uuid.UUID, filters model.ActionItemFilters) ([]model.ActionItem, int, error) {
	items, total, err := s.store.ListActionItems(ctx, tenantID, filters)
	if err != nil {
		return nil, 0, err
	}
	s.enrichMeetingTitles(ctx, tenantID, items)
	return items, total, nil
}

func (s *ActionItemService) GetActionItem(ctx context.Context, tenantID, actionItemID uuid.UUID) (*model.ActionItem, error) {
	item, err := s.store.GetActionItem(ctx, tenantID, actionItemID)
	if err != nil {
		return nil, notFoundError("action item not found")
	}
	titles, _ := s.store.GetMeetingTitles(ctx, tenantID, []uuid.UUID{item.MeetingID})
	if t, ok := titles[item.MeetingID]; ok {
		item.MeetingTitle = t
	}
	return item, nil
}

// enrichMeetingTitles sets MeetingTitle on each action item via a batch lookup.
func (s *ActionItemService) enrichMeetingTitles(ctx context.Context, tenantID uuid.UUID, items []model.ActionItem) {
	ids := make(map[uuid.UUID]struct{})
	for _, item := range items {
		ids[item.MeetingID] = struct{}{}
	}
	meetingIDs := make([]uuid.UUID, 0, len(ids))
	for id := range ids {
		meetingIDs = append(meetingIDs, id)
	}
	titles, err := s.store.GetMeetingTitles(ctx, tenantID, meetingIDs)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to enrich action item meeting titles")
		return
	}
	for i := range items {
		if t, ok := titles[items[i].MeetingID]; ok {
			items[i].MeetingTitle = t
		}
	}
}

func (s *ActionItemService) UpdateActionItem(ctx context.Context, tenantID, userID, actionItemID uuid.UUID, req dto.UpdateActionItemRequest) (*model.ActionItem, error) {
	req.Normalize()
	item, err := s.store.GetActionItem(ctx, tenantID, actionItemID)
	if err != nil {
		return nil, notFoundError("action item not found")
	}
	item.Title = req.Title
	item.Description = req.Description
	item.Priority = model.ActionItemPriority(req.Priority)
	item.AssignedTo = req.AssignedTo
	item.AssigneeName = req.AssigneeName
	item.DueDate = req.DueDate.UTC()
	item.Tags = req.Tags
	item.Metadata = req.Metadata
	item.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateActionItem(ctx, s.store.DB(), item); err != nil {
		return nil, internalError("failed to update action item", err)
	}
	return s.store.GetActionItem(ctx, tenantID, actionItemID)
}

func (s *ActionItemService) UpdateStatus(ctx context.Context, tenantID, userID, actionItemID uuid.UUID, req dto.UpdateActionItemStatusRequest) (*model.ActionItem, error) {
	item, err := s.store.GetActionItem(ctx, tenantID, actionItemID)
	if err != nil {
		return nil, notFoundError("action item not found")
	}
	oldStatus := item.Status
	newStatus := model.ActionItemStatus(req.Status)
	if !isValidActionTransition(oldStatus, newStatus) {
		return nil, validationError("invalid action item status transition", nil)
	}
	now := time.Now().UTC()
	item.Status = newStatus
	item.UpdatedAt = now
	if newStatus == model.ActionItemStatusCompleted {
		item.CompletedAt = &now
		item.CompletionNotes = req.CompletionNotes
		item.CompletionEvidence = req.CompletionEvidence
		if item.CreatedAt.Before(now) && s.metrics != nil {
			s.metrics.ActionItemsCompletionDays.Observe(now.Sub(item.CreatedAt).Hours() / 24)
		}
	}
	if err := s.store.UpdateActionItem(ctx, s.store.DB(), item); err != nil {
		return nil, internalError("failed to update action item status", err)
	}
	eventType := "acta.action_item.status_changed"
	if newStatus == model.ActionItemStatusCompleted {
		eventType = "acta.action_item.completed"
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, eventType, tenantID, &userID, map[string]any{
		"id":         item.ID,
		"title":      item.Title,
		"old_status": oldStatus,
		"new_status": newStatus,
	}, s.logger)
	return s.store.GetActionItem(ctx, tenantID, actionItemID)
}

func (s *ActionItemService) ExtendDueDate(ctx context.Context, tenantID, userID, actionItemID uuid.UUID, req dto.ExtendActionItemRequest) (*model.ActionItem, error) {
	item, err := s.store.GetActionItem(ctx, tenantID, actionItemID)
	if err != nil {
		return nil, notFoundError("action item not found")
	}
	if req.NewDueDate.UTC().Before(item.DueDate.UTC()) {
		return nil, validationError("new due date must not be earlier than current due date", nil)
	}
	reason := normalizeString(req.Reason)
	if reason == "" {
		return nil, validationError("extension reason is required", map[string]string{"reason": "required"})
	}
	oldDueDate := item.DueDate
	item.DueDate = req.NewDueDate.UTC()
	item.ExtendedCount++
	item.ExtensionReason = &reason
	item.UpdatedAt = time.Now().UTC()
	if item.Status == model.ActionItemStatusOverdue {
		item.Status = model.ActionItemStatusInProgress
	}
	if err := s.store.UpdateActionItem(ctx, s.store.DB(), item); err != nil {
		return nil, internalError("failed to extend action item", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.action_item.extended", tenantID, &userID, map[string]any{
		"id":           item.ID,
		"old_due_date": oldDueDate.Format("2006-01-02"),
		"new_due_date": item.DueDate.Format("2006-01-02"),
		"reason":       reason,
	}, s.logger)
	return s.store.GetActionItem(ctx, tenantID, actionItemID)
}

func (s *ActionItemService) ListOverdue(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.ActionItem, error) {
	return s.store.ListOverdueActionItems(ctx, tenantID, limit)
}

func (s *ActionItemService) ListMyActionItems(ctx context.Context, tenantID, userID uuid.UUID) ([]model.ActionItem, error) {
	items, _, err := s.store.ListActionItems(ctx, tenantID, model.ActionItemFilters{
		AssigneeID: &userID,
		Page:       1,
		PerPage:    250,
	})
	return items, err
}

func (s *ActionItemService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ActionItemStats, error) {
	byStatus, err := s.store.CountActionItemsByStatus(ctx, tenantID)
	if err != nil {
		return nil, internalError("failed to count action items by status", err)
	}
	byPriority, err := s.store.CountActionItemsByPriority(ctx, tenantID)
	if err != nil {
		return nil, internalError("failed to count action items by priority", err)
	}
	stats := &model.ActionItemStats{
		ByStatus:   byStatus,
		ByPriority: byPriority,
		Open:       byStatus["pending"] + byStatus["in_progress"] + byStatus["deferred"] + byStatus["overdue"],
		Overdue:    byStatus["overdue"],
		Completed:  byStatus["completed"],
	}
	return stats, nil
}

func (s *ActionItemService) MarkOverdueItems(ctx context.Context) error {
	tenantIDs, err := s.store.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, tenantID := range tenantIDs {
		items, err := s.store.ListOverdueActionItems(ctx, tenantID, 1000)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.Status == model.ActionItemStatusOverdue {
				continue
			}
			if err := s.store.MarkActionItemOverdue(ctx, s.store.DB(), tenantID, item.ID, now); err != nil {
				return err
			}
			committee, err := s.store.GetCommittee(ctx, tenantID, item.CommitteeID)
			if err != nil {
				s.logger.Warn().Err(err).Str("committee_id", item.CommitteeID.String()).Msg("failed to load committee for overdue action item event")
			}
			var chairUserID uuid.UUID
			if committee != nil {
				chairUserID = committee.ChairUserID
			}
			publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.action_item.overdue", tenantID, nil, map[string]any{
				"id":            item.ID,
				"title":         item.Title,
				"assigned_to":   item.AssignedTo,
				"due_date":      item.DueDate.Format("2006-01-02"),
				"chair_user_id": chairUserID,
			}, s.logger)
		}
	}
	return nil
}

func isValidActionTransition(from, to model.ActionItemStatus) bool {
	allowed := map[model.ActionItemStatus]map[model.ActionItemStatus]bool{
		model.ActionItemStatusPending: {
			model.ActionItemStatusInProgress: true,
			model.ActionItemStatusCompleted:  true,
			model.ActionItemStatusCancelled:  true,
			model.ActionItemStatusDeferred:   true,
			model.ActionItemStatusOverdue:    true,
		},
		model.ActionItemStatusInProgress: {
			model.ActionItemStatusCompleted: true,
			model.ActionItemStatusCancelled: true,
			model.ActionItemStatusDeferred:  true,
			model.ActionItemStatusOverdue:   true,
		},
		model.ActionItemStatusDeferred: {
			model.ActionItemStatusPending:    true,
			model.ActionItemStatusInProgress: true,
			model.ActionItemStatusCancelled:  true,
			model.ActionItemStatusOverdue:    true,
		},
		model.ActionItemStatusOverdue: {
			model.ActionItemStatusInProgress: true,
			model.ActionItemStatusCompleted:  true,
			model.ActionItemStatusCancelled:  true,
			model.ActionItemStatusDeferred:   true,
		},
	}
	return allowed[from][to]
}
