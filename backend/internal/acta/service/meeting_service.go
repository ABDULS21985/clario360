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
	workflowmodel "github.com/clario360/platform/internal/workflow/model"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
	workflowservice "github.com/clario360/platform/internal/workflow/service"
)

type MeetingService struct {
	store            *repository.Store
	publisher        Publisher
	metrics          *metrics.Metrics
	logger           zerolog.Logger
	kafkaTopic       string
	workflowDefRepo  *workflowrepo.DefinitionRepository
	workflowInstRepo *workflowrepo.InstanceRepository
}

func NewMeetingService(
	store *repository.Store,
	publisher Publisher,
	metrics *metrics.Metrics,
	kafkaTopic string,
	workflowDefRepo *workflowrepo.DefinitionRepository,
	workflowInstRepo *workflowrepo.InstanceRepository,
	logger zerolog.Logger,
) *MeetingService {
	return &MeetingService{
		store:            store,
		publisher:        publisherOrNoop(publisher),
		metrics:          metrics,
		logger:           logger.With().Str("component", "acta_meeting_service").Logger(),
		kafkaTopic:       kafkaTopic,
		workflowDefRepo:  workflowDefRepo,
		workflowInstRepo: workflowInstRepo,
	}
}

func (s *MeetingService) ScheduleMeeting(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateMeetingRequest) (*model.Meeting, error) {
	req.Normalize()
	if req.Title == "" {
		return nil, validationError("title is required", map[string]string{"title": "required"})
	}
	if req.ScheduledAt.Before(time.Now().UTC()) {
		return nil, validationError("scheduled_at must be in the future", map[string]string{"scheduled_at": "must be in the future"})
	}
	if req.DurationMinutes < 15 || req.DurationMinutes > 480 {
		return nil, validationError("duration_minutes must be between 15 and 480", map[string]string{"duration_minutes": "out of range"})
	}

	committee, err := s.store.GetCommittee(ctx, tenantID, req.CommitteeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	if committee.Status != model.CommitteeStatusActive {
		return nil, validationError("committee must be active", nil)
	}
	isMember, err := s.store.UserIsCommitteeMember(ctx, tenantID, req.CommitteeID, userID)
	if err != nil {
		return nil, internalError("failed to validate committee membership", err)
	}
	if !isMember {
		return nil, forbiddenError("only committee members can schedule meetings")
	}

	scheduledEnd := req.ScheduledAt.Add(time.Duration(req.DurationMinutes) * time.Minute)
	if req.ScheduledEndAt != nil {
		scheduledEnd = req.ScheduledEndAt.UTC()
	}
	conflictStart := req.ScheduledAt.Add(-2 * time.Hour)
	conflictEnd := scheduledEnd.Add(2 * time.Hour)
	memberCount, err := s.store.CountQuorumEligibleMembers(ctx, s.store.DB(), tenantID, committee.ID)
	if err != nil {
		return nil, internalError("failed to count committee members", err)
	}
	required, err := computeQuorumRequired(memberCount, string(committee.QuorumType), committee.QuorumPercentage, committee.QuorumFixedCount)
	if err != nil {
		return nil, validationError(err.Error(), nil)
	}

	now := time.Now().UTC()
	meeting := &model.Meeting{
		ID:              uuid.New(),
		TenantID:        tenantID,
		CommitteeID:     committee.ID,
		CommitteeName:   committee.Name,
		Title:           req.Title,
		Description:     req.Description,
		ScheduledAt:     req.ScheduledAt.UTC(),
		ScheduledEndAt:  &scheduledEnd,
		DurationMinutes: req.DurationMinutes,
		Location:        req.Location,
		LocationType:    model.LocationType(req.LocationType),
		VirtualLink:     req.VirtualLink,
		VirtualPlatform: req.VirtualPlatform,
		Status:          model.MeetingStatusScheduled,
		QuorumRequired:  required,
		Tags:            req.Tags,
		Metadata: map[string]any{
			"reminder_24h_sent": false,
			"reminder_1h_sent":  false,
			"attachments":       []model.MeetingAttachment{},
		},
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if meeting.LocationType == "" {
		meeting.LocationType = model.LocationTypePhysical
	}

	var (
		members     []model.CommitteeMember
		attendeeIDs []uuid.UUID
	)
	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		conflicts, err := s.store.CountMeetingConflicts(ctx, tx, tenantID, committee.ID, conflictStart, conflictEnd, nil)
		if err != nil {
			return err
		}
		if conflicts > 0 {
			return conflictError("meeting conflicts with another committee meeting")
		}
		number, err := s.store.NextMeetingNumber(ctx, tx, tenantID, committee.ID)
		if err != nil {
			return err
		}
		meeting.MeetingNumber = &number
		if err := s.store.CreateMeeting(ctx, tx, meeting); err != nil {
			return err
		}
		members, err = s.store.ListCommitteeMembers(ctx, tenantID, committee.ID, true)
		if err != nil {
			return err
		}
		attendance := make([]model.Attendee, 0, len(members))
		for _, member := range members {
			status := model.AttendanceStatusInvited
			var confirmedAt *time.Time
			if member.Role == model.CommitteeMemberRoleChair {
				status = model.AttendanceStatusConfirmed
				confirmedAt = &now
			}
			attendance = append(attendance, model.Attendee{
				ID:          uuid.New(),
				TenantID:    tenantID,
				MeetingID:   meeting.ID,
				UserID:      member.UserID,
				UserName:    member.UserName,
				UserEmail:   member.UserEmail,
				MemberRole:  member.Role,
				Status:      status,
				ConfirmedAt: confirmedAt,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
		}
		if err := s.store.CreateAttendanceRecords(ctx, tx, attendance); err != nil {
			return err
		}
		attendeeIDs = make([]uuid.UUID, 0, len(attendance))
		for _, attendee := range attendance {
			attendeeIDs = append(attendeeIDs, attendee.UserID)
		}
		meeting.AttendeeCount = len(attendance)
		return s.store.UpdateMeetingAttendanceStats(ctx, tx, tenantID, meeting.ID, len(attendance), 0, nil)
	}); err != nil {
		if appErr := unwrapAppError(err); appErr != nil {
			return nil, appErr
		}
		return nil, internalError("failed to schedule meeting", err)
	}

	if definition, instance := s.ensureWorkflow(ctx, tenantID, userID, meeting, committee); instance != nil {
		meeting.WorkflowInstanceID = ptr(uuid.MustParse(instance.ID))
		if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
			s.logger.Warn().Err(err).Str("meeting_id", meeting.ID.String()).Msg("failed to persist workflow instance on meeting")
		}
		if definition != nil {
			s.logger.Info().
				Str("meeting_id", meeting.ID.String()).
				Str("workflow_definition_id", definition.ID).
				Str("workflow_instance_id", instance.ID).
				Msg("governance workflow attached to meeting")
		}
	}

	if s.metrics != nil {
		s.metrics.MeetingsTotal.WithLabelValues(string(meeting.Status)).Inc()
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.meeting.scheduled", tenantID, &userID, map[string]any{
		"id":             meeting.ID,
		"title":          meeting.Title,
		"committee_id":   meeting.CommitteeID,
		"scheduled_at":   meeting.ScheduledAt,
		"attendee_count": meeting.AttendeeCount,
		"attendee_ids":   attendeeIDs,
		"location":       meeting.Location,
	}, s.logger)
	return s.GetMeeting(ctx, tenantID, meeting.ID)
}

