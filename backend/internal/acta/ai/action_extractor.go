package ai

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/clario360/platform/internal/acta/ai/templates"
	"github.com/clario360/platform/internal/acta/model"
)

type ActionExtractor struct {
	dateLayouts []string
	nonWord     *regexp.Regexp
}

func NewActionExtractor() *ActionExtractor {
	return &ActionExtractor{
		dateLayouts: []string{
			"January 2, 2006",
			"January 2 2006",
			"Jan 2, 2006",
			"Jan 2 2006",
			"2/1/2006",
			"2-1-2006",
			"02/01/2006",
			"02-01-2006",
			"2/1/06",
			"2-1-06",
		},
		nonWord: regexp.MustCompile(`[^a-z0-9]+`),
	}
}

func (e *ActionExtractor) Extract(sourceTitle, notes string) []model.ExtractedAction {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return nil
	}

	candidates := make([]model.ExtractedAction, 0, 4)
	for _, line := range splitSentences(notes) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if match := templates.ActionMarkerPattern.FindStringSubmatch(line); len(match) == 2 {
			candidates = append(candidates, e.buildAction(sourceTitle, line, "Unspecified", match[1]))
			continue
		}
		if match := templates.WillPattern.FindStringSubmatch(line); len(match) == 3 {
			candidates = append(candidates, e.buildAction(sourceTitle, line, strings.TrimSpace(match[1]), strings.TrimSpace(match[2])))
			continue
		}
		if match := templates.AgreedPattern.FindStringSubmatch(line); len(match) == 2 {
			candidates = append(candidates, e.buildAction(sourceTitle, line, "Unspecified", strings.TrimSpace(match[1])))
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	return e.deduplicate(candidates)
}

func (e *ActionExtractor) buildAction(sourceTitle, fullText, assignee, task string) model.ExtractedAction {
	dueDate := e.extractDate(fullText)
	task = strings.TrimSpace(task)
	if len(task) > 100 {
		task = strings.TrimSpace(task[:100])
	}
	return model.ExtractedAction{
		Title:       task,
		Description: strings.TrimSpace(fullText),
		AssignedTo:  defaultString(strings.TrimSpace(assignee), "Unspecified"),
		DueDate:     dueDate,
		Priority:    inferPriority(fullText),
		Source:      sourceTitle,
	}
}

func (e *ActionExtractor) extractDate(text string) *time.Time {
	match := templates.DatePattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return nil
	}
	raw := strings.TrimSpace(match[1])
	for _, layout := range e.dateLayouts {
		if parsed, err := time.ParseInLocation(layout, raw, time.UTC); err == nil {
			value := parsed.UTC()
			return &value
		}
	}
	return nil
}

func (e *ActionExtractor) deduplicate(in []model.ExtractedAction) []model.ExtractedAction {
	merged := make([]model.ExtractedAction, 0, len(in))
	for _, item := range in {
		found := false
		for idx := range merged {
			if !strings.EqualFold(merged[idx].AssignedTo, item.AssignedTo) || !e.similarTask(merged[idx].Title, item.Title) {
				continue
			}
			if len(item.Description) > len(merged[idx].Description) {
				merged[idx].Description = item.Description
			}
			if len(item.Title) > len(merged[idx].Title) {
				merged[idx].Title = item.Title
			}
			if merged[idx].DueDate == nil && item.DueDate != nil {
				merged[idx].DueDate = item.DueDate
			}
			if priorityWeight(item.Priority) > priorityWeight(merged[idx].Priority) {
				merged[idx].Priority = item.Priority
			}
			found = true
			break
		}
		if !found {
			merged = append(merged, item)
		}
	}
	out := make([]model.ExtractedAction, len(merged))
	copy(out, merged)
	sort.Slice(out, func(i, j int) bool {
		if out[i].AssignedTo == out[j].AssignedTo {
			return out[i].Title < out[j].Title
		}
		return out[i].AssignedTo < out[j].AssignedTo
	})
	return out
}

func (e *ActionExtractor) normalize(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	input = e.nonWord.ReplaceAllString(input, " ")
	return strings.Join(strings.Fields(input), " ")
}

func (e *ActionExtractor) similarTask(a, b string) bool {
	left := e.normalize(a)
	right := e.normalize(b)
	if left == right || strings.Contains(left, right) || strings.Contains(right, left) {
		return true
	}
	leftTokens := strings.Fields(left)
	rightTokens := strings.Fields(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return false
	}
	tokenSet := make(map[string]struct{}, len(leftTokens))
	for _, token := range leftTokens {
		tokenSet[token] = struct{}{}
	}
	matches := 0
	for _, token := range rightTokens {
		if _, ok := tokenSet[token]; ok {
			matches++
		}
	}
	threshold := len(leftTokens)
	if len(rightTokens) > threshold {
		threshold = len(rightTokens)
	}
	return float64(matches)/float64(threshold) >= 0.6
}

func splitSentences(input string) []string {
	replacer := strings.NewReplacer("\r\n", "\n", "\n", ". ", ";", ". ")
	input = replacer.Replace(input)
	parts := strings.Split(input, ".")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func inferPriority(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "urgent"), strings.Contains(lower, "urgently"),
		strings.Contains(lower, "immediately"), strings.Contains(lower, "asap"),
		strings.Contains(lower, "critical"):
		return "high"
	case strings.Contains(lower, "when possible"), strings.Contains(lower, "at convenience"):
		return "low"
	default:
		return "medium"
	}
}

func priorityWeight(priority string) int {
	switch priority {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
