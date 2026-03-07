package ai

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/clario360/platform/internal/acta/ai/templates"
	"github.com/clario360/platform/internal/acta/model"
)

type MinutesGenerator struct {
	minutesTemplate *template.Template
	summaryBuilder  *SummaryBuilder
	actionExtractor *ActionExtractor
}

type MinutesTemplateData struct {
	CommitteeName  string
	MeetingNumber  string
	Date           string
	StartTime      string
	EndTime        string
	LocationLine   string
	Chair          string
	Secretary      string
	PresentLine    string
	AbsentLine     string
	ProxyLine      string
	PresentCount   int
	QuorumRequired int
	QuorumMet      bool
	AgendaItems    []MinutesAgendaItem
	ActionItems    []MinutesActionItem
	HasActionItems bool
	NextMeetingDate string
}

type MinutesAgendaItem struct {
	DisplayNumber  string
	Title          string
	Presenter      string
	Category       string
	Notes          string
	RequiresVote   bool
	VoteType       string
	VotesFor       int
	VotesAgainst   int
	VotesAbstained int
	VoteResult     string
	Status         string
}

type MinutesActionItem struct {
	Number     int
	Title      string
	AssignedTo string
	DueDate    string
	Priority   string
}

func NewMinutesGenerator() (*MinutesGenerator, error) {
	tmpl, err := template.New("minutes").Parse(templates.MinutesMarkdownTemplate)
	if err != nil {
		return nil, err
	}
	summaryBuilder, err := NewSummaryBuilder()
	if err != nil {
		return nil, err
	}
	return &MinutesGenerator{
		minutesTemplate: tmpl,
		summaryBuilder:  summaryBuilder,
		actionExtractor: NewActionExtractor(),
	}, nil
}

func (g *MinutesGenerator) Generate(meeting *model.Meeting, agenda []model.AgendaItem, attendance []model.Attendee, actionItems []model.ActionItem, nextMeeting *model.Meeting) (*model.GeneratedMinutes, error) {
	if meeting == nil {
		return nil, fmt.Errorf("meeting is required")
	}

	extracted := g.collectExtractedActions(agenda)
	templateActions := buildTemplateActions(actionItems, extracted)
	data := MinutesTemplateData{
		CommitteeName:  meeting.CommitteeName,
		MeetingNumber:  meetingNumber(meeting.MeetingNumber),
		Date:           meetingDate(meeting),
		StartTime:      meetingStart(meeting),
		EndTime:        meetingEnd(meeting),
		LocationLine:   locationLine(meeting),
		Chair:          attendeeByRole(attendance, model.CommitteeMemberRoleChair),
		Secretary:      attendeeByRole(attendance, model.CommitteeMemberRoleSecretary),
		PresentLine:    attendeeLine(attendance, model.AttendanceStatusPresent),
		AbsentLine:     attendeeLine(attendance, model.AttendanceStatusAbsent),
		ProxyLine:      proxyLine(attendance),
		PresentCount:   countPresent(attendance),
		QuorumRequired: meeting.QuorumRequired,
		QuorumMet:      meeting.QuorumMet != nil && *meeting.QuorumMet,
		AgendaItems:    formatAgenda(agenda),
		ActionItems:    templateActions,
		HasActionItems: len(templateActions) > 0,
		NextMeetingDate: nextMeetingLine(nextMeeting),
	}

	var buf bytes.Buffer
	if err := g.minutesTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}

	summary, err := g.summaryBuilder.Build(meeting, agenda, attendance, actionItems)
	if err != nil {
		return nil, err
	}

	return &model.GeneratedMinutes{
		Content:       buf.String(),
		AISummary:     summary,
		AIActionItems: extracted,
	}, nil
}

func (g *MinutesGenerator) collectExtractedActions(agenda []model.AgendaItem) []model.ExtractedAction {
	combined := make([]model.ExtractedAction, 0)
	for _, item := range agenda {
		combined = append(combined, g.actionExtractor.Extract(item.Title, derefString(item.Notes))...)
	}
	sort.Slice(combined, func(i, j int) bool {
		if combined[i].Source == combined[j].Source {
			return combined[i].Title < combined[j].Title
		}
		return combined[i].Source < combined[j].Source
	})
	return combined
}

func buildTemplateActions(persisted []model.ActionItem, extracted []model.ExtractedAction) []MinutesActionItem {
	out := make([]MinutesActionItem, 0, max(len(persisted), len(extracted)))
	for idx, item := range persisted {
		out = append(out, MinutesActionItem{
			Number:     idx + 1,
			Title:      item.Title,
			AssignedTo: item.AssigneeName,
			DueDate:    item.DueDate.UTC().Format("2006-01-02"),
			Priority:   string(item.Priority),
		})
	}
	if len(out) > 0 {
		return out
	}
	for idx, item := range extracted {
		dueDate := "Not specified"
		if item.DueDate != nil {
			dueDate = item.DueDate.UTC().Format("2006-01-02")
		}
		out = append(out, MinutesActionItem{
			Number:     idx + 1,
			Title:      item.Title,
			AssignedTo: item.AssignedTo,
			DueDate:    dueDate,
			Priority:   item.Priority,
		})
	}
	return out
}

