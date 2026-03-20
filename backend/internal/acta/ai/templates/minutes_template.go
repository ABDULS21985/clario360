package templates

const MinutesMarkdownTemplate = `# Minutes of the {{.CommitteeName}} Meeting #{{.MeetingNumber}}

**Date:** {{.Date}}
**Time:** {{.StartTime}} - {{.EndTime}}
**Location:** {{.LocationLine}}
**Chair:** {{.Chair}}
**Secretary:** {{.Secretary}}

## Attendance

**Present:** {{.PresentLine}}
**Absent:** {{.AbsentLine}}
**By Proxy:** {{.ProxyLine}}
**Quorum:** {{if .QuorumMet}}Met{{else}}NOT MET{{end}} ({{.PresentCount}}/{{.QuorumRequired}} required)

## Agenda Items
{{range .AgendaItems}}
### {{.DisplayNumber}} {{.Title}}
**Presenter:** {{.Presenter}}
**Category:** {{.Category}}

{{.Notes}}
{{if .RequiresVote}}

**Vote:** {{.VoteType}} - For: {{.VotesFor}}, Against: {{.VotesAgainst}}, Abstained: {{.VotesAbstained}}
**Result:** {{.VoteResult}}
{{end}}

**Status:** {{.Status}}

---
{{end}}

## Action Items

| # | Action | Assigned To | Due Date | Priority |
|---|--------|-------------|----------|----------|
{{range .ActionItems}}| {{.Number}} | {{.Title}} | {{.AssignedTo}} | {{.DueDate}} | {{.Priority}} |
{{end}}
{{if not .HasActionItems}}No action items were recorded.{{end}}

## Next Meeting
{{.NextMeetingDate}}

---
*Minutes prepared by {{.Secretary}}*
*Status: Draft*`
