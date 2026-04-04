package jira

import (
	"strings"

	intmodel "github.com/clario360/platform/internal/integration/model"
)

func MapSeverityToPriority(config intmodel.JiraConfig, severity string) string {
	if mapped, ok := config.PriorityMapping[strings.ToLower(severity)]; ok && mapped != "" {
		return mapped
	}
	switch strings.ToLower(severity) {
	case "critical":
		return "Highest"
	case "high":
		return "High"
	case "medium":
		return "Medium"
	case "low":
		return "Low"
	default:
		return "Low"
	}
}

func MapJiraStatusToClario(config intmodel.JiraConfig, status string) string {
	for key, value := range config.StatusMapping {
		if strings.EqualFold(key, status) {
			return value
		}
	}
	switch strings.ToLower(status) {
	case "to do":
		return "new"
	case "in progress", "in review":
		return "investigating"
	case "done", "closed":
		return "resolved"
	case "won't do":
		return "false_positive"
	case "cancelled", "canceled":
		return "closed"
	default:
		return ""
	}
}
