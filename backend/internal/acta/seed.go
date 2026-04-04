package acta

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/database"
)

var demoTenantID = mustUUID("11111111-1111-1111-1111-111111111111")

type seedUser struct {
	ID    uuid.UUID
	Name  string
	Email string
}

type seedCommittee struct {
	ID               uuid.UUID
	Name             string
	Type             model.CommitteeType
	Description      string
	ChairUserID      uuid.UUID
	ViceChairUserID  *uuid.UUID
	SecretaryUserID  *uuid.UUID
	MeetingFrequency model.MeetingFrequency
	QuorumPercentage int
	QuorumType       model.QuorumType
	QuorumFixedCount *int
	Charter          string
	Tags             []string
	Members          []uuid.UUID
}

type seedAttendance struct {
	Status        model.AttendanceStatus
	ProxyUserID   *uuid.UUID
	ProxyUserName *string
}

type seedMeeting struct {
	ID             uuid.UUID
	CommitteeID    uuid.UUID
	Title          string
	Description    string
	MeetingNumber  int
	ScheduledAt    time.Time
	Duration       int
	Location       string
	Status         model.MeetingStatus
	QuorumRequired int
	Attendance     map[uuid.UUID]seedAttendance
}

type seedAgenda struct {
	ID             uuid.UUID
	MeetingID      uuid.UUID
	Title          string
	Description    string
	Presenter      uuid.UUID
	PresenterName  string
	OrderIndex     int
	Status         model.AgendaItemStatus
	RequiresVote   bool
	VoteType       *model.VoteType
	VotesFor       *int
	VotesAgainst   *int
	VotesAbstained *int
	VoteResult     *model.VoteResult
	Category       model.AgendaCategory
	Notes          string
}

type seedMinutes struct {
	ID          uuid.UUID
	MeetingID   uuid.UUID
	Status      model.MinutesStatus
	Version     int
	AIGenerated bool
}

type seedActionItem struct {
	ID           uuid.UUID
	MeetingID    uuid.UUID
	AgendaItemID *uuid.UUID
	CommitteeID  uuid.UUID
	Title        string
	Description  string
	Priority     model.ActionItemPriority
	AssignedTo   uuid.UUID
	AssigneeName string
	DueDate      time.Time
	Status       model.ActionItemStatus
}