func (s *MeetingService) GetMeeting(ctx context.Context, tenantID, meetingID uuid.UUID) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	agenda, err := s.store.ListAgendaItems(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to load agenda items", err)
	}
	attendance, err := s.store.ListAttendance(ctx, tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to load meeting attendance", err)
	}
	minutes, err := s.store.GetLatestMinutes(ctx, tenantID, meetingID)
	if err == nil {
		meeting.LatestMinutes = minutes
	} else if err != nil && err != pgx.ErrNoRows {
		return nil, internalError("failed to load latest minutes", err)
	}
	meeting.Agenda = agenda
	meeting.Attendance = attendance
	return meeting, nil
}

func (s *MeetingService) ListMeetings(ctx context.Context, tenantID uuid.UUID, filters model.MeetingFilters) ([]model.Meeting, int, error) {
	return s.store.ListMeetings(ctx, tenantID, filters)
}

func (s *MeetingService) UpdateMeeting(ctx context.Context, tenantID, userID, meetingID uuid.UUID, req dto.UpdateMeetingRequest) (*model.Meeting, error) {
	req.Normalize()
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusDraft && meeting.Status != model.MeetingStatusScheduled && meeting.Status != model.MeetingStatusPostponed {
		return nil, validationError("only draft, scheduled, or postponed meetings can be modified", nil)
	}
	if req.Title == "" {
		return nil, validationError("title is required", map[string]string{"title": "required"})
	}
	if req.ScheduledAt.Before(time.Now().UTC()) {
		return nil, validationError("scheduled_at must be in the future", map[string]string{"scheduled_at": "must be in the future"})
	}
	if req.DurationMinutes < 15 || req.DurationMinutes > 480 {
		return nil, validationError("duration_minutes must be between 15 and 480", map[string]string{"duration_minutes": "out of range"})
	}
	scheduledEnd := req.ScheduledAt.Add(time.Duration(req.DurationMinutes) * time.Minute)
	if req.ScheduledEndAt != nil {
		scheduledEnd = req.ScheduledEndAt.UTC()
	}
	conflictStart := req.ScheduledAt.Add(-2 * time.Hour)
	conflictEnd := scheduledEnd.Add(2 * time.Hour)
	conflicts, err := s.store.CountMeetingConflicts(ctx, s.store.DB(), tenantID, meeting.CommitteeID, conflictStart, conflictEnd, &meeting.ID)
	if err != nil {
		return nil, internalError("failed to validate meeting conflicts", err)
	}
	if conflicts > 0 {
		return nil, conflictError("meeting conflicts with another committee meeting")
	}
	meeting.Title = req.Title
	meeting.Description = req.Description
	meeting.ScheduledAt = req.ScheduledAt.UTC()
	meeting.ScheduledEndAt = &scheduledEnd
	meeting.DurationMinutes = req.DurationMinutes
	meeting.Location = req.Location
	meeting.LocationType = model.LocationType(req.LocationType)
	meeting.VirtualLink = req.VirtualLink
	meeting.VirtualPlatform = req.VirtualPlatform
	meeting.Tags = req.Tags
	if req.Metadata != nil {
		merged := meeting.Metadata
		if merged == nil {
			merged = map[string]any{}
		}
		for key, value := range req.Metadata {
			merged[key] = value
		}
		meeting.Metadata = merged
	}
	meeting.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
		return nil, internalError("failed to update meeting", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.meeting.updated", tenantID, &userID, map[string]any{
		"id": meeting.ID,
	}, s.logger)
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) CancelMeeting(ctx context.Context, tenantID, userID, meetingID uuid.UUID, reason string) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if normalizeString(reason) == "" {
		return nil, validationError("cancellation reason is required", map[string]string{"reason": "required"})
	}
	meeting.Status = model.MeetingStatusCancelled
	meeting.CancellationReason = ptr(reason)
	meeting.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
		return nil, internalError("failed to cancel meeting", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.meeting.cancelled", tenantID, &userID, map[string]any{
		"id":     meeting.ID,
		"reason": reason,
	}, s.logger)
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) PostponeMeeting(ctx context.Context, tenantID, userID, meetingID uuid.UUID, req dto.PostponeMeetingRequest) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status == model.MeetingStatusCompleted || meeting.Status == model.MeetingStatusCancelled {
		return nil, validationError("completed or cancelled meetings cannot be postponed", nil)
	}
	if req.NewScheduledAt.Before(time.Now().UTC()) {
		return nil, validationError("new_scheduled_at must be in the future", nil)
	}
	scheduledEnd := req.NewScheduledAt.Add(time.Duration(meeting.DurationMinutes) * time.Minute)
	if req.NewScheduledEndAt != nil {
		scheduledEnd = req.NewScheduledEndAt.UTC()
	}
	conflictStart := req.NewScheduledAt.Add(-2 * time.Hour)
	conflictEnd := scheduledEnd.Add(2 * time.Hour)
	conflicts, err := s.store.CountMeetingConflicts(ctx, s.store.DB(), tenantID, meeting.CommitteeID, conflictStart, conflictEnd, &meeting.ID)
	if err != nil {
		return nil, internalError("failed to validate meeting conflicts", err)
	}
	if conflicts > 0 {
		return nil, conflictError("meeting conflicts with another committee meeting")
	}
	if meeting.Metadata == nil {
		meeting.Metadata = map[string]any{}
	}
	meeting.Metadata["postponed_from"] = meeting.ScheduledAt
	meeting.Metadata["postponement_reason"] = req.Reason
	meeting.ScheduledAt = req.NewScheduledAt.UTC()
	meeting.ScheduledEndAt = &scheduledEnd
	meeting.Status = model.MeetingStatusPostponed
	meeting.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
		return nil, internalError("failed to postpone meeting", err)
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.meeting.postponed", tenantID, &userID, map[string]any{
		"id":       meeting.ID,
		"new_date": meeting.ScheduledAt,
	}, s.logger)
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) StartMeeting(ctx context.Context, tenantID, userID, meetingID uuid.UUID) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusScheduled {
		return nil, validationError("meeting must be scheduled before it can be started", nil)
	}
	committee, err := s.store.GetCommittee(ctx, tenantID, meeting.CommitteeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	if userID != committee.ChairUserID && (committee.SecretaryUserID == nil || userID != *committee.SecretaryUserID) {
		return nil, forbiddenError("only the committee chair or secretary can start meetings")
	}
	now := time.Now().UTC()
	presentCount, err := s.store.CountPresentAttendance(ctx, s.store.DB(), tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to count present attendees", err)
	}
	attendeeCount, err := s.store.CountAttendees(ctx, s.store.DB(), tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to count attendees", err)
	}
	meeting.Status = model.MeetingStatusInProgress
	meeting.ActualStartAt = &now
	meeting.PresentCount = presentCount
	meeting.AttendeeCount = attendeeCount
	met := quorumMet(meeting.QuorumRequired, presentCount)
	meeting.QuorumMet = &met
	meeting.UpdatedAt = now
	if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
		return nil, internalError("failed to start meeting", err)
	}
	if s.metrics != nil {
		s.metrics.MeetingsTotal.WithLabelValues(string(meeting.Status)).Inc()
		s.metrics.MeetingsActive.Inc()
	}
	if !met {
		s.logger.Warn().
			Str("meeting_id", meeting.ID.String()).
			Int("present", presentCount).
			Int("required", meeting.QuorumRequired).
			Msg("meeting started without quorum")
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.meeting.started", tenantID, &userID, map[string]any{
		"id":            meeting.ID,
		"present_count": presentCount,
		"quorum_met":    met,
	}, s.logger)
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) EndMeeting(ctx context.Context, tenantID, userID, meetingID uuid.UUID) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	if meeting.Status != model.MeetingStatusInProgress {
		return nil, validationError("meeting must be in progress before it can be ended", nil)
	}
	committee, err := s.store.GetCommittee(ctx, tenantID, meeting.CommitteeID)
	if err != nil {
		return nil, notFoundError("committee not found")
	}
	if userID != committee.ChairUserID && (committee.SecretaryUserID == nil || userID != *committee.SecretaryUserID) {
		return nil, forbiddenError("only the committee chair or secretary can end meetings")
	}
	now := time.Now().UTC()
	presentCount, err := s.store.CountPresentAttendance(ctx, s.store.DB(), tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to count present attendees", err)
	}
	attendeeCount, err := s.store.CountAttendees(ctx, s.store.DB(), tenantID, meetingID)
	if err != nil {
		return nil, internalError("failed to count attendees", err)
	}
	met := quorumMet(meeting.QuorumRequired, presentCount)
	meeting.Status = model.MeetingStatusCompleted
	meeting.ActualEndAt = &now
	meeting.PresentCount = presentCount
	meeting.AttendeeCount = attendeeCount
	meeting.QuorumMet = &met
	if meeting.ActualStartAt != nil {
		duration := int(now.Sub(*meeting.ActualStartAt).Minutes())
		if duration < 15 {
			duration = 15
		}
		if duration > 480 {
			duration = 480
		}
		meeting.DurationMinutes = duration
	}
	meeting.UpdatedAt = now
	if err := database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		if err := s.store.MarkUncheckedAttendeesAbsent(ctx, tx, tenantID, meetingID, now); err != nil {
			return err
		}
		return s.store.UpdateMeeting(ctx, tx, meeting)
	}); err != nil {
		return nil, internalError("failed to end meeting", err)
	}
	if s.metrics != nil {
		s.metrics.MeetingsTotal.WithLabelValues(string(meeting.Status)).Inc()
		s.metrics.MeetingsActive.Dec()
		s.metrics.MeetingDurationMinutes.Observe(float64(meeting.DurationMinutes))
		s.metrics.MeetingQuorumMetTotal.WithLabelValues(fmt.Sprintf("%t", met)).Inc()
	}
	if !met {
		s.logger.Warn().Str("meeting_id", meeting.ID.String()).Msg("meeting ended without quorum")
		check := model.ComplianceCheck{
			ID:             uuid.New(),
			TenantID:       tenantID,
			CommitteeID:    &meeting.CommitteeID,
			CheckType:      model.ComplianceCheckQuorumCompliance,
			CheckName:      "Meeting quorum compliance",
			Status:         model.ComplianceStatusNonCompliant,
			Severity:       model.ComplianceSeverityHigh,
			Description:    "Meeting ended with insufficient attendance for quorum.",
			Finding:        ptr("Meeting ended without quorum. Decisions may not be legally binding."),
			Recommendation: ptr("Ratify decisions in a duly constituted meeting."),
			Evidence: map[string]any{
				"meeting_id":      meeting.ID,
				"present_count":   presentCount,
				"quorum_required": meeting.QuorumRequired,
				"meeting_status":  meeting.Status,
			},
			PeriodStart: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			PeriodEnd:   time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			CheckedAt:   now,
			CheckedBy:   "system",
			CreatedAt:   now,
		}
		if err := s.store.InsertComplianceChecks(ctx, s.store.DB(), []model.ComplianceCheck{check}); err != nil {
			s.logger.Error().Err(err).Str("meeting_id", meeting.ID.String()).Msg("failed to persist quorum compliance finding")
		}
	}
	publishEvent(ctx, s.publisher, "acta-service", events.Topics.ActaEvents, "acta.meeting.completed", tenantID, &userID, map[string]any{
		"id":                meeting.ID,
		"duration_minutes":  meeting.DurationMinutes,
		"quorum_met":        met,
		"action_item_count": meeting.ActionItemCount,
	}, s.logger)
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) GetAttendance(ctx context.Context, tenantID, meetingID uuid.UUID) ([]model.Attendee, error) {
	return s.store.ListAttendance(ctx, tenantID, meetingID)
}

