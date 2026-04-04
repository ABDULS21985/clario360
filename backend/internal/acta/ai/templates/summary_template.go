package templates

const ExecutiveSummaryTemplate = `The {{.CommitteeName}} met on {{.Date}} with {{.PresentCount}} of {{.TotalMembers}} members present ({{.QuorumPhrase}}). {{.AgendaItemCount}} agenda items were discussed. Key decisions: {{.DecisionSummary}}. {{.ActionItemCount}} action items were assigned, with {{.HighPriorityCount}} marked as high priority.{{if .NeedsRatification}} Note: Quorum was not met. Decisions made in this meeting may require ratification.{{end}}`
