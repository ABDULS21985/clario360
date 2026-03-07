package ai

import (
	"bytes"
	"strings"
	"text/template"
	"time"

	"github.com/clario360/platform/internal/acta/ai/templates"
	"github.com/clario360/platform/internal/acta/model"
)

type SummaryBuilder struct {
	tmpl *template.Template
}

func NewSummaryBuilder() (*SummaryBuilder, error) {
	tmpl, err := template.New("executive_summary").Parse(templates.ExecutiveSummaryTemplate)
	if err != nil {
		return nil, err
	}
	return &SummaryBuilder{tmpl: tmpl}, nil
}

func (b *SummaryBuilder) Build(meeting *model.Meeting, agenda []model.AgendaItem, attendance []model.Attendee, actionItems []model.ActionItem) (string, error) {
	presentCount := 0
	totalMembers := 0
	for _, attendee := range attendance {
		totalMembers++
		if attendee.Status == model.AttendanceStatusPresent || attendee.Status == model.AttendanceStatusProxy {
			presentCount++
		}
	}
	highPriority := 0
	for _, action := range actionItems {
		if action.Priority == model.ActionItemPriorityHigh || action.Priority == model.ActionItemPriorityCritical {
			highPriority++
		}
	}

	decisions := make([]string, 0, len(agenda))
	for _, item := range agenda {
		if item.VoteResult == nil {
			continue
		}
		decisions = append(decisions, item.Title+" ("+string(*item.VoteResult)+")")
	}
	if len(decisions) == 0 {
		decisions = []string{"no formal votes were recorded"}
	}

	quorumPhrase := "quorum met"
	needsRatification := meeting.QuorumMet != nil && !*meeting.QuorumMet
	if needsRatification {
		quorumPhrase = "quorum not met"
	}

	data := map[string]any{
		"CommitteeName":    meeting.CommitteeName,
		"Date":             summaryDate(meeting.ActualStartAt, meeting.ScheduledAt),
		"PresentCount":     presentCount,
		"TotalMembers":     totalMembers,
		"QuorumPhrase":     quorumPhrase,
		"AgendaItemCount":  len(agenda),
		"DecisionSummary":  strings.Join(decisions, ", "),
		"ActionItemCount":  len(actionItems),
		"HighPriorityCount": highPriority,
		"NeedsRatification": needsRatification,
	}

	var buf bytes.Buffer
	if err := b.tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func summaryDate(actualStart *time.Time, scheduled time.Time) string {
	if actualStart != nil {
		return actualStart.UTC().Format("January 2, 2006")
	}
	return scheduled.UTC().Format("January 2, 2006")
}