func formatAgenda(items []model.AgendaItem) []MinutesAgendaItem {
	out := make([]MinutesAgendaItem, 0, len(items))
	for _, item := range items {
		out = append(out, MinutesAgendaItem{
			DisplayNumber:  agendaNumber(item),
			Title:          item.Title,
			Presenter:      defaultString(derefString(item.PresenterName), "Unspecified"),
			Category:       defaultString(stringValue(item.Category), "regular"),
			Notes:          defaultString(strings.TrimSpace(derefString(item.Notes)), "No discussion notes recorded."),
			RequiresVote:   item.RequiresVote,
			VoteType:       stringValue(item.VoteType),
			VotesFor:       intValue(item.VotesFor),
			VotesAgainst:   intValue(item.VotesAgainst),
			VotesAbstained: intValue(item.VotesAbstained),
			VoteResult:     stringValue(item.VoteResult),
			Status:         string(item.Status),
		})
	}
	return out
}

func meetingNumber(number *int) string {
	if number == nil {
		return "TBD"
	}
	return fmt.Sprintf("%d", *number)
}

func meetingDate(meeting *model.Meeting) string {
	if meeting.ActualStartAt != nil {
		return meeting.ActualStartAt.UTC().Format("January 2, 2006")
	}
	return meeting.ScheduledAt.UTC().Format("January 2, 2006")
}

func meetingStart(meeting *model.Meeting) string {
	if meeting.ActualStartAt != nil {
		return meeting.ActualStartAt.UTC().Format("3:04 PM")
	}
	return meeting.ScheduledAt.UTC().Format("3:04 PM")
}

func meetingEnd(meeting *model.Meeting) string {
	if meeting.ActualEndAt != nil {
		return meeting.ActualEndAt.UTC().Format("3:04 PM")
	}
	if meeting.ScheduledEndAt != nil {
		return meeting.ScheduledEndAt.UTC().Format("3:04 PM")
	}
	return meeting.ScheduledAt.Add(time.Duration(meeting.DurationMinutes) * time.Minute).UTC().Format("3:04 PM")
}

func locationLine(meeting *model.Meeting) string {
	location := defaultString(derefString(meeting.Location), "Not specified")
	if meeting.VirtualLink != nil && *meeting.VirtualLink != "" {
		return fmt.Sprintf("%s (%s, %s)", location, meeting.LocationType, *meeting.VirtualLink)
	}
	return fmt.Sprintf("%s (%s)", location, meeting.LocationType)
}

func attendeeByRole(attendance []model.Attendee, role model.CommitteeMemberRole) string {
	for _, attendee := range attendance {
		if attendee.MemberRole == role {
			return attendee.UserName
		}
	}
	return "Unspecified"
}

func attendeeLine(attendance []model.Attendee, status model.AttendanceStatus) string {
	names := make([]string, 0)
	for _, attendee := range attendance {
		if attendee.Status == status {
			names = append(names, attendee.UserName)
		}
	}
	if len(names) == 0 {
		return "None"
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

func proxyLine(attendance []model.Attendee) string {
	entries := make([]string, 0)
	for _, attendee := range attendance {
		if attendee.Status == model.AttendanceStatusProxy {
			entry := attendee.UserName
			if attendee.ProxyUserName != nil {
				entry = fmt.Sprintf("%s (proxy: %s)", attendee.UserName, *attendee.ProxyUserName)
			}
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return "None"
	}
	sort.Strings(entries)
	return strings.Join(entries, ", ")
}

func countPresent(attendance []model.Attendee) int {
	count := 0
	for _, attendee := range attendance {
		if attendee.Status == model.AttendanceStatusPresent || attendee.Status == model.AttendanceStatusProxy {
			count++
		}
	}
	return count
}

func nextMeetingLine(meeting *model.Meeting) string {
	if meeting == nil {
		return "To be scheduled"
	}
	return meeting.ScheduledAt.UTC().Format("January 2, 2006 3:04 PM")
}

func agendaNumber(item model.AgendaItem) string {
	if item.ItemNumber != nil && *item.ItemNumber != "" {
		return *item.ItemNumber
	}
	return fmt.Sprintf("%d.", item.OrderIndex+1)
}

func stringValue[T ~string](value *T) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func intValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
