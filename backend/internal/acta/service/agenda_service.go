package service

import (
	"context"
	"fmt"
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

type AgendaService struct {
	store     *repository.Store
	publisher Publisher
	metrics   *metrics.Metrics
	logger    zerolog.Logger
}

func NewAgendaService(store *repository.Store, publisher Publisher, metrics *metrics.Metrics, logger zerolog.Logger) *AgendaService {
	return &AgendaService{
		store:     store,
		publisher: publisherOrNoop(publisher),
		metrics:   metrics,
		logger:    logger.With().Str("component", "acta_agenda_service").Logger(),
	}
}

func (s *AgendaService) AddAgendaItem(ctx context.Context, tenantID, userID uuid.UUID, meetingID uuid.UUID, req dto.CreateAgendaItemRequest) (*model.AgendaItem, error) {
	req.Normalize()
	if req.Title == "" {
		return nil, validationError("title is required", map[string]string{"title": "required"})
	}
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status == model.MeetingStatusCompleted || meeting.Status == model.MeetingStatusCancelled {
		return nil, validationError("agenda cannot be modified for completed or cancelled meetings", nil)
	}

	existing, err := s.store.ListAgendaItems(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to list agenda items", err)
	}
	orderIndex := len(existing)
	if req.OrderIndex != nil && *req.OrderIndex >= 0 && *req.OrderIndex <= len(existing) {
		orderIndex = *req.OrderIndex
	}

	now := time.Now().UTC()
	item := &model.AgendaItem{
		ID:              uuid.New(),
		TenantID:        tenantID,
		MeetingID:       meetingID,
		Title:           req.Title,
		Description:     req.Description,
		ItemNumber:      req.ItemNumber,
		PresenterUserID: req.PresenterUserID,
		PresenterName:   req.PresenterName,
		DurationMinutes: req.DurationMinutes,
		OrderIndex:      orderIndex,
		ParentItemID:    req.ParentItemID,
		Status:          model.AgendaItemStatusPending,
		RequiresVote:    req.RequiresVote,
		AttachmentIDs:   req.AttachmentIDs,
		Confidential:    req.Confidential,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if item.DurationMinutes == 0 {
		item.DurationMinutes = 15
	}
	if req.VoteType != nil && *req.VoteType != "" {
		voteType := model.VoteType(*req.VoteType)
		item.VoteType = &voteType
	}
	if req.Category != nil && *req.Category != "" {
		category := model.AgendaCategory(*req.Category)
		item.Category = &category
	}

	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if orderIndex < len(existing) {
			orderedIDs := make([]uuid.UUID, 0, len(existing)+1)
			for idx, agendaItem := range existing {
				if idx == orderIndex {
					orderedIDs = append(orderedIDs, item.ID)
				}
				orderedIDs = append(orderedIDs, agendaItem.ID)
			}
			if orderIndex == len(existing) {
				orderedIDs = append(orderedIDs, item.ID)
			}
			if err := s.store.CreateAgendaItem(ctx, tx, item); err != nil {
				return err
			}
			return s.store.ReorderAgendaItems(ctx, tx, tenantID, meetingID, orderedIDs)
		}
		if err := s.store.CreateAgendaItem(ctx, tx, item); err != nil {
			return err
		}
		return s.store.UpdateMeetingAgendaCount(ctx, tx, tenantID, meetingID)
	}); err != nil {
		return nil, internalError("failed to create agenda item", err)
	}

	if s.metrics != nil {
		s.metrics.AgendaItemsTotal.Inc()
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.agenda.item_added", tenantID, &userID, map[string]any{
		"id":         item.ID,
		"meeting_id": meetingID,
		"title":      item.Title,
	}, s.logger)
	return s.store.GetAgendaItem(ctx, tenantID, meetingID, item.ID)
}

func (s *AgendaService) ListAgendaItems(ctx context.Context, tenantID, meetingID uuid.UUID) ([]model.AgendaItem, error) {
	items, err := s.store.ListAgendaItems(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to list agenda items", err)
	}
	return items, nil
}

