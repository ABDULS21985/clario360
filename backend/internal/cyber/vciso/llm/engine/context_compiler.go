package engine

import (
	"context"
	"strings"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

const (
	DefaultRecencyWindow = 10
	DefaultSummaryBudget = 256
)

type messageLister interface {
	ListMessages(ctx context.Context, tenantID, conversationID uuid.UUID) ([]chatmodel.Message, error)
}

type Summariser interface {
	Summarise(ctx context.Context, msgs []chatmodel.Message) (string, error)
}

type DefaultSummariser struct {
	MaxMessages int
	MaxCharsPer int
}

func (s *DefaultSummariser) Summarise(_ context.Context, msgs []chatmodel.Message) (string, error) {
	if len(msgs) == 0 {
		return "", nil
	}
	maxMessages := s.MaxMessages
	if maxMessages <= 0 {
		maxMessages = 4
	}
	maxChars := s.MaxCharsPer
	if maxChars <= 0 {
		maxChars = 120
	}
	if len(msgs) > maxMessages {
		msgs = msgs[len(msgs)-maxMessages:]
	}
	parts := make([]string, 0, len(msgs))
	for _, msg := range msgs {
		parts = append(parts, capitalise(string(msg.Role))+": "+truncateStr(strings.TrimSpace(msg.Content), maxChars))
	}
	return strings.Join(parts, " | "), nil
}

type CompiledContext struct {
	Messages      []llmmodel.LLMMessage
	ContextTurns  int
	WasTruncated  bool
	WasSummarised bool
	DroppedTurns  int
	TotalTokens   int
	SummaryTokens int
}

type ContextCompiler struct {
	conversationRepo messageLister
	budget           TokenBudget
	metrics          *Metrics
	recencyWindow    int
	summaryBudget    int
	summariser       Summariser
}

type ContextCompilerOption func(*ContextCompiler)

func WithRecencyWindow(n int) ContextCompilerOption {
	return func(c *ContextCompiler) {
		if n > 0 {
			c.recencyWindow = n
		}
	}
}

func WithSummaryBudget(n int) ContextCompilerOption {
	return func(c *ContextCompiler) {
		if n > 0 {
			c.summaryBudget = n
		}
	}
}

func WithSummariser(s Summariser) ContextCompilerOption {
	return func(c *ContextCompiler) {
		if s != nil {
			c.summariser = s
		}
	}
}

func NewContextCompiler(conversationRepo messageLister, budget TokenBudget, metrics *Metrics, opts ...ContextCompilerOption) *ContextCompiler {
	c := &ContextCompiler{
		conversationRepo: conversationRepo,
		budget:           budget,
		metrics:          metrics,
		recencyWindow:    DefaultRecencyWindow,
		summaryBudget:    DefaultSummaryBudget,
		summariser:       &DefaultSummariser{},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

func (c *ContextCompiler) Compile(ctx context.Context, conversationID *uuid.UUID, tenantID uuid.UUID) (*CompiledContext, error) {
	compiled := &CompiledContext{Messages: []llmmodel.LLMMessage{}}
	if c == nil || c.conversationRepo == nil || conversationID == nil || *conversationID == uuid.Nil {
		return compiled, nil
	}
	messages, err := c.conversationRepo.ListMessages(ctx, tenantID, *conversationID)
	if err != nil {
		return nil, err
	}
	filtered := filterConversationMessages(messages)
	if len(filtered) == 0 {
		return compiled, nil
	}

	older, recent := splitRecent(filtered, c.recencyWindow)
	compiled.ContextTurns = len(recent)
	compiled.DroppedTurns = len(older)

	if len(older) > 0 && c.summariser != nil {
		summary, summaryErr := c.summariser.Summarise(ctx, older)
		if summaryErr == nil && strings.TrimSpace(summary) != "" {
			summary, summaryTokens, summaryTruncated := enforceSummaryBudget(summary, c.summaryBudget)
			compiled.Messages = append(compiled.Messages, llmmodel.LLMMessage{
				Role:    "assistant",
				Content: "Conversation summary: " + summary,
			})
			compiled.WasSummarised = true
			compiled.WasTruncated = summaryTruncated
			compiled.SummaryTokens = summaryTokens
			compiled.TotalTokens += estimateTokens("Conversation summary: " + summary)
		}
	}

	// Compute how many tokens remain for recent messages after the summary.
	available := -1 // -1 = unlimited
	if c.budget.ConversationHistoryBudget > 0 {
		usedSegments := make([]string, 0, len(compiled.Messages))
		for _, msg := range compiled.Messages {
			usedSegments = append(usedSegments, msg.Content)
		}
		available = remainingTokens(c.budget.ConversationHistoryBudget, usedSegments...)
	}

	for idx, item := range recent {
		role := string(item.Role)
		messageTokens := estimateTokens(item.Content)
		if available >= 0 && messageTokens > available && len(compiled.Messages) > 0 {
			compiled.WasTruncated = true
			compiled.DroppedTurns += len(recent) - idx
			break
		}
		compiled.Messages = append(compiled.Messages, llmmodel.LLMMessage{
			Role:    role,
			Content: item.Content,
		})
		compiled.TotalTokens += messageTokens
		if available >= 0 {
			available -= messageTokens
		}
	}

	if c.metrics != nil && c.metrics.ContextTokensUsed != nil {
		c.metrics.ContextTokensUsed.Observe(float64(compiled.TotalTokens))
	}
	return compiled, nil
}

func filterConversationMessages(messages []chatmodel.Message) []chatmodel.Message {
	filtered := make([]chatmodel.Message, 0, len(messages))
	for _, item := range messages {
		if strings.EqualFold(string(item.Role), "system") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func splitRecent(messages []chatmodel.Message, recencyWindow int) ([]chatmodel.Message, []chatmodel.Message) {
	if recencyWindow <= 0 || len(messages) <= recencyWindow {
		return nil, messages
	}
	return messages[:len(messages)-recencyWindow], messages[len(messages)-recencyWindow:]
}

func enforceSummaryBudget(summary string, budget int) (string, int, bool) {
	if budget <= 0 {
		return summary, estimateTokens(summary), false
	}
	if estimateTokens(summary) <= budget {
		return summary, estimateTokens(summary), false
	}
	truncated := truncateToTokenBudget(summary, budget)
	return truncated, estimateTokens(truncated), true
}

func capitalise(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func truncateStr(value string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(value) <= maxLen {
		return value
	}
	if maxLen <= len("…") {
		return "…"
	}
	return value[:maxLen-len("…")] + "…"
}
