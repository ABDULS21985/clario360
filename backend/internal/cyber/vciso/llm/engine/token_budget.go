package engine

import (
	"strings"
	"unicode/utf8"
)

const (
	defaultSystemPromptBudget        = 1800
	defaultConversationHistoryBudget = 2400
	defaultToolResultMaxPerCall      = 1200
	defaultResponseMax               = 900

	minTokenBudget = 64
)

type TokenBudget struct {
	SystemPromptBudget        int `json:"system_prompt_budget"`
	ConversationHistoryBudget int `json:"conversation_history_budget"`
	ToolResultMaxPerCall      int `json:"tool_result_max_per_call"`
	ResponseMax               int `json:"response_max"`
}

func DefaultTokenBudget() TokenBudget {
	return TokenBudget{
		SystemPromptBudget:        defaultSystemPromptBudget,
		ConversationHistoryBudget: defaultConversationHistoryBudget,
		ToolResultMaxPerCall:      defaultToolResultMaxPerCall,
		ResponseMax:               defaultResponseMax,
	}
}

func (b TokenBudget) Normalize() TokenBudget {
	if b.SystemPromptBudget < minTokenBudget {
		b.SystemPromptBudget = defaultSystemPromptBudget
	}
	if b.ConversationHistoryBudget < minTokenBudget {
		b.ConversationHistoryBudget = defaultConversationHistoryBudget
	}
	if b.ToolResultMaxPerCall < minTokenBudget {
		b.ToolResultMaxPerCall = defaultToolResultMaxPerCall
	}
	if b.ResponseMax < minTokenBudget {
		b.ResponseMax = defaultResponseMax
	}
	return b
}

func (b TokenBudget) TotalInputBudget() int {
	b = b.Normalize()
	return b.SystemPromptBudget + b.ConversationHistoryBudget + b.ToolResultMaxPerCall
}

func (b TokenBudget) TotalBudget() int {
	b = b.Normalize()
	return b.SystemPromptBudget + b.ConversationHistoryBudget + b.ToolResultMaxPerCall + b.ResponseMax
}

func (b TokenBudget) FitsPrompt(systemPrompt, history string, toolResults ...string) bool {
	b = b.Normalize()

	if estimateTokens(systemPrompt) > b.SystemPromptBudget {
		return false
	}
	if estimateTokens(history) > b.ConversationHistoryBudget {
		return false
	}

	totalToolTokens := 0
	for _, item := range toolResults {
		totalToolTokens += estimateTokens(item)
	}
	if totalToolTokens > b.ToolResultMaxPerCall {
		return false
	}

	return true
}

func estimateTokens(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	words := len(strings.Fields(value))
	runes := utf8.RuneCountInString(value)

	// Hybrid heuristic:
	// - word count captures natural language fairly well
	// - rune/4 approximates tokenizer behavior for mixed prose / JSON / identifiers
	// Use the larger of the two to avoid undercounting.
	wordEstimate := words
	runeEstimate := (runes + 3) / 4

	if runeEstimate > wordEstimate {
		return runeEstimate
	}
	return wordEstimate
}

func remainingTokens(limit int, segments ...string) int {
	used := 0
	for _, segment := range segments {
		used += estimateTokens(segment)
	}
	if used >= limit {
		return 0
	}
	return limit - used
}

func truncateToTokenBudget(value string, budget int) string {
	value = strings.TrimSpace(value)
	if value == "" || budget <= 0 {
		return ""
	}
	if estimateTokens(value) <= budget {
		return value
	}

	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}

	lo, hi := 0, len(words)
	best := ""

	for lo <= hi {
		mid := (lo + hi) / 2
		candidate := strings.Join(words[:mid], " ")
		if mid < len(words) {
			candidate += "..."
		}

		if estimateTokens(candidate) <= budget {
			best = candidate
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	return best
}