func (s *AgendaService) UpdateAgendaItem(ctx context.Context, tenantID, userID, meetingID, itemID uuid.UUID, req dto.UpdateAgendaItemRequest) (*model.AgendaItem, error) {
	req.Normalize()
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status == model.MeetingStatusCompleted || meeting.Status == model.MeetingStatusCancelled {
		return nil, validationError("agenda cannot be modified for completed or cancelled meetings", nil)
	}
	item, err := s.store.GetAgendaItem(ctx, tenantID, meetingID, itemID)
	if err != nil {
		return nil, notFoundError("agenda item not found")
	}
	item.Title = req.Title
	item.Description = req.Description
	item.ItemNumber = req.ItemNumber
	item.PresenterUserID = req.PresenterUserID
	item.PresenterName = req.PresenterName
	item.DurationMinutes = req.DurationMinutes
	item.ParentItemID = req.ParentItemID
	item.RequiresVote = req.RequiresVote
	item.AttachmentIDs = req.AttachmentIDs
	item.Confidential = req.Confidential
	if req.Status != "" {
		item.Status = model.AgendaItemStatus(req.Status)
	}
	if req.VoteType != nil && *req.VoteType != "" {
		voteType := model.VoteType(*req.VoteType)
		item.VoteType = &voteType
	}
	if req.Category != nil && *req.Category != "" {
		category := model.AgendaCategory(*req.Category)
		item.Category = &category
	}
	item.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateAgendaItem(ctx, s.store.DB(), item); err != nil {
		return nil, internalError("failed to update agenda item", err)
	}
	return s.store.GetAgendaItem(ctx, tenantID, meetingID, itemID)
}

func (s *AgendaService) DeleteAgendaItem(ctx context.Context, tenantID, meetingID, itemID uuid.UUID) error {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return notFoundError("meeting not found")
	}
	if meeting.Status == model.MeetingStatusCompleted || meeting.Status == model.MeetingStatusCancelled {
		return validationError("agenda cannot be modified for completed or cancelled meetings", nil)
	}
	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if err := s.store.DeleteAgendaItem(ctx, tx, tenantID, meetingID, itemID); err != nil {
			return err
		}
		items, err := s.store.ListAgendaItems(ctx, tenantID, meetingID)
		if err != nil {
			return err
		}
		ids := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		if len(ids) > 0 {
			if err := s.store.ReorderAgendaItems(ctx, tx, tenantID, meetingID, ids); err != nil {
				return err
			}
		}
		return s.store.UpdateMeetingAgendaCount(ctx, tx, tenantID, meetingID)
	}); err != nil {
		if appErr := unwrapAppError(err); appErr != nil {
			return appErr
		}
		return internalError("failed to delete agenda item", err)
	}
	return nil
}

func (s *AgendaService) ReorderAgendaItems(ctx context.Context, tenantID, meetingID uuid.UUID, req dto.ReorderAgendaRequest) error {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return notFoundError("meeting not found")
	}
	if meeting.Status == model.MeetingStatusCompleted || meeting.Status == model.MeetingStatusCancelled {
		return validationError("agenda cannot be modified for completed or cancelled meetings", nil)
	}
	items, err := s.store.ListAgendaItems(ctx, tenantID, meetingID)
	if err != nil {
		return internalError("failed to list agenda items", err)
	}
	if len(req.ItemIDs) != len(items) {
		return validationError("item_ids must contain every agenda item exactly once", nil)
	}
	expected := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		expected[item.ID] = struct{}{}
	}
	seen := make(map[uuid.UUID]struct{}, len(req.ItemIDs))
	for _, itemID := range req.ItemIDs {
		if _, ok := expected[itemID]; !ok {
			return validationError("agenda item list contains invalid item id", map[string]string{"item_ids": "contains item not in meeting"})
		}
		if _, ok := seen[itemID]; ok {
			return validationError("agenda item list contains duplicate item id", map[string]string{"item_ids": "duplicate id"})
		}
		seen[itemID] = struct{}{}
	}
	if err := s.store.ReorderAgendaItems(ctx, s.store.DB(), tenantID, meetingID, req.ItemIDs); err != nil {
		return internalError("failed to reorder agenda items", err)
	}
	return nil
}

