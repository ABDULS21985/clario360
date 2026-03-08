package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/ai"
	"github.com/clario360/platform/internal/acta/dto"
	"github.com/clario360/platform/internal/acta/metrics"
	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
)

type MinutesService struct {
	store     *repository.Store
	generator *ai.MinutesGenerator
	publisher Publisher
	metrics   *metrics.Metrics
	logger    zerolog.Logger
}

func NewMinutesService(store *repository.Store, generator *ai.MinutesGenerator, publisher Publisher, metrics *metrics.Metrics, logger zerolog.Logger) *MinutesService {
	return &MinutesService{
		store:     store,
		generator: generator,
		publisher: publisherOrNoop(publisher),
		metrics:   metrics,
		logger:    logger.With().Str("component", "acta_minutes_service").Logger(),
	}
}

func (s *MinutesService) CreateMinutes(ctx context.Context, tenantID, userID, meetingID uuid.UUID, req dto.CreateMinutesRequest) (*model.MeetingMinutes, error) {
	req.Normalize()
	if req.Content == "" {
		return nil, validationError("content is required", map[string]string{"content": "required"})
	}
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusCompleted {
		return nil, validationError("minutes can only be created for completed meetings", nil)
	}
	if _, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID); err == nil {
		return nil, conflictError("minutes already exist for this meeting; use update instead")
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, internalError("failed to inspect existing meeting minutes", err)
	}
	now := time.Now().UTC()
	minutes := &model.MeetingMinutes{
		ID:            uuid.New(),
		TenantID:      tenantID,
		MeetingID:     meetingID,
		Content:       req.Content,
		Status:        model.MinutesStatusDraft,
		Version:       1,
		AIActionItems: []model.ExtractedAction{},
		CreatedBy:     userID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := s.store.CreateMinutes(ctx, s.store.DB(), minutes); err != nil {
		return nil, internalError("failed to create meeting minutes", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr(string(minutes.Status))); err != nil {
		return nil, internalError("failed to update meeting minutes state", err)
	}
	s.recordMinutesTransition("", string(minutes.Status))
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) GetLatestMinutes(ctx context.Context, tenantID, meetingID uuid.UUID) (*model.MeetingMinutes, error) {
	minutes, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("minutes not found")
	}
	return minutes, nil
}

func (s *MinutesService) ListVersions(ctx context.Context, tenantID, meetingID uuid.UUID) ([]model.MeetingMinutes, error) {
	return s.store.ListMinutesVersions(ctx, tenantID, meetingID)
}