func SeedDemoData(ctx context.Context, store *repository.Store, logger zerolog.Logger) (uuid.UUID, error) {
	if store == nil || store.DB() == nil {
		return uuid.Nil, fmt.Errorf("store is not initialized")
	}
	existing, total, err := store.ListCommittees(ctx, demoTenantID, "", 1, 1)
	if err == nil && total > 0 && len(existing) > 0 {
		return demoTenantID, nil
	}

	users := seedUsers()
	userByID := make(map[uuid.UUID]seedUser, len(users))
	for _, user := range users {
		userByID[user.ID] = user
	}

	committees := seedCommittees()
	meetings := seedMeetings()
	agendaItems := seedAgendaItems(meetings)
	minutes := seedMinutesRecords(meetings)
	actionItems := seedActionItems(agendaItems, users)

	if err := database.RunInTx(ctx, store.DB(), func(tx pgx.Tx) error {
		if err := seedCommitteesTx(ctx, tx, store, committees, userByID); err != nil {
			return err
		}
		if err := seedMeetingsTx(ctx, tx, store, meetings, committees, userByID); err != nil {
			return err
		}
		if err := seedAgendaTx(ctx, tx, store, agendaItems); err != nil {
			return err
		}
		if err := seedMinutesTx(ctx, tx, store, meetings, minutes); err != nil {
			return err
		}
		if err := seedActionItemsTx(ctx, tx, store, meetings, actionItems, users[0].ID); err != nil {
			return err
		}
		for _, meeting := range meetings {
			if err := store.UpdateMeetingAgendaCount(ctx, tx, demoTenantID, meeting.ID); err != nil {
				return err
			}
			if err := store.UpdateMeetingActionItemCount(ctx, tx, demoTenantID, meeting.ID); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return uuid.Nil, err
	}

	logger.Info().
		Str("tenant_id", demoTenantID.String()).
		Int("committees", len(committees)).
		Int("meetings", len(meetings)).
		Int("agenda_items", len(agendaItems)).
		Int("minutes", len(minutes)).
		Int("action_items", len(actionItems)).
		Msg("seeded acta demo dataset")

	return demoTenantID, nil
}

func seedCommitteesTx(ctx context.Context, tx pgx.Tx, store *repository.Store, committees []seedCommittee, users map[uuid.UUID]seedUser) error {
	for idx, committee := range committees {
		createdAt := seedTime(2025, time.January, 10+idx, 9, 0)
		item := &model.Committee{
			ID:               committee.ID,
			TenantID:         demoTenantID,
			Name:             committee.Name,
			Type:             committee.Type,
			Description:      committee.Description,
			ChairUserID:      committee.ChairUserID,
			ViceChairUserID:  committee.ViceChairUserID,
			SecretaryUserID:  committee.SecretaryUserID,
			MeetingFrequency: committee.MeetingFrequency,
			QuorumPercentage: committee.QuorumPercentage,
			QuorumType:       committee.QuorumType,
			QuorumFixedCount: committee.QuorumFixedCount,
			Charter:          seedPtr(committee.Charter),
			Status:           model.CommitteeStatusActive,
			Tags:             committee.Tags,
			Metadata: map[string]any{
				"charter_reviewed_at": createdAt.AddDate(0, 10, 0).Format(time.RFC3339),
			},
			CreatedBy: committee.ChairUserID,
			CreatedAt: createdAt,
			UpdatedAt: createdAt.AddDate(0, 10, 0),
		}
		if err := store.CreateCommittee(ctx, tx, item); err != nil {
			return err
		}
		for _, userID := range committee.Members {
			user := users[userID]
			role := model.CommitteeMemberRoleMember
			switch userID {
			case committee.ChairUserID:
				role = model.CommitteeMemberRoleChair
			default:
				if committee.ViceChairUserID != nil && userID == *committee.ViceChairUserID {
					role = model.CommitteeMemberRoleViceChair
				}
				if committee.SecretaryUserID != nil && userID == *committee.SecretaryUserID {
					role = model.CommitteeMemberRoleSecretary
				}
			}
			member := &model.CommitteeMember{
				ID:          mustUUID(fmt.Sprintf("11111111-1111-1111-1111-%012x", int64(len(item.Name))+int64(len(user.Name))+int64(createdAt.Day()))),
				TenantID:    demoTenantID,
				CommitteeID: committee.ID,
				UserID:      user.ID,
				UserName:    user.Name,
				UserEmail:   user.Email,
				Role:        role,
				JoinedAt:    createdAt,
				Active:      true,
				CreatedAt:   createdAt,
				UpdatedAt:   createdAt,
			}
			member.ID = uuid.NewSHA1(committee.ID, []byte(user.ID.String()))
			if err := store.UpsertCommitteeMember(ctx, tx, member); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedMeetingsTx(ctx context.Context, tx pgx.Tx, store *repository.Store, meetings []seedMeeting, committees []seedCommittee, users map[uuid.UUID]seedUser) error {
	committeeByID := make(map[uuid.UUID]seedCommittee, len(committees))
	for _, committee := range committees {
		committeeByID[committee.ID] = committee
	}
	for _, meeting := range meetings {
		committee := committeeByID[meeting.CommitteeID]
		scheduledEnd := meeting.ScheduledAt.Add(time.Duration(meeting.Duration) * time.Minute)
		var actualStart *time.Time
		var actualEnd *time.Time
		var presentCount int
		var quorumMet *bool
		if meeting.Status == model.MeetingStatusCompleted {
			start := meeting.ScheduledAt.Add(5 * time.Minute)
			end := start.Add(time.Duration(meeting.Duration) * time.Minute)
			actualStart = &start
			actualEnd = &end
			presentCount = countSeedPresent(committee.Members, meeting.Attendance)
			met := presentCount >= meeting.QuorumRequired
			quorumMet = &met
		}
		item := &model.Meeting{
			ID:              meeting.ID,
			TenantID:        demoTenantID,
			CommitteeID:     meeting.CommitteeID,
			CommitteeName:   committee.Name,
			Title:           meeting.Title,
			Description:     meeting.Description,
			MeetingNumber:   seedPtr(meeting.MeetingNumber),
			ScheduledAt:     meeting.ScheduledAt,
			ScheduledEndAt:  &scheduledEnd,
			ActualStartAt:   actualStart,
			ActualEndAt:     actualEnd,
			DurationMinutes: meeting.Duration,
			Location:        seedPtr(meeting.Location),
			LocationType:    model.LocationTypePhysical,
			Status:          meeting.Status,
			QuorumRequired:  meeting.QuorumRequired,
			AttendeeCount:   len(committee.Members),
			PresentCount:    presentCount,
			QuorumMet:       quorumMet,
			HasMinutes:      false,
			Tags:            []string{"seed", string(committee.Type)},
			Metadata: map[string]any{
				"reminder_24h_sent": false,
				"reminder_1h_sent":  false,
				"attachments":       []model.MeetingAttachment{},
			},
			CreatedBy: committee.ChairUserID,
			CreatedAt: meeting.ScheduledAt.AddDate(0, 0, -14),
			UpdatedAt: meeting.ScheduledAt.AddDate(0, 0, -14),
		}
		if err := store.CreateMeeting(ctx, tx, item); err != nil {
			return err
		}
		attendance := make([]model.Attendee, 0, len(committee.Members))
		for _, memberID := range committee.Members {
			user := users[memberID]
			role := committeeRoleForUser(committee, memberID)
			seed := meeting.Attendance[memberID]
			status := seed.Status
			if status == "" {
				status = model.AttendanceStatusInvited
				if meeting.Status == model.MeetingStatusCompleted {
					status = model.AttendanceStatusPresent
				} else if memberID == committee.ChairUserID {
					status = model.AttendanceStatusConfirmed
				}
			}
			record := model.Attendee{
				ID:            uuid.NewSHA1(meeting.ID, []byte(memberID.String())),
				TenantID:      demoTenantID,
				MeetingID:     meeting.ID,
				UserID:        memberID,
				UserName:      user.Name,
				UserEmail:     user.Email,
				MemberRole:    role,
				Status:        status,
				ProxyUserID:   seed.ProxyUserID,
				ProxyUserName: seed.ProxyUserName,
				CreatedAt:     item.CreatedAt,
				UpdatedAt:     item.CreatedAt,
			}
			if status == model.AttendanceStatusProxy {
				record.ProxyAuthorizedBy = seedPtr(committee.ChairUserID)
			}
			switch status {
			case model.AttendanceStatusConfirmed:
				record.ConfirmedAt = &item.CreatedAt
			case model.AttendanceStatusPresent, model.AttendanceStatusProxy:
				checkIn := meeting.ScheduledAt.Add(2 * time.Minute)
				record.CheckedInAt = &checkIn
			}
			attendance = append(attendance, record)
		}
		if err := store.CreateAttendanceRecords(ctx, tx, attendance); err != nil {
			return err
		}
	}
	return nil
}

func seedAgendaTx(ctx context.Context, tx pgx.Tx, store *repository.Store, agendaItems []seedAgenda) error {
	for _, agenda := range agendaItems {
		voteType := agenda.VoteType
		category := agenda.Category
		item := &model.AgendaItem{
			ID:              agenda.ID,
			TenantID:        demoTenantID,
			MeetingID:       agenda.MeetingID,
			Title:           agenda.Title,
			Description:     agenda.Description,
			ItemNumber:      seedPtr(fmt.Sprintf("%d", agenda.OrderIndex+1)),
			PresenterUserID: &agenda.Presenter,
			PresenterName:   seedPtr(agenda.PresenterName),
			DurationMinutes: 20,
			OrderIndex:      agenda.OrderIndex,
			Status:          agenda.Status,
			Notes:           seedPtr(agenda.Notes),
			RequiresVote:    agenda.RequiresVote,
			VoteType:        voteType,
			VotesFor:        agenda.VotesFor,
			VotesAgainst:    agenda.VotesAgainst,
			VotesAbstained:  agenda.VotesAbstained,
			VoteResult:      agenda.VoteResult,
			Category:        &category,
			AttachmentIDs:   []uuid.UUID{},
			CreatedAt:       seedTime(2025, time.September, 1, 8, 0),
			UpdatedAt:       seedTime(2025, time.September, 1, 8, 0),
		}
		if err := store.CreateAgendaItem(ctx, tx, item); err != nil {
			return err
		}
	}
	return nil
}

func seedMinutesTx(ctx context.Context, tx pgx.Tx, store *repository.Store, meetings []seedMeeting, minutes []seedMinutes) error {
	meetingByID := make(map[uuid.UUID]seedMeeting, len(meetings))
	for _, meeting := range meetings {
		meetingByID[meeting.ID] = meeting
	}
	for idx, minutesRecord := range minutes {
		meeting := meetingByID[minutesRecord.MeetingID]
		committeeID := committeeIDForMeeting(meeting.ID)
		chairUserID := chairForCommittee(committeeID)
		secretaryUserID := secretaryForCommittee(committeeID)
		statusText := string(minutesRecord.Status)
		content := fmt.Sprintf("# Minutes of %s\n\nThe committee reviewed its standing agenda, resolved matters before it, and recorded follow-up actions.\n", meeting.Title)
		summary := fmt.Sprintf("%s met on %s to review governance matters and assign follow-up actions.", meeting.Title, meeting.ScheduledAt.Format("January 2, 2006"))
		item := &model.MeetingMinutes{
			ID:            minutesRecord.ID,
			TenantID:      demoTenantID,
			MeetingID:     minutesRecord.MeetingID,
			Content:       content,
			AISummary:     seedPtr(summary),
			Status:        minutesRecord.Status,
			Version:       minutesRecord.Version,
			AIGenerated:   minutesRecord.AIGenerated,
			AIActionItems: []model.ExtractedAction{},
			CreatedBy:     secretaryUserID,
			CreatedAt:     meeting.ScheduledAt.AddDate(0, 0, 2),
			UpdatedAt:     meeting.ScheduledAt.AddDate(0, 0, 2+idx),
		}
		if minutesRecord.Status == model.MinutesStatusReview || minutesRecord.Status == model.MinutesStatusApproved || minutesRecord.Status == model.MinutesStatusPublished {
			submittedAt := meeting.ScheduledAt.AddDate(0, 0, 2)
			item.SubmittedForReviewAt = &submittedAt
			item.SubmittedBy = seedPtr(secretaryUserID)
		}
		if minutesRecord.Status == model.MinutesStatusApproved || minutesRecord.Status == model.MinutesStatusPublished {
			approvedAt := meeting.ScheduledAt.AddDate(0, 0, 4)
			item.ApprovedAt = &approvedAt
			item.ApprovedBy = seedPtr(chairUserID)
		}
		if minutesRecord.Status == model.MinutesStatusPublished {
			publishedAt := meeting.ScheduledAt.AddDate(0, 0, 5)
			item.PublishedAt = &publishedAt
		}
		if err := store.CreateMinutes(ctx, tx, item); err != nil {
			return err
		}
		if err := store.UpdateMeetingMinutesState(ctx, tx, demoTenantID, minutesRecord.MeetingID, true, &statusText); err != nil {
			return err
		}
	}
	return nil
}

func seedActionItemsTx(ctx context.Context, tx pgx.Tx, store *repository.Store, meetings []seedMeeting, actionItems []seedActionItem, createdBy uuid.UUID) error {
	for idx, action := range actionItems {
		item := &model.ActionItem{
			ID:              action.ID,
			TenantID:        demoTenantID,
			MeetingID:       action.MeetingID,
			AgendaItemID:    action.AgendaItemID,
			CommitteeID:     action.CommitteeID,
			Title:           action.Title,
			Description:     action.Description,
			Priority:        action.Priority,
			AssignedTo:      action.AssignedTo,
			AssigneeName:    action.AssigneeName,
			AssignedBy:      createdBy,
			DueDate:         action.DueDate,
			OriginalDueDate: action.DueDate,
			Status:          action.Status,
			Tags:            []string{"seed"},
			Metadata:        map[string]any{},
			CreatedBy:       createdBy,
			CreatedAt:       seedTime(2025, time.December, 15+idx%10, 9, 0),
			UpdatedAt:       seedTime(2026, time.January, 10+idx%10, 9, 0),
		}
		if action.Status == model.ActionItemStatusCompleted {
			completedAt := action.DueDate.Add(-48 * time.Hour)
			item.CompletedAt = &completedAt
			item.CompletionNotes = seedPtr("Closed with committee-reviewed evidence.")
		}
		if err := store.CreateActionItem(ctx, tx, item); err != nil {
			return err
		}
	}
	return nil
}

func seedUsers() []seedUser {
	return demoUsersForSeed()
}

func demoUsersForSeed() []seedUser {
	return []seedUser{
		{ID: mustUUID("11111111-1111-1111-1111-111111111001"), Name: "Amina Okafor", Email: "amina.okafor@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111002"), Name: "Daniel Mensah", Email: "daniel.mensah@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111003"), Name: "Grace Nwosu", Email: "grace.nwosu@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111004"), Name: "Ibrahim Bello", Email: "ibrahim.bello@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111005"), Name: "Lara Adeyemi", Email: "lara.adeyemi@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111006"), Name: "Musa Sule", Email: "musa.sule@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111007"), Name: "Nadia Yusuf", Email: "nadia.yusuf@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111008"), Name: "Oliver Dike", Email: "oliver.dike@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111009"), Name: "Priya Raman", Email: "priya.raman@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111010"), Name: "Quentin Udeh", Email: "quentin.udeh@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111011"), Name: "Ruth Ekanem", Email: "ruth.ekanem@clario.demo"},
		{ID: mustUUID("11111111-1111-1111-1111-111111111012"), Name: "Samuel Balogun", Email: "samuel.balogun@clario.demo"},
	}
}

