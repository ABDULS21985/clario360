package scheduler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	actasvc "github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/events"
)

type MeetingReminder struct {
	store     *repository.Store
	publisher actasvc.Publisher
	interval  time.Duration
	logger    zerolog.Logger
}

func NewMeetingReminder(store *repository.Store, publisher actasvc.Publisher, interval time.Duration, logger zerolog.Logger) *MeetingReminder {
	return &MeetingReminder{
		store:     store,
		publisher: publisher,
		interval:  interval,
		logger:    logger.With().Str("component", "acta_meeting_reminder").Logger(),
	}
}

func (r *MeetingReminder) Run(ctx context.Context) error {
	if err := r.runOnce(ctx); err != nil {
		r.logger.Error().Err(err).Msg("initial meeting reminder run failed")
	}
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.runOnce(ctx); err != nil {
				r.logger.Error().Err(err).Msg("scheduled meeting reminder run failed")
			}
		}
	}
}

func (r *MeetingReminder) runOnce(ctx context.Context) error {
	tenantIDs, err := r.store.ListTenantIDs(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, tenantID := range tenantIDs {
		items, _, err := r.store.ListMeetings(ctx, tenantID, model.MeetingFilters{
			Statuses: []model.MeetingStatus{
				model.MeetingStatusDraft,
				model.MeetingStatusScheduled,
				model.MeetingStatusPostponed,
			},
			DateFrom: &now,
			DateTo:   ptrTime(now.Add(24 * time.Hour)),
			Page:     1,
			PerPage:  500,
		})
		if err != nil {
			return err
		}
		for _, meeting := range items {
			hoursUntil, metadataKey, shouldSend := reminderWindow(meeting, now)
			if !shouldSend {
				continue
			}
			attendance, err := r.store.ListAttendance(ctx, tenantID, meeting.ID)
			if err != nil {
				return err
			}
			attendees := make([]uuid.UUID, 0, len(attendance))
			for _, attendee := range attendance {
				attendees = append(attendees, attendee.UserID)
			}
			payload := map[string]any{
				"meeting_id":   meeting.ID,
				"title":        meeting.Title,
				"hours_until":  hoursUntil,
				"scheduled_at": meeting.ScheduledAt,
				"attendee_ids": attendees,
				"committee_id": meeting.CommitteeID,
				"location":     meeting.Location,
			}
			if err := r.publishReminder(ctx, tenantID, payload); err != nil {
				r.logger.Error().Err(err).Str("meeting_id", meeting.ID.String()).Msg("failed to publish meeting reminder")
			}
			if meeting.Metadata == nil {
				meeting.Metadata = map[string]any{}
			}
			meeting.Metadata[metadataKey] = true
			meeting.UpdatedAt = now
			if err := r.store.UpdateMeeting(ctx, r.store.DB(), &meeting); err != nil {
				return err
			}
		}
	}
	return nil
}

func reminderWindow(meeting model.Meeting, now time.Time) (int, string, bool) {
	until := meeting.ScheduledAt.Sub(now)
	if until <= 0 {
		return 0, "", false
	}
	if until <= time.Hour && !metadataBool(meeting.Metadata, "reminder_1h_sent") {
		return 1, "reminder_1h_sent", true
	}
	if until <= 24*time.Hour && !metadataBool(meeting.Metadata, "reminder_24h_sent") {
		return 24, "reminder_24h_sent", true
	}
	return 0, "", false
}

func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	value, ok := metadata[key]
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func (r *MeetingReminder) publishReminder(ctx context.Context, tenantID uuid.UUID, payload any) error {
	if r.publisher == nil {
		return nil
	}
	event, err := events.NewEvent("acta.meeting.reminder", "acta-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return r.publisher.Publish(ctx, events.Topics.ActaEvents, event)
}