func (s *MeetingService) RecordAttendance(ctx context.Context, tenantID, meetingID uuid.UUID, req dto.AttendanceRequest) ([]model.Attendee, error) {
	if err := s.updateAttendanceRecords(ctx, tenantID, meetingID, []dto.AttendanceRequest{req}); err != nil {
		return nil, err
	}
	return s.store.ListAttendance(ctx, tenantID, meetingID)
}

func (s *MeetingService) BulkRecordAttendance(ctx context.Context, tenantID, meetingID uuid.UUID, req dto.BulkAttendanceRequest) ([]model.Attendee, error) {
	if err := s.updateAttendanceRecords(ctx, tenantID, meetingID, req.Attendance); err != nil {
		return nil, err
	}
	return s.store.ListAttendance(ctx, tenantID, meetingID)
}

func (s *MeetingService) UpcomingMeetings(ctx context.Context, tenantID uuid.UUID, limit int) ([]model.MeetingSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.ListUpcomingMeetings(ctx, tenantID, limit)
}

func (s *MeetingService) Calendar(ctx context.Context, tenantID uuid.UUID, month time.Time) ([]model.CalendarDay, error) {
	meetings, err := s.store.ListCalendarMeetings(ctx, tenantID, month)
	if err != nil {
		return nil, internalError("failed to load meeting calendar", err)
	}
	byDate := make(map[string][]model.MeetingSummary)
	for _, meeting := range meetings {
		key := meeting.ScheduledAt.UTC().Format("2006-01-02")
		byDate[key] = append(byDate[key], meeting)
	}
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)
	out := make([]model.CalendarDay, 0)
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		out = append(out, model.CalendarDay{
			Date:     day,
			Meetings: byDate[key],
		})
	}
	return out, nil
}