func seedCommittees() []seedCommittee {
	users := demoUsersForSeed()
	return []seedCommittee{
		{
			ID:               mustUUID("11111111-1111-1111-1111-111111112001"),
			Name:             "Board of Directors",
			Type:             model.CommitteeTypeBoard,
			Description:      "Primary board governance body overseeing enterprise strategy and fiduciary decisions.",
			ChairUserID:      users[0].ID,
			ViceChairUserID:  seedPtr(users[1].ID),
			SecretaryUserID:  seedPtr(users[2].ID),
			MeetingFrequency: model.MeetingFrequencyQuarterly,
			QuorumPercentage: 60,
			QuorumType:       model.QuorumTypePercentage,
			Charter:          "The board provides strategic direction, approves major decisions, and oversees executive performance.",
			Tags:             []string{"board", "governance"},
			Members:          []uuid.UUID{users[0].ID, users[1].ID, users[2].ID, users[3].ID, users[4].ID, users[5].ID, users[6].ID, users[7].ID, users[8].ID},
		},
		{
			ID:               mustUUID("11111111-1111-1111-1111-111111112002"),
			Name:             "Audit Committee",
			Type:             model.CommitteeTypeAudit,
			Description:      "Oversight committee for audit, internal control, and external assurance matters.",
			ChairUserID:      users[3].ID,
			ViceChairUserID:  seedPtr(users[4].ID),
			SecretaryUserID:  seedPtr(users[5].ID),
			MeetingFrequency: model.MeetingFrequencyMonthly,
			QuorumPercentage: 51,
			QuorumType:       model.QuorumTypePercentage,
			Charter:          "The audit committee reviews financial reporting, assurance activity, and control remediation.",
			Tags:             []string{"audit", "assurance"},
			Members:          []uuid.UUID{users[3].ID, users[4].ID, users[5].ID, users[6].ID, users[7].ID},
		},
		{
			ID:               mustUUID("11111111-1111-1111-1111-111111112003"),
			Name:             "Risk Committee",
			Type:             model.CommitteeTypeRisk,
			Description:      "Committee overseeing enterprise risk, resilience, and regulatory exposure.",
			ChairUserID:      users[6].ID,
			ViceChairUserID:  seedPtr(users[7].ID),
			SecretaryUserID:  seedPtr(users[8].ID),
			MeetingFrequency: model.MeetingFrequencyMonthly,
			QuorumPercentage: 51,
			QuorumType:       model.QuorumTypePercentage,
			Charter:          "The risk committee reviews the risk register, key exposures, resilience, and treatment plans.",
			Tags:             []string{"risk", "resilience"},
			Members:          []uuid.UUID{users[5].ID, users[6].ID, users[7].ID, users[8].ID, users[9].ID, users[10].ID, users[11].ID},
		},
		{
			ID:               mustUUID("11111111-1111-1111-1111-111111112004"),
			Name:             "Compensation Committee",
			Type:             model.CommitteeTypeCompensation,
			Description:      "Committee reviewing executive compensation and incentives.",
			ChairUserID:      users[1].ID,
			ViceChairUserID:  seedPtr(users[2].ID),
			SecretaryUserID:  seedPtr(users[9].ID),
			MeetingFrequency: model.MeetingFrequencyQuarterly,
			QuorumPercentage: 51,
			QuorumType:       model.QuorumTypePercentage,
			Charter:          "The compensation committee reviews remuneration policy and executive performance incentives.",
			Tags:             []string{"compensation", "people"},
			Members:          []uuid.UUID{users[1].ID, users[2].ID, users[9].ID, users[10].ID},
		},
	}
}

