package engine

import "strings"

type TokenBudget struct {
	SystemPromptBudget        int
	ConversationHistoryBudget int
	ToolResultMaxPerCall      int
	ResponseMax               int
}

func estimateTokens(value string) int {
	if strings.TrimSpace(value) == "" {
		return 0
	}
	return len(strings.Fields(value)) + len(value)/16
}