func (s *MeetingService) AddAttachment(ctx context.Context, tenantID, meetingID uuid.UUID, req dto.AttachmentRequest) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	attachments := meeting.Attachments
	if meeting.Metadata == nil {
		meeting.Metadata = map[string]any{}
	}
	for _, attachment := range attachments {
		if attachment.FileID == req.FileID {
			return nil, conflictError("attachment already linked to meeting")
		}
	}
	attachments = append(attachments, model.MeetingAttachment{
		FileID:      req.FileID,
		Name:        req.Name,
		ContentType: req.ContentType,
		UploadedBy:  req.UploadedBy,
		UploadedAt:  time.Now().UTC(),
	})
	meeting.Metadata["attachments"] = attachments
	meeting.Attachments = attachments
	meeting.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
		return nil, internalError("failed to attach file to meeting", err)
	}
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) RemoveAttachment(ctx context.Context, tenantID, meetingID, fileID uuid.UUID) (*model.Meeting, error) {
	meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
	if err != nil {
		return nil, notFoundError("meeting not found")
	}
	attachments := make([]model.MeetingAttachment, 0, len(meeting.Attachments))
	found := false
	for _, attachment := range meeting.Attachments {
		if attachment.FileID == fileID {
			found = true
			continue
		}
		attachments = append(attachments, attachment)
	}
	if !found {
		return nil, notFoundError("attachment not found")
	}
	if meeting.Metadata == nil {
		meeting.Metadata = map[string]any{}
	}
	meeting.Metadata["attachments"] = attachments
	meeting.Attachments = attachments
	meeting.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateMeeting(ctx, s.store.DB(), meeting); err != nil {
		return nil, internalError("failed to remove meeting attachment", err)
	}
	return s.GetMeeting(ctx, tenantID, meetingID)
}