func seedMeetings() []seedMeeting {
	users := demoUsersForSeed()
	board := mustUUID("11111111-1111-1111-1111-111111112001")
	audit := mustUUID("11111111-1111-1111-1111-111111112002")
	risk := mustUUID("11111111-1111-1111-1111-111111112003")
	proxyName := users[11].Name
	return []seedMeeting{
		{ID: mustUUID("11111111-1111-1111-1111-111111113001"), CommitteeID: board, Title: "Board Q3 2025 Meeting", Description: "Quarterly board strategy and oversight session.", MeetingNumber: 1, ScheduledAt: seedTime(2025, time.September, 18, 9, 0), Duration: 120, Location: "Lagos HQ Boardroom", Status: model.MeetingStatusCompleted, QuorumRequired: 6, Attendance: map[uuid.UUID]seedAttendance{users[4].ID: {Status: model.AttendanceStatusAbsent}, users[7].ID: {Status: model.AttendanceStatusProxy, ProxyUserID: seedPtr(users[11].ID), ProxyUserName: &proxyName}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113002"), CommitteeID: board, Title: "Board Q4 2025 Meeting", Description: "Year-end board review and governance approvals.", MeetingNumber: 2, ScheduledAt: seedTime(2025, time.December, 12, 9, 0), Duration: 135, Location: "Lagos HQ Boardroom", Status: model.MeetingStatusCompleted, QuorumRequired: 6, Attendance: map[uuid.UUID]seedAttendance{users[5].ID: {Status: model.AttendanceStatusExcused}, users[8].ID: {Status: model.AttendanceStatusAbsent}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113003"), CommitteeID: board, Title: "Board Q2 2026 Meeting", Description: "Scheduled board session for Q2 strategic matters.", MeetingNumber: 3, ScheduledAt: seedTime(2026, time.April, 17, 9, 0), Duration: 120, Location: "Lagos HQ Boardroom", Status: model.MeetingStatusScheduled, QuorumRequired: 6, Attendance: map[uuid.UUID]seedAttendance{}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113004"), CommitteeID: audit, Title: "Audit September 2025 Meeting", Description: "Monthly audit committee review.", MeetingNumber: 1, ScheduledAt: seedTime(2025, time.September, 9, 10, 0), Duration: 90, Location: "Finance Conference Room", Status: model.MeetingStatusCompleted, QuorumRequired: 3, Attendance: map[uuid.UUID]seedAttendance{users[7].ID: {Status: model.AttendanceStatusAbsent}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113005"), CommitteeID: audit, Title: "Audit October 2025 Meeting", Description: "Monthly audit committee review.", MeetingNumber: 2, ScheduledAt: seedTime(2025, time.October, 14, 10, 0), Duration: 90, Location: "Finance Conference Room", Status: model.MeetingStatusCompleted, QuorumRequired: 3, Attendance: map[uuid.UUID]seedAttendance{users[6].ID: {Status: model.AttendanceStatusAbsent}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113006"), CommitteeID: audit, Title: "Audit November 2025 Meeting", Description: "Monthly audit committee review.", MeetingNumber: 3, ScheduledAt: seedTime(2025, time.November, 11, 10, 0), Duration: 90, Location: "Finance Conference Room", Status: model.MeetingStatusCompleted, QuorumRequired: 3, Attendance: map[uuid.UUID]seedAttendance{}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113007"), CommitteeID: audit, Title: "Audit December 2025 Meeting", Description: "Monthly audit committee review.", MeetingNumber: 4, ScheduledAt: seedTime(2025, time.December, 16, 10, 0), Duration: 90, Location: "Finance Conference Room", Status: model.MeetingStatusCompleted, QuorumRequired: 3, Attendance: map[uuid.UUID]seedAttendance{users[4].ID: {Status: model.AttendanceStatusExcused}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113008"), CommitteeID: audit, Title: "Audit April 2026 Meeting", Description: "Scheduled monthly audit committee review.", MeetingNumber: 5, ScheduledAt: seedTime(2026, time.April, 14, 10, 0), Duration: 90, Location: "Finance Conference Room", Status: model.MeetingStatusScheduled, QuorumRequired: 3, Attendance: map[uuid.UUID]seedAttendance{}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113009"), CommitteeID: risk, Title: "Risk October 2025 Meeting", Description: "Monthly enterprise risk review.", MeetingNumber: 1, ScheduledAt: seedTime(2025, time.October, 8, 11, 0), Duration: 105, Location: "Risk War Room", Status: model.MeetingStatusCompleted, QuorumRequired: 4, Attendance: map[uuid.UUID]seedAttendance{users[9].ID: {Status: model.AttendanceStatusAbsent}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113010"), CommitteeID: risk, Title: "Risk November 2025 Meeting", Description: "Monthly enterprise risk review.", MeetingNumber: 2, ScheduledAt: seedTime(2025, time.November, 12, 11, 0), Duration: 105, Location: "Risk War Room", Status: model.MeetingStatusCompleted, QuorumRequired: 4, Attendance: map[uuid.UUID]seedAttendance{users[10].ID: {Status: model.AttendanceStatusAbsent}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113011"), CommitteeID: risk, Title: "Risk December 2025 Meeting", Description: "Monthly enterprise risk review.", MeetingNumber: 3, ScheduledAt: seedTime(2025, time.December, 10, 11, 0), Duration: 105, Location: "Risk War Room", Status: model.MeetingStatusCompleted, QuorumRequired: 4, Attendance: map[uuid.UUID]seedAttendance{users[8].ID: {Status: model.AttendanceStatusAbsent}, users[9].ID: {Status: model.AttendanceStatusAbsent}, users[10].ID: {Status: model.AttendanceStatusAbsent}, users[11].ID: {Status: model.AttendanceStatusExcused}}},
		{ID: mustUUID("11111111-1111-1111-1111-111111113012"), CommitteeID: risk, Title: "Risk April 2026 Meeting", Description: "Scheduled monthly enterprise risk review.", MeetingNumber: 4, ScheduledAt: seedTime(2026, time.April, 9, 11, 0), Duration: 105, Location: "Risk War Room", Status: model.MeetingStatusScheduled, QuorumRequired: 4, Attendance: map[uuid.UUID]seedAttendance{}},
	}
}

