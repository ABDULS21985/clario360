package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/acta/model"
)

func TestGenerateFullMinutesDeterministic(t *testing.T) {
	generator, err := NewMinutesGenerator()
	if err != nil {
		t.Fatalf("NewMinutesGenerator returned error: %v", err)
	}

	start := time.Date(2026, time.March, 1, 9, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)
	quorumMet := false
	voteType := model.VoteTypeMajority
	voteResult := model.VoteResultApproved
	votesFor := 8
	votesAgainst := 2
	votesAbstained := 1

	meeting := &model.Meeting{
		ID:             uuid.New(),
		CommitteeID:    uuid.New(),
		CommitteeName:  "Audit Committee",
		MeetingNumber:  ptr(4),
		ScheduledAt:    start,
		ActualStartAt:  &start,
		ActualEndAt:    &end,
		Location:       ptr("Finance Conference Room"),
		LocationType:   model.LocationTypePhysical,
		QuorumRequired: 6,
		QuorumMet:      &quorumMet,
	}
	agenda := []model.AgendaItem{
		{
			ID:             uuid.New(),
			Title:          "Approve internal controls attestation",
			Status:         model.AgendaItemStatusApproved,
			RequiresVote:   true,
			VoteType:       &voteType,
			VotesFor:       &votesFor,
			VotesAgainst:   &votesAgainst,
			VotesAbstained: &votesAbstained,
			VoteResult:     &voteResult,
			Notes:          ptr("ACTION: John to finalize the attestation package by March 15, 2026."),
			Category:       ptr(model.AgendaCategoryDecision),
		},
	}
	attendance := []model.Attendee{
		{UserName: "Amina Okafor", MemberRole: model.CommitteeMemberRoleChair, Status: model.AttendanceStatusPresent},
		{UserName: "Grace Nwosu", MemberRole: model.CommitteeMemberRoleSecretary, Status: model.AttendanceStatusPresent},
		{UserName: "Daniel Mensah", MemberRole: model.CommitteeMemberRoleMember, Status: model.AttendanceStatusPresent},
		{UserName: "Lara Adeyemi", MemberRole: model.CommitteeMemberRoleMember, Status: model.AttendanceStatusAbsent},
		{UserName: "Oliver Dike", MemberRole: model.CommitteeMemberRoleMember, Status: model.AttendanceStatusProxy, ProxyUserName: ptr("Samuel Balogun")},
	}
	actionItems := []model.ActionItem{
		{
			ID:           uuid.New(),
			Title:        "Finalize attestation package",
			AssigneeName: "John",
			DueDate:      time.Date(2026, time.March, 15, 0, 0, 0, 0, time.UTC),
			Priority:     model.ActionItemPriorityHigh,
		},
	}
	nextMeeting := &model.Meeting{ScheduledAt: time.Date(2026, time.April, 1, 9, 0, 0, 0, time.UTC)}

	first, err := generator.Generate(meeting, agenda, attendance, actionItems, nextMeeting)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	second, err := generator.Generate(meeting, agenda, attendance, actionItems, nextMeeting)
	if err != nil {
		t.Fatalf("Generate returned error on second call: %v", err)
	}

	if first.Content != second.Content || first.AISummary != second.AISummary {
		t.Fatal("Generate is not deterministic for identical input")
	}
	if !strings.Contains(first.Content, "**Present:** Amina Okafor, Daniel Mensah, Grace Nwosu") {
		t.Fatal("minutes content does not include present attendees")
	}
	if !strings.Contains(first.Content, "**Absent:** Lara Adeyemi") {
		t.Fatal("minutes content does not include absent attendees")
	}
	if !strings.Contains(first.Content, "**By Proxy:** Oliver Dike (proxy: Samuel Balogun)") {
		t.Fatal("minutes content does not include proxy attendees")
	}
	if !strings.Contains(first.Content, "**Result:** approved") {
		t.Fatal("minutes content does not include voting result")
	}
	if !strings.Contains(first.Content, "**Quorum:** NOT MET") {
		t.Fatal("minutes content does not include quorum warning")
	}
	if !strings.Contains(first.AISummary, "Approve internal controls attestation (approved)") {
		t.Fatal("summary does not include decision summary")
	}
	if !strings.Contains(first.AISummary, "may require ratification") {
		t.Fatal("summary does not include quorum ratification note")
	}
}

func ptr[T any](value T) *T {
	return &value
}