func (s *MeetingService) updateAttendanceRecords(ctx context.Context, tenantID, meetingID uuid.UUID, requests []dto.AttendanceRequest) error {
	now := time.Now().UTC()
	return database.RunInTx(ctx, s.store.DB(), func(tx pgx.Tx) error {
		for _, request := range requests {
			record, err := s.store.GetAttendanceRecord(ctx, tenantID, meetingID, request.UserID)
			if err != nil {
				return err
			}
			record.Status = model.AttendanceStatus(request.Status)
			record.Notes = request.Notes
			record.ProxyUserID = request.ProxyUserID
			record.ProxyUserName = request.ProxyUserName
			record.ProxyAuthorizedBy = request.ProxyAuthorizedBy
			switch record.Status {
			case model.AttendanceStatusConfirmed:
				record.ConfirmedAt = &now
			case model.AttendanceStatusPresent, model.AttendanceStatusProxy:
				record.CheckedInAt = &now
				record.CheckedOutAt = nil
			case model.AttendanceStatusAbsent, model.AttendanceStatusExcused, model.AttendanceStatusDeclined:
				record.CheckedOutAt = &now
			}
			record.UpdatedAt = now
			if err := s.store.UpdateAttendanceRecord(ctx, tx, tenantID, meetingID, record); err != nil {
				return err
			}
		}
		presentCount, err := s.store.CountPresentAttendance(ctx, tx, tenantID, meetingID)
		if err != nil {
			return err
		}
		attendeeCount, err := s.store.CountAttendees(ctx, tx, tenantID, meetingID)
		if err != nil {
			return err
		}
		meeting, err := s.store.GetMeeting(ctx, tenantID, meetingID)
		if err != nil {
			return err
		}
		met := quorumMet(meeting.QuorumRequired, presentCount)
		return s.store.UpdateMeetingAttendanceStats(ctx, tx, tenantID, meetingID, attendeeCount, presentCount, &met)
	})
}