func seedAgendaItems(meetings []seedMeeting) []seedAgenda {
	users := demoUsersForSeed()
	voteMajority := model.VoteTypeMajority
	voteTwoThirds := model.VoteTypeTwoThirds
	voteUnanimous := model.VoteTypeUnanimous
	approved := model.VoteResultApproved
	rejected := model.VoteResultRejected
	deferred := model.VoteResultDeferred
	counts := []int{3, 4, 3, 4, 4, 3, 3, 3, 3}
	titles := []string{
		"Review prior board resolutions", "Approve capital allocation framework", "CEO performance update",
		"Approve 2026 operating plan", "Review cyber resilience posture", "Board diversity policy draft", "Confirm succession planning review",
		"Review external audit findings", "Approve remediation timeline", "Internal audit update", "Whistleblower case summary",
		"Review control exceptions", "Vendor audit coverage plan", "Financial statement close readiness", "Approve internal controls attestation", "Risk of fraud briefing",
		"Data retention exceptions", "Expense policy update", "Treasury controls review",
		"Risk appetite refresh", "Update risk register", "Third-party concentration review",
		"Business continuity scenario", "Insurance coverage review", "Operational loss report",
		"Emerging regulatory risks", "Liquidity risk triggers", "Incident response tabletop", "Stress testing outcomes", "Model risk governance note",
	}
	out := make([]seedAgenda, 0, 30)
	titleIdx := 0
	agendaIdx := 1
	for meetingIdx := 0; meetingIdx < 9; meetingIdx++ {
		for order := 0; order < counts[meetingIdx]; order++ {
			title := titles[titleIdx]
			presenter := users[(meetingIdx+order)%len(users)]
			item := seedAgenda{
				ID:            mustUUID(fmt.Sprintf("11111111-1111-1111-1111-%012d", 3000+agendaIdx)),
				MeetingID:     meetings[meetingIdx].ID,
				Title:         title,
				Description:   "Governance discussion item: " + title,
				Presenter:     presenter.ID,
				PresenterName: presenter.Name,
				OrderIndex:    order,
				Status:        model.AgendaItemStatusDiscussed,
				Category:      model.AgendaCategoryRegular,
				Notes:         fmt.Sprintf("%s was reviewed in detail. Management highlighted current status and key dependencies. ACTION: %s will advance the next step before the next committee meeting.", title, presenter.Name),
			}
			out = append(out, item)
			titleIdx++
			agendaIdx++
		}
	}
	for _, idx := range []int{1, 5, 9, 13, 18, 22, 26, 28} {
		out[idx].Category = model.AgendaCategoryDecision
		out[idx].RequiresVote = true
	}
	for _, idx := range []int{2, 10, 11, 19, 29} {
		out[idx].Category = model.AgendaCategoryInformation
		out[idx].Status = model.AgendaItemStatusForNoting
	}
	for _, idx := range []int{6, 23} {
		out[idx].Category = model.AgendaCategoryDiscussion
	}
	// Adjust the 8 required vote outcomes.
	out[1].VoteType, out[1].RequiresVote, out[1].VoteResult, out[1].Status = &voteUnanimous, true, &approved, model.AgendaItemStatusApproved
	out[1].VotesFor, out[1].VotesAgainst, out[1].VotesAbstained = seedPtr(8), seedPtr(0), seedPtr(0)
	out[5].VoteType, out[5].RequiresVote, out[5].VoteResult, out[5].Status = &voteTwoThirds, true, &approved, model.AgendaItemStatusApproved
	out[5].VotesFor, out[5].VotesAgainst, out[5].VotesAbstained = seedPtr(7), seedPtr(2), seedPtr(0)
	out[9].VoteType, out[9].RequiresVote, out[9].VoteResult, out[9].Status = &voteMajority, true, &approved, model.AgendaItemStatusApproved
	out[9].VotesFor, out[9].VotesAgainst, out[9].VotesAbstained = seedPtr(4), seedPtr(1), seedPtr(0)
	out[13].VoteType, out[13].RequiresVote, out[13].VoteResult, out[13].Status = &voteMajority, true, &approved, model.AgendaItemStatusApproved
	out[13].VotesFor, out[13].VotesAgainst, out[13].VotesAbstained = seedPtr(4), seedPtr(0), seedPtr(1)
	out[18].VoteType, out[18].RequiresVote, out[18].VoteResult, out[18].Status = &voteMajority, true, &rejected, model.AgendaItemStatusRejected
	out[18].VotesFor, out[18].VotesAgainst, out[18].VotesAbstained = seedPtr(2), seedPtr(3), seedPtr(0)
	out[22].VoteType, out[22].RequiresVote, out[22].VoteResult, out[22].Status = &voteTwoThirds, true, &approved, model.AgendaItemStatusApproved
	out[22].VotesFor, out[22].VotesAgainst, out[22].VotesAbstained = seedPtr(5), seedPtr(2), seedPtr(0)
	out[26].VoteType, out[26].RequiresVote, out[26].VoteResult, out[26].Status = &voteMajority, true, &deferred, model.AgendaItemStatusDeferred
	out[26].VotesFor, out[26].VotesAgainst, out[26].VotesAbstained = seedPtr(3), seedPtr(3), seedPtr(1)
	out[28].VoteType, out[28].RequiresVote, out[28].VoteResult, out[28].Status = &voteMajority, true, &approved, model.AgendaItemStatusApproved
	out[28].VotesFor, out[28].VotesAgainst, out[28].VotesAbstained = seedPtr(5), seedPtr(1), seedPtr(0)
	return out
}