func (s *MinutesService) GenerateMinutes(ctx context.Context, tenantID, userID, meetingID uuid.UUID) (*model.MeetingMinutes, error) {
	start := time.Now()
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusCompleted {
		return nil, validationError("minutes can only be generated for completed meetings", nil)
	}
	agenda, err := s.store.ListAgendaItems(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to load agenda items", err)
	}
	attendance, err := s.store.ListAttendance(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to load attendance", err)
	}
	actionItems, _, err := s.store.ListActionItems(ctx, tenantID, model.ActionItemFilters{MeetingID: &meetingID, Page: 1, PerPage: 250})
	if err != nil {
		return nil, internalError("failed to load action items", err)
	}
	nextMeeting, err := s.store.GetNextMeeting(ctx, tenantID, meeting.CommitteeID, meeting.ScheduledAt)
	if err != nil {
		return nil, internalError("failed to load next meeting", err)
	}

	generated, err := s.generator.Generate(meeting, agenda, attendance, actionItems, nextMeeting)
	if err != nil {
		return nil, internalError("failed to generate deterministic minutes", err)
	}

	latest, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		latest = nil
	case err != nil:
		return nil, internalError("failed to inspect existing minutes", err)
	}

	now := time.Now().UTC()
	var minutes *model.MeetingMinutes
	err = database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		switch {
		case latest == nil:
			minutes = &model.MeetingMinutes{
				ID:            uuid.New(),
				TenantID:      tenantID,
				MeetingID:     meetingID,
				Content:       generated.Content,
				AISummary:     ptr(generated.AISummary),
				Status:        model.MinutesStatusDraft,
				Version:       1,
				AIActionItems: generated.AIActionItems,
				AIGenerated:   true,
				CreatedBy:     userID,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			return s.store.CreateMinutes(ctx, tx, minutes)
		case latest.Status == model.MinutesStatusApproved || latest.Status == model.MinutesStatusPublished:
			minutes = &model.MeetingMinutes{
				ID:                uuid.New(),
				TenantID:          tenantID,
				MeetingID:         meetingID,
				Content:           generated.Content,
				AISummary:         ptr(generated.AISummary),
				Status:            model.MinutesStatusDraft,
				Version:           latest.Version + 1,
				PreviousVersionID: &latest.ID,
				AIActionItems:     generated.AIActionItems,
				AIGenerated:       true,
				CreatedBy:         userID,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			return s.store.CreateMinutes(ctx, tx, minutes)
		default:
			previousStatus := latest.Status
			latest.Content = generated.Content
			latest.AISummary = ptr(generated.AISummary)
			latest.AIActionItems = generated.AIActionItems
			latest.AIGenerated = true
			latest.Status = model.MinutesStatusDraft
			latest.UpdatedAt = now
			minutes = latest
			if previousStatus != latest.Status {
				s.recordMinutesTransition(string(previousStatus), string(latest.Status))
			}
			return s.store.UpdateMinutes(ctx, tx, latest)
		}
	})
	if err != nil {
		return nil, internalError("failed to persist generated minutes", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr("draft")); err != nil {
		return nil, internalError("failed to update meeting minutes state", err)
	}
	if s.metrics != nil {
		s.metrics.MinutesGenerationDurationSeconds.Observe(time.Since(start).Seconds())
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.minutes.generated", tenantID, &userID, map[string]any{
		"id":           minutes.ID,
		"meeting_id":   meetingID,
		"ai_generated": true,
	}, s.logger)
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) UpdateMinutes(ctx context.Context, tenantID, userID, meetingID uuid.UUID, req dto.UpdateMinutesRequest) (*model.MeetingMinutes, error) {
	req.Normalize()
	if req.Content == "" {
		return nil, validationError("content is required", map[string]string{"content": "required"})
	}
	latest, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("minutes not found")
	}
	now := time.Now().UTC()
	var updated *model.MeetingMinutes
	err = database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if latest.Status == model.MinutesStatusApproved || latest.Status == model.MinutesStatusPublished {
			updated = &model.MeetingMinutes{
				ID:                uuid.New(),
				TenantID:          tenantID,
				MeetingID:         meetingID,
				Content:           req.Content,
				AISummary:         latest.AISummary,
				Status:            model.MinutesStatusDraft,
				Version:           latest.Version + 1,
				PreviousVersionID: &latest.ID,
				AIActionItems:     latest.AIActionItems,
				AIGenerated:       latest.AIGenerated,
				CreatedBy:         userID,
				CreatedAt:         now,
				UpdatedAt:         now,
			}
			s.recordMinutesTransition(string(latest.Status), string(updated.Status))
			return s.store.CreateMinutes(ctx, tx, updated)
		}
		updated = latest
		updated.Content = req.Content
		updated.UpdatedAt = now
		return s.store.UpdateMinutes(ctx, tx, updated)
	})
	if err != nil {
		return nil, internalError("failed to update minutes", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr(string(updated.Status))); err != nil {
		return nil, internalError("failed to refresh meeting minutes state", err)
	}
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) SubmitForReview(ctx context.Context, tenantID, userID, meetingID uuid.UUID) (*model.MeetingMinutes, error) {
	minutes, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("minutes not found")
	}
	if minutes.Status != model.MinutesStatusDraft && minutes.Status != model.MinutesStatusRevisionRequested {
		return nil, validationError("only draft or revision requested minutes can be submitted for review", nil)
	}
	now := time.Now().UTC()
	oldStatus := minutes.Status
	minutes.Status = model.MinutesStatusReview
	minutes.SubmittedForReviewAt = &now
	minutes.SubmittedBy = &userID
	minutes.UpdatedAt = now
	if err := s.store.UpdateMinutes(ctx, s.store.DB(), minutes); err != nil {
		return nil, internalError("failed to submit minutes for review", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr(string(minutes.Status))); err != nil {
		return nil, internalError("failed to refresh meeting minutes state", err)
	}
	s.recordMinutesTransition(string(oldStatus), string(minutes.Status))
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.minutes.submitted", tenantID, &userID, map[string]any{
		"id":         minutes.ID,
		"meeting_id": meetingID,
	}, s.logger)
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) RequestRevision(ctx context.Context, tenantID, userID, meetingID uuid.UUID, req dto.ReviewRequest) (*model.MeetingMinutes, error) {
	req.Normalize()
	minutes, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("minutes not found")
	}
	if minutes.Status != model.MinutesStatusReview {
		return nil, validationError("only minutes in review can request revision", nil)
	}
	oldStatus := minutes.Status
	minutes.Status = model.MinutesStatusRevisionRequested
	minutes.ReviewedBy = &userID
	minutes.ReviewNotes = &req.Notes
	minutes.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMinutes(ctx, s.store.DB(), minutes); err != nil {
		return nil, internalError("failed to request minutes revision", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr(string(minutes.Status))); err != nil {
		return nil, internalError("failed to refresh meeting minutes state", err)
	}
	s.recordMinutesTransition(string(oldStatus), string(minutes.Status))
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) Approve(ctx context.Context, tenantID, userID, meetingID uuid.UUID) (*model.MeetingMinutes, error) {
	minutes, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("minutes not found")
	}
	if minutes.Status != model.MinutesStatusReview {
		return nil, validationError("minutes must be in review before approval", nil)
	}
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	committee, err := s.store.GetCommittee(ctx, tenantID, meeting.CommitteeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	if userID != committee.ChairUserID {
		return nil, forbiddenError("only the committee chair can approve minutes")
	}
	oldStatus := minutes.Status
	now := time.Now().UTC()
	minutes.Status = model.MinutesStatusApproved
	minutes.ApprovedBy = &userID
	minutes.ApprovedAt = &now
	minutes.UpdatedAt = now
	if err := s.store.UpdateMinutes(ctx, s.store.DB(), minutes); err != nil {
		return nil, internalError("failed to approve minutes", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr(string(minutes.Status))); err != nil {
		return nil, internalError("failed to refresh meeting minutes state", err)
	}
	s.recordMinutesTransition(string(oldStatus), string(minutes.Status))
	attendance, err := s.store.ListAttendance(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to load meeting attendance", err)
	}
	attendeeIDs := make([]uuid.UUID, 0, len(attendance))
	for _, attendee := range attendance {
		attendeeIDs = append(attendeeIDs, attendee.UserID)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.minutes.approved", tenantID, &userID, map[string]any{
		"id":           minutes.ID,
		"meeting_id":   meetingID,
		"approved_by":  userID,
		"attendee_ids": attendeeIDs,
	}, s.logger)
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) Publish(ctx context.Context, tenantID, userID, meetingID uuid.UUID) (*model.MeetingMinutes, error) {
	minutes, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("minutes not found")
	}
	if minutes.Status != model.MinutesStatusApproved {
		return nil, validationError("only approved minutes can be published", nil)
	}
	oldStatus := minutes.Status
	now := time.Now().UTC()
	minutes.Status = model.MinutesStatusPublished
	minutes.PublishedAt = &now
	minutes.UpdatedAt = now
	if err := s.store.UpdateMinutes(ctx, s.store.DB(), minutes); err != nil {
		return nil, internalError("failed to publish minutes", err)
	}
	if err := s.store.UpdateMeetingMinutesState(ctx, s.store.DB(), tenantID, meetingID, true, ptr(string(minutes.Status))); err != nil {
		return nil, internalError("failed to refresh meeting minutes state", err)
	}
	s.recordMinutesTransition(string(oldStatus), string(minutes.Status))
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.minutes.published", tenantID, &userID, map[string]any{
		"id":         minutes.ID,
		"meeting_id": meetingID,
	}, s.logger)
	return s.store.GetLatestMinutes(ctx, tenantID, meetingID)
}

func (s *MinutesService) recordMinutesTransition(from, to string) {
	if s.metrics == nil || from == to {
		return
	}
	if from != "" {
		s.metrics.MinutesStatus.WithLabelValues(from).Dec()
	}
	if to != "" {
		s.metrics.MinutesStatus.WithLabelValues(to).Inc()
	}
}