func (s *MeetingService) ensureWorkflow(ctx context.Context, tenantID, userID uuid.UUID, meeting *model.Meeting, committee *model.Committee) (*workflowmodel.WorkflowDefinition, *workflowmodel.WorkflowInstance) {
	if s.workflowDefRepo == nil || s.workflowInstRepo == nil {
		return nil, nil
	}
	templateSvc := workflowservice.NewTemplateService(s.workflowDefRepo, s.logger)
	definition, err := s.ensureBoardMeetingDefinition(ctx, templateSvc, tenantID, userID)
	if err != nil {
		s.logger.Warn().Err(err).Str("meeting_id", meeting.ID.String()).Msg("unable to prepare governance workflow definition")
		return nil, nil
	}
	secretaryID := ""
	if committee.SecretaryUserID != nil {
		secretaryID = committee.SecretaryUserID.String()
	}
	currentStep := "agenda_preparation"
	instance := &workflowmodel.WorkflowInstance{
		TenantID:      tenantID.String(),
		DefinitionID:  definition.ID,
		DefinitionVer: definition.Version,
		Status:        workflowmodel.InstanceStatusRunning,
		CurrentStepID: &currentStep,
		Variables: map[string]any{
			"meeting_id":        meeting.ID.String(),
			"committee_id":      committee.ID.String(),
			"scheduled_at":      meeting.ScheduledAt.Format(time.RFC3339),
			"chair_user_id":     committee.ChairUserID.String(),
			"secretary_user_id": secretaryID,
			"workflow_template": "board-meeting",
		},
		StepOutputs: map[string]any{},
		StartedBy:   ptr(userID.String()),
	}
	if err := s.workflowInstRepo.Create(ctx, instance); err != nil {
		s.logger.Warn().Err(err).Str("meeting_id", meeting.ID.String()).Msg("failed to create governance workflow instance")
		return definition, nil
	}
	return definition, instance
}

func (s *MeetingService) ensureBoardMeetingDefinition(ctx context.Context, templateSvc *workflowservice.TemplateService, tenantID, userID uuid.UUID) (*workflowmodel.WorkflowDefinition, error) {
	definitions, _, err := s.workflowDefRepo.List(ctx, tenantID.String(), "active", "Board Meeting", "", "", "", 10, 0)
	if err != nil {
		return nil, err
	}
	for _, definition := range definitions {
		if definition.Name == "Board Meeting" {
			return definition, nil
		}
	}
	definition, err := templateSvc.InstantiateTemplate(ctx, tenantID.String(), userID.String(), "tmpl-board-meeting", "", "")
	if err != nil {
		return nil, err
	}
	definition.Status = workflowmodel.DefinitionStatusActive
	definition.UpdatedBy = userID.String()
	if err := s.workflowDefRepo.Update(ctx, definition); err != nil {
		return nil, err
	}
	return definition, nil
}