func seedMinutesRecords(meetings []seedMeeting) []seedMinutes {
	return []seedMinutes{
		{ID: mustUUID("11111111-1111-1111-1111-111111114001"), MeetingID: meetings[0].ID, Status: model.MinutesStatusPublished, Version: 1, AIGenerated: true},
		{ID: mustUUID("11111111-1111-1111-1111-111111114002"), MeetingID: meetings[1].ID, Status: model.MinutesStatusPublished, Version: 1, AIGenerated: true},
		{ID: mustUUID("11111111-1111-1111-1111-111111114003"), MeetingID: meetings[3].ID, Status: model.MinutesStatusPublished, Version: 1, AIGenerated: true},
		{ID: mustUUID("11111111-1111-1111-1111-111111114004"), MeetingID: meetings[4].ID, Status: model.MinutesStatusPublished, Version: 1, AIGenerated: true},
		{ID: mustUUID("11111111-1111-1111-1111-111111114005"), MeetingID: meetings[5].ID, Status: model.MinutesStatusPublished, Version: 1, AIGenerated: true},
		{ID: mustUUID("11111111-1111-1111-1111-111111114006"), MeetingID: meetings[6].ID, Status: model.MinutesStatusApproved, Version: 1, AIGenerated: false},
		{ID: mustUUID("11111111-1111-1111-1111-111111114007"), MeetingID: meetings[8].ID, Status: model.MinutesStatusApproved, Version: 1, AIGenerated: false},
		{ID: mustUUID("11111111-1111-1111-1111-111111114008"), MeetingID: meetings[9].ID, Status: model.MinutesStatusDraft, Version: 1, AIGenerated: false},
	}
}