func (s *AgendaService) UpdateAgendaNotes(ctx context.Context, tenantID, meetingID, itemID uuid.UUID, req dto.UpdateAgendaNotesRequest) (*model.AgendaItem, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusInProgress {
		return nil, validationError("agenda notes can only be updated while the meeting is in progress", nil)
	}
	item, err := s.store.GetAgendaItem(ctx, tenantID, meetingID, itemID)
	if err != nil {
		return nil, notFoundError("agenda item not found")
	}
	notes := req.Notes
	item.Notes = &notes
	item.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateAgendaItem(ctx, s.store.DB(), item); err != nil {
		return nil, internalError("failed to update agenda notes", err)
	}
	return s.store.GetAgendaItem(ctx, tenantID, meetingID, itemID)
}

func (s *AgendaService) RecordVote(ctx context.Context, tenantID, userID, meetingID, itemID uuid.UUID, req dto.RecordVoteRequest) (*model.AgendaItem, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusInProgress {
		return nil, validationError("meeting must be in progress before recording votes", nil)
	}

	item, err := s.store.GetAgendaItem(ctx, tenantID, meetingID, itemID)
	if err != nil {
		return nil, notFoundError("agenda item not found")
	}
	if !item.RequiresVote {
		return nil, validationError("agenda item does not require a vote", nil)
	}

	totalVotes := req.VotesFor + req.VotesAgainst + req.VotesAbstained
	if totalVotes > meeting.PresentCount {
		return nil, validationError("vote tally cannot exceed present attendees", map[string]string{"votes_for": "vote tally exceeds present_count"})
	}

	result, err := determineVoteResult(model.VoteType(req.VoteType), req.VotesFor, req.VotesAgainst, req.VotesAbstained)
	if err != nil {
		return nil, validationError(err.Error(), nil)
	}

	voteType := model.VoteType(req.VoteType)
	item.VoteType = &voteType
	item.VotesFor = &req.VotesFor
	item.VotesAgainst = &req.VotesAgainst
	item.VotesAbstained = &req.VotesAbstained
	item.VoteResult = &result
	item.Status = model.AgendaItemStatus(result)
	if req.Notes != "" {
		notes := req.Notes
		item.VoteNotes = &notes
	}
	item.UpdatedAt = time.Now().UTC()

	if err := s.store.UpdateAgendaItem(ctx, s.store.DB(), item); err != nil {
		return nil, internalError("failed to record agenda vote", err)
	}
	if s.metrics != nil {
		s.metrics.AgendaVotesTotal.WithLabelValues(string(result)).Inc()
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.agenda.voted", tenantID, &userID, map[string]any{
		"id":          item.ID,
		"meeting_id":  meetingID,
		"vote_result": result,
	}, s.logger)
	return s.store.GetAgendaItem(ctx, tenantID, meetingID, itemID)
}

func determineVoteResult(voteType model.VoteType, votesFor, votesAgainst, votesAbstained int) (model.VoteResult, error) {
	switch voteType {
	case model.VoteTypeUnanimous:
		if votesAgainst != 0 || votesAbstained != 0 {
			return "", fmt.Errorf("unanimous votes cannot include against or abstained counts")
		}
		return model.VoteResultApproved, nil
	case model.VoteTypeMajority:
		if votesFor > votesAgainst {
			return model.VoteResultApproved, nil
		}
		if votesFor == votesAgainst {
			return model.VoteResultTied, nil
		}
		return model.VoteResultRejected, nil
	case model.VoteTypeTwoThirds:
		decidingVotes := votesFor + votesAgainst
		if decidingVotes == 0 {
			return "", fmt.Errorf("two-thirds votes require at least one deciding vote")
		}
		if votesFor == votesAgainst {
			return model.VoteResultTied, nil
		}
		if float64(votesFor) >= (2.0/3.0)*float64(decidingVotes) {
			return model.VoteResultApproved, nil
		}
		return model.VoteResultRejected, nil
	case model.VoteTypeRollCall:
		if votesFor == votesAgainst {
			return model.VoteResultTied, nil
		}
		if votesFor > votesAgainst {
			return model.VoteResultApproved, nil
		}
		return model.VoteResultRejected, nil
	default:
		return "", fmt.Errorf("unsupported vote type")
	}
}