func seedActionItems(agendaItems []seedAgenda, users []seedUser) []seedActionItem {
	titles := []string{
		"Prepare Q1 budget proposal", "Review vendor security audit", "Update risk register", "Draft board diversity policy", "Validate treasury controls",
		"Refresh incident response contacts", "Document whistleblower reporting metrics", "Complete cyber insurance benchmark", "Finalize capital allocation analysis", "Publish internal controls memo",
		"Review continuity testing gaps", "Approve audit issue owner matrix", "Track remediation on control exceptions", "Confirm policy exception register", "Refine succession planning deck",
		"Review regulatory change inventory", "Update liquidity trigger thresholds", "Prepare board education calendar", "Document vendor due diligence findings", "Refresh conduct risk heatmap",
		"Compile stress testing action log", "Review operating model dependencies", "Complete resilience tabletop actions", "Update document retention register", "Finalize governance training schedule",
	}
	statuses := []model.ActionItemStatus{
		model.ActionItemStatusCompleted, model.ActionItemStatusCompleted, model.ActionItemStatusCompleted, model.ActionItemStatusCompleted, model.ActionItemStatusCompleted,
		model.ActionItemStatusCompleted, model.ActionItemStatusCompleted, model.ActionItemStatusCompleted, model.ActionItemStatusCompleted, model.ActionItemStatusCompleted,
		model.ActionItemStatusCompleted, model.ActionItemStatusCompleted,
		model.ActionItemStatusPending, model.ActionItemStatusPending, model.ActionItemStatusPending, model.ActionItemStatusPending, model.ActionItemStatusPending, model.ActionItemStatusPending, model.ActionItemStatusPending, model.ActionItemStatusPending,
		model.ActionItemStatusOverdue, model.ActionItemStatusOverdue, model.ActionItemStatusOverdue,
		model.ActionItemStatusInProgress, model.ActionItemStatusInProgress,
	}
	priorities := []model.ActionItemPriority{
		model.ActionItemPriorityHigh, model.ActionItemPriorityHigh, model.ActionItemPriorityMedium, model.ActionItemPriorityMedium, model.ActionItemPriorityHigh,
		model.ActionItemPriorityMedium, model.ActionItemPriorityLow, model.ActionItemPriorityHigh, model.ActionItemPriorityCritical, model.ActionItemPriorityMedium,
		model.ActionItemPriorityHigh, model.ActionItemPriorityMedium, model.ActionItemPriorityMedium, model.ActionItemPriorityLow, model.ActionItemPriorityHigh,
		model.ActionItemPriorityMedium, model.ActionItemPriorityHigh, model.ActionItemPriorityLow, model.ActionItemPriorityMedium, model.ActionItemPriorityMedium,
		model.ActionItemPriorityHigh, model.ActionItemPriorityHigh, model.ActionItemPriorityCritical, model.ActionItemPriorityMedium, model.ActionItemPriorityHigh,
	}
	out := make([]seedActionItem, 0, len(titles))
	for idx, title := range titles {
		agenda := agendaItems[idx%len(agendaItems)]
		assignee := users[(idx+3)%len(users)]
		dueDate := seedTime(2026, time.March, 15+idx%20, 0, 0)
		switch statuses[idx] {
		case model.ActionItemStatusCompleted:
			dueDate = seedTime(2026, time.January, 10+idx%20, 0, 0)
		case model.ActionItemStatusOverdue:
			dueDate = seedTime(2026, time.February, 1+idx, 0, 0)
		case model.ActionItemStatusInProgress:
			dueDate = seedTime(2026, time.April, 3+idx, 0, 0)
		}
		out = append(out, seedActionItem{
			ID:           mustUUID(fmt.Sprintf("11111111-1111-1111-1111-%012d", 5000+idx+1)),
			MeetingID:    agenda.MeetingID,
			AgendaItemID: &agenda.ID,
			CommitteeID:  committeeIDForMeeting(agenda.MeetingID),
			Title:        title,
			Description:  "Follow-up action arising from committee deliberations: " + title,
			Priority:     priorities[idx],
			AssignedTo:   assignee.ID,
			AssigneeName: assignee.Name,
			DueDate:      dueDate,
			Status:       statuses[idx],
		})
	}
	return out
}

func committeeIDForMeeting(meetingID uuid.UUID) uuid.UUID {
	switch meetingID {
	case mustUUID("11111111-1111-1111-1111-111111113001"), mustUUID("11111111-1111-1111-1111-111111113002"), mustUUID("11111111-1111-1111-1111-111111113003"):
		return mustUUID("11111111-1111-1111-1111-111111112001")
	case mustUUID("11111111-1111-1111-1111-111111113004"), mustUUID("11111111-1111-1111-1111-111111113005"), mustUUID("11111111-1111-1111-1111-111111113006"), mustUUID("11111111-1111-1111-1111-111111113007"), mustUUID("11111111-1111-1111-1111-111111113008"):
		return mustUUID("11111111-1111-1111-1111-111111112002")
	default:
		return mustUUID("11111111-1111-1111-1111-111111112003")
	}
}

func chairForCommittee(committeeID uuid.UUID) uuid.UUID {
	switch committeeID {
	case mustUUID("11111111-1111-1111-1111-111111112001"):
		return mustUUID("11111111-1111-1111-1111-111111111001")
	case mustUUID("11111111-1111-1111-1111-111111112002"):
		return mustUUID("11111111-1111-1111-1111-111111111004")
	case mustUUID("11111111-1111-1111-1111-111111112003"):
		return mustUUID("11111111-1111-1111-1111-111111111007")
	default:
		return mustUUID("11111111-1111-1111-1111-111111111002")
	}
}

func secretaryForCommittee(committeeID uuid.UUID) uuid.UUID {
	switch committeeID {
	case mustUUID("11111111-1111-1111-1111-111111112001"):
		return mustUUID("11111111-1111-1111-1111-111111111003")
	case mustUUID("11111111-1111-1111-1111-111111112002"):
		return mustUUID("11111111-1111-1111-1111-111111111006")
	case mustUUID("11111111-1111-1111-1111-111111112003"):
		return mustUUID("11111111-1111-1111-1111-111111111009")
	default:
		return mustUUID("11111111-1111-1111-1111-111111111010")
	}
}

func committeeRoleForUser(committee seedCommittee, userID uuid.UUID) model.CommitteeMemberRole {
	if userID == committee.ChairUserID {
		return model.CommitteeMemberRoleChair
	}
	if committee.ViceChairUserID != nil && userID == *committee.ViceChairUserID {
		return model.CommitteeMemberRoleViceChair
	}
	if committee.SecretaryUserID != nil && userID == *committee.SecretaryUserID {
		return model.CommitteeMemberRoleSecretary
	}
	return model.CommitteeMemberRoleMember
}

func countSeedPresent(members []uuid.UUID, attendance map[uuid.UUID]seedAttendance) int {
	present := 0
	for _, memberID := range members {
		seed, ok := attendance[memberID]
		if !ok {
			present++
			continue
		}
		if seed.Status == "" || seed.Status == model.AttendanceStatusPresent || seed.Status == model.AttendanceStatusProxy {
			present++
		}
	}
	return present
}

func seedTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

func seedPtr[T any](value T) *T {
	return &value
}

func mustUUID(raw string) uuid.UUID {
	return uuid.MustParse(raw)
}
